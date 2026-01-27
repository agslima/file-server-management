package tasks

import (
    "context"
    "fmt"
    "strings"

    "github.com/example/file-engine/internal/adapters/queue/redisq"
    "github.com/example/file-engine/internal/storage"
)

type Processor struct {
    st storage.Storage
}

func NewProcessorWithStorage(st storage.Storage) *Processor {
    return &Processor{st: st}
}

// NewProcessor kept for compatibility with older wiring (local FS).
// It is intentionally removed in this version to enforce the unified storage interface.
// If you need to keep it, create a local storage adapter and call NewProcessorWithStorage.
func NewProcessor(_ any) *Processor {
    panic("use NewProcessorWithStorage")
}

func (p *Processor) Process(ctx context.Context, t *redisq.TaskPayload) error {
    switch t.Type {
    case "create_folder":
        parent := t.Params["parent"]
        name := t.Params["name"]
        if parent == "" || name == "" {
            return fmt.Errorf("missing params")
        }
        parent = strings.TrimSuffix(parent, "/")
        return p.st.CreateFolder(ctx, parent+"/"+name)
    case "move_file":
        src := t.Params["src"]
        dst := t.Params["dst"]
        if src == "" || dst == "" {
            return fmt.Errorf("missing params")
        }
        return p.st.Move(ctx, src, dst)
    default:
        return fmt.Errorf("unknown task type %s", t.Type)
    }
}
