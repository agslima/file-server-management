# Threat Model & Security Specification (STRIDE)

This document provides a comprehensive threat model for the File Engine system and maps threats to concrete security controls.
It is intended as a living document updated alongside architecture changes.

---

## 1) System Decomposition

### 1.1 Components (High-level)
- **Browser (User UI)**: Web client (Laravel UI or SPA) used by authenticated users.
- **API (App/API Gateway)**: Public-facing application API (Laravel backend or gateway) that orchestrates user flows and enforces business rules.
- **File Engine (Go service)**: gRPC-first service exposing HTTP (gRPC-Gateway) that performs authorization, issues upload sessions, and enqueues filesystem/object operations.
- **Storage**: Backend storage abstraction:
  - Local filesystem (mounted volume / NFS/SMB)
  - S3-compatible object storage (AWS S3 / MinIO)
  - Google Cloud Storage (GCS)
- **Scanner (Malware/Content Scanner)**: Optional asynchronous scanning service (AV engine, DLP, content classification).
- **File Server (Legacy/Hybrid file server)**: SMB/NFS/SFTP or on-prem/hybrid file server (may be accessed via gateway or mounted storage).

### 1.2 Primary Data Flows (Text DFD)

**Flow A — Browse/List**
1) Browser → API (HTTPS): list directories/files for a path
2) API → File Engine (gRPC/HTTP): list operation with path + user context
3) File Engine → Storage: list objects/entries
4) File Engine → API → Browser: returns listing

**Flow B — Create Folder**
1) Browser → API (HTTPS): request folder creation
2) API → File Engine (gRPC/HTTP): create_folder(path, name)
3) File Engine:
   - validates path + naming policy
   - resolves JWT → AuthContext
   - enforces RBAC/ACL (path-based)
   - enqueues task to Redis
4) Worker → Storage/File Server: performs mkdir/prefix creation
5) Browser polls TaskStatus via API/File Engine until completion

**Flow C — Upload (Two-step)**
1) Browser → API: initiate upload (metadata)
2) API → File Engine: upload:initiate
3) File Engine:
   - validates allowed types/size
   - authorizes path
   - returns upload URL (pre-signed) or internal upload endpoint
4) Browser → Storage (direct to S3/GCS) OR Browser → File Engine: upload bytes
5) File Engine/Worker → Scanner (async): scan payload
6) Worker → Storage: finalize move temp → final (atomic write semantics)
7) Browser polls TaskStatus (and scan status if exposed)

**Flow D — Move/Write**
1) Browser → API → File Engine: move/write request
2) File Engine authorizes + enqueues task
3) Worker performs move/atomic write with backend-specific semantics

---

## 2) Trust Boundaries

Trust boundaries are where data crosses between privilege levels, identities, or networks.

### TB1 — Internet boundary (Browser ↔ API)
- Untrusted client input crosses into trusted services.
- Risks: spoofing, tampering, injection, session/token theft.

### TB2 — Service boundary (API ↔ File Engine)
- Internal east-west traffic.
- Risks: service spoofing, missing mTLS, privilege escalation via forged headers.

### TB3 — Queue boundary (File Engine ↔ Redis ↔ Worker)
- Asynchronous execution path.
- Risks: task tampering, replay, poison messages, broken correlation.

### TB4 — Data boundary (Worker/File Engine ↔ Storage)
- High-impact operations affecting persistent data.
- Risks: tampering, unauthorized reads/writes, misconfigured buckets, traversal, object overwrite.

### TB5 — Scanner boundary (Worker/File Engine ↔ Scanner)
- Potentially untrusted content enters a scanning subsystem.
- Risks: scanner RCE, bypass, TOCTOU between scan and publish.

### TB6 — Hybrid boundary (Cloud ↔ File Server / On-prem)
- Cross-network connectivity; often highest friction and highest risk.
- Risks: credential leakage, man-in-the-middle, lateral movement, inconsistent access controls.

---

## 3) Assets, Security Objectives, and Assumptions

### 3.1 Key Assets
- Files and folder structure (primary business asset)
- ACL rules and authorization decisions (security-critical)
- JWT tokens and identity claims
- Task payloads and execution history (audit evidence)
- Upload URLs / pre-signed credentials
- Scanner results and quarantined objects

### 3.2 Security Objectives
- **Confidentiality**: only authorized principals can read/list objects.
- **Integrity**: only authorized principals can mutate data; writes are atomic and auditable.
- **Availability**: service remains usable; failures degrade gracefully and are observable.
- **Non-repudiation/Auditability**: mutations are attributable to a principal and traceable.

