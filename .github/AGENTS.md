# AGENTS.md

Project: File Server Management (PHP + Go)

## Canonical docs (read first)

- Project overview/status: `README.md`
- Capability truth table: `docs/capability-ledger.md`
- Setup/onboarding: `docs/setup.md`
- File-engine scoped guide: `file-engine/AGENTS.md`
- Backend scoped guide: `backend/AGENTS.md`

If guidance conflicts, use this precedence order: capability ledger -> setup -> scoped AGENTS -> architecture deep-dives.

## High-level flow

User -> Frontend -> Backend (Laravel control-plane) -> File Engine (Go API + worker) -> Storage backend.

File Engine is the final execution boundary for async filesystem mutations.

## Engineering Principles (Normative)

These principles are mandatory by default. They are implementation constraints.

### KISS (Keep It Simple, Stupid)

**Why here:** Runtime + security behavior must stay auditable under pressure.

Required:

- Prefer straightforward control flow over clever meta-programming.
- Prefer explicit match branches and typed structs over hidden dynamic behavior.
- Keep error paths obvious and localized.

### YAGNI (You Aren't Gonna Need It)

**Why here:** Premature features increase attack surface and maintenance burden.

Required:

- Do not add new config keys, trait methods, feature flags, or workflow branches without a concrete accepted use case.
- Do not introduce speculative “future-proof” abstractions without at least one current caller.
- Keep unsupported paths explicit (error out) rather than adding partial fake support.

### DRY + Rule of Three

**Why here:** Naive DRY can create brittle shared abstractions across providers/channels/tools.

Required:

- Duplicate small, local logic when it preserves clarity.
- Extract shared utilities only after repeated, stable patterns (rule-of-three).
- When extracting, preserve module boundaries and avoid hidden coupling.

### SRP + ISP (Single Responsibility + Interface Segregation)

**Why here:** Trait-driven architecture already encodes subsystem boundaries.

Required:

- Keep each module focused on one concern.
- Extend behavior by implementing existing narrow traits whenever possible.
- Avoid fat interfaces and “god modules” that mix policy + transport + storage.

### Fail Fast + Explicit Errors

**Why here:** Silent fallback can create unsafe or costly behavior.

Required:

- Prefer explicit `bail!`/errors for unsupported or unsafe states.
- Never silently broaden permissions/capabilities.
- Document fallback behavior when fallback is intentional and safe.

### Secure by Default + Least Privilege

**Why here:** Gateway/tools/runtime can execute actions with real-world side effects.

Required:

- Deny-by-default for access and exposure boundaries.
- Never log secrets, raw tokens, or sensitive payloads.
- Keep network/filesystem/shell scope as narrow as possible unless explicitly justified.

### Determinism + Reproducibility

**Why here:** Reliable CI and low-latency triage depend on deterministic behavior.

Required:

- Prefer reproducible commands and locked dependency behavior in CI-sensitive paths.
- Keep tests deterministic (no flaky timing/network dependence without guardrails).
- Ensure local validation commands map to CI expectations.

### Reversibility + Rollback-First Thinking

**Why here:** Fast recovery is mandatory under high PR volume.

Required:

- Keep changes easy to revert (small scope, clear blast radius).
- For risky changes, define rollback path before merge.
- Avoid mixed mega-patches that block safe rollback.
- 
## Repository layout

- `backend/` Laravel API scaffold
- `file-engine/` Go API, worker, proto, storage/auth components
- `frontend/` UI placeholder
- `docs/` architecture/setup/security/api references
- `docker-compose.yml` root compose for multi-service local stack
- `file-engine/docker-compose.yml` compatibility mirror for file-engine-local workflows

## Baseline validation commands

Run from repository root unless noted:

- `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto`
- `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v`
- `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v`
- `./file-engine/scripts/dev.sh`
- `cd backend && composer validate --strict && php -l app/Http/Controllers/FolderController.php && php -l app/Http/Controllers/TaskController.php && php -l app/Services/FileEngineService.php`
- `test -f frontend/README.md && test ! -f frontend/package.json`

## Current alignment notes (validated)

- Proto mirror sync is expected to hold (`cmp` check).
- File-engine baseline and integration checks are expected to pass via `scripts/dev.sh`.
- Backend is scaffold-level; composer metadata validation is used as baseline signal.
- Frontend is intentionally placeholder (README present, no `package.json`).

## Known gaps to keep in mind

- Some docs still describe target-state capabilities as implemented; verify with capability ledger commands before claiming support.
- HTTP/JSON via gRPC-Gateway is baseline for `CreateFolder` + `GetTaskStatus`; upload routes remain target-state.
- `file-engine/docker-compose.yml` is a compatibility mirror and should not override canonical setup guidance.
- Root `docker-compose.yml` is the canonical developer compose entry point.

## Conventions

- Go: keep imports/module usage clean, run `gofmt`, and preserve error context.
- PHP/Laravel: follow PSR-4 and valid config return structures.
- API contract: keep proto, generated artifacts, and API docs aligned.
- Capability promotion: do not mark features as baseline in top-level docs without a claim ID + runnable validation in `docs/capability-ledger.md`.

## 10) Anti-Patterns (Do Not)

- Do not add heavy dependencies for minor convenience.
- Do not silently weaken security policy or access constraints.
- Do not add speculative config/feature flags “just in case”.
- Do not mix massive formatting-only changes with functional changes.
- Do not modify unrelated modules “while here”.
- Do not bypass failing checks without explicit explanation.
- Do not hide behavior-changing side effects in refactor commits.
- Do not include personal identity or sensitive information in test data, examples, docs, or commits.
- Do not attempt repository rebranding/identity replacement unless maintainers explicitly requested it in the current scope.
- Do not introduce new platform surfaces (for example `web` apps, dashboards, frontend stacks, or UI portals) unless maintainers explicitly requested them in the current scope.

## 11) Handoff Template (Agent -> Agent / Maintainer)

When handing off work, include:

1. What changed
2. What did not change
3. Validation run and results
4. Remaining risks / unknowns
5. Next recommended action

## 12) Vibe Coding Guardrails

When working in fast iterative mode:

- Keep each iteration reversible (small commits, clear rollback).
- Validate assumptions with code search before implementing.
- Prefer deterministic behavior over clever shortcuts.
- Do not “ship and hope” on security-sensitive paths.
- If uncertain, leave a concrete TODO with verification context, not a hidden guess.
