
package worker

import (
    "context"
    "fmt"
    "time"
)

type Task struct {
    ID     string
    Type   string
    Params map[string]string
}

type Queue interface {
    Pop(ctx context.Context) (*Task, error)
    Complete(ctx context.Context, id string, status string) error
}

type Processor interface {
    Process(ctx context.Context, task *Task) error
}

type Worker struct {
    Queue     Queue
    Processor Processor
}

func (w *Worker) Start(ctx context.Context) {
    for {
        task, err := w.Queue.Pop(ctx)
        if err != nil {
            time.Sleep(1 * time.Second)
            continue
        }
        if task == nil {
            continue
        }
        err = w.Processor.Process(ctx, task)
        status := "success"
        if err != nil {
            status = "error: " + err.Error()
        }
        w.Queue.Complete(ctx, task.ID, status)
        fmt.Println("processed task", task.ID, status)
    }
}
