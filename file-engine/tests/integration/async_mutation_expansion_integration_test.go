package integration

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	securityadp "github.com/example/file-engine/internal/adapters/security"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/app/tasks"
	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/services"
)

type uploadMutationExecutor struct{ svc *services.UploadService }

func (e *uploadMutationExecutor) MoveObject(ctx context.Context, actorID, sourcePath, destinationPath string) error {
	_, err := e.svc.MoveObject(ctx, actorID, sourcePath, destinationPath)
	return err
}

func (e *uploadMutationExecutor) DeleteObject(ctx context.Context, actorID, objectPath string) error {
	return e.svc.DeleteObject(ctx, actorID, objectPath)
}

func (e *uploadMutationExecutor) RestoreQuarantinedObject(ctx context.Context, actorID, objectPath string, forceReprocess bool) error {
	_, err := e.svc.RestoreQuarantinedObject(ctx, actorID, objectPath, forceReprocess)
	return err
}

func (q *inMemoryQueue) enqueueTask(taskType string, params map[string]string, correlationID string) string {
	id := fmt.Sprintf("task-%d", time.Now().UnixNano())
	q.mu.Lock()
	q.statuses[id] = redisq.TaskStatus{TaskID: id, Status: "queued", CorrelationID: correlationID, Message: "task accepted"}
	q.transitions[id] = append(q.transitions[id], "queued")
	q.mu.Unlock()
	params["correlation_id"] = correlationID
	params["request_id"] = correlationID
	q.ch <- &redisq.TaskPayload{ID: id, Type: taskType, Params: params}
	return id
}

func TestAsyncMutationMoveObjectFlow(t *testing.T) {
	t.Parallel()
	rootDir := t.TempDir()
	st := localstorage.New(rootDir)
	svc := services.NewUploadService(st, securityadp.NewMalwareScannerStub(), services.UploadPolicy{})
	q := newInMemoryQueue()
	auditor := &inMemoryAuditor{}
	processor := tasks.NewProcessorWithMutationExecutor(st, &uploadMutationExecutor{svc: svc})
	worker := tasks.NewWorkerWithAudit(q, processor, logger.New("debug"), auditor)
	go worker.Start(t.Context())

	src := "/tenants/acme/docs/source.txt"
	dst := "/tenants/acme/docs/dest.txt"
	if _, err := svc.UploadStream(context.Background(), src, bytes.NewReader([]byte("payload")), ""); err != nil {
		t.Fatalf("seed upload failed: %v", err)
	}
	taskID := q.enqueueTask("move_file", map[string]string{"src": src, "dst": dst, "tenant_id": "acme", "actor_id": "admin-user", "idempotency_key": "idem-move-1"}, "req-move-1")

	waitForTaskSuccess(t, q, taskID)
	if _, err := st.Open(context.Background(), dst); err != nil {
		t.Fatalf("expected destination object after move: %v", err)
	}
	if _, err := st.Open(context.Background(), src); err == nil {
		t.Fatalf("expected source object to be moved away")
	}
	assertAuditContains(t, auditor.Snapshot(), "object.mutation.succeeded", taskID, "req-move-1")
}

func TestAsyncMutationGovernedDeleteFinalGateDenies(t *testing.T) {
	t.Parallel()
	rootDir := t.TempDir()
	st := localstorage.New(rootDir)
	svc := services.NewUploadService(st, securityadp.NewMalwareScannerStub(), services.UploadPolicy{})
	if err := svc.SetGovernancePolicy(services.GovernancePolicy{Default: services.TenantGovernancePolicy{}, Tenants: map[string]services.TenantGovernancePolicy{"acme": {RetentionSeconds: 3600}}}); err != nil {
		t.Fatalf("set governance policy: %v", err)
	}
	q := newInMemoryQueue()
	auditor := &inMemoryAuditor{}
	processor := tasks.NewProcessorWithMutationExecutor(st, &uploadMutationExecutor{svc: svc})
	worker := tasks.NewWorkerWithAudit(q, processor, logger.New("debug"), auditor)
	go worker.Start(t.Context())

	path := "/tenants/acme/docs/protected.txt"
	if _, err := svc.UploadStream(context.Background(), path, bytes.NewReader([]byte("payload")), ""); err != nil {
		t.Fatalf("seed upload failed: %v", err)
	}
	taskID := q.enqueueTask("governed_delete", map[string]string{"path": path, "tenant_id": "acme", "actor_id": "admin-user", "idempotency_key": "idem-delete-1"}, "req-delete-1")

	status := waitForTaskTerminal(t, q, taskID)
	if status.Status != "failed" {
		t.Fatalf("expected failed status for retention-blocked delete, got %+v", status)
	}
	if !strings.Contains(status.Message, "TASK_EXECUTION_FAILED") {
		t.Fatalf("expected stable error envelope in status message, got %q", status.Message)
	}
	if _, err := st.Open(context.Background(), path); err != nil {
		t.Fatalf("expected object to remain after governance deny: %v", err)
	}
	assertAuditContains(t, auditor.Snapshot(), "object.mutation.failed", taskID, "req-delete-1")
}

