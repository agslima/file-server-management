
package worker

import (
    "context"
    "fmt"
)

type FSProcessor struct{}

func (p *FSProcessor) Process(ctx context.Context, task *Task) error {
    switch task.Type {
    case "create_folder":
        fmt.Println("Creating folder:", task.Params["path"])
    case "complete_upload":
        fmt.Println("Completing upload:", task.Params["upload_id"])
    default:
        return fmt.Errorf("unknown task type: %s", task.Type)
    }
    return nil
}
