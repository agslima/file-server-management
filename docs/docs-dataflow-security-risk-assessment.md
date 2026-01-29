# Data Flow Security & Risk Assessment (File Processing Pipeline)

This guide focuses on the **data processing pipeline** (upload → quarantine → scan → publish) and how the system maintains confidentiality and integrity across **Browser, API, File Engine, Scanner, and Storage**.

---

## 1) Component Isolation & Trust Management

### 1.1 Browser (Untrusted)
**Trust level:** Untrusted input source  
**Isolation controls (recommended):**
- TLS (HTTPS) to API
- Strict CORS (only approved origins)
- CSP + XSS protections (to reduce token theft)
- Client-side validation is **non-security** (server must enforce all)

### 1.2 API (Entry Point / Control Plane)
**Trust level:** Trusted service boundary; must validate everything from Browser  
**Isolation controls:**
- JWT validation (signature, exp; issuer/audience in prod)
- Rate limiting + request size limits
- Server-side path normalization and naming policy enforcement
- Never trust client-provided user identity headers (derive from JWT only)
- Sanitized error responses (no stack traces, no secrets)

### 1.3 File Engine (Data Plane Orchestrator)
**Trust level:** High privilege; enforces authorization and orchestrates storage operations  
**Isolation controls:**
- mTLS (recommended) between API ↔ File Engine (prevents service spoofing)
- AuthContext derived from verified JWT (sub/roles)
- RBAC + Path ACL enforcement (deny-by-default)
- Task queue isolation (Redis in private network, auth enabled)
- Structured audit logging for mutations (user_id, path, operation, request_id, trace_id)

### 1.4 Scanner (High-risk Execution Boundary)
**Trust level:** Treat as hostile environment (it processes attacker-controlled bytes)  
**Isolation controls:**
- Sandbox/container isolation; no host mounts except controlled scratch
- Minimal network egress (ideally none)
- No long-lived credentials; access to storage only through narrow, short-lived permissions
- Timeouts + size/decompression limits to prevent bombs
- Signed/authenticated scan result channel (mTLS or message signing)

### 1.5 Storage (Durable Data Layer)
**Trust level:** Durable storage; must enforce least privilege and prevent accidental exposure  
**Isolation controls:**
- Private endpoints/VPC access, no public buckets/shares
- Encryption at rest (SSE-KMS/CMEK or encrypted volumes)
- Object versioning (recommended) for tamper recovery
- Strong IAM scoping (prefix-based, environment separation)
- Lifecycle rules for quarantine/temp objects

---

## 2) Data Flow Analysis: Upload → Storage (Step-by-step)

This flow assumes the recommended **two-phase publish model**:

**(1) Initiate upload**
1. Browser calls API: `POST /v1/uploads:initiate` with metadata (path, filename, size, mime).
2. API forwards to File Engine with JWT (or service identity + user context).
3. File Engine:
   - Validates path format and user authorization (RBAC/ACL).
   - Validates file policy: allowed MIME types, size caps, filename rules.
   - Issues upload session:
     - For S3/GCS: returns a **pre-signed upload URL** to a **quarantine prefix**
       - Example key: `quarantine/<tenant>/<uuid>/<filename>`
     - For local FS: may return an internal upload endpoint or stage to a temp directory.

**Security review**
- ✅ Prevents direct publish to final path before scanning.
- ✅ Limits what can be uploaded and where.
- ✅ Removes need for File Engine to proxy bytes (lower DoS risk), when using pre-signed URLs.

**(2) Upload bytes**
4. Browser uploads bytes:
   - Directly to object store via pre-signed URL (preferred), OR
   - To File Engine upload endpoint (if required for local FS).
5. Storage receives object in quarantine location.

**Security review**
- ✅ Pre-signed URLs should be short-lived and scoped to a single object key.
- ✅ Enforce Content-Length and content-type constraints where supported.
- ⚠️ Ensure pre-signed URL cannot be reused to overwrite other objects (key must be fixed).

**(3) Complete upload / finalize**
6. Browser calls `POST /v1/uploads/{uploadId}:complete`.
7. File Engine enqueues a task for post-processing:
   - Fetch object metadata and compute hash (optional but recommended).
   - Trigger scan (Scanner).
   - If scan passes, promote to final location (atomic move/copy semantics).
   - Persist scan result + object version/hash (recommended).

**Security review**
- ✅ “Complete” is the gate that moves the object into a publishable state.
- ⚠️ TOCTOU risk between scan and promote: mitigate by hashing and promoting the exact scanned object version.

**(4) Scan**
8. Scanner reads from quarantine location (read-only).
9. Scanner produces a signed/verified result: `CLEAN`, `MALICIOUS`, `UNKNOWN`.
10. Worker applies outcome:
   - CLEAN → promote to final (`/tenants/<id>/...`)
   - MALICIOUS → move to quarantine hold + alert + deny download
   - UNKNOWN → keep quarantined, require manual review

**Security review**
- ✅ Scanner must be isolated; it processes attacker bytes.
- ✅ Results must be authenticated to prevent “clean spoofing.”

