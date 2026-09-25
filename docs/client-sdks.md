# Client SDKs (Thin)

## Go thin client (`file-engine/client`)

Provides minimal external-consumer helpers for:
- HTTP: readiness + upload lifecycle.
- gRPC: create-folder + task-status methods.

Key files:
- `file-engine/client/http_client.go`
- `file-engine/client/grpc_client.go`
- `file-engine/examples/client/go_http_example.go`

### Typed errors + retry ergonomics (Go)

The HTTP client now returns a typed `*client.APIError` for non-2xx responses.

- Parse safely: `apiErr, ok := client.AsAPIError(err)`
- Retry only temporary/retryable failures: `apiErr.Temporary()`
- Use helper wrapper: `DoWithRetry(ctx, op, RetryOptions{...})`

Recommended default retry profile:
- max attempts: 3
- base delay: 200ms
- max delay: 2s
- retry only when `apiErr.Temporary()` is true

## PHP thin client wrapper (backend)

Backend uses a dedicated thin HTTP wrapper for engine calls:
- `backend/app/Clients/FileEngineClient.php`
- consumed by `backend/app/Services/FileEngineService.php`

### Typed errors + retry guidance (PHP)

`FileEngineClient` exposes strict variants (`postOrThrow`, `getOrThrow`, `putRawOrThrow`) that raise `App\Clients\FileEngineException` with:
- `status`
- `codeValue`
- `reason`
- `retryable`

Recommended consumer retry policy:
- retry only when `$e->isRetryable()` is true
- use bounded retries (3 attempts) with exponential backoff (e.g., 200ms, 400ms, 800ms)

## Compatibility tests

Golden compatibility fixtures for stable public response contracts live in:
- `file-engine/internal/server/testdata/compat/`

Coverage includes:
- upload lifecycle (`initiate`, `complete`)
- readiness endpoint (`/readyz`)
- authz deny envelope
- upload throttling envelope
- governance retention block response

Compatibility check command:

```bash
./scripts/check-api-compatibility.sh
```