func TestAsyncMutationQuarantineRestoreFlow(t *testing.T) {
	t.Parallel()
	rootDir := t.TempDir()
	st := localstorage.New(rootDir)
	svc := services.NewUploadService(st, securityadp.NewMalwareScannerStub(), services.UploadPolicy{RequireCleanScan: true})
	q := newInMemoryQueue()
	auditor := &inMemoryAuditor{}
	processor := tasks.NewProcessorWithMutationExecutor(st, &uploadMutationExecutor{svc: svc})
	worker := tasks.NewWorkerWithAudit(q, processor, logger.New("debug"), auditor)
	go worker.Start(t.Context())

	path := "/tenants/acme/docs/eicar.txt"
	if _, err := svc.UploadStream(context.Background(), path, bytes.NewReader([]byte("payload")), ""); err == nil || !strings.Contains(err.Error(), "malware scan gate blocked commit") {
		t.Fatalf("expected quarantine seed upload to be blocked by scan gate, got err=%v", err)
	}
	taskID := q.enqueueTask("quarantine_restore", map[string]string{"path": path, "force_reprocess": "false", "tenant_id": "acme", "actor_id": "admin-user", "idempotency_key": "idem-restore-1"}, "req-restore-1")

	waitForTaskSuccess(t, q, taskID)
	if _, err := st.Open(context.Background(), path); err != nil {
		t.Fatalf("expected restored path to exist: %v", err)
	}
	if objs, err := st.List(context.Background(), "/quarantine"); err != nil {
		t.Fatalf("list quarantine: %v", err)
	} else {
		for _, obj := range objs {
			if !obj.IsDir {
				t.Fatalf("expected quarantine to have no file objects after restore, got %+v", objs)
			}
		}
	}
	assertAuditContains(t, auditor.Snapshot(), "object.mutation.succeeded", taskID, "req-restore-1")
}

func waitForTaskSuccess(t *testing.T, q *inMemoryQueue, taskID string) {
	t.Helper()
	status := waitForTaskTerminal(t, q, taskID)
	if status.Status != "success" {
		t.Fatalf("expected success status, got %+v", status)
	}
}

func waitForTaskTerminal(t *testing.T, q *inMemoryQueue, taskID string) redisq.TaskStatus {
	t.Helper()
	const timeout = 3 * time.Second
	const interval = 20 * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()
	last := "missing"
	for {
		if st, ok := q.Status(taskID); ok && (st.Status == "success" || st.Status == "failed") {
			return st
		} else if ok {
			last = st.Status
		}
		select {
		case <-ticker.C:
		case <-timeoutTimer.C:
			t.Fatalf("timed out waiting for task %s terminal status (timeout=%s interval=%s last_status=%s)", taskID, timeout, interval, last)
		}
	}
}

func assertAuditContains(t *testing.T, events []auditEvent, expectedEvent, taskID, correlationID string) {
	t.Helper()
	for _, ev := range events {
		if ev.event == expectedEvent && ev.taskID == taskID && ev.correlationID == correlationID {
			return
		}
	}
	t.Fatalf("expected audit event %q for task=%s correlation=%s; got %+v", expectedEvent, taskID, correlationID, events)
}
