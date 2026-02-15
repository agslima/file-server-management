package handlers

import (
	"context"
	"testing"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	"github.com/example/file-engine/internal/auth"
	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeTaskQueue struct {
	enqueued *enqueueRecord
	statuses map[string]*redisq.TaskStatus
}

type enqueueRecord struct {
	parentPath    string
	folderName    string
	requestedBy   string
	correlationID string
}

func (q *fakeTaskQueue) EnqueueCreateFolder(_ context.Context, parentPath, folderName, requestedBy, correlationID string) (string, error) {
	q.enqueued = &enqueueRecord{parentPath: parentPath, folderName: folderName, requestedBy: requestedBy, correlationID: correlationID}
	id := "task-abc"
	if q.statuses == nil {
		q.statuses = map[string]*redisq.TaskStatus{}
	}
	q.statuses[id] = &redisq.TaskStatus{TaskID: id, Status: "queued", CorrelationID: correlationID, Message: "task accepted"}
	return id, nil
}

func (q *fakeTaskQueue) GetStatus(_ context.Context, id string) (*redisq.TaskStatus, error) {
	if q.statuses == nil {
		return nil, redisq.ErrTaskNotFound
	}
	st, ok := q.statuses[id]
	if !ok {
		return nil, redisq.ErrTaskNotFound
	}
	return st, nil
}

func tenantResolverForTests() auth.TenantResolver {
	return auth.NewInMemoryTenantResolver(map[string][]string{
		"alice": {"acme"},
	})
}

func TestCreateFolderRequiresAuthContext(t *testing.T) {
	h := NewGRPCHandler(&fakeTaskQueue{}, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil)

	_, err := h.CreateFolder(context.Background(), &pb.CreateFolderRequest{ParentPath: "/tenants/acme", FolderName: "reports"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestCreateFolderRejectsNonTenantPath(t *testing.T) {
	h := NewGRPCHandler(&fakeTaskQueue{}, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil)
	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})

	_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: "/projects/shared", FolderName: "reports"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for non-tenant path, got %v", err)
	}
}

func TestCreateFolderRejectsUnauthorizedTenant(t *testing.T) {
	h := NewGRPCHandler(&fakeTaskQueue{}, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil)
	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})

	_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: "/tenants/beta", FolderName: "reports"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied for tenant mismatch, got %v", err)
	}
}

func TestCreateFolderEnqueuesWithCorrelationAndActorFallback(t *testing.T) {
	q := &fakeTaskQueue{}
	h := NewGRPCHandler(q, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil)

	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-request-id", "req-123"))

	resp, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: "/tenants/acme", FolderName: "reports"})
	if err != nil {
		t.Fatalf("CreateFolder returned error: %v", err)
	}
	if resp.TaskId == "" || resp.Status != "queued" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if q.enqueued == nil {
		t.Fatalf("expected queue enqueue to be called")
	}
	if q.enqueued.requestedBy != "alice" {
		t.Fatalf("expected requestedBy fallback to auth subject, got %q", q.enqueued.requestedBy)
	}
	if q.enqueued.correlationID != "req-123" {
		t.Fatalf("expected correlation id propagated, got %q", q.enqueued.correlationID)
	}
}

func TestGetTaskStatusRequiresAuthAndReturnsPersistedStatus(t *testing.T) {
	q := &fakeTaskQueue{statuses: map[string]*redisq.TaskStatus{"task-abc": {TaskID: "task-abc", Status: "success", Message: "done"}}}
	h := NewGRPCHandler(q, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil)

	_, err := h.GetTaskStatus(context.Background(), &pb.TaskStatusRequest{TaskId: "task-abc"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}

	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"viewer"}})
	resp, err := h.GetTaskStatus(ctx, &pb.TaskStatusRequest{TaskId: "task-abc"})
	if err != nil {
		t.Fatalf("GetTaskStatus returned error: %v", err)
	}
	if resp.Status != "success" || resp.Message != "done" || resp.Progress != 100 {
		t.Fatalf("unexpected task status response: %+v", resp)
	}
}

func TestCreateFolderEnqueuesNormalizedPath(t *testing.T) {
	q := &fakeTaskQueue{}
	h := NewGRPCHandler(q, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil)

	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})
	_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: `tenants\acme\projects//`, FolderName: "reports"})
	if err != nil {
		t.Fatalf("CreateFolder returned error: %v", err)
	}
	if q.enqueued == nil {
		t.Fatalf("expected queue enqueue to be called")
	}
	if q.enqueued.parentPath != "/tenants/acme/projects" {
		t.Fatalf("expected normalized parent path, got %q", q.enqueued.parentPath)
	}
	if q.enqueued.folderName != "reports" {
		t.Fatalf("expected folder name reports, got %q", q.enqueued.folderName)
	}
}

func TestCreateFolderWithNilResolverDefaultsToDenyAll(t *testing.T) {
	h := NewGRPCHandler(&fakeTaskQueue{}, nil, auth.NewInMemoryACLStore(), nil, nil)
	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})

	_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: "/tenants/acme", FolderName: "reports"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied for deny-all default, got %v", err)
	}
}
