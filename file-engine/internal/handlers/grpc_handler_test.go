package handlers

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/services"
	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeTaskQueue struct {
	enqueued *enqueueRecord
	statuses map[string]*redisq.TaskStatus
}

type mutableTenantResolver struct {
	allowed bool
}

func (m *mutableTenantResolver) ResolveTenants(_ context.Context, _ string) ([]string, error) {
	if m.allowed {
		return []string{"acme"}, nil
	}
	return nil, nil
}

func (m *mutableTenantResolver) UserHasTenant(_ context.Context, _, tenantID string) (bool, error) {
	return m.allowed && tenantID == "acme", nil
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
	h := NewGRPCHandler(&fakeTaskQueue{}, nil, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)

	_, err := h.CreateFolder(context.Background(), &pb.CreateFolderRequest{ParentPath: "/tenants/acme", FolderName: "reports"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestCreateFolderRejectsNonTenantPath(t *testing.T) {
	h := NewGRPCHandler(&fakeTaskQueue{}, nil, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)
	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})

	_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: "/projects/shared", FolderName: "reports"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument for non-tenant path, got %v", err)
	}
}

func TestCreateFolderRejectsUnauthorizedTenant(t *testing.T) {
	h := NewGRPCHandler(&fakeTaskQueue{}, nil, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)
	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})

	_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: "/tenants/beta", FolderName: "reports"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied for tenant mismatch, got %v", err)
	}
}

func TestCreateFolderEnqueuesWithCorrelationAndActorFallback(t *testing.T) {
	q := &fakeTaskQueue{}
	h := NewGRPCHandler(q, nil, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)

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
	h := NewGRPCHandler(q, nil, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)

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
	h := NewGRPCHandler(q, nil, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)

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
	h := NewGRPCHandler(&fakeTaskQueue{}, nil, nil, auth.NewInMemoryACLStore(), nil, nil, nil)
	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})

	_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: "/tenants/acme", FolderName: "reports"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied for deny-all default, got %v", err)
	}
}

func TestListObjectsReturnsEntries(t *testing.T) {
	root := t.TempDir()
	st := localstorage.New(root)
	obj := services.NewObjectService(st)
	h := NewGRPCHandler(&fakeTaskQueue{}, obj, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)

	start := time.Now().Add(-time.Second)
	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"viewer"}})
	if err := st.CreateFolder(ctx, "/tenants/acme/projects"); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if err := st.AtomicWrite(ctx, "/tenants/acme/projects/report.txt", bytes.NewBufferString("ok")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := st.CreateFolder(ctx, "/tenants/acme/projects/reports"); err != nil {
		t.Fatalf("create subdir: %v", err)
	}

	resp, err := h.ListObjects(ctx, &pb.ListObjectsRequest{Prefix: `tenants\acme\projects//`})
	if err != nil {
		t.Fatalf("ListObjects returned error: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	seen := map[string]*pb.ObjectInfo{}
	for _, it := range resp.Items {
		seen[it.Path] = it
	}
	file := seen["/tenants/acme/projects/report.txt"]
	if file == nil || file.IsDir {
		t.Fatalf("expected file /tenants/acme/projects/report.txt, got %+v", resp.Items)
	}
	if file.Size != 2 {
		t.Fatalf("expected size 2 for /tenants/acme/projects/report.txt, got %d", file.Size)
	}
	if file.ModifiedAt == nil || file.ModifiedAt.AsTime().Before(start) {
		t.Fatalf("expected modified_at after %v, got %+v", start, file.ModifiedAt)
	}
	if file.CreatedAt == nil || file.CreatedAt.AsTime().Before(start) {
		t.Fatalf("expected created_at after %v, got %+v", start, file.CreatedAt)
	}
	if file.Owner != strconv.Itoa(os.Getuid()) {
		t.Fatalf("expected owner %d, got %q", os.Getuid(), file.Owner)
	}
	if file.Group != strconv.Itoa(os.Getgid()) {
		t.Fatalf("expected group %d, got %q", os.Getgid(), file.Group)
	}
	dir := seen["/tenants/acme/projects/reports"]
	if dir == nil || !dir.IsDir {
		t.Fatalf("expected dir /tenants/acme/projects/reports, got %+v", resp.Items)
	}
}

func TestListObjectsRequiresAuthContext(t *testing.T) {
	root := t.TempDir()
	st := localstorage.New(root)
	obj := services.NewObjectService(st)
	h := NewGRPCHandler(&fakeTaskQueue{}, obj, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)

	_, err := h.ListObjects(context.Background(), &pb.ListObjectsRequest{Prefix: "/tenants/acme/projects"})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestListObjectsRejectsUnauthorizedTenant(t *testing.T) {
	root := t.TempDir()
	st := localstorage.New(root)
	obj := services.NewObjectService(st)
	h := NewGRPCHandler(&fakeTaskQueue{}, obj, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)

	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"viewer"}})
	_, err := h.ListObjects(ctx, &pb.ListObjectsRequest{Prefix: "/tenants/beta/projects"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

type fakeDownloadStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (f *fakeDownloadStream) Context() context.Context { return f.ctx }
func (f *fakeDownloadStream) Send(*pb.DownloadChunk) error {
	return nil
}

func TestDownloadObjectRejectsUnauthorizedTenant(t *testing.T) {
	root := t.TempDir()
	st := localstorage.New(root)
	obj := services.NewObjectService(st)
	h := NewGRPCHandler(&fakeTaskQueue{}, obj, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)

	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"viewer"}})
	stream := &fakeDownloadStream{ctx: ctx}
	err := h.DownloadObject(&pb.DownloadObjectRequest{Path: "/tenants/beta/projects/report.txt"}, stream)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestUploadObjectRequiresAuthContext(t *testing.T) {
	root := t.TempDir()
	st := localstorage.New(root)
	obj := services.NewObjectService(st)
	h := NewGRPCHandler(&fakeTaskQueue{}, obj, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)

	_, err := h.UploadObject(context.Background(), &pb.UploadObjectRequest{Path: "/tenants/acme/projects/new.txt", Content: []byte("ok")})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated, got %v", err)
	}
}

