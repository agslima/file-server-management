package tasks

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/logger"
)

type Queue interface {
	Pop(ctx context.Context) (*redisq.TaskPayload, error)
	Complete(ctx context.Context, id, status, correlationID, message string) error
}

type AuditEmitter interface {
	EmitTaskEvent(ctx context.Context, event, taskID, correlationID, message string)
}

type logAuditEmitter struct {
	log *logger.Logger
}

func (l *logAuditEmitter) EmitTaskEvent(_ context.Context, event, taskID, correlationID, message string) {
	l.log.Infof("audit_event=%s task_id=%s correlation_id=%s message=%q", event, taskID, correlationID, message)
}

type Worker struct {
	q       Queue
	p       *Processor
	log     *logger.Logger
	auditor AuditEmitter
}

func NewWorker(q Queue, p *Processor, log *logger.Logger) *Worker {
	return &Worker{q: q, p: p, log: log, auditor: &logAuditEmitter{log: log}}
}

func NewWorkerWithAudit(q Queue, p *Processor, log *logger.Logger, auditor AuditEmitter) *Worker {
	if auditor == nil {
		auditor = &logAuditEmitter{log: log}
	}
	return &Worker{q: q, p: p, log: log, auditor: auditor}
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
		correlationID := task.Params["correlation_id"]
		w.auditor.EmitTaskEvent(ctx, "task.processing", task.ID, correlationID, fmt.Sprintf("task_type=%s", task.Type))
		log.Printf("task_id=%s correlation_id=%s stage=processing type=%s", task.ID, correlationID, task.Type)

		if err := w.p.Process(ctx, task); err != nil {
			msg := err.Error()
			log.Printf("task_id=%s correlation_id=%s stage=failed error=%q", task.ID, correlationID, msg)
			_ = w.q.Complete(ctx, task.ID, "failed", correlationID, msg)
			w.auditor.EmitTaskEvent(ctx, "task.failed", task.ID, correlationID, msg)
		} else {
			log.Printf("task_id=%s correlation_id=%s stage=success", task.ID, correlationID)
			_ = w.q.Complete(ctx, task.ID, "success", correlationID, "folder created")
			w.auditor.EmitTaskEvent(ctx, "task.succeeded", task.ID, correlationID, "task completed")
		}
	}
}
