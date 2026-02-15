# AGENTS.md

Project: File Server Management (PHP + Go hybrid skeleton)

## Canonical docs (read first)

- Project overview/status: `README.md`
- Capability truth table: `docs/capability-ledger.md`
- Setup/onboarding: `docs/setup.md`
- File-engine scoped guide: `file-engine/Agents.md`

If docs disagree, prefer: capability-ledger -> setup -> scoped guide -> older service READMEs.

## High-level flow

User -> Frontend -> Backend (Laravel control-plane) -> File Engine (Go API + worker) -> Storage backend.

File Engine is the final execution boundary for async filesystem mutations.

## Repository layout

- `backend/` Laravel API scaffold
- `file-engine/` Go API, worker, proto, storage/auth components
- `frontend/` UI placeholder
- `docs/` architecture/setup/security/api references
- `docker-compose.yml` root compose for multi-service local stack
- `docker/docker-compose.yml` legacy/alternate compose reference

## Baseline validation commands

Run from repository root unless noted:

- `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto`
- `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v`
- `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v`
- `./file-engine/scripts/dev.sh`
- `cd backend && composer validate --strict`
- `test -f frontend/README.md && test ! -f frontend/package.json`

## Current alignment notes (validated)

- Proto mirror sync is expected to hold (`cmp` check).
- File-engine baseline and integration checks are expected to pass via `scripts/dev.sh`.
- Backend is scaffold-level; composer metadata validation is used as baseline signal.
- Frontend is intentionally placeholder (README present, no `package.json`).

## Known gaps to keep in mind

- Some docs still describe target-state capabilities as implemented; verify with capability ledger commands before claiming support.
- HTTP/JSON via gRPC-Gateway is baseline for `CreateFolder` + `GetTaskStatus`; upload routes remain target-state.
- `docker/docker-compose.yml` is a legacy/alternate compose path and should not override canonical setup guidance.
- Root `docker-compose.yml` is experimental and not a baseline-validated runtime path.

## Conventions

- Go: keep imports/module usage clean, run `gofmt`, and preserve error context.
- PHP/Laravel: follow PSR-4 and valid config return structures.
- API contract: keep proto, generated artifacts, and API docs aligned.
