# Route Maturity Matrix

[//]: # (owner: API/Contract)
[//]: # (review_cadence: Per release)
[//]: # (last_reviewed: 2026-02-21)


This matrix summarizes API/operability route maturity by transport, links each route to claim IDs, and points to runnable validations.

## Baseline routes (implemented)

| Route | Transport | Maturity | Claim IDs | Runnable validation |
| :-- | :-- | :--: | :-- | :-- |
| `POST /v1/folders` | HTTP/JSON (gRPC-Gateway) | ✅ baseline | `CL-003`, `CL-013` | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` and `cd file-engine && go test ./internal/server -run TestGatewayCreateFolderAndGetTaskStatusRoutes -v` |
| `GET /v1/tasks/{taskId}` | HTTP/JSON (gRPC-Gateway) | ✅ baseline | `CL-004`, `CL-013` | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` and `cd file-engine && go test ./internal/server -run TestGatewayCreateFolderAndGetTaskStatusRoutes -v` |
| `POST /v1/uploads:initiate` | HTTP/JSON | ✅ baseline | `CL-047` | `docker compose up -d --build redis postgres file-engine file-engine-worker backend && ./scripts/wait-for-http.sh http://localhost:8080/healthz 120 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 120 && ./scripts/e2e/upload_lifecycle.sh && docker compose down -v` |
| `PUT /v1/uploads/{uploadId}:chunk` | HTTP/JSON | ✅ baseline | `CL-047` | `docker compose up -d --build redis postgres file-engine file-engine-worker backend && ./scripts/wait-for-http.sh http://localhost:8080/healthz 120 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 120 && ./scripts/e2e/upload_lifecycle.sh && docker compose down -v` |
| `POST /v1/uploads/{uploadId}:complete` | HTTP/JSON | ✅ baseline | `CL-047` | `docker compose up -d --build redis postgres file-engine file-engine-worker backend && ./scripts/wait-for-http.sh http://localhost:8080/healthz 120 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 120 && ./scripts/e2e/upload_lifecycle.sh && docker compose down -v` |
| `GET /healthz` | HTTP/JSON | ✅ baseline | `CL-020`, `CL-047` | `docker compose up -d --build && ./scripts/wait-for-http.sh http://localhost:8080/healthz 120 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 120 && docker compose down -v` |
| `GET /readyz` | HTTP/JSON | ✅ baseline | `CL-036` | `cd file-engine && go test ./internal/server -run "TestHandleReadyzReturnsReadyWhenChecksPass|TestHandleReadyzReturnsServiceUnavailableWhenAnyCheckFails|TestHandleReadyzWithoutChecksReturnsDeterministicReadyPayload" -v` |
| `ListObjects` | gRPC | ✅ baseline | `CL-012` | `cd file-engine && go test ./internal/handlers -run "TestListObjectsReturnsEntries|TestListObjectsRequiresAuthContext|TestListObjectsRejectsUnauthorizedTenant" -v` |
| `DownloadObject` | gRPC streaming | ✅ baseline | `CL-012` | `cd file-engine && go test ./internal/handlers -run "TestDownloadObjectRejectsUnauthorizedTenant" -v && cd file-engine && go test ./internal/server -run "TestHandleDownloadNormalizesPath|TestHandleDownloadRejectsTraversal" -v` |

## Baseline identity profile route set

| Route/profile | Transport | Maturity | Claim IDs | Runnable validation |
| :-- | :-- | :--: | :-- | :-- |
| OIDC profile end-to-end (`/realms/file-engine/.well-known/openid-configuration` + protected engine call with server-side tenant mapping enforcement) | Docker compose profile + HTTP | ✅ baseline | `CL-042` | `./scripts/e2e/run_oidc_profile.sh` |

## Backend control-plane vertical slice track

To evolve backend from scaffold safely, ship one end-to-end workflow at a time.

| Slice | Workflow | Status | Claim IDs | Validation |
| :-- | :-- | :--: | :-- | :-- |
| `VS-001` | Laravel control-plane request validation + forwarding for create-folder and task status polling | ✅ baseline slice | `CL-008`, `CL-018`, `CL-020`, `CL-031` | `cd backend && composer validate --strict && ./scripts/smoke.sh` and `docker compose build --pull && docker compose up -d && ./scripts/wait-for-http.sh http://localhost:8080/healthz 120 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 120 && ./scripts/e2e/vs001_create_folder.sh && docker compose down -v` |

## Promotion discipline

If a new route is documented before baseline promotion, it must remain marked target-state until:

1. a claim ID exists in `docs/capability-ledger.md`,
2. a runnable validation command is documented,
3. and CI/PR evidence exists for that command.
