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

type authzServerStream struct {
	grpc.ServerStream
	authCtx auth.AuthContext
	perm    auth.Permission
	store   auth.ACLStore
	checked bool
}

func (s *authzServerStream) RecvMsg(m any) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	if !s.checked {
		s.checked = true
		path, err := ExtractPath(m)
		if err != nil {
			return status.Error(codes.InvalidArgument, "cannot extract path")
		}
		if !auth.CanAccess(s.authCtx, path, s.perm, s.store) {
			return status.Error(codes.PermissionDenied, "access denied")
		}
	}
	return nil
}

func GRPCAuthZStreamInterceptor(store auth.ACLStore) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		perm, ok := MethodPermission[info.FullMethod]
		if !ok {
			return status.Error(codes.PermissionDenied, "no permission mapping for method")
		}

		a, ok := auth.FromContext(stream.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "missing auth context")
		}

		wrapped := &authzServerStream{
			ServerStream: stream,
			authCtx:      a,
			perm:         perm,
			store:        store,
		}
		return handler(srv, wrapped)
	}
}
