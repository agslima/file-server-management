package tasks

import (
    "context"
    "fmt"

    "github.com/example/file-engine/internal/adapters/queue/redisq"
    "github.com/example/file-engine/internal/adapters/fs/local"
)

type Processor struct {
    fs *local.LocalFs
}

func NewProcessor(fs *local.LocalFs) *Processor {
    return &Processor{fs: fs}
}

func (p *Processor) Process(ctx context.Context, t *redisq.TaskPayload) error {
    switch t.Type {
    case "create_folder":
        parent := t.Params["parent"]
        name := t.Params["name"]
        if parent == "" || name == "" {
            return fmt.Errorf("missing params")
        }
        return p.fs.CreateFolder(ctx, parent, name)
    case "move_file":
        src := []string{t.Params["src"]}
        dst := []string{t.Params["dst"]}
        return p.fs.MoveUploadedFile(ctx, src, dst)
    default:
        return fmt.Errorf("unknown task type %s", t.Type)
    }
}