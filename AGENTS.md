# AGENTS.md

Project: File Server Management (PHP + Go hybrid skeleton)

High-level flow (from README + docs)
- User -> Frontend (React/Next.js) -> Backend (Laravel API gateway; auth/RBAC) -> File Engine (Go gRPC-first + REST/gateway; async tasks) -> Storage (local or remote backends).
- File Engine is intended to be gRPC source of truth with REST generated via gRPC-Gateway; worker executes filesystem tasks from Redis queue.

Repository layout
- backend/          Laravel API skeleton
- file-engine/      Go services (API server + worker + proto/openapi + docs)
- frontend/         Web UI placeholder (no package.json yet)
- docs/             OpenAPI specs, Postman collection, architecture PDF, file-engine technical doc
- docker/           Nginx config + alternate compose
- docker-compose.yml Root compose (currently outdated; see notes)

Go entrypoints (file-engine/)
- cmd/file-engine/  Main API server (gRPC + HTTP gateway)
- cmd/worker/       Background worker (Redis queue + filesystem ops)
- cmd/server/       gRPC server stub (expects generated code)
- cmd/gateway/      gRPC-Gateway stub (expects generated code)
- cmd/migrate/      Placeholder for future DB migrations
- cmd/main.go       Legacy stub; avoid using unless you wire deps + go.mod

Proto / gRPC / OpenAPI
- Primary proto used by build scripts: file-engine/api/proto/fileengine.proto
- Secondary proto: file-engine/proto/fileengine.proto (currently divergent; reconcile before generating)
- Generate code: from file-engine/ run `make proto` or `./scripts/generate_grpc.sh` (Docker-based script at `scripts/scripts/generate_grpc_docker.sh`)
- Generated output: file-engine/pkg/generated/
- OpenAPI: generated at file-engine/api/openapi/fileengine.yaml; repo also has docs/openapi.yaml, docs/openapi3.yaml, docs/openapi.json, docs/schemas.yaml

Worker behavior (from file-engine/internal/worker/README.md)
- Worker pops tasks from Redis list `tasks` and processes via FSProcessor.
- Expected env: REDIS_ADDR (default localhost:6379), FILE_BASE_ROOT (default /mnt/files).
- Example task payload: {"id":"tsk1","type":"create_folder","params":{"path":"projects/demo","folder":"newfolder"}}

Auth & storage docs (file-engine/docs)
- JWT integration: JWT_SECRET or JWT_PUBLIC_KEY_PEM; optional JWT_ISSUER and JWT_AUDIENCE.
- Storage backends behind common interface: STORAGE_BACKEND=local|s3|gcs with FILE_BASE_ROOT for local, S3_* for S3/MinIO, GCS_* for GCS.

Build/test (after dependencies are aligned)
- file-engine/: `go test ./...` (fails until go.mod + package duplicates are fixed)
- backend/: `composer install`, then `php artisan test`
- frontend/: `npm install`, `npm run build` (frontend is placeholder)

Key env vars referenced in code
- FILE_ENGINE_URL (backend -> file-engine)
- REDIS_ADDR
- FILE_BASE_ROOT
- STORAGE_BACKEND, S3_*, GCS_*
- JWT_SECRET or JWT_PUBLIC_KEY_PEM; optional JWT_ISSUER, JWT_AUDIENCE
- LOG_LEVEL
- GRPC_ADDR
- HTTP_ADDR

Known alignment gaps (important for agents)
- Root docker-compose.yml references file-engine-api/ and file-engine-worker/ which do not exist.
- docker/docker-compose.yml references file-engine-go/ which does not exist.
- Root README and docs diagrams still mention file-engine-go/ and older folder names.
- file-engine/README.md contains two divergent tree layouts; reconcile with actual directory structure.
- backend/config/services.php is not a valid Laravel config file (missing PHP array wrapper).
- file-engine/go.mod lacks required dependencies for the current imports.
- internal/worker/ has duplicate Task/Queue/Processor types; `go test` will not compile until deduped.

Conventions
- Go: keep module imports consistent with go.mod, run gofmt, remove unused imports, handle errors.
- PHP/Laravel: follow PSR-4 (App\\Services => app/Services); config files must return arrays.
- API: keep REST paths and payload fields in sync with docs/openapi*.yaml and file-engine/api/proto/fileengine.proto.
