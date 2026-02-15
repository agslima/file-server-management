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

## Target-state claims (documented, not baseline-validated)

The following areas remain target-state and are not currently baseline-gated by CI:

- Enterprise identity integrations (AD/LDAP/OIDC broker)
- Malware-gated upload promotion pipeline end-to-end
- Full OpenTelemetry backend export + alerting pipeline
- Immutable external audit sink integration

Promote these to baseline only when each has a dedicated runnable validation command.
