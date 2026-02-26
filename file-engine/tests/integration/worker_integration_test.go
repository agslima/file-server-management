package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/app/tasks"
	"github.com/example/file-engine/internal/logger"
)

type inMemoryQueue struct {
	ch          chan *redisq.TaskPayload
	statuses    map[string]redisq.TaskStatus
	transitions map[string][]string
	mu          sync.RWMutex
}

func newInMemoryQueue() *inMemoryQueue {
	return &inMemoryQueue{
		ch:          make(chan *redisq.TaskPayload, 1),
		statuses:    map[string]redisq.TaskStatus{},
		transitions: map[string][]string{},
	}
}

func (q *inMemoryQueue) Pop(ctx context.Context) (*redisq.TaskPayload, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case task := <-q.ch:
		return task, nil
	}
}

func (q *inMemoryQueue) Complete(_ context.Context, id, status, correlationID, message string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.statuses[id] = redisq.TaskStatus{TaskID: id, Status: status, CorrelationID: correlationID, Message: message}
	q.transitions[id] = append(q.transitions[id], status)
	return nil
}

func (q *inMemoryQueue) EnqueueCreateFolder(_ context.Context, parentPath, folderName, requestedBy, correlationID string) (string, error) {
	id := fmt.Sprintf("task-%d", time.Now().UnixNano())
	q.mu.Lock()
	q.statuses[id] = redisq.TaskStatus{TaskID: id, Status: "queued", CorrelationID: correlationID, Message: "task accepted"}
	q.transitions[id] = append(q.transitions[id], "queued")
	q.mu.Unlock()

	q.ch <- &redisq.TaskPayload{
		ID:   id,
		Type: "create_folder",
		Params: map[string]string{
			"parent":         parentPath,
			"name":           folderName,
			"by":             requestedBy,
			"correlation_id": correlationID,
		},
	}
	return id, nil
}

func (q *inMemoryQueue) Status(id string) (redisq.TaskStatus, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	status, ok := q.statuses[id]
	return status, ok
}

func (q *inMemoryQueue) TransitionHistory(id string) []string {
	q.mu.RLock()
	defer q.mu.RUnlock()
	h := q.transitions[id]
	cpy := make([]string, len(h))
	copy(cpy, h)
	return cpy
}

type inMemoryAuditor struct {
	mu     sync.Mutex
	events []auditEvent
}

type auditEvent struct {
	event         string
	taskID        string
	correlationID string
}

func (a *inMemoryAuditor) EmitTaskEvent(_ context.Context, event, taskID, correlationID, _ string, _ ...map[string]string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, auditEvent{event: event, taskID: taskID, correlationID: correlationID})
}

func (a *inMemoryAuditor) Snapshot() []auditEvent {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]auditEvent, len(a.events))
	copy(out, a.events)
	return out
}

// TestAsyncCreateFolderFlow validates one end-to-end async folder flow:
// enqueue -> worker process -> status success -> folder created.
func TestAsyncCreateFolderFlow(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	queue := newInMemoryQueue()
	auditor := &inMemoryAuditor{}
	processor := tasks.NewProcessorWithStorage(localstorage.New(rootDir))
	worker := tasks.NewWorkerWithAudit(queue, processor, logger.New("debug"), auditor)

	ctx := t.Context()

	go worker.Start(ctx)

	const parentPath = "tenants/acme/projects"
	const folderName = "week2-e2e"
	const correlationID = "req-week3-123"
	taskID, err := queue.EnqueueCreateFolder(context.Background(), parentPath, folderName, "integration-test", correlationID)
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	if status, ok := queue.Status(taskID); !ok || status.Status != "queued" {
		t.Fatalf("expected queued task status right after enqueue, got: %+v", status)
	}

	const timeout = 3 * time.Second
	const interval = 50 * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()
	lastStatus := "missing"
	for {
		taskStatus, ok := queue.Status(taskID)
		if ok {
			lastStatus = taskStatus.Status
		}
		if ok && taskStatus.Status == "success" {
			if taskStatus.CorrelationID != correlationID {
				t.Fatalf("expected correlation id %q, got %q", correlationID, taskStatus.CorrelationID)
			}
			if taskStatus.Message == "" {
				t.Fatalf("expected completion message to be persisted")
			}
			expectedPath := filepath.Join(rootDir, parentPath, folderName)
			info, statErr := os.Stat(expectedPath)
			if statErr != nil {
				t.Fatalf("expected folder to exist: %v", statErr)
			}
			if !info.IsDir() {
				t.Fatalf("expected %s to be a directory", expectedPath)
			}
			history := queue.TransitionHistory(taskID)
			expected := []string{"queued", "running", "success"}
			if len(history) != len(expected) {
				t.Fatalf("expected transition history %v, got %v", expected, history)
			}
			for i := range expected {
				if history[i] != expected[i] {
					t.Fatalf("expected transition history %v, got %v", expected, history)
				}
			}

			auditEvents := auditor.Snapshot()
			expectedAudit := []auditEvent{
				{event: "task.dequeued", taskID: taskID, correlationID: correlationID},
				{event: "task.processing", taskID: taskID, correlationID: correlationID},
				{event: "task.succeeded", taskID: taskID, correlationID: correlationID},
				{event: "folder.mutation.succeeded", taskID: taskID, correlationID: correlationID},
			}
			if len(auditEvents) != len(expectedAudit) {
				t.Fatalf("expected %d audit events, got %d (%+v)", len(expectedAudit), len(auditEvents), auditEvents)
			}
			for i := range expectedAudit {
				if auditEvents[i] != expectedAudit[i] {
					t.Fatalf("expected audit event[%d]=%+v, got %+v", i, expectedAudit[i], auditEvents[i])
				}
			}
			return
		}
		select {
		case <-ticker.C:
		case <-timeoutTimer.C:
			t.Fatalf("timed out waiting for task completion: %s (timeout=%s interval=%s last_status=%s)", taskID, timeout, interval, lastStatus)
		}
	}
}
