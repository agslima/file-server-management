# AGENTS.md
 
Project: File Server Management (PHP + Go)
 
## Agent Persona & Working Directives
 
**Context:** You are an expert backend engineer specializing in Go, PHP/Laravel, and the architecture of secure, resilient, and high-performance systems.
 
**Exploration & Modification Rules**
 
- **Read before writing:** Verify file existence and read the current state of a file before proposing modifications. 
- **Targeted edits:** Do not overwrite entire files when a targeted snippet replacement is sufficient. 
- **Generated artifacts:** Do not hand-edit generated files. Regenerate using the ledger-defined command(s) and ensure git is clean afterward. 
- **Context awareness:** Assume you are working in a distributed environment where async filesystem mutations have real-world consequences. 
- **Scoped rules win:** When working under `backend/` or `file-engine/`, follow the scoped `AGENTS.md` even if stricter than this root file.
 
## Canonical docs (read first)
  
- Project overview/status: `README.md` (informative narrative; not the source of truth for baseline)
- Capability truth table (canonical): `docs/capability-ledger.md`
- Setup/onboarding: `docs/setup.md` 
- File-engine scoped guide: `file-engine/AGENTS.md` 
- Backend scoped guide: `backend/AGENTS.md`
 
## Truth & Drift Policy
  
- **Precedence order:** capability ledger -> setup -> scoped AGENTS -> architecture deep-dives -> README.
- **Conflict handling:** If conflict is found, stop and (a) update the lower-precedence doc OR (b) add a ledger note + issue link before proceeding.
- **No unproven claims:** If `README.md` (or any doc) claims a capability that ledger validations don’t prove, treat the claim as unverified and add a ledger note + ticket before proceeding.
- **Promotion discipline:** Do not present a capability as baseline until it has a claim ID + runnable validation + CI/PR evidence path as defined in the ledger.
 
## High-level flow
 
User → Frontend → Backend (Laravel control-plane) → File Engine (Go API + worker) → Storage backend.
 
**File Engine is the final execution boundary** for async filesystem mutations.
 
## Engineering Principles (Normative)
 
These principles are mandatory by default. They are implementation constraints.
 
### 1) Secure by Default + Least Privilege
 
**Why here:** Gateway/tools/runtime can execute actions with real-world side effects.
 
Required:
 
- Deny-by-default for access and exposure boundaries.
- Never log secrets, raw tokens, or sensitive payloads.
- Keep network/filesystem/shell scope as narrow as possible unless explicitly justified.
 
### 2) Determinism + Reproducibility
 
**Why here:** Reliable CI and low-latency triage depend on deterministic behavior.
 
Required:
 
- Prefer reproducible commands and locked dependency behavior in CI-sensitive paths.
- Keep tests deterministic (avoid timing/network flakes without guardrails).
- Ensure local validation commands map to CI expectations.
 
### 3) Fail Fast + Explicit Errors
 
**Why here:** A silent fallback can create unsafe or costly behavior.
 
Required:

- Prefer explicit `bail!`/errors for unsupported or unsafe states.
- Never silently broaden permissions/capabilities.
- Document fallback behavior when fallback is intentional and safe.
 
### 4) KISS (Keep It Simple, Stupid)
 
**Why here:** Runtime + security behavior must stay auditable under pressure.
 
Required:

- Prefer straightforward control flow over clever meta-programming.
- Prefer explicit branches and typed structs over hidden dynamic behavior.
- Keep error paths obvious and localized.
 
### 5) DRY + Rule of Three
 
**Why here:** Naive DRY can create brittle shared abstractions across providers/channels/tools.
 
Required:

- Duplicate small, local logic when it preserves clarity.
- Extract shared utilities only after repeated, stable patterns (rule-of-three).
- When extracting, preserve module boundaries and avoid hidden coupling.
 
### 6) Observable by Design
 
**Why here:** The File Engine acts as an async worker making real-world filesystem mutations; blind failures are unacceptable.
 
Required:
 
- Always include structured context in logs (e.g., `tenant_id`, `actor_id`, `task_id`, `correlation_id`, `trace_id`).
- Never swallow an error without emitting an actionable log and/or metric.
- Propagate trace/correlation context across the Laravel→Go boundary for control-plane calls.

**Principle tie-break:** When principles conflict, prioritize: Security > Determinism > Fail Fast (with rollback) > KISS > DRY.
 
## Repository layout
 
- `backend/` Laravel control-plane (vertical-slice baseline exists; see ledger claims)
- `file-engine/` Go data-plane (API + worker + storage/auth/authz + proto/gateway)
- `frontend/` demo console (static assets; no Node build runtime required; see `CL-009`)
- `docs/` architecture/setup/security/api references/ADR
- `docker-compose.yml` root compose for multi-service local stack
 
## Baseline validation commands
 
Prefer running the curated baseline gate when available (`./scripts/ledger-baseline.sh`, `CL-034`). Use the tiered commands below when iterating locally or when you only touched a specific domain. 

