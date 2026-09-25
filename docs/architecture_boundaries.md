# Architecture boundaries

This document defines enforceable package/module boundaries for the repo.

## File Engine boundaries

- `internal/app/*` → application orchestration: ports and use-cases.
- `internal/services/*` → domain services and governance/domain policy logic.
- `internal/adapters/*` → infra adapters (queue, storage, security, config).
- `internal/server/*`, `internal/handlers/*` → transport boundary (HTTP/gRPC).
- `internal/logger/*` → canonical structured logger package.

## Boundary rules

1. **Single logger implementation:** imports must use `internal/logger`; `internal/infra/logger` is deprecated and must not be reintroduced.
2. **Use-case vs domain split:** new app orchestration code should go under `internal/app/usecases` (or other `internal/app/*` packages), while domain behavior remains in `internal/services`.
3. **Transport isolation:** HTTP/gRPC handlers should orchestrate auth, decode/encode, and call services/use-cases; business policy should not be duplicated in handlers.
4. **Adapter boundary protection:** Transport packages (`internal/server/*`, `internal/handlers/*`) avoid direct storage/security adapter imports; queue wiring in `internal/handlers/grpc_handler.go` is the current scoped exception.
5. **Service boundary protection:** Service packages (`internal/services/*`) do not import transport packages.

## Conformance gate

`./scripts/architecture-conformance-check.sh` enforces core documentation and boundary invariants.
