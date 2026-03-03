# AGENTS.md

Project: File Server Management (PHP + Go hybrid skeleton)

## Canonical docs (read first)

- Project overview/status: `README.md`
- Capability truth table: `docs/capability-ledger.md`
- Setup/onboarding: `docs/setup.md`
- File-engine scoped guide: `file-engine/AGENTS.md`
- Backend scoped guide: `backend/AGENTS.md`

If guidance conflicts, use this precedence order: capability ledger -> setup -> scoped AGENTS -> architecture deep-dives.

## High-level flow

User -> Frontend -> Backend (Laravel control-plane) -> File Engine (Go API + worker) -> Storage backend.

File Engine is the final execution boundary for async filesystem mutations.

## Repository layout

- `backend/` Laravel API scaffold
- `file-engine/` Go API, worker, proto, storage/auth components
- `frontend/` UI placeholder
- `docs/` architecture/setup/security/api references
- `docker-compose.yml` root compose for multi-service local stack
- `file-engine/docker-compose.yml` compatibility mirror for file-engine-local workflows

## Baseline validation commands

Run from repository root unless noted:

- `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto`
- `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v`
- `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v`
- `./file-engine/scripts/dev.sh`
- `cd backend && composer validate --strict && php -l app/Http/Controllers/FolderController.php && php -l app/Http/Controllers/TaskController.php && php -l app/Services/FileEngineService.php`
- `cd backend && ./scripts/smoke.sh`
- `docker compose build --pull && docker compose up -d && ./scripts/wait-for-http.sh http://localhost:8080/healthz 120 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 120 && ./scripts/e2e/vs001_create_folder.sh && docker compose down -v`
- `test -f frontend/README.md && test ! -f frontend/package.json`

## Current alignment notes (validated)

- Proto mirror sync is expected to hold (`cmp` check).
- File-engine baseline and integration checks are expected to pass via `scripts/dev.sh`.
- Backend has a baseline-validated VS-001 slice (scaffold-plus-forwarding + polling validation), but is not feature-complete.
- Frontend is intentionally placeholder (README present, no `package.json`).

## Known gaps to keep in mind

- Capability maturity truth is `docs/capability-ledger.md`; current promoted baseline spans `CL-001`..`CL-073` with no open target-state exclusions as of 2026-02-19.
- HTTP/JSON via gRPC-Gateway is baseline for `CreateFolder`, `GetTaskStatus`, and upload lifecycle routes (`/v1/uploads:initiate`, `/v1/uploads/{uploadId}:chunk`, `/v1/uploads/{uploadId}:complete`).
- `file-engine/docker-compose.yml` is a compatibility mirror and should not override canonical setup guidance.
- Root `docker-compose.yml` is the canonical developer compose entry point.

## Conventions

- Go: keep imports/module usage clean, run `gofmt`, and preserve error context.
- PHP/Laravel: follow PSR-4 and valid config return structures.
- API contract: keep proto, generated artifacts, and API docs aligned.
- Capability promotion: do not mark features as baseline in top-level docs without a claim ID + runnable validation in `docs/capability-ledger.md`.
