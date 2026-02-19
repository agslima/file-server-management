# Capability Ledger

This ledger is the canonical claim-to-validation source for the repository.

## How to use

- Run commands from repository root unless a command explicitly changes directories.
- If a validation command fails, treat that claim as **unverified**.
- Claims marked **target-state** are intentionally excluded from current baseline CI.

## Promotion discipline (required)

- Do not present a capability as baseline in `README.md` or other top-level docs until it has:
  - a claim ID in this ledger,
  - a runnable validation command,
  - and matching CI/PR verification evidence.

## Baseline claims (implemented)

| Claim ID | Capability claim | Status | Runnable validation | Expected result |
| :-- | :-- | :--: | :-- | :-- |
| `CL-001` | Canonical proto mirror is synchronized (`file-engine/api/proto` -> `file-engine/proto`) | ✅ | `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto` | Exit code `0` |
| `CL-002` | File Engine baseline module checks compile in baseline scope | ✅ | `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v` | Tests pass (packages may report `[no test files]`) |
| `CL-003` | Async create-folder flow works end-to-end (enqueue -> worker -> folder created) | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | `PASS` for `TestAsyncCreateFolderFlow` |
| `CL-004` | Task status persistence is present for async flow (`queued -> running -> success`) | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | Test asserts transition history and persisted status payload |
| `CL-005` | Basic audit task events are emitted for async flow | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | Test asserts `task.processing` and `task.succeeded` events |
| `CL-006` | Correlation IDs are propagated in async flow state and logs | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | Test asserts persisted correlation ID; script output includes `correlation_id=` lines |
| `CL-007` | Known-working local baseline script remains green | ✅ | `./file-engine/scripts/dev.sh` | Script completes with `[dev] all checks passed` |
| `CL-008` | Backend scaffold baseline remains valid (composer metadata) | ✅ | `cd backend && composer validate --strict` | Exit code `0` |
| `CL-009` | Frontend is intentionally placeholder-level (no Node runtime scaffold yet) | 🔒 | `test -f frontend/README.md && test ! -f frontend/package.json` | Exit code `0` |
| `CL-010` | Structured JSON logs with correlation IDs and baseline queue/task metrics exposure are wired for API + worker | ✅ | `cd file-engine && go test ./internal/handlers ./internal/observability -v` | Tests pass; handler logs include `correlation_id` and metrics snapshot assertions pass |
| `CL-011` | Documentation link/path drift check remains green | ✅ | `./scripts/doc-drift-check.sh` | Script completes with `doc drift check passed` |
| `CL-012` | Read-path behavior (list results + size/timestamps/ownership metadata + download) enforces path normalization + final authz at File Engine boundary (tenant membership + ACL/RBAC) | ✅ | `cd file-engine && go test ./internal/handlers -run "TestListObjectsReturnsEntries|TestListObjectsRequiresAuthContext|TestListObjectsRejectsUnauthorizedTenant|TestDownloadObjectRejectsUnauthorizedTenant" -v && go test ./internal/adapters/storage/local -run TestLocalStorageListMetadata -v && go test ./internal/authz -run "TestGRPCAuthZInterceptorListObjects" -v && go test ./internal/server -run "TestHandleDownloadNormalizesPath|TestHandleDownloadRejectsTraversal" -v && go test -tags integration_authz ./tests/integration -run TestReadListBehaviorAndAuthzRejection -v` | Tests pass |
| `CL-013` | HTTP gateway routes for CreateFolder + GetTaskStatus are generated and respond | ✅ | `cd file-engine && go test ./internal/server -run TestGatewayCreateFolderAndGetTaskStatusRoutes -v` | Tests pass |
| `CL-014` | AuthZ precedence (ACL vs RBAC) behaves as specified | ✅ | `cd file-engine && go test ./internal/auth -run "TestRBACFallback|TestUserACLOverridesRBAC|TestACLPathInheritance|TestUserDenyPrecedesRoleAllowAndRBAC|TestRoleDenyPrecedesRoleAllowAtSamePath|TestClosestPathACLWinsBeforeParentACLs|TestUserACLPrecedenceOnSamePath|TestUserACLWithoutPermissionFallsThroughToRoleACL" -v` | Tests pass |
| `CL-015` | Path normalization guarantees (traversal rejection + canonicalization) | ✅ | `cd file-engine && go test ./internal/authz -run "TestExtractPathNormalizesCreateFolder|TestExtractPathRejectsTraversal|TestNormalizePathHandlesWindowsAndWhitespace|TestNormalizePathAllowsDotContainingNames|TestTenantFromPath|TestTenantFromPathRejectsNonTenantRoot" -v` | Tests pass |
| `CL-016` | Generated gateway artifacts are in sync with proto | ✅ | `cd file-engine && ./scripts/generate_grpc_docker.sh && cd .. && git diff --exit-code && test -z "$(git status --porcelain)"` | No diff after generation |
| `CL-017` | Worker performance guardrails for async create-folder are enforced (bounded status retries + processing timeout) | ✅ | `cd file-engine && go test ./internal/app/tasks -run "TestWorkerRetriesStatusPersistence|TestWorkerMarksTaskFailedOnProcessingTimeout" -v` | Tests pass |
| `CL-018` | Backend vertical-slice VS-001 is explicitly tracked (create-folder control-plane forwarding + task status polling contract) with runnable scaffold validations | ✅ | `cd backend && ./scripts/smoke.sh` | Exit code `0` and `No syntax errors detected` for each file |
| `CL-020` | Backend VS-001 E2E integration validates backend forwarding to File Engine and task-status polling to success (docker-compose) | ✅ | `docker compose build --pull && docker compose up -d && ./scripts/wait-for-http.sh http://localhost:8080/healthz 120 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 120 && ./scripts/e2e/vs001_create_folder.sh && docker compose down -v` | Script exits `0`; output contains `task_status=success`, `folder_exists=true`, and deterministic `E2E_OK` |
| `CL-022` | Audit coverage: read/list/download emit `audit_events` records with correlation_id + actor + tenant + action | ✅ | `cd file-engine && go test ./tests/integration -run TestAuditEventsEmittedForReadListDownload -v` | `PASS`; test resets DB state (`TRUNCATE ... RESTART IDENTITY`) per test and asserts >= 3 persisted rows for (list, read, download) include correlation_id + tenant_id + actor_id + action + result |
| `CL-025` | Upload pipeline baseline: staged upload writes to quarantine, then atomic promote moves object to final path (no partial final objects) | ✅ | `cd file-engine && go test ./tests/integration -run TestStagedUploadAtomicPromote -v` | `PASS`; test asserts deterministic quarantine->promote sequencing where final object is absent during staging and only materializes after promotion completes |
| `CL-031` | Backend baseline smoke suite runs via dependency install + PHPUnit smoke checks | ✅ | `docker compose run --rm --no-deps backend sh -lc 'composer install --no-interaction && ./vendor/bin/phpunit -c phpunit.xml'` | Exit code `0`; PHPUnit reports `OK`/`PASS` |
| `CL-032` | Audit table append-only enforcement rejects UPDATE/DELETE for app DB user | ✅ | `cd file-engine && go test ./tests/integration -run TestAuditEventsAppendOnlyEnforced -v` | `PASS`; test inserts seed row, then asserts UPDATE/DELETE fail and row remains |
| `CL-033` | Upload malware gate enforces quarantine→scan→promote policy: dirty scan blocks promote, clean scan allows promote | ✅ | `cd file-engine && go test ./tests/integration -run TestUploadScanGateDirtyPreventsPromotion -v` | `PASS`; test asserts dirty verdict keeps file quarantined and final path absent |
| `CL-034` | Ledger baseline gate runs curated baseline validations in CI and blocks regressions | ✅ | `./scripts/ledger-baseline.sh` | Exit code `0`; CI `Ledger Baseline Gate` job passes |
| `CL-035` | Audit external sink delivery supports S3 WORM/Loki/SIEM adapters with retries + DLQ and publishes sink lag metric | ✅ | `cd file-engine && go test ./tests/integration -run TestAuditExternalSinkDeliveryWithDLQAndLagMetrics -v` | `PASS`; test validates S3 WORM adapter writes JSONL object, Loki payload delivery, SIEM retry->DLQ behavior, and `fileengine_audit_sink_lag_ms` updates |
| `CL-036` | `/readyz` executes deterministic DB+queue+storage dependency checks and returns stable per-check JSON state | ✅ | `cd file-engine && go test ./internal/server -run "TestHandleReadyzReturnsReadyWhenChecksPass|TestHandleReadyzReturnsServiceUnavailableWhenAnyCheckFails|TestHandleReadyzWithoutChecksReturnsDeterministicReadyPayload" -v` | `PASS`; tests assert deterministic check ordering and explicit status for `db`, `queue`, and `storage` checks |
| `CL-037` | Storage contract suite enforces shared backend behavior and passes for local baseline (S3/GCS optional via env) | ✅ | `cd file-engine && go test ./internal/adapters/storage/local -run TestLocalStorageContractSuite -v` | `PASS`; local backend satisfies create/write/open/list/move/delete contract; S3/GCS suite entrypoints are present and env-gated |

