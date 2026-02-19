<div align="center">

<a name="back-to-top"></a>

# Server File Manager Platform (PHP + Go File Engine)

[![CI](https://github.com/agslima/file-server-management/actions/workflows/ci.yml/badge.svg)](https://github.com/agslima/file-server-management/actions/workflows/ci.yml)
![Go Version](https://img.shields.io/badge/go-1.24+-blue)
![Laravel](https://img.shields.io/badge/laravel-10%2B-red)
![gRPC](https://img.shields.io/badge/API-gRPC%20-5e5e5e)
[![Docs](https://img.shields.io/badge/docs-architecture%20%7C%20adr-brightgreen)](https://github.com/agslima/file-server-management/tree/main/docs)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
<!--
![Go Tests](https://github.com/<org>/<repo>/actions/workflows/go-test.yaml/badge.svg)
![Laravel Tests](https://github.com/<org>/<repo>/actions/workflows/phpunit.yaml/badge.svg)
[![codecov](https://codecov.io/gh/<org>/<repo>/branch/main/graph/badge.svg)](https://codecov.io/gh/<org>/<repo>)
![Dependency Review](https://github.com/<org>/<repo>/actions/workflows/dependency-review.yml/badge.svg)
![Trivy](https://github.com/<org>/<repo>/actions/workflows/trivy.yml/badge.svg)
-->

⚡ A governance-focused, multi-tenant file management platform written in Go an PHP ⚡\
Designed to operate directly on **real storage backends** (SMB, NFS, SFTP, and local mounts) \
The system enforces centralized access control through **Role-Based Access Control (RBAC) and path-based ACLs**, leverages asynchronous operations for all mutations, and secures file ingestion via a **quarantine, anti-malware scan, and promotion pipeline** (**target-state**), with a complete record of all actions maintained through immutable audit trails.

</div>

## TL;DR

- **Multi-tenant:** tenant scope is resolved **server-side** (not trusted from JWT/client).
- **AuthZ:** RBAC + path-based ACL with inheritance, **deny-by-default**, enforced at the File Engine boundary.
- **Async mutations:** create (baseline) returns a `taskId`; move/upload are target-state; clients poll task status.
- **Secure uploads:** quarantine -> scan -> promote workflow (**target-state, not fully baseline-validated**).
- **Auditing:** persisted task status + basic task audit events in async folder flow baseline; dual-layer sink is target-state.
- **Observability:** correlation IDs are propagated in baseline async flow logs/status; full OTEL pipeline is target-state.

> [!Note]
> **Honest status:** The **Go File Engine** is the current working nucleus (baseline-validated). The **Laravel control plane** is scaffold/in-progress and becomes the orchestration layer as features are promoted via the capability ledger.
---

## Canonical doc map

**Architecture & Implementation:**

- **API Reference:** [`docs/api-reference.md`](docs/api-reference.md)
- **Architecture Overview:** [`docs/architecture.md`](docs/architecture.md)
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
- **Agent Constraints:** [`.github/AGENTS.md`](.github/AGENTS.md)
- **File Engine scoped operating guide:** [`file-engine/AGENTS.md`](file-engine/AGENTS.md)
- **Backend operating guide:** [`backend/AGENTS.md`](backend/AGENTS.md)

> If guidance conflicts, use this precedence order: capability ledger -> setup -> scoped AGENTS -> architecture deep-dives.

---

## Project status

This repository documents an evolving architecture.

Legend:

- ✅ implemented
- 🟡 in progress
- 🔒 planned / target state

> [!Note]
> **Current maturity note:** Some controls are documented as target state. The roadmap tracks what is enforced vs intended.
> **Validation source of truth:** See [`docs/capability-ledger.md`](docs/capability-ledger.md) for runnable commands that validate each implemented claim.

### Implementation status (baseline)

Every baseline claim is mapped to a claim ID and runnable command in the capability ledger.

<details><summary><b>See more details</b></summary>
  
| Claim ID | Capability | Status | Runnable validation |
| :-- | :-- | :--: | :-- |
| [`CL-001`](docs/capability-ledger.md#baseline-claims-implemented) | Canonical proto contract sync | ✅ | `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto` |
| [`CL-002`](docs/capability-ledger.md#baseline-claims-implemented) | File Engine baseline module checks | ✅ | `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v` |
| [`CL-003`](docs/capability-ledger.md#baseline-claims-implemented) | Async folder flow (enqueue -> worker -> folder created) | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` |
| [`CL-004`](docs/capability-ledger.md#baseline-claims-implemented) | Task status persistence (`queued -> running -> success`) | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` |
| [`CL-005`](docs/capability-ledger.md#baseline-claims-implemented) | Basic audit event emission (`task.processing`, `task.succeeded`) | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` |
| [`CL-006`](docs/capability-ledger.md#baseline-claims-implemented) | Correlation ID propagation in async flow | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` |
| [`CL-007`](docs/capability-ledger.md#baseline-claims-implemented) | Known-working local dev script | ✅ | `./file-engine/scripts/dev.sh` |
| [`CL-008`](docs/capability-ledger.md#baseline-claims-implemented) | Backend scaffold validation | ✅ | `cd backend && composer validate --strict` |
| [`CL-009`](docs/capability-ledger.md#baseline-claims-implemented) | Frontend placeholder scaffold | 🔒 | `test -f frontend/README.md && test ! -f frontend/package.json` |
| [`CL-010`](docs/capability-ledger.md#baseline-claims-implemented) | Structured logs + queue/task metrics baseline | ✅ | `cd file-engine && go test ./internal/handlers ./internal/observability -v` |
| [`CL-011`](docs/capability-ledger.md#baseline-claims-implemented) | Documentation drift checks (links + governance hygiene) | ✅ | `./scripts/doc-drift-check.sh` |
| [`CL-012`](docs/capability-ledger.md#baseline-claims-implemented) | Read-path behavior + final authz (list/download + path normalization + tenant enforcement) | ✅ | `cd file-engine && go test ./internal/handlers -run "TestListObjectsReturnsEntries|TestListObjectsRequiresAuthContext|TestListObjectsRejectsUnauthorizedTenant|TestDownloadObjectRejectsUnauthorizedTenant" -v && go test ./internal/adapters/storage/local -run TestLocalStorageListMetadata -v && go test ./internal/authz -run "TestGRPCAuthZInterceptorListObjects" -v && go test ./internal/server -run "TestHandleDownloadNormalizesPath|TestHandleDownloadRejectsTraversal" -v && go test -tags integration_authz ./tests/integration -run TestReadListBehaviorAndAuthzRejection -v` |
| [`CL-013`](docs/capability-ledger.md#baseline-claims-implemented) | HTTP gateway routes for `CreateFolder` + `GetTaskStatus` are generated and responsive | ✅ | `cd file-engine && go test ./internal/server -run TestGatewayCreateFolderAndGetTaskStatusRoutes -v` |
| [`CL-014`](docs/capability-ledger.md#baseline-claims-implemented) | AuthZ precedence behavior (ACL vs RBAC) | ✅ | `cd file-engine && go test ./internal/auth -run "TestRBACFallback|TestUserACLOverridesRBAC|TestACLPathInheritance|TestUserDenyPrecedesRoleAllowAndRBAC|TestRoleDenyPrecedesRoleAllowAtSamePath|TestClosestPathACLWinsBeforeParentACLs|TestUserACLPrecedenceOnSamePath|TestUserACLWithoutPermissionFallsThroughToRoleACL" -v` |
| [`CL-015`](docs/capability-ledger.md#baseline-claims-implemented) | Path normalization guarantees (traversal rejection + canonicalization) | ✅ | `cd file-engine && go test ./internal/authz -run "TestExtractPathNormalizesCreateFolder|TestExtractPathRejectsTraversal|TestNormalizePathHandlesWindowsAndWhitespace|TestNormalizePathAllowsDotContainingNames|TestTenantFromPath|TestTenantFromPathRejectsNonTenantRoot" -v` |
| [`CL-016`](docs/capability-ledger.md#baseline-claims-implemented) | Generated gateway artifacts remain in sync with proto | ✅ | `cd file-engine && ./scripts/generate_grpc_docker.sh && cd .. && git diff --exit-code && test -z "$(git status --porcelain)"` |
| [`CL-017`](docs/capability-ledger.md#baseline-claims-implemented) | Worker performance guardrails (status retries + task timeout) for async create-folder | ✅ | `cd file-engine && go test ./internal/app/tasks -run "TestWorkerRetriesStatusPersistence|TestWorkerMarksTaskFailedOnProcessingTimeout" -v` |
| [`CL-018`](docs/capability-ledger.md#baseline-claims-implemented) | Backend VS-001 scaffold contract (create-folder forward + task polling wiring checks) | ✅ | `cd backend && composer validate --strict && php -l app/Http/Controllers/FolderController.php && php -l app/Http/Controllers/TaskController.php && php -l app/Services/FileEngineService.php` |
| [`CL-020`](docs/capability-ledger.md#baseline-claims-implemented) | Backend VS-001 docker-compose E2E (forward create-folder + poll task to success + folder existence) | 🟡 | `docker compose up -d --build && ./scripts/wait-for-http.sh http://localhost:8080/healthz 60 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 60 && ./scripts/e2e/vs001_create_folder.sh && docker compose down -v` |
| [`CL-022`](docs/capability-ledger.md#baseline-claims-implemented) | Audit coverage for read/list/download actions (`object.list`, `object.read`, `object.download`) | 🟡 | `cd file-engine && go test ./tests/integration -run TestAuditEventsEmittedForReadListDownload -v` |
| [`CL-025`](docs/capability-ledger.md#baseline-claims-implemented) | Upload pipeline baseline: staged quarantine write + atomic promote (no partial final object visibility) | 🟡 | `cd file-engine && go test ./tests/integration -run TestStagedUploadAtomicPromote -v` |
| [`CL-031`](docs/capability-ledger.md#baseline-claims-implemented) | Backend baseline smoke suite (composer install + phpunit) | ✅ | `docker compose run --rm --no-deps backend sh -lc 'composer install --no-interaction && ./vendor/bin/phpunit -c phpunit.xml'` |
| [`CL-032`](docs/capability-ledger.md#baseline-claims-implemented) | Audit table append-only enforcement (UPDATE/DELETE rejected for app DB user) | ✅ | `cd file-engine && go test ./tests/integration -run TestAuditEventsAppendOnlyEnforced -v` |
| [`CL-033`](docs/capability-ledger.md#baseline-claims-implemented) | Upload malware gating: dirty scan blocks promote, clean scan promotes from quarantine | ✅ | `cd file-engine && go test ./tests/integration -run TestUploadScanGateDirtyPreventsPromotion -v` |
| [`CL-034`](docs/capability-ledger.md#baseline-claims-implemented) | Curated ledger baseline gate script runs in CI to catch regressions | ✅ | `./scripts/ledger-baseline.sh` |

> For target-state exclusions and promotion criteria, see [`docs/capability-ledger.md`](docs/capability-ledger.md).

</details>

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
- **Baseline-validated read path:** list results + size/timestamps/ownership metadata + download path normalization validated by [`CL-012`](docs/capability-ledger.md#baseline-claims-implemented)
- **Final authz enforcement for reads:** gRPC list/download enforce tenant-scoped paths, server-side tenant membership, and ACL/RBAC checks at File Engine boundary; verified by unit + integration coverage in [`file-engine/internal/handlers/grpc_handler_test.go`](file-engine/internal/handlers/grpc_handler_test.go) and [`file-engine/tests/integration/read_list_authz_integration_test.go`](file-engine/tests/integration/read_list_authz_integration_test.go)

### Write path (async)

- Create folders (policy-enforced naming)
- Upload files (two-step: initiate → complete) *(target-state)*
- Move/rename/write operations *(as tasks; target-state)*

### Governance & security

- JWT auth (Bearer)
- RBAC + path-based ACL (inheritance)
- Multi-tenant enforcement via **server-side tenant mapping**
- Upload quarantine + malware scan gate before publish *(target-state)*
- Dual-layer audit (queryable + tamper-resistant sink) *(target-state)*

---

## Architecture

### Control plane vs data plane

**Control Plane — Laravel (PHP):**

- UI/API orchestration and business validation (e.g., naming conventions)
- Integrations (planned/target): enterprise identity patterns (AD/LDAP/OIDC broker)
- Admin/UX aggregation (task status, audit views)

**Data Plane — Go File Engine + Worker:**

- gRPC-first API + HTTP/JSON via gRPC-Gateway (baseline for CreateFolder + GetTaskStatus; uploads target-state)
- **Final authorization gate** (tenant membership + RBAC/ACL + safe-path execution)
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
  FE --> DB["(Postgres<br/>audit_events (append-only, target-state)<br/>ACL / mappings)"]
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
- `InitiateUpload` / `CompleteUpload` *(target-state)*

HTTP/JSON routes (baseline for CreateFolder + GetTaskStatus):

- `POST /v1/folders` → `CreateFolder`
- `GET /v1/tasks/{taskId}` → `GetTaskStatus`

Upload HTTP routes remain target-state until the upload pipeline is implemented.

Task state model (canonical):

- `queued → running → success | failed | quarantined`

---

## Current Implementation: Folder Flow

While the architecture supports the full Upload Quarantine flow (see below), the currently verifiable baseline is the Async Folder Creation flow.

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

**Target-state upload flow (not baseline-validated):**

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
- Quarantine → scan → promote gating *(target-state)*
- Redaction policy: never log tokens or pre-signed URLs

Known gaps / planned hardening (examples):

- Explicit deny rules in ACL (deny > allow) 🔒 (ADR candidate)
- Signed task payloads / replay defense 🔒 (ADR candidate)
- Stronger immutability guarantees for the secondary audit sink 🔒

> Detailed STRIDE model: `docs/threat-model.md`

---

## Auditing

**Dual-layer audit (target-state vision):**

- **Primary (queryable)**: Postgres `audit_events` table (append-only, target-state)
- **Secondary (tamper-resistant)**: external sink (SIEM / Loki / S3 WORM, target-state)

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

- Go 1.24+
- Docker Engine / Docker Desktop + Compose v2 (optional; only needed for containerized dependencies)
- curl (optional; only needed for manual API calls)

### 1) Run the validated baseline checks (recommended)

This is the only **baseline-validated** quickstart today.

```bash
./file-engine/scripts/dev.sh
```

### 2) Optional: run the async folder flow integration test alone

```bash
cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v
```

### 3) Optional: local File Engine run (scaffold-level, for debugging)

This brings up Redis/Postgres in Docker and runs the API/worker locally. REST endpoints are baseline for `CreateFolder` + `GetTaskStatus`; uploads remain target-state. Treat this path as a debug path beyond baseline behavior.

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
export JWT="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJkZXYtYWRtaW4iLCJyb2xlcyI6WyJhZG1pbiJdLCJleHAiOjQxMDI0NDQ4MDB9.Y-JdrUO96XS3odOeBWtYSIjPwR7z7g7IytvBLxTbCus"
```

### 4) Canonical compose entry point

Use **repository-root `docker-compose.yml`** as the primary developer compose entry point.

`file-engine/docker-compose.yml` remains only as a compatibility mirror and should not be treated as the canonical source.

**Default ports:**

- HTTP: `8080`
- gRPC: `50051`
- Redis: `6379`
- Postgres: `5432`

> [!Note]
> All setup flows (local File Engine run, canonical root compose, dev JWT) are documented in `docs/setup.md`.

---

## Repository structure

```text
file-server-management/
├─ frontend/                  # React / Next.js UI
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
