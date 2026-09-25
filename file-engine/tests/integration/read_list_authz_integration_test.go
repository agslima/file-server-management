//go:build integration_authz

package integration

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/example/file-engine/internal/adapters/queue/redisq"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/handlers"
	"github.com/example/file-engine/internal/services"
	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type integrationQueueStub struct{}

func (integrationQueueStub) EnqueueCreateFolder(_ context.Context, _, _, _, _ string) (string, error) {
	return "task-integration", nil
}

func (integrationQueueStub) GetStatus(_ context.Context, _ string) (*redisq.TaskStatus, error) {
	return nil, redisq.ErrTaskNotFound
}

type downloadCaptureStream struct {
	grpc.ServerStream
	ctx    context.Context
	chunks [][]byte
}

func (s *downloadCaptureStream) Context() context.Context { return s.ctx }

func (s *downloadCaptureStream) Send(chunk *pb.DownloadChunk) error {
	s.chunks = append(s.chunks, append([]byte(nil), chunk.GetData()...))
	return nil
}

func tenantResolverSeed() auth.TenantResolver {
	return auth.NewInMemoryTenantResolver(map[string][]string{
		"alice": {"acme"},
	})
}

func TestReadListBehaviorAndAuthzRejection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storage := localstorage.New(root)
	objectService := services.NewObjectService(storage)
	h := handlers.NewGRPCHandler(integrationQueueStub{}, objectService, nil, auth.NewInMemoryACLStore(), tenantResolverSeed(), nil, nil)

	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"viewer"}})

	if err := storage.CreateFolder(ctx, "/tenants/acme/projects"); err != nil {
		t.Fatalf("create tenant folder: %v", err)
	}
	if err := storage.AtomicWrite(ctx, "/tenants/acme/projects/readme.txt", bytes.NewBufferString("ok")); err != nil {
		t.Fatalf("write tenant file: %v", err)
	}

	start := time.Now().Add(-1 * time.Second)
	resp, err := h.ListObjects(ctx, &pb.ListObjectsRequest{Prefix: "/tenants/acme/projects"})
	if err != nil {
		t.Fatalf("ListObjects for authorized tenant failed: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].Path != "/tenants/acme/projects/readme.txt" {
		t.Fatalf("expected file path /tenants/acme/projects/readme.txt, got %q", resp.Items[0].Path)
	}
	if resp.Items[0].IsDir {
		t.Fatalf("expected listed item to be file, got directory")
	}
	if resp.Items[0].ModifiedAt == nil || resp.Items[0].ModifiedAt.AsTime().Before(start) {
		t.Fatalf("expected modified_at after %v, got %+v", start, resp.Items[0].ModifiedAt)
	}
	if resp.Items[0].CreatedAt == nil || resp.Items[0].CreatedAt.AsTime().Before(start) {
		t.Fatalf("expected created_at after %v, got %+v", start, resp.Items[0].CreatedAt)
	}

	cases := []struct {
		name string
		run  func(context.Context) error
	}{
		{
			name: "list unauthorized tenant",
			run: func(ctx context.Context) error {
				_, err := h.ListObjects(ctx, &pb.ListObjectsRequest{Prefix: "/tenants/beta/projects"})
				return err
			},
		},
		{
			name: "read/download unauthorized tenant",
			run: func(ctx context.Context) error {
				stream := &downloadCaptureStream{ctx: ctx}
				return h.DownloadObject(&pb.DownloadObjectRequest{Path: "/tenants/beta/projects/readme.txt"}, stream)
			},
		},
		{
			name: "upload unauthorized tenant",
			run: func(ctx context.Context) error {
				_, err := h.UploadObject(ctx, &pb.UploadObjectRequest{Path: "/tenants/beta/projects/upload.txt", Content: []byte("payload")})
				return err
			},
		},
		{
			name: "create folder unauthorized tenant",
			run: func(ctx context.Context) error {
				_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: "/tenants/beta/projects", FolderName: "reports"})
				return err
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run(ctx)
			if status.Code(err) != codes.PermissionDenied {
				t.Fatalf("expected permission denied, got %v", err)
			}
		})
	}
}

