# Platform Architecture — Server File Manager (Laravel + Go)

This document describes the platform-level architecture: components, trust boundaries, data flows, and operational characteristics.

---

## Goals

- Provide a **web-based file explorer** over a real filesystem (SMB/NFS/SFTP/local)
- Enforce **centralized access control** for reads and mutations
- Ensure **auditability** of every action (who/what/when/where)
- Gate uploads with **malware scanning** before final commit
- Support **asynchronous operations** for reliability and API responsiveness
- Maintain strong **path safety** (anti-traversal, canonicalization, root constraints)

---

## Non-goals (initially)

- Full file versioning and diffing
- End-user sharing links / external collaboration
- Full content indexing / search (future add-on)
- Native desktop clients (web-first)

---

## Core Components

### 1) Frontend (React / Next.js)
**Responsibilities**
- Render folder tree, listing, and breadcrumbs
- Initiate folder creation and upload requests
- Display job/status updates (polling initially; WebSockets optional)

**Security**
- Uses short-lived access tokens (JWT/OAuth)
- Never sends raw filesystem paths without server-issued context (target state)

---

### 2) API / Orchestrator (Laravel)
**Responsibilities**
- Authentication (JWT/OAuth2; optional LDAP/AD integration)
- Authorization (RBAC + per-folder/path permissions via Policies)
- Input validation (folder naming conventions, size/type limits)
- Job creation and status tracking
- Audit log persistence (immutable append-only design goal)
- Service-to-service calls or queue publishing to File Engine

**Trust Boundary**
- Laravel is the **business authorization** authority.
- Laravel does **not** execute filesystem operations directly.

---

### 3) File Engine (Go)
**Responsibilities**
- Perform filesystem operations safely and efficiently:
  - list directory contents
  - create folder
  - move/commit scanned uploads
  - rename/move/delete (controlled; roadmap)
- Enforce *execution safety*:
  - canonicalize paths
  - root-jail constraints (deny escaping allowed roots)
  - safe file mode/permissions (where applicable)
- Consume async jobs from queue or accept synchronous RPC requests

**Trust Boundary**
- Go does not trust user input.
- Go only trusts **signed/scoped requests** from Laravel (target state: mTLS + scoped token).

---

### 4) Database (PostgreSQL)
**Stores**
- Users, roles, groups
- Folder/path permissions
- Jobs and job states (async operations)
- Audit logs (append-only goal; retention policy required)

---

### 5) Queue (Redis)
**Used for**
- Async folder creation
- Upload finalization after scanning
- Potentially long-running operations (compression, archive extraction, sync)

**Required behaviors**
- Retry policy with capped attempts
- Dead-letter queue (DLQ) or quarantine topic
- Idempotency key handling

---

### 6) Temporary Upload Storage (S3/MinIO or hardened staging)
**Used for**
- Receiving large uploads without blocking API
- Standardizing the malware scanning gate before final commit

---

### 7) Malware Scanning (ClamAV)
**Used for**
- Scanning staged uploads before moving into final filesystem location
- Producing a scan verdict: `CLEAN | INFECTED | ERROR`

**Operational requirements**
- timeouts & max file size scanning policy
- concurrency limits
- quarantine storage for infected files (recommended)

---

## Primary Data Flows

### A) Authentication & Authorization
1. User authenticates via Laravel (JWT/OAuth2).
2. Frontend receives token + user context.
3. Every request is authorized via Laravel Policies (RBAC + per-path permissions).

---

### B) Directory listing (read path)
1. Frontend requests a directory listing.
2. Laravel authorizes access to the path (RBAC/permissions).
3. Laravel calls Go File Engine for listing (RPC) **or** uses a job if listing can be heavy.
4. Go canonicalizes the path, enforces root constraints, lists contents.
5. Laravel returns results and writes optional access logs.

---

### C) Folder creation (async mutation)
1. Frontend calls `POST /folders` with `parent` + `folderName`.
2. Laravel validates:
   - RBAC permission to create in that parent
   - naming convention (regex)
   - parent path is allowed for this user/tenant
3. Laravel creates a `job` record (`PENDING`) and publishes a message to the queue with:
   - jobId
   - canonical parent reference (recommended: server-issued path token)
   - folderName
   - idempotency key
4. Go consumes the job:
   - resolves server-issued reference
   - canonicalizes and root-checks the resulting path
   - executes folder creation
   - reports status back (DB update or callback)
5. Laravel marks job `SUCCEEDED` or `FAILED` and writes audit log.
6. Frontend polls job status (or gets push notification).

---

### D) Secure upload (staged → scanned → committed)
1. Frontend requests an upload session:
   - Laravel checks `can_upload` for the destination
   - Laravel returns pre-signed URL (S3/MinIO) or upload token
2. Frontend uploads file to staging (not final filesystem).
3. A scan job starts:
   - ClamAV scans staged file
   - verdict is recorded (`CLEAN/INFECTED/ERROR`)
4. If `CLEAN`:
   - Laravel publishes a “commit upload” job to Go
   - Go moves the staged object into final filesystem path
   - Laravel writes audit log + marks job success
5. If `INFECTED`:
   - file is quarantined (recommended)
   - job state becomes `QUARANTINED`
   - incident/audit events are generated

---

## Security Model

### Core principles
- **Zero Trust for paths:** never trust user-supplied paths directly.
- **Least privilege:** service accounts have minimal access.
- **Defense-in-depth:** Laravel authorizes intent; Go enforces safe execution.

### Required controls (target state)
- mTLS between Laravel ↔ Go
- Scoped service tokens with audience + expiry
- Centralized allowlist for:
  - root directories
  - file extensions/MIME types
  - max sizes and rate limits
- Path canonicalization and root-jail enforcement in Go
- Malware scanning gate before commit
- Immutable audit logs + retention policies

---

## Operational Characteristics (requirements)

### Observability
- Correlation IDs across all services
- Structured logs (JSON)
- Metrics:
  - job queue depth
  - scan duration
  - commit latency
  - error rates per operation
- Tracing (OpenTelemetry recommended)

### Reliability
- Idempotency for all mutations
- Retries with backoff
- DLQ for poisoned jobs
- Timeouts and circuit breakers

### Scaling
- Horizontal scaling for:
  - Laravel API
  - Go workers
  - scanning pool
- Concurrency caps per filesystem share/path to avoid overload

---

## Open Decisions / TODO (must be finalized via ADRs)
- Redis vs Kafka as primary queue
- REST vs gRPC between Laravel ↔ Go
- Path tokenization scheme (server-issued references)
- Filesystem credential model (single service user vs impersonation)
- Audit retention/partition strategy
