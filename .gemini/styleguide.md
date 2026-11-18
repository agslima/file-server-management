# Code Review Style Guide — File Server Management (PHP + Go)

## Priority Order
When trade-offs arise during review, evaluate in this strict order:
**Security → Determinism → Fail Fast → KISS → DRY**

---

## 1. Architectural Boundaries (Critical)
- **Strict Execution Boundary:** The system flow is `Frontend → Backend (Laravel control plane) → File Engine (Go data plane) → Storage`.
- **No Direct Filesystem Mutations Outside File Engine:** Flag as **Critical** any direct filesystem reads/writes, storage mutations, or local disk manipulation inside `backend/` (Laravel) or `frontend/`. All filesystem mutations, async task execution, upload lifecycle processing, and worker orchestration must reside exclusively in `file-engine/`.
- **Backend Scope:** `backend/` must only handle authorization, orchestration, task tracking, and control-plane logic.

---

## 2. Generated Artifacts & Protobuf Discipline
- **Do Not Hand-Edit Generated Artifacts:** Flag manual edits to generated protobuf/gRPC files, gateway bindings, or lockfiles unless accompanied by corresponding source changes (`file-engine/api/proto/fileengine.proto`).
- **Proto Mirror & Contract Sync:** When `file-engine/api/proto/fileengine.proto` changes, verify that `file-engine/proto/fileengine.proto`, generated Go/gateway stubs, and API docs are updated in lockstep.
- **Capability Ledger Integrity (`docs/capability-ledger.md`):** Flag documentation or `README.md` changes that present a feature as baseline/implemented without a corresponding Claim ID (`CL-XXX`), validation command, and evidence path in `docs/capability-ledger.md`.

---

## 3. Security & Hardening Triggers
Perform a strict security audit whenever a PR touches authentication/RBAC, tenant isolation, filesystem syscalls, path resolution, upload quarantine/malware scanning, storage adapters, secrets, or Docker/CI configs:
- **Deny-by-Default & Fail Fast:** Flag silent fallbacks, overly broad catch blocks that swallow errors, or logic that silently widens permissions.
- **Path & Filesystem Safety (`file-engine/`):** Check for path traversal, unsafe symlink resolution, non-atomic file operations, and missing tenant isolation boundaries.
- **Secrets & Sensitive Data:** Flag any logged secrets, raw tokens, credentials, or PII in application code, error messages, tests, or documentation.

---

## 4. Language-Specific Conventions

### Go (`file-engine/`)
- **Context Propagation:** `context.Context` must be propagated across API, service, worker, and storage boundaries. Flag functions performing I/O or async work that drop or replace caller context with `context.Background()`/`context.TODO()`.
- **Error Wrapping:** Errors must be wrapped using `%w` (`fmt.Errorf("...: %w", err)`) to preserve structured error context and sentinel/type inspection.
- **Formatting & Control Flow:** Require `gofmt`-compliant formatting, explicit control flow, typed structs over `map[string]any` where schemas are known, and deterministic behavior (no unseeded randomness or timing-dependent logic in tests).
- **Observability:** Async filesystem workers and execution paths must emit structured logs containing correlation/tracing fields (`tenant_id`, `actor_id`, `task_id`, `correlation_id`, `trace_id`) where available in context.

### PHP / Laravel (`backend/`)
- **Strict Types & Standards:** Every new or modified PHP file should include `declare(strict_types=1);` and adhere to PSR-4 directory/namespace conventions.
- **Request Validation:** Flag inline `$request->validate(...)` calls inside controllers; require dedicated **Form Request** classes.
- **Config Files:** Configuration files must return valid, deterministic array structures without side effects.

---

## 5. Testing, Maintainability & PR Hygiene
- **Missing Edge & Failure Tests:** For non-trivial changes, check for missing negative tests, boundary conditions, tenant-isolation tests, and concurrency/failure-recovery cases.
- **KISS & Rule of Three:** Flag speculative abstractions, premature generalizations, heavy dependencies added for minor convenience, and speculative config flags. Small duplication is acceptable until a pattern repeats three times.
- **PR Hygiene:** Flag commits/PRs that mix formatting-only churn with functional logic changes, or hide behavioral changes inside refactor PRs.
