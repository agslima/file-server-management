# Server File Manager Platform (PHP + Go File Engine)

## Multi-tenant, governance-first file operations with RBAC, audit trails, and malware-gated uploads

[![CI](https://github.com/agslima/file-server-management/actions/workflows/ci.yml/badge.svg)](https://github.com/agslima/file-server-management/actions/workflows/ci.yml)
[![CodeQL](https://github.com/agslima/file-server-management/actions/workflows/codeql.yml/badge.svg)](https://github.com/agslima/file-server-management/actions/workflows/codeql.yml)
![Go Version](https://img.shields.io/badge/go-1.21+-blue)
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

## TL;DR

A multi-tenant, governance-first file management platform that operates on **real storage backends** (mounted SMB/NFS/SFTP/local, or S3/GCS via adapters). It centralizes access to shared storage with **RBAC + path-based ACL**, **async mutations**, **dual-layer auditing**, and a **quarantine → scan → promote** upload pipeline.

**Key points:**

- **Multi-tenant:** tenant scope is resolved **server-side** (not trusted from JWT/client).
- **AuthZ:** RBAC + path-based ACL with inheritance, **deny-by-default**, enforced at the File Engine boundary.
- **Async mutations:** create/move/upload return a `taskId`; clients poll task status.
- **Secure uploads:** **quarantine → scan → promote** (only `tenants/<tenant>/...` is publishable).
- **Auditing:** dual-layer — Postgres append-only + immutable external sink (SIEM/Loki/S3 WORM).
- **Observability:** structured JSON logs + OpenTelemetry tracing + correlation across HTTP/gRPC/queue.

---

## Canonical doc map

Use this map to avoid documentation drift and find the correct source of truth quickly:

- **Project capability truth (implemented vs target):** [`docs/capability-ledger.md`](docs/capability-ledger.md)
- **Project alignment review and improvement plan:** [`docs/project-alignment-review.md`](docs/project-alignment-review.md)
- **Contributor/agent operating constraints (repo-wide governance notes):** [`.github/AGENTS.md`](.github/AGENTS.md)
- **File Engine scoped operating guide:** [`file-engine/Agents.md`](file-engine/Agents.md)
- **Setup/onboarding guide:** [`docs/setup.md`](docs/setup.md)
- **Legacy compose reference (non-canonical):** [`docker/docker-compose.yml`](docker/docker-compose.yml)

> If guidance conflicts, prefer this order: capability ledger → setup guide → scoped agent docs.

---

## Project status

This repository documents an evolving architecture.

Legend:

- ✅ implemented
- 🟡 in progress
- 🔒 planned / target state

> **Current maturity note:** Some controls are documented as target state. The roadmap tracks what is enforced vs intended.

> **Validation source of truth:** See [`docs/capability-ledger.md`](docs/capability-ledger.md) for runnable commands that validate each implemented claim.


### Implementation status (baseline)

| Capability | Status | Validation |
| :-- | :--: | :-- |
| Canonical proto contract sync | ✅ | `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto` |
| File Engine baseline module checks | ✅ | `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v` |
| Async folder flow (enqueue → worker → folder created) | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` |
| Task status persistence + audit + correlation IDs | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` |
| Known-working local dev script | ✅ | `./file-engine/scripts/dev.sh` |
| Backend scaffold validation | 🟡 | `cd backend && composer validate --strict` |
| Frontend placeholder scaffold | 🔒 | `test -f frontend/README.md && test ! -f frontend/package.json` |

For detailed claim-to-check mapping (including target-state exclusions), see the [capability ledger](docs/capability-ledger.md).

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
- Metadata display (size, timestamps, ownership, etc.) *(as applicable per backend)*

### Write path (async)

- Create folders (policy-enforced naming)
- Upload files (two-step: initiate → complete)
- Move/rename/write operations *(as tasks)*

### Governance & security

- JWT auth (Bearer)
- RBAC + path-based ACL (inheritance)
- Multi-tenant enforcement via **server-side tenant mapping**
- Upload quarantine + malware scan gate before publish
- Dual-layer audit (queryable + tamper-resistant sink)

---

## Architecture

### Control plane vs data plane

**Control Plane — Laravel (PHP):**

- UI/API orchestration and business validation (e.g., naming conventions)
- Integrations (planned/target): enterprise identity patterns (AD/LDAP/OIDC broker)
- Admin/UX aggregation (task status, audit views)

**Data Plane — Go File Engine + Worker:**

- gRPC-first API + HTTP/JSON via gRPC-Gateway
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
  FE --> DB["(Postgres<br/>audit_events (append-only)<br/>ACL / mappings)"]
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

> Full reference: docs/api-reference.md

**Contract source of truth**

- Canonical proto: `file-engine/api/proto/fileengine.proto`
- Compatibility mirror (kept in sync): `file-engine/proto/fileengine.proto`

Base URLs:

- HTTP: `http://<host>:8080`
- gRPC: `<host>:50051`

Core endpoints (HTTP/JSON via gRPC-Gateway):

- `POST /v1/folders` → returns `taskId` (async)
- `POST /v1/uploads:initiate` → returns `uploadId, uploadUrl`
- `POST /v1/uploads/{uploadId}:complete` → returns `taskId`
- `GET /v1/tasks/{taskId}` → poll task status
- `GET /healthz` → liveness (200 OK if process + HTTP server responsive)

Task state model (canonical):

- `queued → running → success | failed | quarantined`

---

## Key flows

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
- Quarantine → scan → promote gating
- Redaction policy: never log tokens or pre-signed URLs

Known gaps / planned hardening (examples):

- Explicit deny rules in ACL (deny > allow) 🔒 (ADR candidate)
- Signed task payloads / replay defense 🔒 (ADR candidate)
- Stronger immutability guarantees for the secondary audit sink 🔒

> Detailed STRIDE model: `docs/threat-model.md`

---

## Auditing

**Dual-layer audit:**

- **Primary (queryable)**: Postgres `audit_events` table (append-only)
- **Secondary (tamper-resistant)**: external sink (SIEM / Loki / S3 WORM)

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

- Go 1.21+
- Docker Engine / Docker Desktop + Compose v2
- curl

### 1) Start dependencies (Redis + Postgres)

```bash
docker compose up -d postgres redis
```

### 2) Apply migrations

```bash
export POSTGRES_DSN="postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
go run ./cmd/migrate
```

### 3) Run the stack (API + Worker)

```bash
docker compose up --build
```

### 4) Smoke test (liveness)

```bash
curl -i http://localhost:8080/healthz
```

### 5) Run unit tests

```bash
go test ./... -v
```

**Default ports:**

- HTTP: `8080`
- gRPC: `50051`
- Redis: `6379`
- Postgres: `5432`

---

## Repository structure

```text
file-server-management/
├─ frontend/                  # React / Next.js UI
├─ backend/                   # Laravel control plane
├─ file-engine/               # Go File Engine (API + Worker)
├─ docker/                    # Dockerfiles / Compose helpers
└─ docs/
   ├─ architecture/           # Platform architecture + contracts
   ├─ security/               # Threat model, pipeline security, STRIDE
   ├─ readmes/                # Role-specific docs (platform, security, contributors)
   └─ adr/                    # Architectural Decision Records
```

---

## Roadmap

| Phase | Goal | Status |
| :---- | :---- | :---- |
| Phase 1 | Browse directories + read authz baseline | 🟡 |
| Phase 2 | Folder creation (async) + audit events | 🟡 |
| Phase 3 | Quarantine → scan → promote + observability baseline | 🟡 |
| Phase 4 | Advanced governance (fine-grained ACL, workflows, notifications) | 🔒 |
| Phase 5 | Enterprise features (retention, eDiscovery-friendly audit, versioning) | 🔒 |

Queue strategy:

- Redis (simplicity) — see ADRs in `docs/adr/`.

---

## Documentation map

For detailed implementation guides, please refer to:

- Documentation Overview - [`docs/readme.md`](https://github.com/agslima/file-server-management/blob/main/docs/README.md)
- Decisions and rationale — [`docs/adr`](https://github.com/agslima/file-server-management/blob/main/docs/adr)
- Platform Architecture Overview — [`docs/architecture.md`](https://github.com/agslima/file-server-management/blob/main/docs/architecture.md)
- File Engine API (gRPC + HTTP/JSON) — [`docs/api-reference.md`](https://github.com/agslima/file-server-management/blob/main/docs/api-reference.md)
- JWT + RBAC/ACL model — [`docs/auth.md`](https://github.com/agslima/file-server-management/blob/main/docs/auth.md)
- Threat Model & Security Specification (STRIDE)  — [`docs/threat-model.md`](https://github.com/agslima/file-server-management/blob/main/docs/threat-model.md)
- Logging, metrics, tracing standards — [`docs/observability.md`](https://github.com/agslima/file-server-management/blob/main/docs/observability.md)
- Setud & Installation Guide — [`docs/setup.md`](https://github.com/agslima/file-server-management/blob/main/docs/setup.md)

---

## Disclaimer

This project is a work in progress. Some controls are documented as “target state” and may not be fully implemented yet. Each milestone aims to move documented intent into enforced reality.

---

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.
