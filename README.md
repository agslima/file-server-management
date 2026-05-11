<div align="center">

<a name="back-to-top"></a>

# Server File Manager Platform (PHP + Go File Engine)

[//]: # (owner: Project Maintainers)
[//]: # (review_cadence: Quarterly)
[//]: # (last_reviewed: 2026-02-19)

[![CI](https://github.com/agslima/file-server-management/actions/workflows/ci.yml/badge.svg)](https://github.com/agslima/file-server-management/actions/workflows/ci.yml)
![Go Version](https://img.shields.io/badge/go-1.26+-yellowgreen)
![Laravel](https://img.shields.io/badge/laravel-10%2B-blue)
![gRPC](https://img.shields.io/badge/API-gRPC%20-4e6e6e)
[![Docs](https://img.shields.io/badge/docs-architecture%20%7C%20adr-green)](https://github.com/agslima/file-server-management/tree/main/docs)
![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)

<!--
![Go Tests](https://github.com/<org>/<repo>/actions/workflows/go-test.yaml/badge.svg)
![Laravel Tests](https://github.com/<org>/<repo>/actions/workflows/phpunit.yaml/badge.svg)
[![codecov](https://codecov.io/gh/<org>/<repo>/branch/main/graph/badge.svg)](https://codecov.io/gh/<org>/<repo>)
![Dependency Review](https://github.com/<org>/<repo>/actions/workflows/dependency-review.yml/badge.svg)
![Trivy](https://github.com/<org>/<repo>/actions/workflows/trivy.yml/badge.svg)
-->

⚡️ **A governance-focused, multi-tenant file management platform written in Go and PHP** ⚡️ \
Designed to operate directly on **real storage backends** (local/mounted SMB/NFS/SFTP, with adapter-based extensibility for S3/GCS). It centralizes access to shared storage with **RBAC + path-based ACL**, **async mutations**, baseline **task audit events**, and a baseline-validated **quarantine → scan → promote** guardrail flow (local semantics).

</div>

> [!Note]
> **Honest status:** The **Go File Engine** is the current working nucleus (baseline-validated). The **Laravel control plane** is scaffold/in-progress and will evolve into the orchestration layer as features are developed.
<!--
## TL;DR

- **Multi-tenant:** tenant scope is resolved **server-side** (not trusted from JWT/client).
- **AuthZ:** RBAC + path-based ACL with inheritance, **deny-by-default**, enforced at the File Engine boundary.
- **Async mutations:** create-folder and upload lifecycle are baseline-validated async workflows returning task/status-oriented outcomes; clients poll task status and complete upload flows through deterministic contract checks.
- **Secure uploads:** staged quarantine write + scan-gated promote behavior are baseline-validated, including non-stub ClamAV scanner integration evidence (clean + quarantined paths) and operational scanner closure controls (threshold alerts + runbook/escalation drill evidence).
- **Auditing:** persisted task status + task audit events + append-only DB enforcement + external sink delivery are baseline-validated.
- **Observability:** correlation IDs are baseline; OTEL export wiring is baseline-validated for API + worker entrypoints; collector/backend deployment hardening is baseline-validated with deterministic connectivity + drill scripts; paging-provider delivery is baseline-validated through a deterministic webhook drill path.
-->
  
---

## Project status

This repository documents an evolving architecture.

Legend:

- ✅ implemented
- 🟡 in progress
- 🔒 planned / target state

> [!Note]
> **Current maturity note:** Some controls are documented as target state. The roadmap tracks what is enforced vs intended.
> **Validation source of truth:** See [`docs/capability-ledger.md`](docs/capability-ledger.md) for runnable commands that validate each implementation.

---

## Canonical doc map

**Architecture & Implementation:**

- **API Reference:** [`docs/api-reference.md`](docs/api-reference.md)
- **API Versioning Policy:** [`docs/api-versioning-policy.md`](docs/api-versioning-policy.md)
- **Client SDKs (thin):** [`docs/client-sdks.md`](docs/client-sdks.md)
- **Architecture Overview:** [`docs/architecture.md`](docs/architecture.md)
- **Architecture Boundaries:** [`docs/architecture_boundaries.md`](docs/architecture_boundaries.md)
- **Auth Model (RBAC/JWT):** [`docs/auth.md`](docs/auth.md)
- **Threat Model:** [`docs/threat-model.md`](docs/threat-model.md)
- **Observability:** [`docs/observability.md`](docs/observability.md)
- **Roadmap (staged milestones):** [`docs/roadmap.md`](docs/roadmap.md)
- **Setup/onboarding guide:** [`docs/setup.md`](docs/setup.md)
- **Decisions and rationale:** [`docs/adr`](docs/adr)

**Governance & Status:**

- **Capability Ledger (Truth):** [`docs/capability-ledger.md`](docs/capability-ledger.md)
- **Route maturity matrix:** [`docs/route-maturity-matrix.md`](docs/route-maturity-matrix.md)
- **Project Alignment:** [`docs/project-alignment-review.md`](docs/project-alignment-review.md)
- **Governance (merge gates):** [`docs/governance.md`](docs/governance.md)
- **Branch protection mapping:** [`docs/branch-protection-mapping.md`](docs/branch-protection-mapping.md)
- **Ownership source of truth:** [`.github/OWNERS`](.github/OWNERS)
- **Ownership backup matrix:** [`docs/ownership-backup-matrix.md`](docs/ownership-backup-matrix.md)

<!--
> [!Warning]
> If guidance conflicts, use this precedence order: capability ledger -> setup -> scoped AGENTS -> architecture deep-dives.
-->

---

## Why this exists

Many organizations rely on direct file server access (shared drives/SSH/FTP) to create folders, upload documents, and manage structured storage. This is:

- hard to audit,
- easy to misuse (authorization drift, unsafe paths),
- inconsistent with compliance requirements,
- operationally fragile under load.

This platform provides a centralized, permissioned interface that **controls and records every filesystem mutation**.

---

## What it does

### Read path

- Browse folders (tree navigation, directory listing)
- Metadata display (size, timestamps, ownership) with backend-specific best-effort fields
- **Baseline-validated read path:** list results + size/timestamps/ownership metadata + download path normalization valid
- **Final authz enforcement for reads:** gRPC list/download enforce tenant-scoped paths, server-side tenant membership, and ACL/RBAC checks at File Engine boundary.

### Write path (async)

- Create folders (policy-enforced naming)
- Upload lifecycle is baseline-validated end-to-end (`Initiate → Upload chunk → Complete`) with scan-gated promote semantics and deterministic clean/dirty outcomes.
- Move/rename/delete/restore object operations *(API-level baseline validated; async task variants for move, governed delete, and quarantine restore are baseline-validated*

### Governance & security

- JWT auth (Bearer)
- RBAC + path-based ACL (inheritance)
- Multi-tenant enforcement via **server-side tenant mapping**
- Upload quarantine + malware scan gate before publish (baseline guardrails + non-stub scanner adapter integration are validated)
- Dual-layer audit (queryable + tamper-resistant sink) with baseline-validated external sink delivery adapters
- Access review compliance exports are available via stable JSON contract + monthly operator report generator.

---

## Architecture

### Control plane vs data plane

**Control Plane — Laravel (PHP):**

- UI/API orchestration and business validation (e.g., naming conventions)
- Integrations (planned/target): enterprise identity patterns (AD/LDAP/OIDC broker)
- Admin/UX aggregation (task status, audit views)

**Data Plane — Go File Engine + Worker:**

- gRPC-first API + HTTP/JSON via gRPC-Gateway (baseline for CreateFolder, GetTaskStatus, and upload lifecycle endpoints)
- **Final authorization gate** (tenant membership + RBAC/ACL + safe-path execution)
- OTEL tracer provider wiring is initialized in both API + worker entrypoints when `OTEL_EXPORTER_OTLP_ENDPOINT` is set (
- Enqueues tasks; worker executes storage operations with least privilege

### Diagram (trust boundaries)

```mermaid
flowchart TB
  U[User / Browser] -->|HTTPS| L[Laravel Control Plane<br/>UI + Business Validation]

  %% TB2: Service boundary
  L -->|"gRPC/HTTP (mTLS recommended)"| FE[Go File Engine API<br/>AuthContext + Final AuthZ Gate]

  %% TB3: Queue boundary
  FE --> Q[Redis Queue]
  Q --> W[Worker<br/>Executes tasks]

  %% TB4: Data boundary
  W --> ST["(Storage Backend<br/>Local/NFS/SMB/SFTP mounts<br/>S3/MinIO<br/>GCS)"]

  %% TB5: Scanner boundary
  W --> AV[Scanner Boundary<br/>ClamAV / pluggable]
  AV -->|verdict| W

  %% Audit
  FE --> DB["(Postgres<br/>audit_events (append-only, baseline-validated)<br/>ACL / mappings)"]
  W --> DB
  DB --> SINK[Immutable Audit Sink<br/>SIEM / Loki / S3 WORM]
```

---

### Multi-tenancy model

#### Server-side tenant mapping (source of truth)

- The system does not trust the client or JWT to define tenant scope.
- The File Engine resolves which tenants a user can act on using server-owned data (e.g., a mapping table/service).
- A request is authorized only if:
  - a. the user is mapped to the tenant, and
  - b. RBAC/ACL permits the operation on the target path within that tenant namespace.

#### Namespacing strategy

- Final (publishable): `tenants/<tenant_id>/...`
- Quarantine: `quarantine/<tenant_id>/<uploadId>/...`
- Malware hold: `malware/<tenant_id>/<uploadId>/...`

> Only objects/paths under `tenants/<tenant_id>/...` are listable/downloadable.

---

## Authentication & Authorization

### Authentication (JWT Bearer)

All endpoints require:

```Http
Authorization: Bearer <JWT>
```

Required claims:

- `sub` → user identifier
- `roles` → array of role names

Recommended production validation:

- RSA public-key verification
  - enforce `iss`, `aud`
  - validate `exp`

### Authorization (RBAC + path-based ACL with inheritance)

Authorization is enforced **before operations are executed/enqueued at the File Engine boundary**.

Resolution order:

1. Closest ACL for `user:<sub>` on path
2. Closest ACL for `role:<role>` on path
3. RBAC fallback (role defaults)
4. Deny by default

Inheritance walks up the path: `/a/b/c → /a/b → /a → /`

### No authorization drift (explicit responsibility split)

- Laravel may validate business intent (naming policies, UX flow), but must not be the final gate.
- **File Engine** is the final enforcement point for:
  - tenant membership (server-side mapping),
  - RBAC/ACL decision,
  - path normalization + safe execution constraints.

---

## File Engine API

> Full reference: `docs/api-reference.md`
> Route maturity by endpoint: `docs/route-maturity-matrix.md`

Contract source of truth

- Canonical proto: `file-engine/api/proto/fileengine.proto`
- Compatibility mirror (kept in sync): `file-engine/proto/fileengine.proto`

Base URLs:

- HTTP (gRPC-Gateway): `http://<host>:8080`
- gRPC: `<host>:50051`

Core gRPC methods (canonical):

- `CreateFolder` → returns `taskId` (async)
- `GetTaskStatus` → poll task status
- `InitiateUpload` / `CompleteUpload` *(baseline-validated)*

HTTP/JSON routes (baseline-validated):

- `POST /v1/folders` → `CreateFolder`
- `GET /v1/tasks/{taskId}` → `GetTaskStatus`
- `POST /v1/uploads:initiate` → `InitiateUpload`
- `PUT /v1/uploads/{uploadId}:chunk` → upload chunk
- `POST /v1/uploads/{uploadId}:complete` → `CompleteUpload`

Task state model (canonical):

- `queued → running → success | failed | quarantined`

---

## Current Implementation: Folder Flow

The platform has baseline-validated async folder creation and upload lifecycle flows; folder creation remains the minimal reference walkthrough below.

Implemented baseline reference flow:

1. `CreateFolder` (gRPC) receives request metadata and JWT-authenticated context at File Engine boundary.
2. API enqueues async task in Redis and persists initial task status (`queued`).
2a. API resolves tenant membership from server-side source-of-truth and rejects non-tenant-scoped or unauthorized tenant paths.
3. Worker consumes queue, executes filesystem folder creation, persists terminal status (`success`/`failed`), and emits audit-style events.
4. Client polls `GetTaskStatus` until completion.

Validation command:

```bash
cd file-engine && go test ./internal/handlers -run "TestCreateFolderRequiresAuthContext|TestCreateFolderRejectsNonTenantPath|TestCreateFolderRejectsUnauthorizedTenant|TestCreateFolderEnqueuesWithCorrelationAndActorFallback|TestGetTaskStatusRequiresAuthAndReturnsPersistedStatus" -v && go test ./tests/integration -run TestAsyncCreateFolderFlow -v
```

---

## Key flows

**Baseline-validated upload API flow (`CL-047`) with storage guardrails (`CL-033`, `CL-040`):**

```mermaid
sequenceDiagram
    autonumber
    participant U as User / UI
    participant FE as File Engine (API)
    participant DB as Postgres
    participant Q as Redis Queue
    participant W as Worker
    participant ST as Storage (S3/Local)

    Note over U, ST: Phase 1: Initiate & Upload

    U->>FE: POST /v1/uploads:initiate<br/>(path, size, mime)
    FE->>FE: Validate JWT, RBAC, Policy
    FE->>DB: Create Record (State: PENDING, Path: quarantine/...)
    FE-->>U: Return uploadId + uploadUrl

    U->>ST: PUT binary to uploadUrl<br/>(Writes to quarantine/tenant/id/...)
    
    Note over U, ST: Phase 2: Completion & Scan

    U->>FE: POST /v1/uploads/{id}:complete
    FE->>DB: Update State: QUEUED
    FE->>Q: Enqueue Scan Task
    FE-->>U: HTTP 202 Accepted (taskId)

    Q->>W: Pop Task
    W->>ST: Read File Stream
    W->>W: ClamAV Scan (Stream)

    alt Verdict: CLEAN
        W->>ST: Atomic Move<br/>(quarantine/... -> tenants/...)
        W->>DB: Update State: SUCCESS
        W->>DB: Log Audit (Promote)
    else Verdict: MALICIOUS
        W->>ST: Move to malware/... (Hold)
        W->>DB: Update State: QUARANTINED
        W->>DB: Log Audit (Security Alert)
    end
```

---

## Security model

Trust boundaries:

- TB1: Browser ↔ Laravel (untrusted input)
- TB2: Laravel ↔ File Engine (east-west; mTLS)
- TB3: Queue boundary (tamper/replay/poison messages)
- TB4: Storage boundary (least privilege; private endpoints)
- TB5: Scanner boundary (hostile bytes; sandboxed)

Secure-by-default controls:

- Deny-by-default authorization at File Engine
- Tenant scope from server-side mapping (not JWT)
- Strict path normalization + traversal rejection
- Quarantine → scan → promote gating (baseline guardrails + non-stub scanner integration validated)
- Redaction policy: never log tokens or pre-signed URLs

Known gaps / planned hardening (examples):

- Explicit deny rules in ACL (deny > allow) 🔒 (ADR candidate)
- Signed task payloads / replay defense 🔒 (ADR candidate)
- Stronger immutability guarantees for the secondary audit sink 🔒

> Detailed STRIDE model: `docs/threat-model.md`

---

## Auditing

**Dual-layer audit:**

- **Primary (queryable)**: Postgres `audit_events` table (append-only enforcement baseline-validated in `CL-032`)
- **Secondary (tamper-resistant)**: external sink (SIEM / Loki / S3 WORM) with retries + DLQ + lag metric baseline-validated in `CL-035`

**Baseline audit behavior:**

- Task audit events are emitted for the async folder flow (see `CL-005`).

Audit coverage (target baseline):

- Mutation events: create/move/rename/write, upload lifecycle, scan verdict, promote/hold decision
- Security events: authz denials, policy failures (rate-limited + aggregated as needed)
- Correlation fields: `request_id, trace_id, task_id, user_id, tenant_id, operation, path`, outcome

---

## Observability

Standards:

- JSON structured logs (consistent envelope, redaction)
- Request correlation across HTTP ↔ gRPC ↔ queue ↔ worker
  - X-Request-Id, traceparent (W3C)
- Distributed tracing via OpenTelemetry (OTLP exporter)

Operational signals to monitor:

- Queue depth / worker saturation
- Scan duration + pass/fail ratio
- Promotion failures
- Quarantine growth
- 403 spikes (probing / misconfig)

> Full spec: `docs/observability.md`

---

## Quickstart (local development)

Requirements:

- Go 1.26+
- Docker Engine / Docker Desktop + Compose v2 (optional; only needed for containerized dependencies)
- curl (optional; only needed for manual API calls)

### 1) Run the validated baseline checks (recommended)

This is the only **baseline-validated** quickstart today.

```bash
./file-engine/scripts/dev.sh
```

### 2) One-command onboarding + demo evidence

```bash
make bootstrap && make demo
```

This command pair regenerates docs, enforces architecture boundaries, runs doc drift checks, executes the deterministic 5-minute demo script, and prints evidence links for generated docs.

### 3) Optional: run the async folder flow integration test alone

```bash
cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v
```

### 4) Optional: local File Engine run (scaffold-level, for debugging)

This brings up Redis/Postgres in Docker and runs the API/worker locally for debugging. REST endpoints include baseline create-folder/task-status and upload lifecycle paths; treat this path as local debugging rather than the canonical baseline verification flow.

```bash
cd file-engine
docker compose up -d redis postgres

export REDIS_ADDR="localhost:6379"
export POSTGRES_DSN="postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
export STORAGE_BACKEND="local"
export FILE_BASE_ROOT="$PWD/data"
export JWT_SECRET="dev-secret"
export TENANT_MEMBERSHIPS="dev-admin=dev-tenant"

# Optional worker guardrails (defaults shown in parentheses):
# export WORKER_STATUS_RETRY_ATTEMPTS="3"
# export WORKER_STATUS_RETRY_DELAY_MS="25"
# export WORKER_TASK_PROCESS_TIMEOUT_MS="30000"

go run ./cmd/migrate
```

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
export JWT="***"
```

### 5) Canonical compose entry point

Use **repository-root `docker-compose.yml`** as the primary developer compose entry point.

`file-engine/docker-compose.yml` remains only as a compatibility mirror and should not be treated as the canonical source.

**Default ports:**

- HTTP: `8080`
- gRPC: `50051`
- Redis: `6379`
- Postgres: `5432`

> [!Note]
> All setup flows (local File Engine run, canonical root compose, dev JWT) are documented in `docs/setup.md`.


## Deployment (dev/stage/prod + kind + rollback)

- Environment profile templates are versioned in `env/.env.dev.example`, `env/.env.stage.example`, and `env/.env.prod.example`.
- Config/secret separation and required runtime wiring checks are documented in `docs/deployment-profiles.md` and validated with `./scripts/check-runtime-wiring.sh --profile prod`.
- Kubernetes smoke and rollback drill paths are script-backed via `./scripts/k8s/kind_smoke.sh` and `./scripts/drills/k8s_rollback_drill.sh`.
- Release versioning + changelog + rollback discipline is documented in `docs/release/versioning-and-rollback.md`.

---

## Repository structure

```text
file-server-management/
├─ frontend/                  # Static thin-client demo console (no Node build)
├─ backend/                   # Laravel control plane
├─ file-engine/               # Go File Engine (API + Worker)
└─ docs/
   ├─ adr/                    # Architectural Decision Records
   ├─ architecture.md         # Platform architecture
   ├─ api-reference.md        # API surface (gRPC + HTTP)
   ├─ auth.md                 # AuthN/AuthZ model
   ├─ observability.md        # Logs/metrics/tracing expectations
   ├─ threat-model.md         # Security model + STRIDE notes
   └─ setup.md                # Local development setup
```

---

## Disclaimer

This project is a work in progress. Some controls are documented as “target state” and may not be fully implemented yet. Each milestone aims to move documented intent into enforced reality.

---

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

<br><hr>
[🔼 Back to top](#back-to-top)
