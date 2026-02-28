package tasks

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/storage"
)

type MutationExecutor interface {
	MoveObject(ctx context.Context, actorID, sourcePath, destinationPath string) error
	DeleteObject(ctx context.Context, actorID, objectPath string) error
	RestoreQuarantinedObject(ctx context.Context, actorID, objectPath string, forceReprocess bool) error
}

const (
	defaultSeenKeyTTL        = time.Hour
	defaultSeenKeyMaxEntries = 10_000
)

type Processor struct {
	st                storage.Storage
	executor          MutationExecutor
	mu                sync.Mutex
	seenKeys          map[string]*seenKeyEntry
	seenKeyExpiries   seenKeyExpiryHeap
	seenKeyTTL        time.Duration
	seenKeyMaxEntries int
	now               func() time.Time
}

type seenKeyEntry struct {
	key       string
	expiresAt time.Time
	inFlight  bool
	index     int
}

type seenKeyExpiryHeap []*seenKeyEntry

func (h seenKeyExpiryHeap) Len() int { return len(h) }

func (h seenKeyExpiryHeap) Less(i, j int) bool {
	return h[i].expiresAt.Before(h[j].expiresAt)
}

func (h seenKeyExpiryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *seenKeyExpiryHeap) Push(x any) {
	entry := x.(*seenKeyEntry)
	entry.index = len(*h)
	*h = append(*h, entry)
}

func (h *seenKeyExpiryHeap) Pop() any {
	old := *h
	n := len(old)
	entry := old[n-1]
	entry.index = -1
	*h = old[:n-1]
	return entry
}

func NewProcessorWithStorage(st storage.Storage) *Processor {
	return &Processor{
		st:                st,
		seenKeys:          map[string]*seenKeyEntry{},
		seenKeyExpiries:   seenKeyExpiryHeap{},
		seenKeyTTL:        defaultSeenKeyTTL,
		seenKeyMaxEntries: defaultSeenKeyMaxEntries,
		now:               time.Now,
	}
}

func NewProcessorWithMutationExecutor(st storage.Storage, executor MutationExecutor) *Processor {
	p := NewProcessorWithStorage(st)
	p.executor = executor
	return p
}

// NewProcessor kept for compatibility with older wiring (local FS).
// It is intentionally removed in this version to enforce the unified storage interface.
// If need to keep it, create a local storage adapter and call NewProcessorWithStorage.
func NewProcessor(_ any) *Processor {
	panic("use NewProcessorWithStorage")
}

func (p *Processor) Process(ctx context.Context, t *redisq.TaskPayload) error {
	reservation := p.reserveIdempotencyKey(t)
	if reservation == reservationDuplicate || reservation == reservationInFlight {
		return nil
	}
	actorID := strings.TrimSpace(t.Params["actor_id"])
	if actorID == "" {
		actorID = strings.TrimSpace(t.Params["by"])
	}

	var err error
	switch t.Type {
	case "create_folder":
		parent := t.Params["parent"]
		name := t.Params["name"]
		missing := make([]string, 0, 2)
		if parent == "" {
			missing = append(missing, "parent")
		}
		if name == "" {
			missing = append(missing, "name")
		}
		if len(missing) > 0 {
			err = missingParamsError(missing...)
			break
		}
		parent = strings.TrimSuffix(parent, "/")
		err = p.st.CreateFolder(ctx, parent+"/"+name)
	case "move_file":
		src := t.Params["src"]
		dst := t.Params["dst"]
		missing := make([]string, 0, 2)
		if src == "" {
			missing = append(missing, "src")
		}
		if dst == "" {
			missing = append(missing, "dst")
		}
		if len(missing) > 0 {
			err = missingParamsError(missing...)
			break
		}
		if p.executor != nil {
			err = p.executor.MoveObject(ctx, actorID, src, dst)
			break
		}
		err = p.st.Move(ctx, src, dst)
	case "governed_delete":
		path := t.Params["path"]
		if path == "" {
			err = missingParamsError("path")
			break
		}
		if p.executor == nil {
			err = errors.New("governed delete requires mutation executor")
			break
		}
		err = p.executor.DeleteObject(ctx, actorID, path)
	case "quarantine_restore":
		path := t.Params["path"]
		if path == "" {
			err = missingParamsError("path")
			break
		}
		if p.executor == nil {
			err = errors.New("quarantine restore requires mutation executor")
			break
		}
		err = p.executor.RestoreQuarantinedObject(ctx, actorID, path, strings.EqualFold(t.Params["force_reprocess"], "true"))
	default:
		err = fmt.Errorf("unknown task type %s", t.Type)
	}

	if err == nil {
		p.markSeen(t)
		return nil
	}
	if reservation == reservationReserved {
		p.releaseReservation(t)
	}
	return fmt.Errorf("task processing failed type=%s id=%s actor_id=%s: %w", t.Type, t.ID, actorID, err)
}

