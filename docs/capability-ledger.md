# Capability Ledger (Week 4 Alignment)

This ledger maps every **implemented baseline claim** to a runnable validation command.

## How to use

- Run commands from repository root unless noted.
- If a command fails, the corresponding capability claim should be treated as unverified.

| Capability claim | Status | Runnable validation | Expected result |
| :-- | :--: | :-- | :-- |
| Canonical proto mirror is synchronized (`api/proto` -> `proto`) | ✅ | `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto` | Exit code `0` |
| File Engine baseline modules compile/test in current baseline scope | ✅ | `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v` | Tests pass (packages may report `[no test files]`) |
| Async create-folder flow works end-to-end (enqueue -> worker -> folder created) | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | `PASS` for `TestAsyncCreateFolderFlow` |
| Task status persistence for async flow is present (`queued`/`success` with structured payload fields) | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | Test asserts status/correlation/message persistence |
| Basic audit task events are emitted for async flow | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | Test asserts audit success event emission |
| Correlation IDs are propagated from request metadata to task processing logs/status | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` | Test asserts correlation ID persistence |
| Known-working local baseline script remains green | ✅ | `./file-engine/scripts/dev.sh` | Script completes with `[dev] all checks passed` |
| Backend scaffold baseline remains valid (composer metadata) | ✅ | `cd backend && composer validate --strict` | Exit code `0` |
| Frontend is intentionally placeholder (no Node runtime scaffold yet) | 🔒 | `test -f frontend/README.md && test ! -f frontend/package.json` | Exit code `0` |

## Non-baseline / target-state claims

The following themes are intentionally documented as target-state and are **not** baseline-validated in CI yet:

- Enterprise identity integrations (AD/LDAP/OIDC broker)
- Malware-gated upload promotion pipeline end-to-end
- Full observability stack (OpenTelemetry backend export + alerting)
- Immutable external audit sink integration

Track these in roadmap milestones before promoting them to baseline claims.
