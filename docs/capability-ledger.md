# Capability Ledger

This ledger is the canonical claim-to-validation source for the repository.

## How to use

- Run commands from repository root unless a command explicitly changes directories.
- If a validation command fails, treat that claim as **unverified**.
- Claims marked **target-state** are intentionally excluded from current baseline CI.

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
| `CL-008` | Backend scaffold baseline remains valid (composer metadata) | 🟡 | `cd backend && composer validate --strict` | Exit code `0` |
| `CL-009` | Frontend is intentionally placeholder-level (no Node runtime scaffold yet) | 🔒 | `test -f frontend/README.md && test ! -f frontend/package.json` | Exit code `0` |
| `CL-010` | Structured JSON logs with correlation IDs and baseline queue/task metrics exposure are wired for API + worker | ✅ | `cd file-engine && go test ./internal/handlers ./internal/observability -v` | Tests pass; handler logs include `correlation_id` and metrics snapshot assertions pass |
| `CL-011` | Documentation link/path drift check remains green | ✅ | `./scripts/doc-drift-check.sh` | Script completes with `doc drift check passed` |
| `CL-012` | Read-path behavior (list results + size/timestamps/ownership metadata + download) enforces path normalization + ACLs | ✅ | `cd file-engine && go test ./internal/handlers -run TestListObjectsReturnsEntries -v && go test ./internal/adapters/storage/local -run TestLocalStorageListMetadata -v && go test ./internal/authz -run "TestGRPCAuthZInterceptorListObjects" -v && go test ./internal/server -run "TestHandleDownloadNormalizesPath|TestHandleDownloadRejectsTraversal" -v` | Tests pass |
| `CL-013` | HTTP gateway routes for CreateFolder + GetTaskStatus are generated and respond | ✅ | `cd file-engine && go test ./internal/server -run TestGatewayCreateFolderAndGetTaskStatusRoutes -v` | Tests pass |
| `CL-014` | AuthZ precedence (ACL vs RBAC) behaves as specified | ✅ | `cd file-engine && go test ./internal/auth -run "TestRBACFallback|TestUserACLOverridesRBAC|TestACLPathInheritance|TestUserDenyPrecedesRoleAllowAndRBAC|TestRoleDenyPrecedesRoleAllowAtSamePath|TestClosestPathACLWinsBeforeParentACLs|TestUserACLPrecedenceOnSamePath|TestUserACLWithoutPermissionFallsThroughToRoleACL" -v` | Tests pass |
| `CL-015` | Path normalization guarantees (traversal rejection + canonicalization) | ✅ | `cd file-engine && go test ./internal/authz -run "TestExtractPathNormalizesCreateFolder|TestExtractPathRejectsTraversal|TestNormalizePathHandlesWindowsAndWhitespace|TestNormalizePathAllowsDotContainingNames|TestTenantFromPath|TestTenantFromPathRejectsNonTenantRoot" -v` | Tests pass |
| `CL-016` | Generated gateway artifacts are in sync with proto | ✅ | `cd file-engine && ./scripts/generate_grpc_docker.sh && cd .. && git diff --exit-code && test -z "$(git status --porcelain)"` | No diff after generation |

## Domain capability ledger (by area)

This section summarizes capability status by domain with ownership, acceptance tests, and target milestones. Target milestones reference the roadmap phases in `README.md`.

| Domain | Current state | Owner | Acceptance tests | Target milestone |
| :-- | :-- | :-- | :-- | :-- |
| AuthZ | Baseline RBAC + path-based ACL with inheritance is implemented; tenant scoping enforced at File Engine boundary. | Unassigned (TBD) | `cd file-engine && go test ./internal/auth -v` | Phase 1 (read authz baseline) |
| Tasks | Async task enqueue + worker execution + status persistence validated for create-folder flow. | Unassigned (TBD) | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | Phase 2 (folder creation + audit) |
| Uploads | Quarantine → scan → promote is target-state only; no baseline validation yet. | Unassigned (TBD) | TBD (promote to ledger when runnable) | Phase 3 (upload + scan + observability) |
| Audit | Baseline task audit events are emitted for async folder flow; external sink is target-state. | Unassigned (TBD) | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | Phase 2 baseline; Phase 3 for sink |
| Observability | Structured logs + queue/task metrics baseline in place; OTEL export is target-state. | Unassigned (TBD) | `cd file-engine && go test ./internal/handlers ./internal/observability -v` | Phase 3 (observability baseline) |

## Target-state claims (documented, not baseline-validated)

The following areas remain target-state and are not currently baseline-gated by CI:

- Enterprise identity integrations (AD/LDAP/OIDC broker)
- Malware-gated upload promotion pipeline end-to-end
- Full OpenTelemetry backend export + alerting pipeline
- Immutable external audit sink integration

Promote these to baseline only when each has a dedicated runnable validation command.
