# AGENTS.md

Project: File Server Management (multi-service skeleton)

Repository layout
- backend/          Laravel API (skeleton)
- file-engine/      Go services (file-engine server, worker, gRPC/gateway, proto tooling)
- frontend/         Web UI placeholder (no package.json yet)
- docs/             OpenAPI spec + Postman collection
- docker/           Nginx config + alternate compose
- docker-compose.yml Root compose (currently outdated; see notes)

Go entrypoints (file-engine/)
- cmd/file-engine/  Main Go service (HTTP + gRPC stubs)
- cmd/worker/       Background worker (Redis + local FS)
- cmd/server/       gRPC server stub (expects generated code)
- cmd/gateway/      gRPC-Gateway stub (expects generated code)
- cmd/main.go       Legacy stub; avoid using unless you wire deps + go.mod

Proto / gRPC
- Source used by build scripts: file-engine/api/proto/fileengine.proto
- There is a second proto at file-engine/proto/fileengine.proto (currently divergent). Reconcile before generating.
- Generate code: from file-engine/ run `make proto` or `./scripts/generate_grpc.sh`.
- Generated output: file-engine/pkg/generated/

Build/test (after dependencies are aligned)
- file-engine/: `go test ./...` (fails until go.mod + package duplicates are fixed)
- backend/: `composer install`, then `php artisan test`
- frontend/: `npm install`, `npm run build` (frontend is placeholder)

Key env vars referenced in code
- FILE_ENGINE_URL (backend -> file-engine)
- REDIS_ADDR (Go worker + server)
- FILE_BASE_ROOT
- LOG_LEVEL
- GRPC_ADDR
- HTTP_ADDR

Known alignment gaps (important for agents)
- Root docker-compose.yml references file-engine-api/ and file-engine-worker/ which do not exist.
- docker/docker-compose.yml references file-engine-go/ which does not exist.
- backend/config/services.php is not a valid Laravel config file (missing PHP array wrapper).
- file-engine/go.mod lacks required dependencies for the current imports.
- internal/worker/ has duplicate Task/Queue/Processor types; `go test` will not compile until deduped.

Conventions
- Go: keep module imports consistent with go.mod, run gofmt, remove unused imports, handle errors.
- PHP/Laravel: follow PSR-4 (App\\Services => app/Services); config files must return arrays.
- API: keep REST paths and payload fields in sync with docs/openapi.yaml.