## Domain capability ledger (by area)

This section summarizes capability status by domain with ownership, acceptance tests, and target milestones. Target milestones reference the roadmap phases in `README.md`.

| Domain | Current state | Primary owner | Backup reviewer | Acceptance tests | Target milestone |
| :-- | :-- | :-- | :-- | :-- | :-- |
| AuthZ | Baseline RBAC + path-based ACL with inheritance is implemented; tenant scoping enforced at File Engine boundary. | @agslima (Agnaldo Silva Lima) | Security reviewer rotation (`docs/security-reviewers.md`) | `cd file-engine && go test ./internal/auth -v` | Phase 1 (read authz baseline) |
| Tasks | Async task enqueue + worker execution + status persistence validated for create-folder flow, with worker guardrails for bounded retries and processing timeout. | @agslima (Agnaldo Silva Lima) | Platform engineer rotation (`docs/platform-engineers.md`) | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` and `cd file-engine && go test ./internal/app/tasks -run "TestWorkerRetriesStatusPersistence|TestWorkerMarksTaskFailedOnProcessingTimeout" -v` | Phase 2 (folder creation + audit) |
| Uploads | Staged upload + atomic promote are baseline-validated, and scan-gated promotion guardrails (CL-033) are baseline-enforced for local storage semantics; real scanner integration remains target-state. | @agslima (Agnaldo Silva Lima) | Security + platform rotations (`docs/security-reviewers.md`, `docs/platform-engineers.md`) | `cd file-engine && go test ./tests/integration -run TestStagedUploadAtomicPromote -v` and `cd file-engine && go test ./tests/integration -run TestUploadScanGateDirtyPreventsPromotion -v` | Phase 3 (upload + scan + observability) |
| Audit | Baseline task audit events are emitted for async/read flows, append-only table enforcement is baseline-validated (CL-032), and external sink delivery (S3 WORM/Loki/SIEM + DLQ/lag metrics) is baseline-validated (CL-035). | @agslima (Agnaldo Silva Lima) | Security reviewer rotation (`docs/security-reviewers.md`) | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` and `cd file-engine && go test ./tests/integration -run "TestAuditEventsAppendOnlyEnforced|TestAuditExternalSinkDeliveryWithDLQAndLagMetrics" -v` | Phase 2 baseline; Phase 3 for sink |
| Observability | Structured logs + queue/task metrics baseline in place; OTEL export is target-state. | @agslima (Agnaldo Silva Lima) | Platform engineer rotation (`docs/platform-engineers.md`) | `cd file-engine && go test ./internal/handlers ./internal/observability -v` | Phase 3 (observability baseline) |
| Backend control-plane | Vertical-slice delivery model in place; VS-001 now includes docker-compose E2E forwarding + polling validation before broader backend expansion. | @agslima (Agnaldo Silva Lima) | Repo reviewer backup (`.github/OWNERS`) | `docker compose build --pull && docker compose up -d && ./scripts/wait-for-http.sh http://localhost:8080/healthz 120 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 120 && ./scripts/e2e/vs001_create_folder.sh && docker compose down -v` | Phase 2 (first validated control-plane slice) |

## Target-state claims (documented, not baseline-validated)

The following areas remain target-state and are not currently baseline-gated by CI:

- Enterprise identity integrations (AD/LDAP/OIDC broker)
- Malware-gated upload promotion pipeline end-to-end with **real scanner integration** (non-stub) + operational controls (DLQ/metrics/runbooks)
- Full OpenTelemetry backend export + alerting pipeline

Promote these to baseline only when each has a dedicated runnable validation command.
