# Server File Manager Platform (Laravel + Go File Engine)

[![CI](https://github.com/agslima/file-server-management/actions/workflows/ci.yml/badge.svg)](https://github.com/agslima/file-server-management/actions/workflows/ci.yml)
![Docs](https://img.shields.io/badge/docs-architecture%20%7C%20security%20%7C%20adr-brightgreen)
![gRPC](https://img.shields.io/badge/API-gRPC%20%2B%20HTTP%20Gateway-5e5e5e)
![CodeQL](https://github.com/<org>/<repo>/actions/workflows/codeql.yml/badge.svg)
![Go Version](https://img.shields.io/badge/go-1.21+-blue)
![Laravel](https://img.shields.io/badge/laravel-10%2B-red)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
<!--
![Go Tests](https://github.com/<org>/<repo>/actions/workflows/go-test.yaml/badge.svg)
![Laravel Tests](https://github.com/<org>/<repo>/actions/workflows/phpunit.yaml/badge.svg)
[![codecov](https://codecov.io/gh/<org>/<repo>/branch/main/graph/badge.svg)](https://codecov.io/gh/<org>/<repo>)
![Dependency Review](https://github.com/<org>/<repo>/actions/workflows/dependency-review.yml/badge.svg)
![Trivy](https://github.com/<org>/<repo>/actions/workflows/trivy.yml/badge.svg)
-->

A multi-tenant, governance-first file management platform that operates on **real storage backends** (mounted SMB/NFS/SFTP/local, or S3/GCS via adapters). It centralizes access to shared storage with **RBAC + path-based ACL**, **async mutations**, **dual-layer auditing**, and a **quarantine → scan → promote** upload pipeline.

> **Core principle:** JWT authenticates the user; **tenant scope is resolved server-side**. The **Go File Engine is the final authorization gate** for tenant membership, RBAC/ACL decisions, and safe-path execution.


## TL;DR

A multi-tenant, governance-first file manager for **real storage backends** (mounted SMB/NFS/SFTP/local or S3/GCS).  
**Laravel** handles UI + business validation; the **Go File Engine** is the **final authorization gate** and runs mutations **as async tasks**.

**Key points**
- **Multi-tenant:** tenant scope is resolved **server-side** (not trusted from JWT/client).
- **AuthZ:** RBAC + path-based ACL with inheritance, **deny-by-default**, enforced at the File Engine boundary.
- **Async mutations:** create/move/upload return a `taskId`; clients poll task status.
- **Secure uploads:** **quarantine → scan → promote** (only `tenants/<tenant>/...` is publishable).
- **Auditing:** dual-layer — Postgres append-only + immutable external sink (SIEM/Loki/S3 WORM).
- **Observability:** structured JSON logs + OpenTelemetry tracing + correlation across HTTP/gRPC/queue.

---

## Project status

This repository documents an evolving architecture.

Legend:
- ✅ implemented
- 🟡 in progress
- 🔒 planned / target state

> **Current maturity note:** Some controls are documented as target state. The roadmap tracks what is enforced vs intended.

---

## Why this exists

Many organizations still rely on direct file server access (shared drives/SSH/FTP) to create folders, upload documents, and manage structured storage. This is:
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

## Architecture at a glance

### Control plane vs data plane

**Control Plane — Laravel**
- UI/API orchestration and business validation (e.g., naming conventions)
- Integrations (planned/target): enterprise identity patterns (AD/LDAP/OIDC broker)
- Admin/UX aggregation (task status, audit views)

**Data Plane — Go File Engine + Worker**
- gRPC-first API + HTTP/JSON via gRPC-Gateway
- **Final authorization gate** (tenant membership + RBAC/ACL + safe-path execution)
- Enqueues tasks; worker executes storage operations with least privilege

### High-level diagram (trust boundaries)

```mermaid
flowchart TB
  U[User / Browser] -->|HTTPS| L[Laravel Control Plane<br/>UI + Business Validation]

  %% TB2: Service boundary
  L -->|gRPC/HTTP (mTLS recommended)| FE[Go File Engine API<br/>AuthContext + Final AuthZ Gate]

  %% TB3: Queue boundary
  FE --> Q[(Redis Queue<br/>Kafka later)]
  Q --> W[Worker<br/>Executes tasks]

  %% TB4: Data boundary
  W --> ST[(Storage Backend<br/>Local/NFS/SMB/SFTP mounts<br/>S3/MinIO<br/>GCS)]

  %% TB5: Scanner boundary
  W --> AV[Scanner Boundary<br/>ClamAV / pluggable]
  AV -->|verdict| W

  %% Audit
  FE --> DB[(Postgres<br/>audit_events (append-only)<br/>ACL / mappings)]
  W --> DB
  DB --> SINK[Immutable Audit Sink<br/>SIEM / Loki / S3 WORM]
```

---

Multi-tenancy model

Server-side tenant mapping (source of truth)

The system does not trust the client or JWT to define tenant scope.

The File Engine resolves which tenants a user can act on using server-owned data (e.g., a mapping table/service).

A request is authorized only if:

1. the user is mapped to the tenant, and


2. RBAC/ACL permits the operation on the target path within that tenant namespace.




Namespacing strategy (recommended layout)

Final (publishable): tenants/<tenant_id>/...

Quarantine: quarantine/<tenant_id>/<uploadId>/...

Malware hold: malware/<tenant_id>/<uploadId>/...


> Only objects/paths under tenants/<tenant_id>/... are listable/downloadable.




---

Authentication & Authorization

Authentication (JWT Bearer)

All endpoints require:

Authorization: Bearer <JWT>

Required claims:

sub → user identifier

roles → array of role names


Recommended production validation:

RSA public-key verification

enforce iss, aud

validate exp


Authorization (RBAC + path-based ACL with inheritance)

Authorization is enforced before operations are executed/enqueued at the File Engine boundary.

Resolution order:

1. Closest ACL for user:<sub> on path


2. Closest ACL for role:<role> on path


3. RBAC fallback (role defaults)


4. Deny by default



Inheritance walks up the path: /a/b/c → /a/b → /a → /

No authorization drift (explicit responsibility split)

Laravel may validate business intent (naming policies, UX flow), but must not be the final gate.

File Engine is the final enforcement point for:

tenant membership (server-side mapping),

RBAC/ACL decision,

path normalization + safe execution constraints.




---

File Engine API (contract snapshot)

> Full reference: docs/api-reference.md (link target)



Base URLs:

HTTP: http://<host>:8080

gRPC: <host>:50051


Core endpoints (HTTP/JSON via gRPC-Gateway):

POST /v1/folders → returns taskId (async)

POST /v1/uploads:initiate → returns uploadId, uploadUrl

POST /v1/uploads/{uploadId}:complete → returns taskId

GET /v1/tasks/{taskId} → poll task status

GET /healthz → liveness (200 OK if process + HTTP server responsive)


Task state model (canonical):

queued → running → success | failed | quarantined



---

Key flows

1) Create folder (async)

1. User → Laravel: request folder creation


2. Laravel: validates naming policy (business rule)


3. Laravel → File Engine: create folder (with JWT)


4. File Engine:

validates JWT → AuthContext

resolves tenant scope via server mapping

enforces RBAC/ACL + safe-path constraints

enqueues task



5. Worker executes mkdir/prefix creation


6. Audit event is written (Postgres) and forwarded to immutable sink


7. UI polls task status and updates view



2) Upload (two-phase publish: quarantine → scan → promote)

Initiate

1. UI requests POST /v1/uploads:initiate with metadata (path, filename, size, mime)


2. File Engine validates:

tenant membership (server mapping)

RBAC/ACL on target path

file policy (allowlists, size caps, naming rules)



3. Returns an upload session + upload URL scoped to a quarantine prefix



Complete 4. UI calls POST /v1/uploads/{uploadId}:complete 5. Worker:

scans quarantined object (scanner boundary)

if CLEAN: promotes to final namespace (tenants/<tenant>/...) using backend-appropriate atomic semantics

if MALICIOUS/UNKNOWN: moves to malware hold and marks task quarantined


6. Audit records: upload complete, scan verdict, promote/hold decision




---

Security model (README-level)

Trust boundaries:

TB1: Browser ↔ Laravel (untrusted input)

TB2: Laravel ↔ File Engine (east-west; mTLS recommended)

TB3: Queue boundary (tamper/replay/poison messages)

TB4: Storage boundary (least privilege; private endpoints)

TB5: Scanner boundary (hostile bytes; sandboxed)


Secure-by-default controls:

Deny-by-default authorization at File Engine

Tenant scope from server-side mapping (not JWT)

Strict path normalization + traversal rejection

Quarantine → scan → promote gating

Redaction policy: never log tokens or pre-signed URLs


Known gaps / planned hardening (examples):

Explicit deny rules in ACL (deny > allow) 🔒 (ADR candidate)

Signed task payloads / replay defense 🔒 (ADR candidate)

Stronger immutability guarantees for the secondary audit sink 🔒


> Detailed STRIDE model: docs/security/threat-model.md




---

Auditing

Dual-layer audit

Primary (queryable): Postgres audit_events table (append-only)

Secondary (tamper-resistant): external sink (SIEM / Loki / S3 WORM)


Audit coverage (target baseline):

Mutation events: create/move/rename/write, upload lifecycle, scan verdict, promote/hold decision

Security events: authz denials, policy failures (rate-limited + aggregated as needed)

Correlation fields: request_id, trace_id, task_id, user_id, tenant_id, operation, path, outcome



---

Observability

Standards:

JSON structured logs (consistent envelope, redaction)

Request correlation across HTTP ↔ gRPC ↔ queue ↔ worker

X-Request-Id, traceparent (W3C)


Distributed tracing via OpenTelemetry (OTLP exporter)


Operational signals to monitor:

Queue depth / worker saturation

Scan duration + pass/fail ratio

Promotion failures

Quarantine growth

403 spikes (probing / misconfig)


> Full spec: docs/observability.md




---

Quickstart (local development)

Requirements:

Go 1.21+

Docker Engine / Docker Desktop + Compose v2

curl


1) Start dependencies (Redis + Postgres)

docker compose up -d postgres redis

2) Apply migrations

export POSTGRES_DSN="postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
go run ./cmd/migrate

3) Run the stack (API + Worker)

docker compose up --build

4) Smoke test (liveness)

curl -i http://localhost:8080/healthz

5) Run unit tests

go test ./... -v

Default ports:

HTTP: 8080

gRPC: 50051

Redis: 6379

Postgres: 5432



---

Repository structure

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

Roadmap

Phase	Goal	Status

Phase 1	Browse directories + read authz baseline	🟡
Phase 2	Folder creation (async) + audit events	🟡
Phase 3	Quarantine → scan → promote + observability baseline	🟡
Phase 4	Advanced governance (fine-grained ACL, workflows, notifications)	🔒
Phase 5	Enterprise features (retention, eDiscovery-friendly audit, versioning)	🔒


Queue strategy:

Redis initially (simplicity), Kafka later (scale/replay) — see ADRs in docs/adr/.



---

Documentation map

docs/api-reference.md — File Engine API (gRPC + HTTP/JSON)

docs/auth.md — JWT + RBAC/ACL model

docs/security/threat-model.md — STRIDE + trust boundaries

docs/security/pipeline-security.md — Upload → scan → promote security analysis

docs/observability.md — logging, metrics, tracing standards

docs/STORAGE_BACKENDS.md — local/s3/gcs adapters + configuration

docs/adr/ — decisions and rationale



---

License

MIT License. See LICENSE.
