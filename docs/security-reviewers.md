# Security Reviewer Guide — Threat Model & Controls

This document is for security review and assurance: design controls, trust boundaries, and verification steps.

---

## Security objectives

- Prevent unauthorized filesystem access and mutations
- Prevent path traversal and directory escape
- Ensure malware does not reach final storage
- Provide tamper-resistant audit trails
- Minimize blast radius via least privilege

---

## Trust boundaries

### External boundary
- Browser ↔ API (TLS)
- Token-based auth (JWT/OAuth2)

### Internal boundary
- API ↔ File Engine
- API ↔ Queue
- File Engine ↔ Filesystem
- ClamAV ↔ Staging storage

**Design principle:** The File Engine treats all requests as untrusted unless authenticated and scoped.

---

## Access control model

### RBAC (application-level)
- Roles grant capabilities (read/list/create/upload/delete etc.)
- Policies enforce per-path permissions at Laravel boundary

### Per-path authorization (fine-grained)
- Permissions contain folder_path + allowed actions
- Default deny

### Filesystem ACL strategy (must be finalized)
One of:
1) single service account with strict root constraints + internal RBAC, or
2) per-tenant service identity, or
3) user impersonation (highest complexity)

This choice must be locked via ADR.

---

## Path safety (anti-traversal)

### Required controls
- Canonicalize paths (resolve symlinks and `..`)
- Enforce root-jail (operation must remain inside allowed root)
- Deny special devices and unsafe filenames
- Prefer server-issued path references over raw user-supplied strings

### Verification checklist
- Attempt `../` traversal
- Attempt encoded traversal (`..%2F`)
- Attempt symlink escape
- Attempt absolute path injection (`/etc/…`)
- Ensure failures are logged with correlationId

---

## Upload security flow (staged → scanned → committed)

1) Upload goes to staging storage, not final filesystem  
2) Extension/MIME/size allowlist enforced  
3) Malware scan runs before commit  
4) If clean: File Engine moves/commits to final path  
5) If infected: quarantine and audit event recorded

### Required behaviors
- Scan timeouts and max file size policy
- Quarantine storage for infected uploads
- Audit log for verdict and final action

---

## Service-to-service security (target state)

- mTLS between API and File Engine
- Scoped tokens:
  - audience-restricted
  - short TTL
  - least-privilege claims (operation + jobId)
- Replay protection:
  - idempotency key per mutation job
  - strict expiration on requests

---

## Audit logging

### What must be recorded
- userId, role, tenant (if applicable)
- action type (create_folder, upload_commit, delete_file)
- target path (normalized; avoid leaking secrets)
- timestamp
- outcome (success/failure + reason)
- correlationId + jobId

### Hardening
- Append-only semantics (or WORM storage)
- Retention policy and periodic export

---

## Vulnerability assessment approach

Recommended minimum:
- SAST for PHP and Go
- Dependency scanning (SCA)
- Container scanning
- IaC scanning (K8s manifests)
- DAST for API endpoints
- Periodic threat modeling review

---

## Security test plan (starter)

- Authorization:
  - verify deny-by-default
  - verify least privilege roles
- Path safety:
  - traversal attempts (encoded and unencoded)
  - symlink escape attempts
- Upload:
  - EICAR test file should be quarantined
  - oversize file must be rejected
  - forbidden MIME must be rejected
- Service-to-service:
  - reject unauthenticated File Engine calls
  - reject expired/scoped token misuse
