# Platform Engineer Guide — Deployment & Operations

This document is for operating the platform in environments like Kubernetes.

---

## Services

- `frontend` (Next.js)
- `api-laravel` (Laravel)
- `file-engine` (Go workers + optional RPC server)
- `postgres`
- `queue` (Redis)
- `clamav`
- `object-storage` (S3/MinIO staging)

---

## Deployment model

### Kubernetes (recommended)
- Separate deployments:
  - `api-laravel`
  - `file-engine-worker` (scaled horizontally)
  - `clamav` (scaled based on scan throughput)
  - optional `file-engine-rpc`
- Use:
  - HPA for API + workers
  - PodDisruptionBudgets
  - resource limits and requests

### Docker Compose (dev only)
- Intended for local testing, not production

---

## Configuration & secrets

**Never store secrets in ConfigMaps.** Use one of:
- Vault
- AWS SSM / Secrets Manager
- GCP Secret Manager
- Kubernetes Secrets (with encryption-at-rest)

Key secrets:
- DB credentials
- queue credentials
- service-to-service auth keys/certs
- object storage credentials

---

## Observability

### Logs
- JSON structured logs for all services
- Include:
  - correlationId / requestId
  - userId (when safe)
  - jobId
  - operation type
  - filesystem target (redacted / normalized)

### Metrics (minimum)
- API:
  - request rate, latency, error rate
- Queue:
  - depth, retry count, DLQ count
- File Engine:
  - job execution latency, failures by type
- ClamAV:
  - scan duration, verdict counts, timeouts

### Tracing (recommended)
- OpenTelemetry
- Distributed trace across:
  - API request → job publish → worker execution → status update

---

## Scaling guidance

### API (Laravel)
Scale based on:
- p95 latency
- CPU / memory
- request rate

### File Engine workers
Scale based on:
- queue depth
- job duration
- filesystem share performance constraints (apply concurrency caps!)

### ClamAV
Scale based on:
- scan throughput
- timeouts
- CPU and I/O constraints

---

## Reliability patterns (must-have)

- Idempotency keys on mutations
- Retries with exponential backoff
- Dead-letter queue for poisoned jobs
- Timeouts for:
  - scan operations
  - filesystem operations
- Circuit breakers between API and file engine (if synchronous RPC used)

---

## Runbooks (starter)

### 1) Queue backlog incident
Symptoms:
- queue depth rising continuously
Actions:
- verify file-engine worker health
- check filesystem latency
- temporarily reduce inbound mutation rate (rate limiting)
- scale workers if filesystem supports it
- inspect DLQ / retry storm

### 2) Malware scanning slowdown
Symptoms:
- scan duration p95 spikes
Actions:
- scale clamav pods
- enforce scan concurrency limits
- validate max file size policy
- check storage I/O saturation

### 3) Filesystem errors / permissions mismatch
Symptoms:
- Go workers failing with access denied
Actions:
- validate service account permissions
- validate root allowlist configuration
- check SMB/NFS credentials and mounts
- confirm API permission model matches filesystem ACL strategy

---

## Backup & DR (required for production)

- Postgres: PITR + restore drills
- File server: snapshots or backup system depending on storage provider
- Object storage: lifecycle policies for staging + quarantine buckets
- RPO/RTO must be defined per environment
