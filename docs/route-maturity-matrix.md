# Route Maturity Matrix

[//]: # (owner: API/Contract)
[//]: # (review_cadence: Per release)
[//]: # (last_reviewed: 2026-02-19)


This matrix summarizes API route maturity by transport, links each route to claim IDs, and points to runnable validations.

## Baseline routes (implemented)

| Route | Transport | Maturity | Claim IDs | Runnable validation |
| :-- | :-- | :--: | :-- | :-- |
| `POST /v1/folders` | HTTP/JSON (gRPC-Gateway) | ✅ baseline | `CL-003`, `CL-013` | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` and `cd file-engine && go test ./internal/server -run TestGatewayCreateFolderAndGetTaskStatusRoutes -v` |
| `GET /v1/tasks/{taskId}` | HTTP/JSON (gRPC-Gateway) | ✅ baseline | `CL-004`, `CL-013` | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` and `cd file-engine && go test ./internal/server -run TestGatewayCreateFolderAndGetTaskStatusRoutes -v` |
| `ListObjects` | gRPC | ✅ baseline | `CL-012` | `cd file-engine && go test ./internal/handlers -run "TestListObjectsReturnsEntries|TestListObjectsRequiresAuthContext|TestListObjectsRejectsUnauthorizedTenant" -v` |
| `DownloadObject` | gRPC streaming | ✅ baseline | `CL-012` | `cd file-engine && go test ./internal/handlers -run "TestDownloadObjectRejectsUnauthorizedTenant" -v && cd file-engine && go test ./internal/server -run "TestHandleDownloadNormalizesPath|TestHandleDownloadRejectsTraversal" -v` |

## Target-state routes (documented, not baseline-validated)

| Route | Transport | Maturity | Claim IDs | Promotion rule |
| :-- | :-- | :--: | :-- | :-- |
| `POST /v1/uploads:initiate` | HTTP/JSON (gRPC-Gateway) | 🔒 target-state | Future claim ID (TBD) | Add claim ID + runnable command in `docs/capability-ledger.md`, then promote from target-state. |
| `POST /v1/uploads/{uploadId}:complete` | HTTP/JSON (gRPC-Gateway) | 🔒 target-state | Future claim ID (TBD) | Add claim ID + runnable command in `docs/capability-ledger.md`, then promote from target-state. |

## Backend control-plane vertical slice track

To evolve backend from scaffold safely, ship one end-to-end workflow at a time.

| Slice | Workflow | Status | Claim IDs | Validation |
| :-- | :-- | :--: | :-- | :-- |
| `VS-001` | Laravel control-plane request validation + forwarding for create-folder and task status polling | 🟡 in progress | `CL-008`, `CL-018` | `cd backend && composer validate --strict` and `cd backend && php -l app/Http/Controllers/FolderController.php && php -l app/Http/Controllers/TaskController.php && php -l app/Services/FileEngineService.php` |

When `VS-001` gains executable backend tests, move its validation command(s) into the baseline table and update this matrix.
