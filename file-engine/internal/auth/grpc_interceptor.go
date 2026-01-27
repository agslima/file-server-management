package auth

import (
    "context"

    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/codes"
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
