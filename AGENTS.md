# AGENTS.md
 
Project: File Server Management (PHP + Go)

## 1. Core Operating Rules (Non-Negotiable)

Agents working in this repository must follow these rules.

**Read before editing**
- Verify the file exists.
- Read the relevant section and nearby context before modifying behavior.
- Check for related callers/tests when changing logic.

**Prefer targeted edits**
- Do **not rewrite entire files** when a localized change is sufficient.

**Do not hand-edit generated artifacts**
Generated files must not be manually edited.
If a generated artifact is incorrect:
  1. Regenerate using the canonical repository command.
  2. Commit the regenerated output if the repository tracks it.
  3. Verify no unintended diffs remain.

**Do not claim capabilities without verification**
Implemented capability status is defined **only** by: `docs/capability-ledger.md`

If a capability claim fails its validation command:
  - treat the capability as unverified
  - do not present it as implemented
  - document the discrepancy before proceeding

**Do not bypass validation failures**
If a validation command fails:
  - investigate the cause
  - or mark the claim **unverified**
Do not skip validation gates.

**Preserve architectural boundaries**
System flow:
Frontend → Backend (Laravel control plane) → File Engine (Go execution boundary)

**Filesystem mutations must only occur in the File Engine.**
Backend and frontend must not directly mutate storage.

**Scoped rules override root rules**

If working within a directory containing its own `AGENTS.md`:
  - follow the **nearest scoped AGENTS.md**
  - even if it is stricter than this root document
Current scoped guides:
  - `backend/AGENTS.md`
  - `file-engine/AGENTS.md`
  - `frontend/AGENTS.md`
  
## 2. Source of Truth Model
Different documents govern different domains.

| Domain | Canonical Source |
|-----|--------------------|
| Implemented capability status | `docs/capability-ledger.md` |
| Local development / bootstrap | `docs/setup.md` |
| Subsystem rules | nearest scoped `AGENTS.md` |
| Architecture narrative | `README.md` and deep-dive docs |

**Conflict handling**

If documentation conflicts:
  1. **Capability status** is decided by the **capability ledger**.
  2. **Subsystem behavior/workflow** is decided by the **nearest scoped AGENTS.md**.
  3. **Local setup behavior** follows `docs/setup.md` unless a claim validation specifies otherwise.

When conflicts are discovered:
  - update the lower-authority document when possible
  - otherwise record the discrepancy during handoff.
  
## Canonical Documentation Map

Agents should consult these documents first.

| Task | Canonical Location |
|-----|--------------------|
| Feature truth / implemented status | `docs/capability-ledger.md` |
| Project overview | `README.md` |
| Setup and local dev | `docs/setup.md` |
| File Engine subsystem guide | `file-engine/AGENTS.md` |
| Backend subsystem guide | `backend/AGENTS.md` |

**Rule:** When uncertain, start with the Capability Ledger and follow its validation commands.
- If a task affects filesystem mutations, treat `file-engine/` as the authority.

## 4. Architecture Overview

High-level system flow:
User
 → Frontend
 → Backend (Laravel control plane)
 → File Engine (Go API + async workers)
 → Storage backend

### Execution boundary

The **File Engine** is the **final execution boundary**.

Responsibilities of the File Engine:
- filesystem mutations
- async task execution
- upload lifecycle processing
- worker orchestration

Backend responsibilities:
- authorization
- orchestration
- task tracking
- control-plane logic


## 5. Capability Ledger Workflow

The **capability ledger** defines the implementation truth for features.
`docs/capability-ledger.md`

Each claim includes:
- Claim ID
- Feature description
- Validation command
- Evidence path

**Required workflow**
1. Identify the feature area.
2. Locate the relevant claim(s).
3. Run the associated validation command before modifying behavior.

Example feature areas:
| **Area** | **Example Claim IDs** |
|---|---|
| Proto / API contract | CL-001, CL-013, CL-016 |
| Async task system | CL-003, CL-004, CL-017 |
| Upload pipeline | CL-025, CL-033, CL-040, CL-047 |
| Auth / AuthZ | CL-012, CL-014, CL-015 |
| Observability | CL-010, CL-038, CL-044 |
| Governance controls | CL-046, CL-049 |
| Audit & compliance | CL-022, CL-032, CL-035 |
| Backend control-plane | CL-008, CL-018, CL-020, CL-031 |

A feature should **not be documented as baseline** unless it has:
- a claim ID
- runnable validation
- CI or PR evidence path


## 6. Validation Workflow

Prefer running the curated baseline validation when available.
`./scripts/ledger-baseline.sh`

Use the tiered commands below when iterating locally.

### Tier 0 — Fast sanity checks

Run for most changes.

**Proto + gateway artifacts**
- Proto mirror must match:
`cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto`
(CL-001)

If proto changes:
```bash
cd file-engine
./scripts/generate_grpc_docker.sh
cd ..
git diff --exit-code
test -z "$(git status --porcelain)"
```
(CL-016)

