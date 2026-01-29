package auth

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCAuthInterceptor extracts `authorization` metadata and stores AuthContext in context.
func GRPCAuthInterceptor(verifier *JWTVerifier) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		vals := md.Get("authorization")
		if len(vals) == 0 {
			// Some gateways use "Authorization"
			vals = md.Get("Authorization")
		}
		if len(vals) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization")
		}

		a, err := verifier.ParseAuthContext(vals[0])
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = WithAuthContext(ctx, a)
		return handler(ctx, req)
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// GRPCStreamAuthInterceptor extracts `authorization` metadata and stores AuthContext in stream context.
func GRPCStreamAuthInterceptor(verifier *JWTVerifier) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(stream.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}
		vals := md.Get("authorization")
		if len(vals) == 0 {
			vals = md.Get("Authorization")
		}
		if len(vals) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization")
		}

		a, err := verifier.ParseAuthContext(vals[0])
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx := WithAuthContext(stream.Context(), a)
		return handler(srv, &wrappedServerStream{ServerStream: stream, ctx: ctx})
	}
}
