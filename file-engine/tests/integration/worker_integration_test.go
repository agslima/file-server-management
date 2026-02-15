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
	ch       chan *redisq.TaskPayload
	statuses map[string]redisq.TaskStatus
	mu       sync.RWMutex
}

func newInMemoryQueue() *inMemoryQueue {
	return &inMemoryQueue{
		ch:       make(chan *redisq.TaskPayload, 1),
		statuses: map[string]redisq.TaskStatus{},
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
	return nil
}

func (q *inMemoryQueue) EnqueueCreateFolder(parentPath, folderName, requestedBy, correlationID string) string {
	id := fmt.Sprintf("task-%d", time.Now().UnixNano())
	q.mu.Lock()
	q.statuses[id] = redisq.TaskStatus{TaskID: id, Status: "queued", CorrelationID: correlationID, Message: "task accepted"}
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
	return id
}

func (q *inMemoryQueue) Status(id string) (redisq.TaskStatus, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	status, ok := q.statuses[id]
	return status, ok
}

type inMemoryAuditor struct {
	mu     sync.Mutex
	events []string
}

func (a *inMemoryAuditor) EmitTaskEvent(_ context.Context, event, taskID, correlationID, _ string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, fmt.Sprintf("%s|%s|%s", event, taskID, correlationID))
}

func (a *inMemoryAuditor) Contains(prefix string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, e := range a.events {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func TestAsyncCreateFolderFlow(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	queue := newInMemoryQueue()
	auditor := &inMemoryAuditor{}
	processor := tasks.NewProcessorWithStorage(localstorage.New(rootDir))
	worker := tasks.NewWorkerWithAudit(queue, processor, logger.New("debug"), auditor)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go worker.Start(ctx)

	const parentPath = "tenants/acme/projects"
	const folderName = "week2-e2e"
	const correlationID = "req-week3-123"
	taskID := queue.EnqueueCreateFolder(parentPath, folderName, "integration-test", correlationID)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		taskStatus, ok := queue.Status(taskID)
		if ok && taskStatus.Status == "success" {
			if taskStatus.CorrelationID != correlationID {
				t.Fatalf("expected correlation id %q, got %q", correlationID, taskStatus.CorrelationID)
			}
			if taskStatus.Message == "" {
				t.Fatalf("expected completion message to be persisted")
			}
			expectedPath := filepath.Join(rootDir, parentPath, folderName)
			info, err := os.Stat(expectedPath)
			if err != nil {
				t.Fatalf("expected folder to exist: %v", err)
			}
			if !info.IsDir() {
				t.Fatalf("expected %s to be a directory", expectedPath)
			}
			if !auditor.Contains("task.succeeded|" + taskID + "|" + correlationID) {
				t.Fatalf("expected audit success event for task %s", taskID)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for task completion: %s", taskID)
}
