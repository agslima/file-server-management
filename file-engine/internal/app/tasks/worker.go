package tasks

import (
	"context"
	"log"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/logger"
)

type Queue interface {
	Pop(ctx context.Context) (*redisq.TaskPayload, error)
	Complete(ctx context.Context, id, status string) error
}

type Worker struct {
	q   Queue
	p   *Processor
	log *logger.Logger
}

func NewWorker(q Queue, p *Processor, log *logger.Logger) *Worker {
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
