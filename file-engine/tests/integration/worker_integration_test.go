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
	statuses map[string]string
	mu       sync.RWMutex
}

func newInMemoryQueue() *inMemoryQueue {
	return &inMemoryQueue{
		ch:       make(chan *redisq.TaskPayload, 1),
		statuses: map[string]string{},
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

func (q *inMemoryQueue) Complete(_ context.Context, id, status string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.statuses[id] = status
	return nil
}

func (q *inMemoryQueue) EnqueueCreateFolder(parentPath, folderName, requestedBy string) string {
	id := fmt.Sprintf("task-%d", time.Now().UnixNano())
	q.ch <- &redisq.TaskPayload{
		ID:   id,
		Type: "create_folder",
		Params: map[string]string{
			"parent": parentPath,
			"name":   folderName,
			"by":     requestedBy,
		},
	}
	return id
}

func (q *inMemoryQueue) Status(id string) (string, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	status, ok := q.statuses[id]
	return status, ok
}

func TestAsyncCreateFolderFlow(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	queue := newInMemoryQueue()
	processor := tasks.NewProcessorWithStorage(localstorage.New(rootDir))
	worker := tasks.NewWorker(queue, processor, logger.New("debug"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Start(ctx)
	}()

	const parentPath = "tenants/acme/projects"
	const folderName = "week2-e2e"
	taskID := queue.EnqueueCreateFolder(parentPath, folderName, "integration-test")

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		status, ok := queue.Status(taskID)
		if ok {
			if status != "success" {
				t.Fatalf("task completed with unexpected status: %s", status)
			}

			expectedPath := filepath.Join(rootDir, parentPath, folderName)
			info, err := os.Stat(expectedPath)
			if err != nil {
				t.Fatalf("expected folder to exist: %v", err)
			}
			if !info.IsDir() {
				t.Fatalf("expected %s to be a directory", expectedPath)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for task completion: %s", taskID)
}
