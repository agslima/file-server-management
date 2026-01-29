# JWT Integration (HTTP + gRPC)

## Configuration
Set either:
- `JWT_SECRET` (HMAC) OR
- `JWT_PUBLIC_KEY_PEM` (RSA public key PEM)

Optional:
- `JWT_ISSUER`
- `JWT_AUDIENCE`

## Claims mapping
- `sub` → `AuthContext.UserID`
- `roles[]` → `AuthContext.Roles`

## HTTP middleware
Wrap the HTTP mux/handler with JWT middleware to inject AuthContext into the request context.

## gRPC interceptor
Use a unary interceptor to read `authorization` metadata and inject AuthContext into the gRPC context.
