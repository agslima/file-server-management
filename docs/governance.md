# Governance — Merge Gates & Workflow Policy

This document defines the **required checks for merge** and how advanced security workflows are handled. The goal is a compact, reliable core gate that scales with the project, while keeping advanced security checks running in CI with clear promotion criteria into branch-protection requirements.

---

## Core merge gates (required)

These checks are required for **all** merges to `main`:

1. **Build & tests (baseline)**
   - `ci.yml`:
     - Backend scaffold validation
     - Frontend scaffold validation
     - File Engine baseline checks (`./file-engine/scripts/dev.sh`)
     - File Engine gateway route race test (`go test -race ./internal/server -run TestGatewayCreateFolderAndGetTaskStatusRoutes -v`)
     - File Engine generated gateway artifact drift check
     - Doc drift check
2. **Lint (required category)**
   - **Status:** Implemented in CI for Go (`lint-go` job in `.github/workflows/ci.yml`, scoped to `file-engine/**` changes).
   - **Policy:** `lint-go` is required for merge when triggered by file-engine changes.
3. **Dependency scan (required category)**
   - **Status:** Snyk PR scan is active in `.github/workflows/snyk-scan.yaml`.
   - **Policy:** Snyk is branch-protection gated for PR merges; merge is blocked until the scan is green.

---

## Advanced security workflows (non-blocking for now)

These workflows continue to run but do not gate merges until they are stable and have low false positives:

- CodeQL
- Snyk snapshot / monitoring (for post-merge visibility beyond PR gate)
- Dependabot auto-merge workflows (scoped to patch/minor)

When any of these are promoted to required, update this doc and the branch protection rules together.

---

## Enforcement checklist (for maintainers)

Before merging:

1. All core merge gates are green.
2. Docs are updated if behavior, contracts, or setup guidance changed.
