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
| `CL-020` | Backend VS-001 E2E integration validates backend forwarding to File Engine and task-status polling to success (docker-compose) | 🟡 | `docker compose up -d --build && ./scripts/wait-for-http.sh http://localhost:8080/healthz 60 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 60 && ./scripts/e2e/vs001_create_folder.sh && docker compose down -v` | Script exits `0`; output contains `task_status=success` and `folder_exists=true` |
| `CL-022` | Audit coverage: read/list/download emit `audit_events` records with correlation_id + actor + tenant + action | 🟡 | `cd file-engine && go test ./tests/integration -run TestAuditEventsEmittedForReadListDownload -v` | `PASS`; test asserts >= 3 audit rows exist for (list, download, read/metadata) and each contains correlation_id + tenant_id + actor_id |
| `CL-025` | Upload pipeline baseline: staged upload writes to quarantine, then atomic promote moves object to final path (no partial final objects) | 🟡 | `cd file-engine && go test ./tests/integration -run TestStagedUploadAtomicPromote -v` | `PASS`; test asserts (1) quarantine file exists during staging, (2) final object appears only after promote, (3) no partially-written final file is observable |
| `CL-031` | Backend baseline smoke suite runs via dependency install + PHPUnit smoke checks | ✅ | `cd backend && composer install --no-interaction && ./vendor/bin/phpunit -c phpunit.xml` | Exit code `0`; PHPUnit reports `OK`/`PASS` |
| `CL-032` | Audit table append-only enforcement rejects UPDATE/DELETE for app DB user | ✅ | `cd file-engine && go test ./tests/integration -run TestAuditEventsAppendOnlyEnforced -v` | `PASS`; test inserts seed row, then asserts UPDATE/DELETE fail and row remains |

## Domain capability ledger (by area)

This section summarizes capability status by domain with ownership, acceptance tests, and target milestones. Target milestones reference the roadmap phases in `README.md`.

| Domain | Current state | Primary owner | Backup reviewer | Acceptance tests | Target milestone |
| :-- | :-- | :-- | :-- | :-- | :-- |
| AuthZ | Baseline RBAC + path-based ACL with inheritance is implemented; tenant scoping enforced at File Engine boundary. | @agslima (Agnaldo Silva Lima) | Security reviewer rotation (`docs/security-reviewers.md`) | `cd file-engine && go test ./internal/auth -v` | Phase 1 (read authz baseline) |
| Tasks | Async task enqueue + worker execution + status persistence validated for create-folder flow, with worker guardrails for bounded retries and processing timeout. | @agslima (Agnaldo Silva Lima) | Platform engineer rotation (`docs/platform-engineers.md`) | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` and `cd file-engine && go test ./internal/app/tasks -run "TestWorkerRetriesStatusPersistence|TestWorkerMarksTaskFailedOnProcessingTimeout" -v` | Phase 2 (folder creation + audit) |
| Uploads | Staged upload + atomic promote baseline is validated for local storage semantics; malware-gated policy remains target-state hardening. | @agslima (Agnaldo Silva Lima) | Security + platform rotations (`docs/security-reviewers.md`, `docs/platform-engineers.md`) | `cd file-engine && go test ./tests/integration -run TestStagedUploadAtomicPromote -v` | Phase 3 (upload + scan + observability) |
| Audit | Baseline task audit events are emitted for async folder flow and append-only persistence guardrails are validated; external sink is target-state. | @agslima (Agnaldo Silva Lima) | Security reviewer rotation (`docs/security-reviewers.md`) | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` and `cd file-engine && go test ./tests/integration -run TestAuditEventsAppendOnlyEnforced -v` | Phase 2 baseline; Phase 3 for sink |
| Observability | Structured logs + queue/task metrics baseline in place; OTEL export is target-state. | @agslima (Agnaldo Silva Lima) | Platform engineer rotation (`docs/platform-engineers.md`) | `cd file-engine && go test ./internal/handlers ./internal/observability -v` | Phase 3 (observability baseline) |
| Backend control-plane | Vertical-slice delivery model in place; VS-001 now includes docker-compose E2E forwarding + polling validation before broader backend expansion. | @agslima (Agnaldo Silva Lima) | Repo reviewer backup (`.github/OWNERS`) | `docker compose up -d --build && ./scripts/wait-for-http.sh http://localhost:8080/healthz 60 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 60 && ./scripts/e2e/vs001_create_folder.sh && docker compose down -v` | Phase 2 (first validated control-plane slice) |

## Target-state claims (documented, not baseline-validated)

The following areas remain target-state and are not currently baseline-gated by CI:

- Enterprise identity integrations (AD/LDAP/OIDC broker)
- Malware-gated upload promotion pipeline end-to-end
- Full OpenTelemetry backend export + alerting pipeline
- Immutable external audit sink integration

Promote these to baseline only when each has a dedicated runnable validation command.
