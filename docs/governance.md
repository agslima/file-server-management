# Governance — Merge Gates & Workflow Policy

[//]: # (owner: Project Maintainers)
[//]: # (review_cadence: Quarterly)
[//]: # (last_reviewed: 2026-03-03)


This document defines the **required checks for merge** and how advanced security workflows are handled. The goal is a compact, reliable core gate that scales with the project, while keeping advanced security checks running in CI with clear promotion criteria into branch-protection requirements.

---

## Core merge gates (required)

These checks are required for merges to `main`, with explicit scope where applicable:

1. **Build & tests (baseline)**
   - `ci.yml` path-scoped checks:
     - Backend scaffold contract checks (`composer validate --strict` + PHP syntax checks for VS-001 files) when backend files change.
     - Frontend scaffold validation when frontend files change.
     - File Engine baseline checks (`./file-engine/scripts/dev.sh`) when file-engine files change.
     - File Engine gateway route race test (`go test -race ./internal/server -run TestGatewayCreateFolderAndGetTaskStatusRoutes -v`) when file-engine files change.
     - File Engine generated gateway artifact drift check when file-engine files change.
   - `ci.yml` always-on governance checks:
     - Doc drift check.
     - Governance hygiene checks (README/AGENTS precedence + canonical guide links).
2. **Lint (required category)**
   - **Status:** Implemented in CI for Go (`lint-go` job in `.github/workflows/ci.yml`, scoped to `file-engine/**` changes).
   - **Policy:** `lint-go` is required for merge when triggered by file-engine changes.
3. **Security scan (required category)**
   - **Status:** PR security scan gates run in `.github/workflows/ci-pr-security-scan.yaml`:
     - `Secret Scan (Gitleaks)`
     - `Trivy Gate (Dependencies)` (filesystem OS/library vulnerabilities, `HIGH/CRITICAL`)
     - `Trivy Gate - Misconfig (IaC)` (`HIGH/CRITICAL`)
   - **Policy:** These PR security scan checks are branch-protection gated for merges to `main`.

---

## Advanced security workflows and policy

These workflows run continuously for security visibility, with explicit merge-gate policy:

- CodeQL (`.github/workflows/codeql.yml`) runs on `pull_request`, `push` to `main`, and scheduled cadence.
  - **Current policy:** advisory/non-blocking in branch protection by default.
  - **Promotion rule:** if elevated to required, gate both matrix checks (`Analyze (actions)` and `Analyze (go)`) and update this document plus `docs/branch-protection-mapping.md` in the same change set.
- Dependabot auto-merge workflows (scoped to patch/minor)

---

## Scheduled promoted-claim validation policy

To reduce drift between promoted claims and real CI evidence, periodic validation runs in `.github/workflows/promoted-claim-cadence.yml`.

- Weekly schedule-gated claims:
  - `CL-009` (demo console contract)
  - `CL-070` (deployment realism runtime-wiring/syntax checks)
  - `CL-072` (security posture scripts + focused regression suite)
- Explicit periodic/manual claims:
  - `CL-071` and `CL-073` run via `workflow_dispatch` inputs (`run_cl071`, `run_cl073`) due heavier runtime/cost profile.
- Policy requirement:
  - Any promoted claim not covered by per-PR required checks must be either schedule-gated or marked explicit periodic/manual validation in `docs/capability-ledger.md`.

---


## Branch protection mapping (Milestone 8)

- Canonical mapping: `docs/branch-protection-mapping.md`.
- Ownership source of truth: `.github/OWNERS`.
- Named domain backup maintainers: `docs/ownership-backup-matrix.md`.
- Path-scoped required reviewer checks are enforced in CI (`Security reviewer` for `file-engine/internal/authz/**`, `Platform reviewer` for `monitoring/**`), and reviewer continuity enforces distinct reviewer identities when multiple critical approval groups are required.
- Keep this document and branch-protection settings synchronized at each quarterly review.

## Quarterly alignment review cadence

- Alignment review is updated **quarterly** in `docs/project-alignment-review.md` (minimum cadence: once per quarter).
- Each quarterly review must refresh evidence commands and confirm README/ledger/governance consistency.
- Ownership metadata for key docs is enforced by `./scripts/doc-ownership-check.sh`.

## Enforcement checklist (for maintainers)

Before merging:

1. All core merge gates are green.
2. Docs are updated if behavior, contracts, or setup guidance changed.
3. Governance hygiene check is green (precedence statement and canonical AGENTS link consistency across top-level docs).

---

## Capability promotion policy (README and top-level claims)

To move a capability from target-state to baseline (or to market it as currently operational in top-level docs), all of the following are required in the same change set:

1. A claim ID in `docs/capability-ledger.md`.
2. A runnable validation command with a clear expected result.
3. Verification evidence in CI or PR checks.
4. Matching updates in `README.md` and any route/status docs that reference the capability.
