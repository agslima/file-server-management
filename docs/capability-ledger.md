# Capability Ledger

This ledger is the source of truth for **what is currently implemented** and **how each claim is validated**.

## Domain capability matrix

| Domain | Current state | Owner | Acceptance tests / runnable validation | Target milestone |
| :-- | :-- | :-- | :-- | :-- |
| AuthZ (JWT + RBAC/ACL enforcement at File Engine boundary) | **Baseline implemented** for the CreateFolder slice: handler requires auth context and authz interceptors remain in server wiring. | File Engine team (`file-engine/` maintainers) | `cd file-engine && go test ./internal/handlers -run TestCreateFolderRequiresAuthContext -v`<br>`cd file-engine && go test ./internal/auth -v` | M1 - Thin vertical slice hardened (complete) |
| Tasks (async enqueue/consume + status retrieval) | **Baseline implemented** for create-folder flow: queue enqueue, worker consume, persisted task status (`queued/success/failed`), and `GetTaskStatus`. | File Engine team (`file-engine/` maintainers) | `cd file-engine && go test ./internal/handlers -run TestGetTaskStatusRequiresAuthAndReturnsPersistedStatus -v`<br>`cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | M1 - Thin vertical slice hardened (complete) |
| Uploads (initiate/complete API and execution pipeline) | **Scaffold only / partial**: backend endpoints and service proxy exist; full malware-gated async upload pipeline is not baseline-validated. | Backend team (`backend/`) + File Engine team (`file-engine/`) | `cd backend && composer validate --strict`<br>`php -l backend/app/Http/Controllers/UploadController.php` | M2 - Upload flow parity with folder slice |
| Audit (task lifecycle emission + durable audit sink) | **Baseline partial**: task lifecycle emits audit-style log events in worker/API path; durable append-only sink integration remains target state. | File Engine team (`file-engine/`) | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` (asserts audit event emission in test flow) | M2 - Durable audit persistence |
| Observability (correlation IDs + operational signals) | **Baseline partial**: request correlation IDs (`x-request-id`/`x-correlation-id`) propagate into task payload/status/logs for folder flow; metrics/alerts/tracing are still target-state. | Platform + File Engine teams | `cd file-engine && go test ./internal/handlers -run TestCreateFolderEnqueuesWithCorrelationAndActorFallback -v`<br>`cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | M2 - Metrics/tracing baseline |

## Cross-cutting baseline checks

Use this set for quick regression checks from repository root:

```bash
cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto
./file-engine/scripts/dev.sh
cd backend && composer validate --strict
```

## Notes on status labels

- **Baseline implemented**: runnable and validated by at least one command above.
- **Baseline partial**: implemented in a narrow slice; broader target-state promises are not fully validated.
- **Scaffold only / partial**: API or structure exists, but end-to-end behavior is not yet validated to baseline standard.
