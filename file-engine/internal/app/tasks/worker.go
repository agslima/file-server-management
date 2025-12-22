package tasks

import (
    "context"
    "log"
    "time"

    "github.com/agslima/file-server-management/file-engine/internal/adapters/queue/redisq"
    "github.com/example/file-engine/internal/logger"
)

type Worker struct {
    q *redisq.RedisQueue
    p *Processor
    log *logger.Logger
}

func NewWorker(q *redisq.RedisQueue, p *Processor, log *logger.Logger) *Worker {
    return &Worker{q: q, p: p, log: log}
}

func (w *Worker) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            w.log.Info("worker context done")
            return
        default:
        }
        task, err := w.q.Pop(ctx)
        if err != nil {
            time.Sleep(1 * time.Second)
            continue
        }
        log.Printf("processing task %s type=%s", task.ID, task.Type)
        if err := w.p.Process(ctx, task); err != nil {
            log.Printf("task %s failed: %v", task.ID, err)
            _ = w.q.Complete(ctx, task.ID, "failed")
        } else {
            _ = w.q.Complete(ctx, task.ID, "success")
        }
    }
}
