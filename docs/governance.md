# Governance — Merge Gates & Workflow Policy

[//]: # (owner: Project Maintainers)
[//]: # (review_cadence: Quarterly)
[//]: # (last_reviewed: 2026-02-19)


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
3. **Dependency scan (required category)**
   - **Status:** Snyk PR scan is active in `.github/workflows/snyk-scan.yaml`.
   - **Policy:** Snyk is branch-protection gated for PR merges when backend/frontend/file-engine scopes are touched.

---

## Advanced security workflows (non-blocking for now)

These workflows continue to run but do not gate merges until they are stable and have low false positives:

- CodeQL
- Snyk snapshot / monitoring (for post-merge visibility beyond PR gate)
- Dependabot auto-merge workflows (scoped to patch/minor)

When any of these are promoted to required, update this doc and the branch protection rules together.

---

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
