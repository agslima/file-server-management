package tasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/observability"
)

const (
	defaultStatusRetryAttempts = 3
	defaultStatusRetryDelay    = 25 * time.Millisecond
	defaultTaskProcessTimeout  = 30 * time.Second
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
		"request_id":     correlationID,
		"audit_message":  message,
	})
}

type Worker struct {
	q                     Queue
	p                     *Processor
	log                   *logger.Logger
	auditor               AuditEmitter
	failureAlertThreshold int64
	statusRetryAttempts   int
	statusRetryDelay      time.Duration
	taskProcessTimeout    time.Duration
}

func NewWorker(q Queue, p *Processor, log *logger.Logger) *Worker {
	return &Worker{
		q:                     q,
		p:                     p,
		log:                   log,
		auditor:               &logAuditEmitter{log: log},
		failureAlertThreshold: failureThresholdFromEnv(),
		statusRetryAttempts:   statusRetryAttemptsFromEnv(),
		statusRetryDelay:      statusRetryDelayFromEnv(),
		taskProcessTimeout:    taskProcessTimeoutFromEnv(),
	}
}

func NewWorkerWithAudit(q Queue, p *Processor, log *logger.Logger, auditor AuditEmitter) *Worker {
	if auditor == nil {
		auditor = &logAuditEmitter{log: log}
	}
	return &Worker{
		q:                     q,
		p:                     p,
		log:                   log,
		auditor:               auditor,
		failureAlertThreshold: failureThresholdFromEnv(),
		statusRetryAttempts:   statusRetryAttemptsFromEnv(),
		statusRetryDelay:      statusRetryDelayFromEnv(),
		taskProcessTimeout:    taskProcessTimeoutFromEnv(),
	}
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
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				w.log.Info("worker context done")
				return
			}
			if !w.sleepWithContext(ctx, 1*time.Second) {
				w.log.Info("worker context done")
				return
			}
			continue
		}
		correlationID := task.Params["correlation_id"]
		if correlationID == "" {
			correlationID = task.Params["request_id"]
		}
		w.auditor.EmitTaskEvent(ctx, "task.dequeued", task.ID, correlationID, fmt.Sprintf("task_type=%s", task.Type))
		w.auditor.EmitTaskEvent(ctx, "task.processing", task.ID, correlationID, fmt.Sprintf("task_type=%s", task.Type))
		w.log.Event("info", "worker task processing", map[string]any{
			"event":          "task.processing",
			"task_id":        task.ID,
			"correlation_id": correlationID,
			"request_id":     correlationID,
			"task_type":      task.Type,
		})
		if err := w.persistStatus(ctx, task.ID, "running", correlationID, "task is running"); err != nil {
			w.log.Event("warn", "task status update failed", map[string]any{"event": "task.status_update_failed", "task_id": task.ID, "correlation_id": correlationID, "request_id": correlationID, "error": err.Error()})
		}

		processCtx := ctx
		cancel := func() {}
		if w.taskProcessTimeout > 0 {
			processCtx, cancel = context.WithTimeout(ctx, w.taskProcessTimeout)
		}
		err = w.p.Process(processCtx, task)
		cancel()

		if err != nil {
			msg := err.Error()
			if errors.Is(err, context.DeadlineExceeded) {
				msg = fmt.Sprintf("task processing timed out after %s", w.taskProcessTimeout)
			}
			w.log.Event("warn", "worker task failed", map[string]any{"event": "task.failed", "task_id": task.ID, "correlation_id": correlationID, "request_id": correlationID, "error": msg})
			if completeErr := w.persistStatus(ctx, task.ID, "failed", correlationID, msg); completeErr != nil {
				w.log.Event("warn", "task status update failed", map[string]any{"event": "task.status_update_failed", "task_id": task.ID, "correlation_id": correlationID, "request_id": correlationID, "error": completeErr.Error()})
			}
			w.auditor.EmitTaskEvent(ctx, "task.failed", task.ID, correlationID, msg)
			if task.Type == "create_folder" {
				w.auditor.EmitTaskEvent(ctx, "folder.mutation.failed", task.ID, correlationID, msg)
			}
			w.maybeAlertOnFailures(correlationID)
		} else {
			w.log.Event("info", "worker task succeeded", map[string]any{"event": "task.succeeded", "task_id": task.ID, "correlation_id": correlationID, "request_id": correlationID})
			if completeErr := w.persistStatus(ctx, task.ID, "success", correlationID, "folder created"); completeErr != nil {
				w.log.Event("warn", "task status update failed", map[string]any{"event": "task.status_update_failed", "task_id": task.ID, "correlation_id": correlationID, "request_id": correlationID, "error": completeErr.Error()})
			}
			w.auditor.EmitTaskEvent(ctx, "task.succeeded", task.ID, correlationID, "task completed")
			if task.Type == "create_folder" {
				w.auditor.EmitTaskEvent(ctx, "folder.mutation.succeeded", task.ID, correlationID, "folder created")
			}
		}
	}
}

func (w *Worker) persistStatus(ctx context.Context, id, status, correlationID, message string) error {
	attempts := w.statusRetryAttempts
	if attempts < 1 {
		attempts = 1
	}
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		err = w.q.Complete(ctx, id, status, correlationID, message)
		if err == nil {
			return nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if attempt < attempts && !w.sleepWithContext(ctx, w.statusRetryDelay) {
			return context.Canceled
		}
	}
	return err
}

func (w *Worker) sleepWithContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
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
			"request_id":      correlationID,
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

func statusRetryAttemptsFromEnv() int {
	attempts := defaultStatusRetryAttempts
	if raw := os.Getenv("WORKER_STATUS_RETRY_ATTEMPTS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			attempts = parsed
		}
	}
	return attempts
}

func statusRetryDelayFromEnv() time.Duration {
	delay := defaultStatusRetryDelay
	if raw := os.Getenv("WORKER_STATUS_RETRY_DELAY_MS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			delay = time.Duration(parsed) * time.Millisecond
		}
	}
	return delay
}

func taskProcessTimeoutFromEnv() time.Duration {
	timeout := defaultTaskProcessTimeout
	if raw := os.Getenv("WORKER_TASK_PROCESS_TIMEOUT_MS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			if parsed <= 0 {
				return 0
			}
			timeout = time.Duration(parsed) * time.Millisecond
		}
	}
	return timeout
}