func TestCrossTenantPolicyCoverageWithTableDrivenCases(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storage := localstorage.New(root)
	objectService := services.NewObjectService(storage)
	h := handlers.NewGRPCHandler(integrationQueueStub{}, objectService, nil, auth.NewInMemoryACLStore(), tenantResolverSeed(), nil, nil)

	baseCtx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})
	ctx := metadata.NewIncomingContext(baseCtx, metadata.Pairs("x-request-id", "integration-tenant-deny"))

	if err := storage.CreateFolder(ctx, "/tenants/acme/projects"); err != nil {
		t.Fatalf("create authorized tenant folder: %v", err)
	}
	if err := storage.AtomicWrite(ctx, "/tenants/acme/projects/existing.txt", bytes.NewBufferString("ok")); err != nil {
		t.Fatalf("write authorized tenant file: %v", err)
	}

	tests := []struct {
		name         string
		run          func(context.Context) error
		expectDenied bool
	}{
		{
			name: "list authorized tenant",
			run: func(ctx context.Context) error {
				_, err := h.ListObjects(ctx, &pb.ListObjectsRequest{Prefix: "/tenants/acme/projects"})
				return err
			},
			expectDenied: false,
		},
		{
			name: "list unauthorized tenant",
			run: func(ctx context.Context) error {
				_, err := h.ListObjects(ctx, &pb.ListObjectsRequest{Prefix: "/tenants/beta/projects"})
				return err
			},
			expectDenied: true,
		},
		{
			name: "upload authorized tenant",
			run: func(ctx context.Context) error {
				_, err := h.UploadObject(ctx, &pb.UploadObjectRequest{Path: "/tenants/acme/projects/new.txt", Content: []byte("ok")})
				return err
			},
			expectDenied: false,
		},
		{
			name: "upload unauthorized tenant",
			run: func(ctx context.Context) error {
				_, err := h.UploadObject(ctx, &pb.UploadObjectRequest{Path: "/tenants/beta/projects/new.txt", Content: []byte("no")})
				return err
			},
			expectDenied: true,
		},
		{
			name: "create folder authorized tenant",
			run: func(ctx context.Context) error {
				_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: "/tenants/acme/projects", FolderName: "reports"})
				return err
			},
			expectDenied: false,
		},
		{
			name: "create folder unauthorized tenant",
			run: func(ctx context.Context) error {
				_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{ParentPath: "/tenants/beta/projects", FolderName: "reports"})
				return err
			},
			expectDenied: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(ctx)
			if tc.expectDenied {
				if status.Code(err) != codes.PermissionDenied {
					t.Fatalf("expected permission denied, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected success, got %v", err)
			}
		})
	}
}

func TestEngineBoundaryDeniesCrossTenantEvenWithBuggyUpstreamInputs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storage := localstorage.New(root)
	objectService := services.NewObjectService(storage)
	h := handlers.NewGRPCHandler(integrationQueueStub{}, objectService, nil, auth.NewInMemoryACLStore(), tenantResolverSeed(), nil, nil)

	baseCtx := auth.WithAuthContext(context.Background(), auth.AuthContext{UserID: "alice", Roles: []string{"admin"}})
	ctx := metadata.NewIncomingContext(baseCtx, metadata.Pairs("x-request-id", "integration-buggy-upstream"))

	if err := storage.CreateFolder(ctx, "/tenants/acme/projects"); err != nil {
		t.Fatalf("create authorized tenant folder: %v", err)
	}
	if err := storage.AtomicWrite(ctx, "/tenants/acme/projects/ok.txt", bytes.NewBufferString("ok")); err != nil {
		t.Fatalf("write authorized tenant file: %v", err)
	}

	// requestedBy below intentionally simulates buggy upstream behavior: it claims a privileged
	// identity different from the authenticated subject. The engine must still deny by auth context
	// user-to-tenant mapping and final in-engine authz checks.
	tests := []struct {
		name string
		run  func(context.Context) error
	}{
		{
			name: "list denied on unauthorized tenant despite upstream context",
			run: func(ctx context.Context) error {
				_, err := h.ListObjects(ctx, &pb.ListObjectsRequest{Prefix: "/tenants/beta/projects"})
				return err
			},
		},
		{
			name: "upload denied on unauthorized tenant despite upstream context",
			run: func(ctx context.Context) error {
				_, err := h.UploadObject(ctx, &pb.UploadObjectRequest{Path: "/tenants/beta/projects/pwned.txt", Content: []byte("no")})
				return err
			},
		},
		{
			name: "folder op denied on unauthorized tenant despite spoofed requestedBy",
			run: func(ctx context.Context) error {
				_, err := h.CreateFolder(ctx, &pb.CreateFolderRequest{
					ParentPath:  "/tenants/beta/projects",
					FolderName:  "finance",
					RequestedBy: "superadmin@upstream.local",
				})
				return err
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(ctx)
			if status.Code(err) != codes.PermissionDenied {
				t.Fatalf("expected permission denied, got %v", err)
			}
		})
	}
}
