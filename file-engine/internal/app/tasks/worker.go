package tasks

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/observability"
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
	l.log.Event("info", "task audit event", map[string]any{
		"event":          event,
		"task_id":        taskID,
		"correlation_id": correlationID,
		"audit_message":  message,
	})
}

type Worker struct {
	q                     Queue
	p                     *Processor
	log                   *logger.Logger
	auditor               AuditEmitter
	failureAlertThreshold int64
}

func NewWorker(q Queue, p *Processor, log *logger.Logger) *Worker {
	return &Worker{q: q, p: p, log: log, auditor: &logAuditEmitter{log: log}, failureAlertThreshold: failureThresholdFromEnv()}
}

func NewWorkerWithAudit(q Queue, p *Processor, log *logger.Logger, auditor AuditEmitter) *Worker {
	if auditor == nil {
		auditor = &logAuditEmitter{log: log}
	}
	return &Worker{q: q, p: p, log: log, auditor: auditor, failureAlertThreshold: failureThresholdFromEnv()}
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
		w.log.Event("info", "worker task processing", map[string]any{
			"event":          "task.processing",
			"task_id":        task.ID,
			"correlation_id": correlationID,
			"task_type":      task.Type,
		})
		if err := w.q.Complete(ctx, task.ID, "running", correlationID, "task is running"); err != nil {
			w.log.Event("warn", "task status update failed", map[string]any{"event": "task.status_update_failed", "task_id": task.ID, "correlation_id": correlationID, "error": err.Error()})
		}

		if err := w.p.Process(ctx, task); err != nil {
			msg := err.Error()
			w.log.Event("warn", "worker task failed", map[string]any{"event": "task.failed", "task_id": task.ID, "correlation_id": correlationID, "error": msg})
			if completeErr := w.q.Complete(ctx, task.ID, "failed", correlationID, msg); completeErr != nil {
				w.log.Event("warn", "task status update failed", map[string]any{"event": "task.status_update_failed", "task_id": task.ID, "correlation_id": correlationID, "error": completeErr.Error()})
			}
			w.auditor.EmitTaskEvent(ctx, "task.failed", task.ID, correlationID, msg)
			w.maybeAlertOnFailures(correlationID)
		} else {
			w.log.Event("info", "worker task succeeded", map[string]any{"event": "task.succeeded", "task_id": task.ID, "correlation_id": correlationID})
			if completeErr := w.q.Complete(ctx, task.ID, "success", correlationID, "folder created"); completeErr != nil {
				w.log.Event("warn", "task status update failed", map[string]any{"event": "task.status_update_failed", "task_id": task.ID, "correlation_id": correlationID, "error": completeErr.Error()})
			}
			w.auditor.EmitTaskEvent(ctx, "task.succeeded", task.ID, correlationID, "task completed")
		}
	}
}

func (w *Worker) maybeAlertOnFailures(correlationID string) {
	if w.failureAlertThreshold <= 0 {
		return
	}
	total := observability.DefaultMetrics.FailedTasksTotal()
	if total > 0 && total%w.failureAlertThreshold == 0 {
		w.log.Event("warn", "task failure alert threshold reached", map[string]any{
			"event":           "task.failure.alert",
			"correlation_id":  correlationID,
			"failed_tasks":    total,
			"alert_threshold": w.failureAlertThreshold,
		})
	}
}

func failureThresholdFromEnv() int64 {
	threshold := int64(5)
	if raw := os.Getenv("ALERT_TASK_FAILURE_THRESHOLD"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			threshold = parsed
		}
	}
	return threshold
}
