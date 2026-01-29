# Production Checklist (File Engine)

This document is a practical checklist for deploying the File Engine in a production-like environment (staging/prod).
It focuses on security, reliability, observability, and operational hygiene.

---

## 0) Reference Architecture (Minimum)

- **API** (gRPC + HTTP gateway) behind a load balancer / ingress
- **Worker** deployment (horizontally scalable)
- **Redis** (managed or HA) for queue
- **PostgreSQL** (managed or HA) for ACL persistence (and optionally task status/audit)
- **Storage backend**: Local (NFS/SMB) OR S3 OR GCS
- **Identity Provider** (IdP) that issues JWTs (RSA/EC signing)

---

## 1) Security & Identity

### JWT (Strongly recommended)
- [ ] Use **asymmetric verification** (RSA/EC). Avoid shared `JWT_SECRET` in production.
- [ ] Enforce `iss` and `aud` validation (`JWT_ISSUER`, `JWT_AUDIENCE`).
- [ ] Enforce `exp` and acceptable clock skew.
- [ ] Define a roles strategy (IdP group → `roles[]`) and document it.

### Secrets management
- [ ] Store secrets in **Vault / AWS Secrets Manager / GCP Secret Manager / Kubernetes Secrets + Sealed Secrets**.
- [ ] No secrets in Git, images, or CI logs.
- [ ] Rotate secrets on a schedule and after incidents.

### Network security
- [ ] TLS termination at the edge (Ingress / LB). Prefer **mTLS** between internal services if required.
- [ ] Restrict ingress to API only; do not expose Redis/Postgres publicly.
- [ ] Add IP allowlists or private networking for internal/admin operations.

### Authorization (ACL/RBAC)
- [ ] Document the **permission model** and path inheritance rules.
- [ ] Add admin-only controls for ACL management (control plane).
- [ ] Decide policy for “deny” semantics (currently model is allow-only unless you implement explicit deny).

### File safety (recommended)
- [ ] Enforce allowed MIME types and max sizes.
- [ ] Malware scanning workflow (async) before making uploads available.
- [ ] Content-disposition/preview policies to prevent XSS in browser contexts.

---

## 2) Reliability & Data Integrity

### Queue semantics (Redis)
- [ ] Ensure at-least-once processing semantics (idempotent tasks).
- [ ] Implement retries with exponential backoff and a dead-letter queue (DLQ).
- [ ] Define max concurrency per worker and per tenant/path (optional).

### Storage semantics
- [ ] Confirm atomic write strategy fits backend:
  - Local FS: temp file + rename
  - S3/GCS: temp object + copy + delete temp (best effort)
- [ ] Decide on object versioning (S3/GCS) for recovery.
- [ ] Define lifecycle policies for temp objects and incomplete uploads.

### Database
- [ ] Migrations applied via a controlled pipeline step (not ad hoc).
- [ ] Backups enabled (point-in-time restore if possible).
- [ ] Connection pooling tuned (`max_conns`, timeouts).

---

## 3) Observability

### Logging
- [ ] Structured JSON logs (include requestId / traceId).
- [ ] Log auth failures safely (no tokens, no secrets).
- [ ] Log critical mutations: mkdir/move/write/delete with principal + path.

### Metrics
- [ ] API metrics: request rate, latency, error rate (RED/USE).
- [ ] Worker metrics: queue depth, task duration, failures, retries, DLQ count.
- [ ] Backend metrics: storage errors, throughput.

### Tracing
- [ ] Add distributed tracing (OpenTelemetry) for API → queue → worker.
- [ ] Propagate trace context via task payloads.

### Alerts (minimum)
- [ ] API 5xx rate / latency SLO burn
- [ ] Worker failure rate / DLQ growth
- [ ] Redis CPU/memory saturation
- [ ] Postgres connection exhaustion / replication lag
- [ ] Storage error spikes

---

## 4) Deployment & Operations

### Containers
- [ ] Run as non-root where possible.
- [ ] Read-only root filesystem (if feasible).
- [ ] Drop Linux capabilities.
- [ ] Set resource requests/limits (CPU/mem).

### Kubernetes (recommended)
- [ ] Separate deployments: `file-engine-api`, `file-engine-worker`
- [ ] HPA on API (RPS/CPU) and Worker (queue depth/CPU)
- [ ] PodDisruptionBudgets for HA
- [ ] NetworkPolicies to restrict lateral movement

### CI/CD (recommended)
- [ ] Build: reproducible builds, pinned dependencies
- [ ] Security: SAST + dependency scanning + container scanning
- [ ] Supply chain: SBOM + image signing (Cosign)
- [ ] Promote images via environments (dev → staging → prod)

---

## 5) Hardening the HTTP Surface

- [ ] Enable CORS only for approved origins.
- [ ] Add rate limiting and request size limits at gateway/ingress.
- [ ] Add WAF rules (if internet-facing).
- [ ] Disable directory listing and ensure path traversal protections are enforced.
- [ ] Validate and normalize all paths server-side.

---

## 6) Disaster Recovery & Incident Response

- [ ] Document RTO/RPO targets.
- [ ] Test restore procedures (Postgres + storage data).
- [ ] Run incident drills: Redis failure, storage outage, bad deploy.
- [ ] Keep audit logs immutable (append-only or external sink).

---

## 7) Multi-tenancy (if applicable)

- [ ] Namespace all paths by tenant (`/tenants/<id>/...`).
- [ ] Enforce tenant isolation at auth layer (claims contain tenant id or mapped via directory rules).
- [ ] Consider per-tenant rate limits and quotas.

---

## 8) Go-Live Gate

A minimal go-live gate you can treat as “definition of done”:

- [ ] RSA/EC JWT verification, issuer/audience enforced
- [ ] ACL admin controls restricted and audited
- [ ] Redis + Postgres not publicly accessible
- [ ] Migrations automated
- [ ] Monitoring dashboards + alerting enabled
- [ ] Backups enabled and restore tested
- [ ] Rate limiting enabled (HTTP)
- [ ] Known failure modes documented (runbook)

---

## Appendix: Recommended Next Enhancements (Roadmap)

- Persist task status/history in Postgres for auditability and reporting
- Add explicit deny rules + precedence model (deny > allow)
- Add fine-grained permissions per operation (mkdir/move/write/delete/list)
- Add antivirus scanning pipeline + quarantining
- Add signed URLs / download tokens for controlled access
