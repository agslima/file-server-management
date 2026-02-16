package authz

import (
	"context"
	"testing"

	"github.com/example/file-engine/internal/auth"
	pb "github.com/example/file-engine/pkg/generated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func runListObjectsAuthz(ctx context.Context, req *pb.ListObjectsRequest, store auth.ACLStore) error {
	interceptor := GRPCAuthZInterceptor(store)
	info := &grpc.UnaryServerInfo{FullMethod: "/fileengine.FileEngine/ListObjects"}
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	_, err := interceptor(ctx, req, info, handler)
	return err
}

func TestGRPCAuthZInterceptorListObjectsAllowsAuthorized(t *testing.T) {
	store := auth.NewInMemoryACLStore()
	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{
		UserID: "alice",
		Roles:  []string{"viewer"},
	})
	req := &pb.ListObjectsRequest{Prefix: "/tenants/acme/projects"}

	if err := runListObjectsAuthz(ctx, req, store); err != nil {
		t.Fatalf("expected allow, got %v", err)
	}
}

func TestGRPCAuthZInterceptorListObjectsRejectsUnauthorized(t *testing.T) {
	store := auth.NewInMemoryACLStore()
	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{
		UserID: "bob",
		Roles:  []string{"guest"},
	})
	req := &pb.ListObjectsRequest{Prefix: "/tenants/acme/projects"}

	err := runListObjectsAuthz(ctx, req, store)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestGRPCAuthZInterceptorListObjectsRejectsTraversal(t *testing.T) {
	store := auth.NewInMemoryACLStore()
	ctx := auth.WithAuthContext(context.Background(), auth.AuthContext{
		UserID: "alice",
		Roles:  []string{"viewer"},
	})
	req := &pb.ListObjectsRequest{Prefix: "/tenants/acme/../secrets"}

	err := runListObjectsAuthz(ctx, req, store)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}