### 3.3 Assumptions (Explicit)
- JWT tokens are issued by a trusted IdP.
- File Engine validates `sub` and roles and does not trust user-supplied identity headers.
- Storage backend credentials are managed securely (prefer workload identity/roles).
- Scanner is isolated and treated as a high-risk component.

---

## 4) STRIDE Threat Modeling by Component

### 4.1 Browser (Client)
**S — Spoofing**
- Stolen JWT (XSS, token leakage), session fixation.
**T — Tampering**
- Manipulate request body to access another path/tenant.
**R — Repudiation**
- User disputes action due to missing audit trails.
**I — Information Disclosure**
- Sensitive data in browser storage, cache, or logs.
**D — DoS**
- Automated abuse: massive listing requests, upload floods.
**E — Elevation of Privilege**
- CSRF-style actions if relying on cookies; role escalation via forged claims (if JWT not verified).

### 4.2 API (Gateway / Laravel)
**S**
- Service impersonation; JWT validation misconfig.
**T**
- Parameter tampering; path traversal strings; command injection via folder names.
**R**
- Missing request IDs and audit logs.
**I**
- Leaking pre-signed URLs, stack traces, verbose errors.
**D**
- High QPS, expensive queries, large payloads.
**E**
- Broken authz checks; trusting client-supplied user/role headers.

### 4.3 File Engine (Go API + gRPC)
**S**
- Forged identity if JWT not validated; internal caller spoofing without mTLS.
**T**
- Task payload tampering; path normalization bugs; object key confusion (`..`, unicode tricks).
**R**
- Insufficient audit logs (who changed what and when).
**I**
- Logging sensitive values; exposing internal errors; listing across tenants.
**D**
- Queue flood; slow storage calls; goroutine leaks; unbounded list.
**E**
- ACL bypass via inheritance bugs; confused deputy (API acts on behalf of user without enforcing path auth).

### 4.4 Worker (Task Processor)
**S**
- Fake tasks from Redis if auth missing; compromised Redis.
**T**
- Modify task payload to write to unauthorized paths; overwrite objects.
**R**
- Missing durable execution logs; no task state persistence.
**I**
- Leaking file paths or metadata to logs; exposing scanner results.
**D**
- Hot loop retries; poison pill tasks; storage throttling.
**E**
- Worker runs with too-broad storage credentials; can mutate any tenant.

### 4.5 Storage (S3/GCS/Local FS)
**S**
- Misconfigured IAM allows unauthorized access; public bucket exposure.
**T**
- Object overwrite; tampering with placeholder folder objects; TOCTOU between scan and publish.
**R**
- No versioning; cannot prove what changed.
**I**
- Public read, overly broad list; metadata leakage.
**D**
- Storage throttling; large object spam.
**E**
- Privilege escalation via bucket policy; local FS mounts granting root access.

### 4.6 Scanner
**S**
- Spoofed “clean” result if scanner channel not authenticated.
**T**
- Tampering with scan results; bypass via race conditions.
**R**
- Missing scan audit; disputes about content.
**I**
- Scanner logs leaking file content/signatures.
**D**
- Scanner overload; archive bombs; decompression bombs.
**E**
- Scanner RCE from malicious samples; lateral movement from scanner host.

### 4.7 File Server (Hybrid SMB/NFS/SFTP)
**S**
- Credential theft; service spoofing.
**T**
- Unauthorized writes if share permissions too broad.
**R**
- Incomplete audit logs; ambiguous ownership.
**I**
- Share enumerations; accidental exposure.
**D**
- Network link saturation; lock contention.
**E**
- SMB relay / privilege escalation; gateway compromise.

---

## 5) Security Controls (Threat → Mitigation Mapping)

### 5.1 Identity & Access Controls
| Threat | Controls |
|---|---|
| Spoofing via forged JWT | Verify signature (RSA/EC preferred), enforce `iss`/`aud`, validate `exp`, reject unsigned/weak algs |
| Privilege escalation via roles | Treat roles as authoritative only from JWT; avoid trusting request headers for identity |
| Cross-tenant access | Namespace paths by tenant; enforce tenant binding (claim → path prefix mapping) |
| Worker over-privilege | Use least-privileged IAM: per-bucket/prefix roles; separate credentials per env/tenant if needed |