**File Engine**
Baseline packages:
```bash
cd file-engine
go test ./internal/config ./internal/logger ./internal/worker -v
```
(CL-002)

Integration slice:
```bash
cd file-engine
go test ./tests/integration -run TestAsyncCreateFolderFlow -v
```
(CL-003 / CL-004 / CL-005 / CL-006)
Local guardrail:
./file-engine/scripts/dev.sh

(CL-007)

Backend
Composer metadata validation:
cd backend
composer validate --strict

(CL-008)
VS-001 smoke test:
cd backend
./scripts/smoke.sh

(CL-018)

Tier 1 — Cross-service integration
Run when modifying:
API envelopes
routing
headers
cross-service forwarding
task polling
contracts
docker compose up -d

./scripts/wait-for-http.sh http://localhost:8080/healthz 120
./scripts/wait-for-http.sh http://localhost:8081/healthz 120

./scripts/e2e/vs001_create_folder.sh

# Always teardown
docker compose down -v

(CL-020)

Tier 2 — Runtime / dependency checks
Run when modifying:
container images
dependency versions
bootstrap logic
test harnesses
docker compose run --rm --no-deps backend \
sh -lc 'composer install --no-interaction && ./vendor/bin/phpunit -c phpunit.xml'

(CL-031)
Use:
docker compose build --pull

when testing base-image changes or reproducing CI environments.


## 7. Language and Code Conventions

### Go (File Engine)

Required:
- run `gofmt` on touched files
- propagate `context.Context` across boundaries
- wrap errors with `%w`
- preserve structured error context

Prefer:
- explicit control flow
- typed structs
- deterministic behavior

### PHP / Laravel (Backend)

Prefer:
- declare(strict_types=1);

Follow:
- PSR-4 structure
- valid configuration return structures

Validation should prefer:
**Form Requests**
instead of inline controller validation.

### API and Contract Discipline

Keep the following aligned:
- protobuf definitions
- generated artifacts
- gateway bindings
- API documentation
Top-level docs must not mark a feature as baseline unless the ledger includes a validated claim.

## 8. Engineering Principles (Guidance)

These principles guide implementation decisions.

**Secure by default**
- deny-by-default boundaries
- never log secrets or raw tokens
- minimize network and filesystem exposure

**Deterministic behavior**
- CI and local commands should behave consistently
- avoid timing or network flakiness in tests

**Fail fast**
- prefer explicit errors over silent fallback
- never silently widen permissions

**KISS**
- prefer clear control flow
- avoid unnecessary abstractions

**DRY with the rule of three**
- small duplication is acceptable
- extract shared utilities only after repeated patterns

**Observable systems**
- Async filesystem workers must emit actionable telemetry.

Prefer structured logging fields such as:
  - tenant_id
  - actor_id
  - task_id
  - correlation_id
  - trace_id

> **When principles conflict, prioritize:**
> Security → Determinism → Fail Fast → KISS → DRY


## 9. Repository Layout

- `backend/` Laravel control-plane
- `file-engine/` Go data-plane
- `frontend/` demo console
- `docs/` architecture/setup/security/api references/ADR
- `scripts/` validation, drill, and helper scripts 
- `docker-compose.yml` canonical multi-service dev stack


##10. Anti-Patterns (Do Not)

Avoid introducing:
- heavy dependencies for minor convenience
- speculative configuration flags
- speculative abstractions
- silent security policy weakening
- behavior changes hidden inside refactor commits

Do not:
- mix formatting-only changes with logic changes
- bypass failing validation gates
- include personal or sensitive data in tests or docs


## 11. Rapid Iteration Guardrails

When iterating quickly:
- keep commits small and reversible
- validate assumptions with code search
- favor deterministic behavior
- avoid “ship and hope” on security-sensitive paths

If uncertain:
leave an explicit **TODO with verification context** rather than hidden assumptions.

## 12. Handoff Template

When ending work or handing off changes include:

1. **What changed**
2. **What did not change** (intentional scope)
4. **Config or environment changes**
5. **Validation performed and results**
6. **Remaining risks or unknowns**
7. **Recommended next action**



----


## Canonical docs (read first)
  
- Project overview/status: `README.md` (informative narrative; not the source of truth for baseline)
- Capability truth table (canonical): `docs/capability-ledger.md`
- Setup/onboarding: `docs/setup.md` 
- File-engine scoped guide: `file-engine/AGENTS.md` 
- Backend scoped guide: `backend/AGENTS.md`

## Agent Quick Map

Use this map to find the authoritative source for each task type.

| Task | Canonical Location |
|-----|--------------------|
| Feature truth / implemented status | `docs/capability-ledger.md` |
| Architecture / system overview | `README.md` |
| Local run / debugging | `docs/setup.md` |
| File Engine changes | `file-engine/AGENTS.md` |
| Backend changes | `backend/AGENTS.md` |
| Baseline validation | `./scripts/ledger-baseline.sh` |

**Rule:** When uncertain, start with the Capability Ledger and follow its validation commands.
- If a task affects filesystem mutations, treat `file-engine/` as the authority.

