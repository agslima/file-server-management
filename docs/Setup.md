# Setup & Developer Onboarding (File Engine)

This guide explains how to run the File Engine locally for development (Docker Compose recommended), how to run it natively for debugging, and how to configure a production-like setup (RSA JWT + S3/GCS credentials + hardening).

---

## Day 1 Checklist (Fast Onboarding)

1) **Clone + bootstrap**
```bash
git clone <repo>
cd <repo>
cp .env.example .env
```

2) **Start dependencies**
```bash
docker compose up -d postgres redis
```

3) **Apply migrations**
```bash
export POSTGRES_DSN="postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
go run ./cmd/migrate
```

4) **Run API + Worker (Docker)**
```bash
docker compose up --build
```

5) **Smoke test**
- Check HTTP port is up (or a real endpoint if `/health` is not implemented):
```bash
curl -i http://localhost:8080/health
```

6) **Run unit tests**
```bash
go test ./... -v
```

---

## 1. System Requirements

### Supported OS
- macOS (Intel/Apple Silicon)
- Linux (Ubuntu/Debian/Fedora)
- Windows 11 via WSL2 (recommended)

### Required CLI Tools
- **Git** (2.30+)
- **Go** (1.21+) — builds, tests, codegen, migrations
- **Docker Engine / Docker Desktop** (24+)
- **Docker Compose v2**
- **curl**

### Recommended Tools
- **grpcurl** (gRPC testing)
- **psql** (Postgres troubleshooting)
- **jq** (JSON formatting)

### Cloud SDKs (only if using S3/GCS locally)
- **AWS CLI** (optional)
- **Google Cloud SDK** (optional)

---

## 2. Repository Layout (Quick Orientation)

- `cmd/file-engine/` – API server entrypoint
- `cmd/worker/` – background worker entrypoint
- `cmd/migrate/` – simple migration runner (local/dev)
- `internal/auth/` – JWT + RBAC + ACL resolver
- `internal/storage/` – storage abstraction + backend factory
- `internal/adapters/storage/{local,s3,gcs}/` – backend implementations
- `db/migrations/` – PostgreSQL migrations (ACL persistence)
- `docker-compose.yml` – local dev stack
- `build/docker/` – Dockerfiles

---

## 3. Configuration

### 3.1 .env files

- `.env.example` is the **committed template**
- `.env` is your **local override** (do **not** commit)

Create `.env`:
```bash
cp .env.example .env
```

### 3.2 Environment Variables (Reference)

The project can run in **three modes** (storage):
- `local` (filesystem mounted volume)
- `s3` (AWS S3 / MinIO)
- `gcs` (Google Cloud Storage)

JWT auth supports:
- HMAC secret (easy for dev)
- RSA public key (recommended for production)

See `.env.example` in the repo root for the full matrix.

---

## 4. Docker Setup (Recommended)

### 4.1 Start dependencies (Redis + Postgres)
```bash
docker compose up -d postgres redis
```

### 4.2 Apply migrations (ACL tables)
```bash
export POSTGRES_DSN="postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
go run ./cmd/migrate
```

### 4.3 Run the full stack (API + Worker)
```bash
docker compose up --build
```

### 4.4 Ports
- HTTP: `http://localhost:8080`
- gRPC: `localhost:50051`
- Redis: `localhost:6379`
- Postgres: `localhost:5432`

---

## 5. Local Execution (Without Containers)

This is ideal for debugging with breakpoints.

### 5.1 Start Redis/Postgres via Docker
```bash
docker compose up -d postgres redis
```

### 5.2 Apply migrations
```bash
export POSTGRES_DSN="postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
go run ./cmd/migrate
```

### 5.3 Run API locally
```bash
export REDIS_ADDR="localhost:6379"
export POSTGRES_DSN="postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
export STORAGE_BACKEND="local"
export FILE_BASE_ROOT="./data"
export JWT_SECRET="dev-secret"
go run ./cmd/file-engine
```

### 5.4 Run Worker locally (second terminal)
```bash
export REDIS_ADDR="localhost:6379"
export POSTGRES_DSN="postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
export STORAGE_BACKEND="local"
export FILE_BASE_ROOT="./data"
go run ./cmd/worker
```

---

## 6. Service Architecture & Connectivity

| Service | Responsibility | Default Port | Host (Compose) | Host (Local) |
|---|---|---:|---|---|
| file-engine (API) | gRPC + HTTP gateway, auth, task enqueue | 8080 / 50051 | `file-engine` | `localhost` |
| worker | consumes tasks, executes storage ops | — | `worker` | `localhost` |
| redis | queue/broker | 6379 | `redis` | `localhost` |
| postgres | ACL persistence | 5432 | `postgres` | `localhost` |

**Data flow**
1) Client → API (JWT)  
2) API → RBAC/ACL check → enqueue Redis task  
3) Worker → Storage backend executes operation  
4) Client polls task status (if enabled)

---

## 7. Storage Backend Setup

### 7.1 Local (default)
```bash
export STORAGE_BACKEND=local
export FILE_BASE_ROOT=./data
```

### 7.2 S3 (AWS or MinIO)
```bash
export STORAGE_BACKEND=s3
export S3_BUCKET=my-bucket
export S3_REGION=us-east-1
export S3_PREFIX=file-engine
# For MinIO:
# export S3_ENDPOINT=http://localhost:9000
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
```

### 7.3 GCS
```bash
export STORAGE_BACKEND=gcs
export GCS_BUCKET=my-bucket
export GCS_PREFIX=file-engine
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
```

---

## 8. Production-like Setup (Recommended Defaults)

This section describes a setup closer to real production, even for staging environments.

### 8.1 JWT: prefer RSA verification (no shared HMAC secret)

1) Generate an RSA keypair (example):
```bash
openssl genrsa -out jwt_private.pem 2048
openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem
```

2) Configure verifier:
```bash
export JWT_PUBLIC_KEY_PEM="$(awk '{printf "%s\\n",$0}' jwt_public.pem)"
export JWT_ISSUER="your-issuer"
export JWT_AUDIENCE="file-engine"
```

3) Ensure clients mint tokens signed by your private key with:
- `sub`
- `roles[]`
- `iss`
- `aud`
- `exp`

> Note: the service only needs the **public key**. The private key stays in your identity provider / auth service.

### 8.2 Storage: use cloud IAM instead of static creds

**S3**
- Prefer instance roles / IRSA (EKS) / workload identity equivalents.
- Avoid `AWS_ACCESS_KEY_ID` in long-lived env vars when possible.

**GCS**
- Prefer Workload Identity / service account bindings.
- Use `GOOGLE_APPLICATION_CREDENTIALS` only for local/dev.

### 8.3 Hardening & operational defaults (recommended)
- Run with non-root containers (where possible)
- Enforce strict CORS (if public-facing)
- Add rate limiting (429) on HTTP gateway
- Structured logging with request IDs
- Persist task status to DB for auditability (future improvement)

---

## 9. Troubleshooting

### 401 Unauthorized
- Missing `Authorization: Bearer <JWT>`
- Wrong `JWT_SECRET` / `JWT_PUBLIC_KEY_PEM`
- Missing `sub` claim

### 403 Forbidden
- ACL/RBAC denies permission on path

### Redis/Postgres connectivity
- Local runs should use `localhost`
- Compose runs should use service names (`redis`, `postgres`)

---

## 10. References
- API docs: `docs/api-reference.md`
- Auth model: `docs/auth.md`
- Errors: `docs/errors.md`
- Storage: `docs/STORAGE_BACKENDS.md`
- JWT wiring: `docs/JWT_INTEGRATION.md`
