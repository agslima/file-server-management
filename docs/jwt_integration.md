# JWT Middleware + Claims -> AuthContext

This project supports extracting identity from JWT and producing `auth.AuthContext`:

- `sub` claim -> `AuthContext.UserID`
- `roles` claim -> `AuthContext.Roles`

## Configuration

Set either:

- `JWT_SECRET` (HMAC), OR
- `JWT_PUBLIC_KEY_PEM` (RSA public key PEM)

Optional (recommended):

- `JWT_ISSUER`
- `JWT_AUDIENCE`

## HTTP integration (grpc-gateway)

Wrap your HTTP handler/mux:

```go
verifier, _ := auth.NewJWTVerifier(cfg.JWTSecret, cfg.JWTPublicKeyPEM, cfg.JWTIssuer, cfg.JWTAudience)
handler := auth.HTTPAuthMiddleware(verifier, mux)
http.ListenAndServe(cfg.HTTPAddr, handler)
```

## gRPC integration

Register interceptor on the gRPC server:

```go
verifier, _ := auth.NewJWTVerifier(cfg.JWTSecret, cfg.JWTPublicKeyPEM, cfg.JWTIssuer, cfg.JWTAudience)

grpcServer := grpc.NewServer(
  grpc.UnaryInterceptor(auth.GRPCAuthInterceptor(verifier)),
)

pb.RegisterFileEngineServer(grpcServer, yourHandler)
```

## Accessing AuthContext in handlers

```go
a, ok := auth.FromContext(ctx)
if !ok { ... }
userID := a.UserID
roles := a.Roles
```

## Example JWT payload

```json
{
  "sub": "user-42",
  "roles": ["admin", "editor"],
  "iss": "your-issuer",
  "aud": "file-engine"
}
```