func TestUploadObjectRejectsUnauthorizedTenant(t *testing.T) {
	root := t.TempDir()
	st := localstorage.New(root)
	obj := services.NewObjectService(st)
	h := NewGRPCHandler(&fakeTaskQueue{}, obj, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, nil)

	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})
	_, err := h.UploadObject(ctx, &pb.UploadObjectRequest{Path: "/tenants/beta/projects/new.txt", Content: []byte("ok")})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestMembershipRevocationBlocksAccessImmediately(t *testing.T) {
	root := t.TempDir()
	st := localstorage.New(root)
	obj := services.NewObjectService(st)
	resolver := &mutableTenantResolver{allowed: true}
	h := NewGRPCHandler(&fakeTaskQueue{}, obj, nil, auth.NewInMemoryACLStore(), resolver, nil, nil)
	if err := st.CreateFolder(context.Background(), "/tenants/acme"); err != nil {
		t.Fatalf("seed tenant dir: %v", err)
	}

	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})
	if _, err := h.ListObjects(ctx, &pb.ListObjectsRequest{Prefix: "/tenants/acme"}); err != nil {
		t.Fatalf("expected access before revoke, got %v", err)
	}

	resolver.allowed = false
	_, err := h.ListObjects(ctx, &pb.ListObjectsRequest{Prefix: "/tenants/acme"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied after revoke, got %v", err)
	}
}

type auditSpyEvent struct {
	event         string
	taskID        string
	correlationID string
	message       string
}

type auditSpy struct {
	events []auditSpyEvent
}

func (a *auditSpy) EmitTaskEvent(_ context.Context, event, taskID, correlationID, message string, _ ...map[string]string) {
	a.events = append(a.events, auditSpyEvent{event: event, taskID: taskID, correlationID: correlationID, message: message})
}

func TestCreateFolderEmitsAuditForAuthDecisionAndEnqueue(t *testing.T) {
	q := &fakeTaskQueue{}
	auditor := &auditSpy{}
	h := NewGRPCHandler(q, nil, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, auditor)

	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-request-id", "req-audit-1"))

	_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: "/tenants/acme", FolderName: "reports"})
	if err != nil {
		t.Fatalf("CreateFolder returned error: %v", err)
	}

	if len(auditor.events) < 3 {
		t.Fatalf("expected at least 3 audit events, got %+v", auditor.events)
	}
	if auditor.events[0].event != "auth.decision.allowed" {
		t.Fatalf("expected first event auth.decision.allowed, got %+v", auditor.events[0])
	}
	if auditor.events[1].event != "task.enqueued" {
		t.Fatalf("expected second event task.enqueued, got %+v", auditor.events[1])
	}
	if auditor.events[2].event != "folder.mutation.requested" {
		t.Fatalf("expected third event folder.mutation.requested, got %+v", auditor.events[2])
	}
}

func TestUploadObjectEmitsAuditStartAndFinish(t *testing.T) {
	root := t.TempDir()
	st := localstorage.New(root)
	obj := services.NewObjectService(st)
	auditor := &auditSpy{}
	h := NewGRPCHandler(&fakeTaskQueue{}, obj, nil, auth.NewInMemoryACLStore(), tenantResolverForTests(), nil, auditor)

	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-request-id", "req-upload-audit"))

	_, err := h.UploadObject(ctx, &pb.UploadObjectRequest{Path: "/tenants/acme/projects/a.txt", Content: []byte("ok")})
	if err != nil {
		t.Fatalf("UploadObject returned error: %v", err)
	}

	seenStart := false
	seenFinish := false
	for _, ev := range auditor.events {
		if ev.event == "upload.started" {
			seenStart = true
		}
		if ev.event == "upload.completed" {
			seenFinish = true
		}
	}
	if !seenStart || !seenFinish {
		t.Fatalf("expected upload.started and upload.completed events, got %+v", auditor.events)
	}
}
