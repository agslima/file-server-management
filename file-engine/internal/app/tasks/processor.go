package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/storage"
)

type MutationExecutor interface {
	MoveObject(ctx context.Context, actorID, sourcePath, destinationPath string) error
	DeleteObject(ctx context.Context, actorID, objectPath string) error
	RestoreQuarantinedObject(ctx context.Context, actorID, objectPath string, forceReprocess bool) error
}

type Processor struct {
	st       storage.Storage
	executor MutationExecutor
	mu       sync.Mutex
	seenKeys map[string]struct{}
}

func NewProcessorWithStorage(st storage.Storage) *Processor {
	return &Processor{st: st, seenKeys: map[string]struct{}{}}
}

func NewProcessorWithMutationExecutor(st storage.Storage, executor MutationExecutor) *Processor {
	return &Processor{st: st, executor: executor, seenKeys: map[string]struct{}{}}
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
	switch t.Type {
	case "create_folder":
		parent := t.Params["parent"]
		name := t.Params["name"]
		if parent == "" || name == "" {
			return errors.New("missing params")
		}
		parent = strings.TrimSuffix(parent, "/")
		return p.st.CreateFolder(ctx, parent+"/"+name)
	case "move_file":
		src := t.Params["src"]
		dst := t.Params["dst"]
		if src == "" || dst == "" {
			return errors.New("missing params")
		}
		if p.executor != nil {
			return p.executor.MoveObject(ctx, actorID, src, dst)
		}
		return p.st.Move(ctx, src, dst)
	case "governed_delete":
		path := t.Params["path"]
		if path == "" {
			return errors.New("missing params")
		}
		if p.executor == nil {
			return errors.New("governed delete requires mutation executor")
		}
		return p.executor.DeleteObject(ctx, actorID, path)
	case "quarantine_restore":
		path := t.Params["path"]
		if path == "" {
			return errors.New("missing params")
		}
		if p.executor == nil {
			return errors.New("quarantine restore requires mutation executor")
		}
		return p.executor.RestoreQuarantinedObject(ctx, actorID, path, strings.EqualFold(t.Params["force_reprocess"], "true"))
	default:
		return fmt.Errorf("unknown task type %s", t.Type)
	}
}

func (p *Processor) isDuplicate(t *redisq.TaskPayload) bool {
	key := strings.TrimSpace(t.Params["idempotency_key"])
	if key == "" {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.seenKeys[key]; ok {
		return true
	}
	p.seenKeys[key] = struct{}{}
	return false
}