### 5.2 Input Validation & Path Safety
| Threat | Controls |
|---|---|
| Path traversal (`../`, unicode) | Normalize paths server-side; reject `..`, invalid UTF-8; enforce absolute path patterns |
| Folder naming injection | Strict naming policy (regex + length); server-side enforcement only |
| Object key confusion | Canonicalize keys; consistent separator rules; avoid double-encoding issues |

### 5.3 Transport Security (Trust Boundaries)
| Boundary | Controls |
|---|---|
| TB1 Browser ↔ API | TLS, HSTS, secure cookies (if used), CSP; rate limiting; CSRF protection if cookie auth |
| TB2 API ↔ File Engine | mTLS (service identity), allowlist networks, mutual auth |
| TB3 File Engine ↔ Redis ↔ Worker | Private network, auth on Redis, TLS where possible, rotate credentials |
| TB4 Worker ↔ Storage | Private endpoints/VPC peering; IAM roles; deny public access |
| TB5 Worker ↔ Scanner | mTLS, signed results, strict timeouts, isolation |
| TB6 Cloud ↔ File Server | VPN/Private Link; mTLS; jump host segmentation; least privilege |

### 5.4 Data Protection
| Threat | Controls |
|---|---|
| Data disclosure at rest | Encryption at rest (S3 SSE-KMS / GCS CMEK), encrypted volumes for local FS |
| Data disclosure in transit | TLS/mTLS everywhere; avoid logging secrets |
| Tampering / overwrite | Object versioning, write-once policy for finalized artifacts, strong ETags/checksums |
| Audit gaps | Append-only audit log sink; include user_id, request_id, trace_id, operation, path, outcome |

### 5.5 Queue & Task Integrity
| Threat | Controls |
|---|---|
| Task tampering/replay | Sign/encrypt task payload (optional), include nonce/created_at, idempotency keys, DLQ |
| Poison messages | Validation on worker, max retry limits, quarantine poison tasks |
| DoS via queue flood | Rate limit per user/tenant, queue depth alerts, autoscale worker, backpressure |

### 5.6 Malware Scanning Controls
| Threat | Controls |
|---|---|
| TOCTOU scan bypass | Two-phase publish: upload → quarantine prefix → scan → promote to final |
| Scanner RCE | Run scanner in sandbox/container, no shared credentials, network egress restrict, seccomp/AppArmor |
| Archive bombs | Depth/size limits, timeouts, content-type enforcement, decompression limits |
| Fake scan results | Signed results, authenticated channel, store scan result with object version/hash |

### 5.7 Availability & Resilience
| Threat | Controls |
|---|---|
| Dependency outage | Timeouts, retries with backoff, circuit breakers; graceful degradation |
| Storage throttling | Concurrency limits, exponential backoff, queue smoothing |
| Burst traffic | Rate limits, caching for list, pagination, request size limits |

---

## 6) Recommended Security Baseline (MVP → Production)

### MVP (must-have)
- JWT verification (RSA/EC preferred), `iss`/`aud` optional for dev but recommended for prod
- Strict path normalization + naming policy enforcement
- RBAC + ACL checks at API boundary
- Private network for Redis/Postgres; credentials not in code
- Structured audit logs for mutations

### Production additions (strongly recommended)
- mTLS between API ↔ File Engine and Worker ↔ Scanner (service identity)
- Quarantine upload + scan + promote workflow
- Object versioning + retention policies
- DLQ + idempotency keys for tasks
- Continuous security scanning (SAST/SCA/container) + SBOM + signing
- WAF/rate limiting at ingress; tenant quotas
- Immutable audit sink (SIEM/Loki + retention + access controls)

---

## 7) Open Questions / Decisions Needed (ADR candidates)
- Do we need explicit **deny rules** in ACL (deny > allow) and precedence semantics?
- Where do we persist **task execution history** (Postgres) for audit and replay safety?
- Do we require **per-tenant buckets/prefixes** for stronger isolation?
- Should uploads be always **direct-to-object-store** via pre-signed URLs, or proxied via File Engine?
- Scanner integration: synchronous gate vs async quarantine model?

---

## 8) Appendix: Suggested Threat Modeling Artifacts
- Data Flow Diagram (DFD) per flow (Create Folder, Upload, Move)
- Trust boundary map and network segmentation diagram
- STRIDE worksheet per component (this doc)
- Controls checklist mapped to security requirements (CI/CD + runtime)
