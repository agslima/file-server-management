# API Storage + Automatic Authorization Enforcement

This increment wires the **same storage backend** (Local / S3 / GCS) into the API layer and enforces **RBAC + Path ACL** automatically for every gRPC call.

## Architecture

- JWT -> `auth.AuthContext` (user + roles)
- gRPC interceptor enforces method -> permission mapping and extracts the target path from each request
- Storage operations use `storage.Storage` (same factory config as the worker)

## Enforced Permissions

Mapping lives in `internal/authz/method_map.go`.

Example:
- `ListObjects` -> `list`
- `UploadObject` -> `write`
- `DownloadObject` -> `read`
- `CreateFolder` -> `write`

## Path extraction

`internal/authz/path_extract.go` extracts the target path from request messages and normalizes it.

## HTTP behavior

- REST endpoints are served via grpc-gateway for non-streaming calls
- Download is exposed as a raw HTTP endpoint at:
  - `GET /v1/objects:download?path=/...`
  - Enforced with the same ACL store and `read` permission

## Generate code

This repo expects protobuf code generation:

```bash
./scripts/generate_grpc.sh
# or with Docker
./scripts/generate_grpc_docker.sh
```

## Example curl

```bash
curl "http://localhost:8080/v1/objects?prefix=/tenants/1" -H "Authorization: Bearer <JWT>"
curl -X POST "http://localhost:8080/v1/objects:upload" -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" -d '{"path":"/tenants/1/a.txt","content":"aGVsbG8="}'
curl "http://localhost:8080/v1/objects:download?path=/tenants/1/a.txt" -H "Authorization: Bearer <JWT>"
```
