# Governance — Merge Gates & Workflow Policy

This document defines the **required checks for merge** and how advanced security workflows are handled. The goal is a compact, reliable core gate that scales with the project, while keeping advanced security checks running without blocking progress until they are stable.

---

## Core merge gates (required)

These checks are required for **all** merges to `main`:

1. **Build & tests (baseline)**
   - `ci.yml`:
     - Backend scaffold validation
     - Frontend scaffold validation
     - File Engine baseline checks (`./file-engine/scripts/dev.sh`)
     - Doc drift check
2. **Lint (required category)**
   - **Status:** Not yet implemented in CI.
   - **Policy:** Once a lint job exists, it becomes required. Until then, maintainers should run lint locally for touched components (Go, PHP, JS) and note the command in the PR.
3. **Dependency scan (required category)**
   - **Status:** Snyk scan workflow exists; dependency review is optional.
   - **Policy:** Snyk scan should be required once the workflow is stable and consistently green. Until then, it runs as non-blocking but must be reviewed for new high/critical findings in each PR.

---

## Advanced security workflows (non-blocking for now)

These workflows continue to run but do not gate merges until they are stable and have low false positives:

- CodeQL
- Snyk snapshot / monitoring
- Dependabot auto-merge workflows (scoped to patch/minor)

When any of these are promoted to required, update this doc and the branch protection rules together.

---

## Enforcement checklist (for maintainers)

Before merging:

1. All core merge gates are green.
2. If lint or dependency scan is not yet required in CI, the PR includes:
   - the commands run, and
   - a brief summary of results.
3. Docs are updated if behavior, contracts, or setup guidance changed.
