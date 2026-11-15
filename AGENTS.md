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

**High-level system flow:**
User → Frontend → Backend (Laravel control plane) → File Engine (Go API + async workers) → Storage backend

### Execution boundary

The **File Engine** is the **final execution boundary**.

**Responsibilities of the File Engine:**
- filesystem mutations
- async task execution
- upload lifecycle processing
- worker orchestration

**Backend responsibilities:**
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
`./file-engine/scripts/dev.sh`
(CL-007)

**Backend**
Composer metadata validation:
```bash
cd backend
composer validate --strict
```
(CL-008)

VS-001 smoke test:
```bash
cd backend
./scripts/smoke.sh
```
(CL-018)

### Tier 1 — Cross-service integration

**Run when modifying:*"
API envelopes
routing
headers
cross-service forwarding
task polling
contracts

```bash
docker compose up -d

./scripts/wait-for-http.sh http://localhost:8080/healthz 120
./scripts/wait-for-http.sh http://localhost:8081/healthz 120

./scripts/e2e/vs001_create_folder.sh

# Always teardown
docker compose down -v
```
(CL-020)

### Tier 2 — Runtime / dependency checks

**Run when modifying:**
container images
dependency versions
bootstrap logic
test harnesses

```bash
docker compose run --rm --no-deps backend \
sh -lc 'composer install --no-interaction && ./vendor/bin/phpunit -c phpunit.xml'
```
(CL-031)
Use: `docker compose build --pull` when testing base-image changes or reproducing CI environments.


## 7. Adversarial Review & Change Verification

### Purpose

Every meaningful change must undergo a critical review before being reported as complete.

The objective is to identify defects that ordinary implementation and validation may miss, particularly security vulnerabilities, incorrect assumptions, missing test coverage, architectural violations, and unintended regressions.

Assume changes may be merged upstream unless explicitly stated otherwise.

### Review Timing

Perform adversarial review only when:

1. Implementation is complete.
2. Build and relevant tests have been executed.
3. Applicable lint and static analysis checks have been executed.
4. The change is ready for handoff.

Do not perform final adversarial review on intermediate edits.

Skip the full review process when:

- The change is purely exploratory and will not be merged.
- The human explicitly requests that review be skipped.
- The change is documentation-only or test-only, in which case perform the relevant lightweight review.

### Core Review Requirements

Every non-documentation implementation change must receive a review covering:

1. **Correctness:** Identify logical errors, incorrect assumptions, missing edge cases, race conditions, and unintended behavior.
2. **Security:** Investigate vulnerabilities, authorization bypasses, tenant isolation failures, unsafe input handling, and weakened security controls.
3. **Testing:** Identify missing positive, negative, boundary, failure-recovery, concurrency, and compatibility test cases.
4. **Maintainability:** Identify architectural violations, unnecessary complexity, duplicated logic, and deviations from repository conventions.
5. **Documentation:** Identify existing documentation, API contracts, and capability claims invalidated by the change.

Scale review depth according to change complexity, blast radius, and security impact.

### Specialist Review Triggers

Additional specialist reviews are required when applicable:

**Security review**
- Authentication, authorization, RBAC, and tenant isolation.
- Filesystem operations, path normalization, and symlink handling.
- Upload lifecycle, quarantine, and malware scanning.
- Storage adapters and access policies.
- Secrets, identity, and sensitive-data handling.

**Security hardening review**
- Changes involving filesystem syscalls, path resolution, permissions, symlinks, or storage isolation.
- Investigate safer APIs, restrictive syscall flags, atomic operations, and least-privilege enforcement.

**Performance regression review**
- Upload/download streaming.
- Filesystem traversal and metadata operations.
- Storage adapters.
- Database-intensive operations.
- Redis queues, worker concurrency, and high-throughput paths.

**API and contract review**
- Protobuf definitions and generated artifacts.
- gRPC and HTTP/JSON gateway behavior.
- Laravel-to-Go communication.
- Request/response compatibility and error semantics.

**Infrastructure and supply-chain review**
- Dockerfiles, container permissions, and network exposure.
- GitHub Actions, dependency updates, and CI/CD security.
- Image provenance, vulnerability scanning, and deployment configuration.

### Independent Reviewer Guidelines

When independent reviewers or sub-agents are available:

- Assign review responsibilities according to the affected components.
- Prefer independent reviewers who were not responsible for the implementation.
- Initially provide only the changed files and the specific review objective.
- Do not disclose implementation assumptions unless additional context is needed.
- Require findings to reference concrete code locations and explain their impact.

Do not require multiple reviewers for trivial changes. Use additional reviewers when the risk or complexity justifies their cost.

If independent reviewers are unavailable, perform an explicit adversarial self-review and disclose this limitation in the handoff.

### Findings Evaluation

Classify findings using the following severity levels:

- **Critical:** Exploitable security vulnerability, major authorization bypass, or serious data-integrity failure.
- **High:** Significant correctness, security, or reliability defect.
- **Medium:** Meaningful weakness, missing coverage, or maintainability concern.
- **Low:** Minor improvement or non-blocking observation.

For every actionable finding:

1. Verify it against the actual implementation.
2. Determine whether it is within the intended scope.
3. Fix confirmed defects or document justified deferrals.
4. Record disputed findings with supporting reasoning.
5. Do not silently dismiss material security or correctness concerns.

Reviewers must not modify the main working tree. If additional modifications are required to verify a finding, use an isolated worktree when available.

### Revalidation After Review

After applying review findings:

- Re-run relevant build and test commands.
- Re-run applicable lint and static analysis.
- Re-run claim-specific Capability Ledger validations.
- Re-run integration tests when cross-component behavior is affected.
- Run benchmarks when changes may materially affect performance.

Do not report a change as fully validated if required checks were not executed or failed.

### Review Completion and Handoff

Before reporting completion, summarize:

1. Review scope and reviewer responsibilities.
2. Reviewers used, if available.
3. Findings identified and corrections applied.
4. Findings rejected or deferred, with justification.
5. Any reviewers or checks not executed and why.
6. Final validation results.
7. Remaining risks and unknowns.

Stop additional review rounds when no new actionable findings emerge and remaining observations only repeat previously evaluated concerns.

The final implementation report must distinguish verified behavior from assumptions and unverified capabilities.

## 8. Language and Code Conventions

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
- `declare(strict_types=1)`;

Follow:
- PSR-4 structure
- valid configuration return structures

Validation should prefer: **Form Requests** instead of inline controller validation.

### API and Contract Discipline

Keep the following aligned:
- protobuf definitions
- generated artifacts
- gateway bindings
- API documentation
Top-level docs must not mark a feature as baseline unless the ledger includes a validated claim.

## 9. Engineering Principles (Guidance)

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

## 10. Repository Layout

- `backend/` Laravel control-plane
- `file-engine/` Go data-plane
- `frontend/` demo console
- `docs/` architecture/setup/security/api references/ADR
- `scripts/` validation, drill, and helper scripts 
- `docker-compose.yml` canonical multi-service dev stack

## 11. Anti-Patterns (Do Not)

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


## 12. Rapid Iteration Guardrails

When iterating quickly:
- keep commits small and reversible
- validate assumptions with code search
- favor deterministic behavior
- avoid “ship and hope” on security-sensitive paths

If uncertain:
leave an explicit **TODO with verification context** rather than hidden assumptions.

## 13. Handoff Template

When ending work or handing off changes include:

1. **What changed**
2. **What did not change** (intentional scope)
4. **Config or environment changes**
5. **Validation performed and results**
6. **Remaining risks or unknowns**
7. **Recommended next action**
