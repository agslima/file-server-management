package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/handlers"
	"github.com/example/file-engine/internal/services"
	pb "github.com/example/file-engine/pkg/generated"
	"github.com/jackc/pgx/v5/pgxpool"
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

type postgresAuditStore struct{ pool *pgxpool.Pool }

func (s *postgresAuditStore) EmitTaskEvent(ctx context.Context, event, taskID, correlationID, message string, metadata ...map[string]string) {
	metaJSON, _ := json.Marshal(map[string]string{})
	if len(metadata) > 0 && metadata[0] != nil {
		metaJSON, _ = json.Marshal(metadata[0])
	}
	_, _ = s.pool.Exec(ctx, `INSERT INTO audit_events (event_type, task_id, correlation_id, message, metadata) VALUES ($1,$2,$3,$4,$5::jsonb)`, event, taskID, correlationID, message, string(metaJSON))
}

func TestAuditEventsEmittedForReadListDownload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool := mustConnectAuditDB(t, ctx)
	defer pool.Close()
	mustResetAuditEvents(t, ctx, pool)

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

	h := handlers.NewGRPCHandler(auditQueueStub{}, objectService, nil, acl, auth.NewInMemoryTenantResolver(map[string][]string{"alice": {"acme"}}), nil, &postgresAuditStore{pool: pool})

	correlationID := fmt.Sprintf("corr-read-audit-%d", time.Now().UnixNano())
	baseCtx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{}})
	requestCtx := metadata.NewIncomingContext(baseCtx, metadata.Pairs("x-request-id", correlationID))

	if err := storage.CreateFolder(requestCtx, "/tenants/acme/projects"); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if err := storage.AtomicWrite(requestCtx, "/tenants/acme/projects/readme.txt", bytes.NewBufferString("hello")); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := h.ListObjects(requestCtx, &pb.ListObjectsRequest{Prefix: "/tenants/acme/projects"}); err != nil {
		t.Fatalf("list objects failed: %v", err)
	}
	stream := &auditDownloadCaptureStream{ctx: requestCtx}
	if err := h.DownloadObject(&pb.DownloadObjectRequest{Path: "/tenants/acme/projects/readme.txt"}, stream); err != nil {
		t.Fatalf("download object failed: %v", err)
	}

	rows, err := pool.Query(ctx, `SELECT event_type, correlation_id, metadata::text FROM audit_events WHERE correlation_id = $1 AND event_type IN ('object.list','object.read','object.download')`, correlationID)
	if err != nil {
		t.Fatalf("query audit rows: %v", err)
	}
	defer rows.Close()

	seen := map[string]bool{}
	count := 0
	for rows.Next() {
		var eventType, rowCorrelationID, metadataJSON string
		if err := rows.Scan(&eventType, &rowCorrelationID, &metadataJSON); err != nil {
			t.Fatalf("scan audit row: %v", err)
		}
		var meta map[string]any
		if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
			t.Fatalf("decode metadata json: %v", err)
		}
		count++
		seen[eventType] = true
		if rowCorrelationID == "" || meta["tenant_id"] == "" || meta["actor_id"] == "" || meta["action"] == "" || meta["result"] == "" {
			t.Fatalf("expected required audit metadata fields for event %q, got correlation_id=%q metadata=%s", eventType, rowCorrelationID, metadataJSON)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate audit rows: %v", err)
	}
	if count < 3 {
		t.Fatalf("expected at least 3 audit rows for read/list/download, got %d", count)
	}
	for _, action := range []string{"object.list", "object.read", "object.download"} {
		if !seen[action] {
			t.Fatalf("expected action %q in audit rows", action)
		}
	}
}
