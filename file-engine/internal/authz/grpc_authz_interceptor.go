package authz

import (
    "context"

    "github.com/example/file-engine/internal/auth"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

func GRPCAuthZInterceptor(store auth.ACLStore) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        perm, ok := MethodPermission[info.FullMethod]
        if !ok {
            return nil, status.Error(codes.PermissionDenied, "no permission mapping for method")
        }

        a, ok := auth.FromContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing auth context")
        }

        path, err := ExtractPath(req)
        if err != nil {
            return nil, status.Error(codes.InvalidArgument, "cannot extract path")
        }

        if !auth.CanAccess(a, path, perm, store) {
            return nil, status.Error(codes.PermissionDenied, "access denied")
        }

        return handler(ctx, req)
    }
}
