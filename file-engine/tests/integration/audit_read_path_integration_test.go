package integration

import (
	"bytes"
	"context"
	"maps"
	"sync"
	"testing"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/handlers"
	"github.com/example/file-engine/internal/services"
	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type auditQueueStub struct{}

func (auditQueueStub) EnqueueCreateFolder(_ context.Context, _, _, _, _ string) (string, error) {
	return "task-audit", nil
}

func (auditQueueStub) GetStatus(_ context.Context, _ string) (*redisq.TaskStatus, error) {
	return nil, redisq.ErrTaskNotFound
}

type auditDownloadCaptureStream struct {
	grpc.ServerStream
	ctx    context.Context
	chunks [][]byte
}

func (s *auditDownloadCaptureStream) Context() context.Context { return s.ctx }

func (s *auditDownloadCaptureStream) Send(chunk *pb.DownloadChunk) error {
	s.chunks = append(s.chunks, append([]byte(nil), chunk.GetData()...))
	return nil
}

type auditEventRow struct {
	eventType     string
	correlationID string
	metadata      map[string]string
}

type auditEventStore struct {
	mu   sync.Mutex
	rows []auditEventRow
}

func (s *auditEventStore) EmitTaskEvent(_ context.Context, event, _, correlationID, _ string, metadata ...map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta := map[string]string{}
	if len(metadata) > 0 && metadata[0] != nil {
		maps.Copy(meta, metadata[0])
	}

	s.rows = append(s.rows, auditEventRow{
		eventType:     event,
		correlationID: correlationID,
		metadata:      meta,
	})
}

func (s *auditEventStore) queryByActions(actions map[string]bool) []auditEventRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []auditEventRow
	for _, row := range s.rows {
		if actions[row.eventType] {
			out = append(out, row)
		}
	}
	return out
}

func TestAuditEventsEmittedForReadListDownload(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storage := localstorage.New(root)
	objectService := services.NewObjectService(storage)

	acl := auth.NewInMemoryACLStore()
	if err := acl.SetACL(auth.ACL{
		Path:        "/tenants/acme/projects",
		PrincipalID: "user:alice",
		Permissions: map[auth.Permission]bool{auth.PermList: true, auth.PermRead: true},
	}); err != nil {
		t.Fatalf("set acl: %v", err)
	}

	audits := &auditEventStore{}
	h := handlers.NewGRPCHandler(
		auditQueueStub{},
		objectService,
		nil,
		acl,
		auth.NewInMemoryTenantResolver(map[string][]string{"alice": {"acme"}}),
		nil,
		audits,
	)

	baseCtx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{}})
	ctx := metadata.NewIncomingContext(baseCtx, metadata.Pairs("x-request-id", "corr-read-audit-1"))

	if err := storage.CreateFolder(ctx, "/tenants/acme/projects"); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if err := storage.AtomicWrite(ctx, "/tenants/acme/projects/readme.txt", bytes.NewBufferString("hello")); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := h.ListObjects(ctx, &pb.ListObjectsRequest{Prefix: "/tenants/acme/projects"}); err != nil {
		t.Fatalf("list objects failed: %v", err)
	}

	stream := &auditDownloadCaptureStream{ctx: ctx}
	if err := h.DownloadObject(&pb.DownloadObjectRequest{Path: "/tenants/acme/projects/readme.txt"}, stream); err != nil {
		t.Fatalf("download object failed: %v", err)
	}

	actionRows := audits.queryByActions(map[string]bool{
		"object.list":     true,
		"object.read":     true,
		"object.download": true,
	})
	if len(actionRows) < 3 {
		t.Fatalf("expected at least 3 read/list/download audit rows, got %d (%+v)", len(actionRows), actionRows)
	}

	seen := map[string]bool{}
	for _, row := range actionRows {
		seen[row.eventType] = true
		if row.correlationID == "" {
			t.Fatalf("expected correlation_id for row %+v", row)
		}
		if row.metadata["tenant_id"] == "" {
			t.Fatalf("expected tenant_id metadata for row %+v", row)
		}
		if row.metadata["actor_id"] == "" {
			t.Fatalf("expected actor_id metadata for row %+v", row)
		}
		if row.metadata["action"] == "" {
			t.Fatalf("expected action metadata for row %+v", row)
		}
		if row.metadata["result"] == "" {
			t.Fatalf("expected result metadata for row %+v", row)
		}
	}

	for _, action := range []string{"object.list", "object.read", "object.download"} {
		if !seen[action] {
			t.Fatalf("expected action %q in audit rows, got %+v", action, actionRows)
		}
	}
}