## Truth & Drift Policy
  
- **Precedence order:** capability ledger -> setup -> scoped AGENTS -> architecture deep-dives -> README.
- **Conflict handling:** If conflict is found, stop and (a) update the lower-precedence doc OR (b) add a ledger note + issue link before proceeding.
- **No unproven claims:** If `README.md` (or any doc) claims a capability that ledger validations don’t prove, treat the claim as unverified and add a ledger note + ticket before proceeding.
- **Promotion discipline:** Do not present a capability as baseline until it has:
  - a claim ID in the ledger,
  - a runnable validation command,
  - CI/PR evidence path as defined in the ledger.

## Capability Discovery Shortcut

The capability ledger is the canonical truth for implemented features.

When working on a feature, locate the relevant claim quickly:

| Feature Area | Example Claim IDs |
|---|---|
| Proto / API contract | CL-001, CL-013, CL-016 |
| Async task system | CL-003, CL-004, CL-017 |
| Upload pipeline | CL-025, CL-033, CL-040, CL-047 |
| Auth / AuthZ | CL-012, CL-014, CL-015 |
| Observability | CL-010, CL-038, CL-044 |
| Governance controls | CL-046, CL-049 |
| Audit & compliance | CL-022, CL-032, CL-035 |
| Backend control-plane | CL-008, CL-018, CL-020, CL-031 |

Workflow:

1. Identify the feature area.
2. Find and run the related claim validations before changing code or docs.

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

> **Principle tie-break:** When principles conflict, prioritize: Security > Determinism > Fail Fast (with rollback) > KISS > DRY.
 
## Repository layout
 
- `backend/` Laravel control-plane
- `file-engine/` Go data-plane
- `frontend/` demo console
- `docs/` architecture/setup/security/api references/ADR
- `scripts/` validation, drill, and helper scripts 
- `docker-compose.yml` canonical multi-service dev stack
## Baseline validation commands
 
Prefer running the curated baseline gate when available (`./scripts/ledger-baseline.sh`, `CL-034`). Use the tiered commands below when iterating locally or when you only touched a specific domain. 

### Tier 0 (fast sanity — run for most changes)

**Proto + gateway artifacts**
- Proto mirror must match:  
  `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto` (`CL-001`)
- If proto changed (or `cmp` fails), regenerate gateway artifacts and ensure git is clean:  
  `cd file-engine && ./scripts/generate_grpc_docker.sh && cd .. && git diff --exit-code && test -z "$(git status --porcelain)"` (`CL-016`)

**File Engine**
- Baseline packages: `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v` (`CL-002`)
- Integration slice: `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` (`CL-003`/`CL-004`/`CL-005`/`CL-006`)
- Local guardrail: `./file-engine/scripts/dev.sh` (`CL-007`)

**Backend**
- Composer metadata: `cd backend && composer validate --strict` (`CL-008`)
- VS-001 smoke: `cd backend && ./scripts/smoke.sh` (`CL-018`)
 
### Tier 1 integration checks

Run when touching cross-service contracts, forwarding, API envelopes, headers, routing, or task polling.

```bash
docker compose up -d
./scripts/wait-for-http.sh http://localhost:8080/healthz 120
./scripts/wait-for-http.sh http://localhost:8081/healthz 120
./scripts/e2e/vs001_create_folder.sh
# ALWAYS run teardown, even if the e2e script fails:
docker compose down -v
```
(`CL-020`)
 
### Tier 2 — runtime / dependency checks

Run when changing dependencies, containers, test harnesses, bootstrap assumptions
- Backend PHPUnit smoke: `docker compose run --rm --no-deps backend sh -lc 'composer install --no-interaction && ./vendor/bin/phpunit -c phpunit.xml'` (`CL-031`)

Note: Use `docker compose build --pull` when changing base images, Dockerfiles, or when you need strict CI parity.
 
## Current alignment notes (ledger-aligned)
 
- The capability ledger is the canonical implemented-vs-target status.
- If a validation command fails, treat the associated claim as **unverified**.
- **Uploads are baseline:** Upload lifecycle and scan-gated behavior are baseline-validated (`CL-025`, `CL-033`, `CL-040`, `CL-047`). Treat uploads as baseline and verify behavior via ledger commands before documenting changes.
- **Gateway routes:** HTTP/JSON via gRPC-Gateway is baseline for `CreateFolder`, `GetTaskStatus`, and upload lifecycle routes; `chunk` and `complete` require auth at the File Engine boundary (`CL-047`).
- **Proto + generated artifacts:** follow Tier 0 commands for mirror validation (`CL-001`) and gateway regeneration (`CL-016`).
- **Compose usage:** `docker compose up --build` is an experimental convenience path (see `docs/setup.md`). Only compose flows explicitly claimed in the ledger (e.g., `CL-020`, `CL-047`) should be treated as baseline validations.
- `file-engine/docker-compose.yml` is a compatibility mirror and must not override canonical setup guidance.
- For debugging beyond baseline validations, use the scaffold local-run workflow in `docs/setup.md`.

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