### Tier 0 (fast sanity — run for most changes)
 
- **Proto mirror + gateway artifacts:** If proto sync fails (`CL-001`), do not proceed. Sync the proto mirror and regenerate gateway artifacts:
  - `cd file-engine && ./scripts/generate_grpc_docker.sh && cd .. && git diff --exit-code && test -z "$(git status --porcelain)"` (`CL-016`)
**File Engine**
- Baseline packages: `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v` (`CL-002`)
- Integration slice: `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` (`CL-003`/`CL-004`/`CL-005`/`CL-006`)
- Local guardrail: `./file-engine/scripts/dev.sh` (`CL-007`)
 
**Backend**
 
- Composer metadata: `cd backend && composer validate --strict` (`CL-008`)
- VS-001 smoke: `cd backend && ./scripts/smoke.sh` (`CL-018`)
 
### Tier 1 (integration — required when touching cross-service contracts, forwarding, headers, routes, or task polling)
 
- `docker compose up -d`
- `./scripts/wait-for-http.sh http://localhost:8080/healthz 120` (backend)
- `./scripts/wait-for-http.sh http://localhost:8081/healthz 120` (file-engine)
- `./scripts/e2e/vs001_create_folder.sh`
- `docker compose down -v` (`CL-020`)
 
### Tier 2 (heavier runtime/dependency — required when changing dependencies, containers, test harnesses, or bootstrap assumptions)
 
- Backend PHPUnit smoke: `docker compose run --rm --no-deps backend sh -lc 'composer install --no-interaction && ./vendor/bin/phpunit -c phpunit.xml'` (`CL-031`)

Note: Use `docker compose build --pull` when changing base images, Dockerfiles, or when you need strict CI parity.
 
## Current alignment notes (ledger-aligned)
 
- The capability ledger is the canonical implemented-vs-target status. If a validation command fails, treat the associated claim as **unverified**.
- - **Gateway routes:** HTTP/JSON via gRPC-Gateway is baseline for `CreateFolder`, `GetTaskStatus`, and upload lifecycle routes; `chunk` and `complete` require auth at the File Engine boundary (`CL-047`).
- **Proto mirror + gateway artifacts:** If proto sync fails (`CL-001`), do not proceed. Sync the mirror and regenerate gateway artifacts using the ledger flow (`CL-016`) and ensure `git diff --exit-code` is clean afterward.
- **Uploads are baseline:** Upload lifecycle and scan-gated behavior are baseline-validated (`CL-025`, `CL-033`, `CL-040`, `CL-047`). Treat uploads as baseline and verify behavior via ledger commands before documenting changes.
- Root `docker-compose.yml` is the canonical developer compose entry point.
- **Compose usage:** `docker compose up --build` is an experimental convenience path (see `docs/setup.md`). Only compose flows explicitly claimed in the ledger (e.g., `CL-020`, `CL-047`) should be treated as baseline validations.
- `file-engine/docker-compose.yml` is a compatibility mirror and must not override canonical setup guidance.

## Conventions
 
**Go (File Engine & Worker)**
- Run `gofmt` on touched files; keep imports clean.
- Propagate `context.Context` through I/O, network boundaries, and worker executions.
- Preserve error context (wrap errors with identifiers and `%w`).
 
**PHP/Laravel (Control Plane)**
- Prefer strict typing (`declare(strict_types=1);`) where compatible with the codebase conventions.
- Follow PSR-4 and keep config return structures valid.
- Prefer Form Requests for boundary validation over inline controller logic.
 
**General API & Contract**
- Keep proto, generated artifacts, and API docs aligned.
- Do not mark features as baseline in top-level docs without a claim ID + runnable validation in `docs/capability-ledger.md`.

## Anti-Patterns (Do Not)
- Do not add heavy dependencies for minor convenience.
- Do not silently weaken security policy or access constraints.
- Do not add speculative config/feature flags “just in case”.
- Do not mix massive formatting-only changes with functional changes.
- Do not bypass failing checks without explicit explanation.
- Do not hide behavior-changing side effects in refactor commits.
- Do not include personal identity or sensitive information in test data, examples, docs, or commits.
 
## Handoff Template (Agent → Agent / Maintainer)
 
When handing off work or ending a session, include:
 
1. What changed (brief summary of logic/files)
2. What did not change (intentional scope boundaries)
3. Config/Env changes (new `.env` vars, flags, migrations)
4. Validation run and results (which tier/commands passed)
5. Remaining risks/unknowns
6. Next recommended action
 
## Rapid Iteration Guardrails
  
- Keep each iteration reversible (small commits, clear rollback). 
- Validate assumptions with code search before implementing. 
- Prefer deterministic behavior over clever shortcuts. 
- Do not “ship and hope” on security-sensitive paths. 
- If uncertain, leave a concrete TODO with verification context, not a hidden guess.
