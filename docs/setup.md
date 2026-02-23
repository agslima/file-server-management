# Setup & Developer Onboarding

This guide is the canonical setup reference for local development.

## Canonical startup paths

- **Baseline validation (supported):** run `./file-engine/scripts/dev.sh` from repository root.
- **File Engine local run (scaffold-level):** run Redis/Postgres in Docker, then run API/worker locally for debugging.
- **Primary compose entry point:** use repository-root `docker-compose.yml`.

---

## 1) Baseline validation (recommended)

From repository root:

```bash
./file-engine/scripts/dev.sh
```

This is the only baseline-validated quickstart today (see `docs/capability-ledger.md`).

---

## 2) File Engine local run (scaffold-level)

### Start dependencies

```bash
cd file-engine
docker compose up -d redis postgres
```

### Configure environment

```bash
export REDIS_ADDR="localhost:6379"
export POSTGRES_DSN="postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
export STORAGE_BACKEND="local"
export FILE_BASE_ROOT="$PWD/data"
export JWT_SECRET="dev-secret"
export TENANT_MEMBERSHIPS="dev-admin=dev-tenant"
# Optional governance policy (quota/retention/legal-hold/lifecycle):
# export GOVERNANCE_POLICY_FILE="$PWD/config/governance-policy.example.json"
```

### Apply migrations

```bash
go run ./cmd/migrate
```

### Run API + worker (separate terminals)

API terminal:

```bash
cd file-engine && go run ./cmd/file-engine
```

Worker terminal:

```bash
cd file-engine && go run ./cmd/worker
```

Dev JWT (HS256 with `JWT_SECRET=dev-secret`, `sub=dev-admin`, `roles=["admin"]`):

```bash
export JWT="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJkZXYtYWRtaW4iLCJyb2xlcyI6WyJhZG1pbiJdLCJleHAiOjQxMDI0NDQ4MDB9.Y-JdrUO96XS3odOeBWtYSIjPwR7z7g7IytvBLxTbCus"
```

Note: HTTP/JSON via gRPC-Gateway is baseline for `CreateFolder` and `GetTaskStatus`. Upload routes remain target-state. Use this path for debugging beyond baseline behavior.

---

## 3) Full stack quickstart (experimental)

From repository root:

```bash
docker compose up --build
```

This path is useful to validate container builds, but it is **not** a baseline-validated runtime. Use the baseline validation or File Engine local run for reproducible checks.

### OIDC profile (Keycloak) for enterprise identity testing

Use compose profile `oidc` to start a local IdP and validate JWT->actor normalization while still enforcing tenant access from server-side DB mappings:

```bash
docker compose --profile oidc up --build -d keycloak postgres redis file-engine file-engine-worker
```

Keycloak URL: `http://localhost:8082` (admin/admin). Realm import: `infra/keycloak/dev-realm.json`.

Identity lifecycle helpers:

```bash
./file-engine/scripts/seed_identity.sh
TOKEN='<admin-access-token>' ./file-engine/scripts/export_access_review.sh
TOKEN='<admin-access-token>' REPORT_MONTH='<YYYY-MM>' ./file-engine/scripts/generate_monthly_access_review_report.sh
```

### Observability profile (OTEL collector + Jaeger)

Start observability stack and core services:

```bash
docker compose --profile observability up --build -d otel-collector jaeger redis postgres file-engine file-engine-worker backend
```

Quick checks:

```bash
./scripts/wait-for-http.sh http://localhost:8080/metrics 120
./scripts/wait-for-http.sh http://localhost:16686 120
./scripts/drills/observability_incident_drill.sh
```

---

## 4) Notes on alternate compose files

- `file-engine/docker-compose.yml` is a compatibility mirror for file-engine-local workflows; the canonical source is root `docker-compose.yml`.
