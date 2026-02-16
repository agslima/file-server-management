//go:build integration_authz
// +build integration_authz

package integration

import (
	"bytes"
	"context"
	"testing"
	"time"

	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/handlers"
	"github.com/example/file-engine/internal/services"
	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func tenantResolverSeed() auth.TenantResolver {
	return auth.NewInMemoryTenantResolver(map[string][]string{
		"alice": []string{"acme"},
	})
}

func TestReadListBehaviorAndAuthzRejection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storage := localstorage.New(root)
	objectService := services.NewObjectService(storage)
	h := handlers.NewGRPCHandler(nil, objectService, auth.NewInMemoryACLStore(), tenantResolverSeed(), nil)

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

	_, err = h.ListObjects(ctx, &pb.ListObjectsRequest{Prefix: "/tenants/beta/projects"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied for unauthorized tenant list, got %v", err)
	}
}