type idempotencyReservationState int

const (
	reservationNotApplicable idempotencyReservationState = iota
	reservationReserved
	reservationDuplicate
	reservationInFlight
)

func (p *Processor) reserveIdempotencyKey(t *redisq.TaskPayload) idempotencyReservationState {
	key := strings.TrimSpace(t.Params["idempotency_key"])
	if key == "" {
		return reservationNotApplicable
	}

	now := p.now()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.evictExpiredLocked(now)

	if entry, ok := p.seenKeys[key]; ok {
		if now.After(entry.expiresAt) {
			delete(p.seenKeys, key)
		} else if entry.inFlight {
			return reservationInFlight
		} else {
			return reservationDuplicate
		}
	}

	entry := &seenKeyEntry{
		key:       key,
		expiresAt: now.Add(p.seenKeyTTL),
		inFlight:  true,
	}
	p.seenKeys[key] = entry
	heap.Push(&p.seenKeyExpiries, entry)
	p.evictOverflowLocked()
	return reservationReserved
}

func (p *Processor) releaseReservation(t *redisq.TaskPayload) {
	key := strings.TrimSpace(t.Params["idempotency_key"])
	if key == "" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.seenKeys, key)
}

func (p *Processor) markSeen(t *redisq.TaskPayload) {
	key := strings.TrimSpace(t.Params["idempotency_key"])
	if key == "" {
		return
	}

	now := p.now()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.evictExpiredLocked(now)

	entry, ok := p.seenKeys[key]
	if !ok {
		entry = &seenKeyEntry{key: key, index: -1}
		p.seenKeys[key] = entry
	}
	entry.inFlight = false
	entry.expiresAt = now.Add(p.seenKeyTTL)
	if entry.index >= 0 && entry.index < len(p.seenKeyExpiries) {
		heap.Fix(&p.seenKeyExpiries, entry.index)
	} else {
		heap.Push(&p.seenKeyExpiries, entry)
	}
	p.evictOverflowLocked()
}

func missingParamsError(params ...string) error {
	if len(params) == 1 {
		return fmt.Errorf("missing param: %s", params[0])
	}
	return fmt.Errorf("missing params: %s", strings.Join(params, ","))
}

func (p *Processor) evictExpiredLocked(now time.Time) {
	for len(p.seenKeyExpiries) > 0 {
		oldest := p.seenKeyExpiries[0]
		if !now.After(oldest.expiresAt) {
			return
		}
		heap.Pop(&p.seenKeyExpiries)
		if current, ok := p.seenKeys[oldest.key]; ok && current == oldest {
			delete(p.seenKeys, oldest.key)
		}
	}
}

func (p *Processor) evictOverflowLocked() {
	if p.seenKeyMaxEntries <= 0 {
		return
	}
	for len(p.seenKeys) > p.seenKeyMaxEntries {
		if !p.evictOldestActiveKeyLocked() {
			return
		}
	}
}

func (p *Processor) evictOldestActiveKeyLocked() bool {
	for len(p.seenKeyExpiries) > 0 {
		entry := heap.Pop(&p.seenKeyExpiries).(*seenKeyEntry)
		if current, ok := p.seenKeys[entry.key]; ok && current == entry {
			delete(p.seenKeys, entry.key)
			return true
		}
	}
	return false
}
