# Client SDKs (Thin)

## Go thin client (`file-engine/client`)

Provides minimal external-consumer helpers for:
- HTTP: readiness + upload lifecycle.
- gRPC: create-folder + task-status methods.

Key files:
- `file-engine/client/http_client.go`
- `file-engine/client/grpc_client.go`
- `file-engine/examples/client/go_http_example.go`

## PHP thin client wrapper (backend)

Backend now uses a dedicated thin HTTP wrapper for engine calls:
- `backend/app/Clients/FileEngineClient.php`
- consumed by `backend/app/Services/FileEngineService.php`

This keeps request wiring/versioned paths centralized while preserving existing controller/service contracts.

## Compatibility tests

Golden compatibility fixtures for stable public response contracts live in:
- `file-engine/internal/server/testdata/compat/`

Compatibility check command:

```bash
./scripts/check-api-compatibility.sh
```
