package tasks

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/storage"
)

type workerTestQueue struct {
	task             *redisq.TaskPayload
	completeCalls    map[string]int
	failuresByStatus map[string]int
	statuses         []redisq.TaskStatus
	terminalStatusCh chan redisq.TaskStatus
	mu               sync.Mutex
}

func newWorkerTestQueue(task *redisq.TaskPayload) *workerTestQueue {
	return &workerTestQueue{
		task:             task,
		completeCalls:    map[string]int{},
		failuresByStatus: map[string]int{},
		terminalStatusCh: make(chan redisq.TaskStatus, 1),
	}
}

func (q *workerTestQueue) Pop(ctx context.Context) (*redisq.TaskPayload, error) {
	if q.task != nil {
		t := q.task
		q.task = nil
		return t, nil
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func (q *workerTestQueue) Complete(_ context.Context, id, status, correlationID, message string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.completeCalls[status]++
	if pending := q.failuresByStatus[status]; pending > 0 {
		q.failuresByStatus[status] = pending - 1
		return errors.New("transient queue write error")
	}
	current := redisq.TaskStatus{TaskID: id, Status: status, CorrelationID: correlationID, Message: message}
	q.statuses = append(q.statuses, current)
	if status == "success" || status == "failed" {
		select {
		case q.terminalStatusCh <- current:
		default:
		}
	}
	return nil
}

func (q *workerTestQueue) setStatusFailures(status string, failures int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.failuresByStatus[status] = failures
}

func (q *workerTestQueue) completeCallsFor(status string) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.completeCalls[status]
}

type workerTestStorage struct {
	processDelay time.Duration
}

func (s *workerTestStorage) CreateFolder(ctx context.Context, path string) error {
	if s.processDelay <= 0 {
		return nil
	}
	timer := time.NewTimer(s.processDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *workerTestStorage) AtomicWrite(context.Context, string, io.Reader) error { return nil }
func (s *workerTestStorage) Move(context.Context, string, string) error           { return nil }
func (s *workerTestStorage) Delete(context.Context, string) error                 { return nil }
func (s *workerTestStorage) Exists(context.Context, string) (bool, error)         { return false, nil }
func (s *workerTestStorage) List(context.Context, string) ([]storage.ObjectInfo, error) {
	return nil, nil
}
func (s *workerTestStorage) Open(context.Context, string) (io.ReadCloser, error) { return nil, nil }

func TestWorkerRetriesStatusPersistence(t *testing.T) {
	t.Setenv("WORKER_STATUS_RETRY_ATTEMPTS", "3")
	t.Setenv("WORKER_STATUS_RETRY_DELAY_MS", "1")
	t.Setenv("WORKER_TASK_PROCESS_TIMEOUT_MS", "1000")

	q := newWorkerTestQueue(&redisq.TaskPayload{
		ID:   "task-retry",
		Type: "create_folder",
		Params: map[string]string{
			"parent":         "tenants/acme/projects",
			"name":           "guardrail",
			"correlation_id": "req-retry-1",
		},
	})
	q.setStatusFailures("running", 2)

	worker := NewWorkerWithAudit(q, NewProcessorWithStorage(&workerTestStorage{}), logger.New("debug"), nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Start(ctx)

	select {
	case terminal := <-q.terminalStatusCh:
		if terminal.Status != "success" {
			t.Fatalf("expected success terminal status, got %+v", terminal)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for worker terminal status")
	}

	if attempts := q.completeCallsFor("running"); attempts != 3 {
		t.Fatalf("expected 3 attempts for running status persistence, got %d", attempts)
	}
}

func TestWorkerMarksTaskFailedOnProcessingTimeout(t *testing.T) {
	t.Setenv("WORKER_TASK_PROCESS_TIMEOUT_MS", "20")
	t.Setenv("WORKER_STATUS_RETRY_ATTEMPTS", "1")
	t.Setenv("WORKER_STATUS_RETRY_DELAY_MS", "0")

	q := newWorkerTestQueue(&redisq.TaskPayload{
		ID:   "task-timeout",
		Type: "create_folder",
		Params: map[string]string{
			"parent":         "tenants/acme/projects",
			"name":           "slow-folder",
			"correlation_id": "req-timeout-1",
		},
	})

	worker := NewWorkerWithAudit(q, NewProcessorWithStorage(&workerTestStorage{processDelay: 250 * time.Millisecond}), logger.New("debug"), nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.Start(ctx)

	select {
	case terminal := <-q.terminalStatusCh:
		if terminal.Status != "failed" {
			t.Fatalf("expected failed terminal status due to timeout, got %+v", terminal)
		}
		if !strings.Contains(terminal.Message, "timed out") {
			t.Fatalf("expected timeout message, got %q", terminal.Message)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for worker terminal status")
	}
}