**(5) Publish / Download**
11. Only final path is listed/downloadable by authorized users.
12. Download should be controlled:
   - signed download URLs, or
   - proxy download through API with authorization checks and content headers.

---

## 3) Vulnerability Assessment (Primary Threats)

### 3.1 Malicious File Uploads (Malware, Exploits, Phishing)
**Risks**
- Compromising endpoints that process files (scanner, previewers).
- Malware stored and later downloaded by users.
- Polyglot files and content-type deception.

### 3.2 Injection Attacks (Path, Naming, Command)
**Risks**
- Path traversal (`../`, unicode normalization tricks)
- Folder naming injection causing unintended paths
- Command injection if filenames are passed to shell commands (scanner wrappers)

### 3.3 Authorization Bypass (ACL/RBAC flaws)
**Risks**
- Miscomputed “closest path wins” inheritance
- Confused deputy: API calls File Engine without enforcing user path permission
- Over-broad worker credentials allowing cross-tenant access

### 3.4 Data Leakage (Confidentiality)
**Risks**
- Public buckets/shares, misconfigured IAM
- Logging of tokens, pre-signed URLs, sensitive file metadata
- Overly verbose errors that reveal internal paths or configs

### 3.5 Tampering & Integrity Loss
**Risks**
- Overwrite existing objects without versioning
- Replaying tasks or altering payloads in queue
- Scan results mismatch (scan A, publish B)

### 3.6 DoS / Abuse (Availability)
**Risks**
- Upload floods, massive listing requests
- Zip/decompression bombs for scanners
- Queue floods causing worker saturation

---

## 4) Defense Mechanisms (Countermeasures per Risk)

### 4.1 Malicious File Uploads
**Controls**
- Allowlist MIME types and extensions (server-side)
- Max size limits + per-tenant quotas
- Two-phase publish: **quarantine → scan → promote**
- Scanner sandbox + no privileged access
- Hashing and verification (promote only the scanned object version)
- Optional: block active content types (HTML, JS) unless required

### 4.2 Injection & Path Safety
**Controls**
- Canonical path normalization server-side
- Reject `..`, invalid UTF-8, mixed separators, and reserved device names (Windows)
- Strict naming regex and length caps (folderName/filename)
- Never call shell with untrusted strings; use exec with arg arrays (if needed)
- Encode/escape keys consistently (avoid double-encoding)

### 4.3 Authorization & Isolation
**Controls**
- JWT signature verification (RSA/EC recommended)
- Enforce `iss` and `aud` in prod
- RBAC + ACL checks at File Engine boundary (deny-by-default)
- mTLS between internal services (API ↔ File Engine; Worker ↔ Scanner)
- Least privilege IAM for worker: bucket/prefix scoping; separate envs (dev/stage/prod)
- Optional: sign task payloads or use authenticated queue channels

### 4.4 Confidentiality (Leak Prevention)
**Controls**
- Private networking for Redis/Postgres/Storage endpoints
- Encryption at rest (KMS/CMEK) and in transit (TLS/mTLS)
- Redaction policy: never log tokens, pre-signed URLs, raw file content
- Strict error handling: no stack traces to client; stable error codes

### 4.5 Integrity & Tamper Resistance
**Controls**
- Atomic writes:
  - Local: temp file → rename
  - S3/GCS: temp object → copy → delete temp
- Object versioning (recommended) + retention policies
- Idempotency keys for tasks and uploads (prevent replay causing duplicates)
- Persist task execution history for audit (recommended)
- Store scan result with object hash/version; enforce “scan-to-publish binding”

### 4.6 Availability & Abuse Resistance
**Controls**
- Rate limiting at ingress/API, per user/tenant
- Request body size limits
- Pagination for listing endpoints (and max depth)
- Worker concurrency limits + backpressure
- DLQ + bounded retries for poison messages
- Scanner timeouts and decompression limits

---

## 5) Practical “Secure-by-Default” Pipeline Recommendation

**Recommended object layout**
- Quarantine: `quarantine/<tenant>/<uploadId>/<filename>`
- Final: `tenants/<tenant>/<path>/<filename>`
- Malware hold: `malware/<tenant>/<uploadId>/<filename>`

**Publishing rule**
- Only objects under `tenants/<tenant>/...` are listable/downloadable.

**Minimum gates**
1) AuthN: JWT verified + issuer/audience (prod)
2) AuthZ: RBAC/ACL on path (both initiate and complete)
3) File policy: size + MIME allowlist
4) Scan required before promote
5) Audit log every mutation

---

## 6) Operational Checks (What SRE should monitor)
- Upload initiation rate / tenant
- Quarantine size growth
- Scan pass/fail ratio and scan time
- Promotion failures (copy/move errors)
- Access denied (403) spikes (could indicate attack probing)
- Storage public access policy drift

---

## 7) Next Engineering Enhancements
- Persist task status + audit trail in Postgres (append-only)
- Explicit deny rules in ACL (deny > allow)
- Signed task payloads (HMAC) + nonce/timestamps for replay defense
- Content fingerprinting, DLP rules, and classification labels
- Secure download tokens or signed URLs with short TTL and IP binding
