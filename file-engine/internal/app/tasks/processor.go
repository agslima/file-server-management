package tasks

import (
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
	seenKeys          map[string]time.Time
	seenKeyTTL        time.Duration
	seenKeyMaxEntries int
	now               func() time.Time
}

func NewProcessorWithStorage(st storage.Storage) *Processor {
	return &Processor{
		st:                st,
		seenKeys:          map[string]time.Time{},
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
	if p.isDuplicate(t) {
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
		if parent == "" || name == "" {
			err = errors.New("missing params")
			break
		}
		parent = strings.TrimSuffix(parent, "/")
		err = p.st.CreateFolder(ctx, parent+"/"+name)
	case "move_file":
		src := t.Params["src"]
		dst := t.Params["dst"]
		if src == "" || dst == "" {
			err = errors.New("missing params")
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
			err = errors.New("missing params")
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
			err = errors.New("missing params")
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
	}
	return err
}

func (p *Processor) isDuplicate(t *redisq.TaskPayload) bool {
	key := strings.TrimSpace(t.Params["idempotency_key"])
	if key == "" {
		return false
	}

	now := p.now()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.evictExpiredLocked(now)

	expiresAt, ok := p.seenKeys[key]
	if !ok {
		return false
	}
	if now.After(expiresAt) {
		delete(p.seenKeys, key)
		return false
	}
	return true
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
	p.seenKeys[key] = now.Add(p.seenKeyTTL)
	p.evictOverflowLocked()
}

func (p *Processor) evictExpiredLocked(now time.Time) {
	for key, expiresAt := range p.seenKeys {
		if now.After(expiresAt) {
			delete(p.seenKeys, key)
		}
	}
}

func (p *Processor) evictOverflowLocked() {
	if p.seenKeyMaxEntries <= 0 {
		return
	}
	for len(p.seenKeys) > p.seenKeyMaxEntries {
		oldestKey := ""
		var oldestExpiry time.Time
		for key, expiresAt := range p.seenKeys {
			if oldestKey == "" || expiresAt.Before(oldestExpiry) {
				oldestKey = key
				oldestExpiry = expiresAt
			}
		}
		if oldestKey == "" {
			return
		}
		delete(p.seenKeys, oldestKey)
	}
}
