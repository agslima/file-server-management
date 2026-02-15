# Setup & Developer Onboarding

This guide is the canonical setup reference for local development.

## Canonical startup paths

- **Full stack (root compose):** run from repository root using `docker-compose.yml`.
- **File Engine focused development:** run from `file-engine/` using `file-engine/docker-compose.yml`.

---

## 1) Full stack quickstart (repository root)

```bash
git clone <repo>
cd file-server-management
docker compose up --build
```

Services started by root compose include Redis, File Engine API/worker, Backend, and Nginx.

### Useful checks

```bash
curl -i http://localhost:8080/healthz
cd backend && composer validate --strict
./file-engine/scripts/dev.sh
```

---

## 2) File Engine focused setup

```bash
cd file-engine
docker compose up --build
```

This starts Redis + Postgres + File Engine API + worker using `file-engine/docker-compose.yml`.

### Validate baseline behavior

From repository root:

```bash
cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto
cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v
cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v
./file-engine/scripts/dev.sh
```

---

## 3) Native local run (without containerized API/worker)

Keep dependencies in Docker and run binaries locally for debugging.

```bash
cd file-engine
docker compose up -d redis postgres

export REDIS_ADDR="localhost:6379"
export POSTGRES_DSN="postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
export STORAGE_BACKEND="local"
export FILE_BASE_ROOT="./data"
export JWT_SECRET="dev-secret"
```

API terminal:

```bash
go run ./cmd/file-engine
```

Worker terminal:

```bash
go run ./cmd/worker
```

---

## 4) Notes on legacy compose

`docker/docker-compose.yml` is a legacy/alternate compose definition. It should not be treated as the primary onboarding path.

Use one of the canonical startup paths above unless you are intentionally validating legacy behavior.
