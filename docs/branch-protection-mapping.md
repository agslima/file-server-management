# Branch Protection Mapping

[//]: # (owner: Project Maintainers)
[//]: # (review_cadence: Quarterly)
[//]: # (last_reviewed: 2026-03-03)

This document maps branch-protection requirements for `main` across always-required checks, path-scoped checks, and approval ownership.

## Approval ownership model

Reference files:

- `.github/OWNERS` (baseline owner/approver/reviewer mapping)
- `.github/codeowners` (path ownership hints)
- `docs/ownership-backup-matrix.md` (named backup maintainers per domain)

Approval policy:

1. At least one approval from a listed approver in `.github/OWNERS` is required.
2. For domain-sensitive changes (`docs/security-reviewers.md`, `docs/platform-engineers.md`, backend/file-engine core flows), request review from the matching primary or backup maintainer listed in `docs/ownership-backup-matrix.md`.
3. Governance/policy files (`docs/governance.md`, `.github/workflows/*`, `docs/capability-ledger.md`) require maintainer review plus one domain backup reviewer when available.

## Required checks: always-on

These checks are required regardless of changed paths:

- `doc-drift` (documentation link + governance hygiene drift)
- `governance-hygiene` checks in CI pipeline
- PR security scan gates from `.github/workflows/ci-pr-security-scan.yaml`:
  - `Secret Scan (Gitleaks)`
  - `Trivy Gate (Dependencies)`
  - `Trivy Gate - Misconfig (IaC)`

## CodeQL policy

- CodeQL runs in `.github/workflows/codeql.yml` on `pull_request`, `push` to `main`, and scheduled cadence.
- Current governance policy treats CodeQL as advisory/non-blocking by default (not listed in always-required branch-protection checks).
- If/when promoted to required, add both matrix checks (`Analyze (actions)` and `Analyze (go)`) to required checks and update this document and `docs/governance.md` together.

## Scheduled (non-branch-protection) claim checks

- Workflow: `.github/workflows/promoted-claim-cadence.yml`
- Weekly schedule-gated claim checks:
  - `CL-009` (`CL-009 Demo Console Validation`)
  - `CL-070` (`CL-070 Deployment Realism Validation`)
  - `CL-072` (`CL-072 Security Posture Validation`)
- Manual periodic claim checks (dispatch-only):
  - `CL-071` (`CL-071 Performance Validation (Manual)`)
  - `CL-073` (`CL-073 Onboarding Validation (Manual)`)

These checks are intentionally outside branch-protection required status checks, but are mandatory evidence paths for claim maintenance cadence.

## Required checks: path-scoped

These checks become required when matching paths are touched:

| Path scope | Required checks |
| :-- | :-- |
| `file-engine/**` | `file-engine-dev` (`./file-engine/scripts/dev.sh`), gateway route race test, generated gateway drift check, `lint-go` |
| `file-engine/internal/auth*` | `Security reviewer`, `Reviewer continuity` |
| `monitoring/**`, `observability/**`, `file-engine/internal/observability/**` | `Platform reviewer`, `Reviewer continuity` |
| `backend/**` | Backend scaffold contract checks (`composer validate --strict` + VS-001 syntax/smoke checks) |
| `frontend/**` | Frontend scaffold validation |
| `docs/**` | Doc drift + governance hygiene checks (always-on) |
| `docs/capability-ledger.md` | `Reviewer continuity` (maintainer approval) |

Reviewer check expectations:

- `Security reviewer` and `Platform reviewer` are path-scoped CI checks that codify reviewer rotation expectations into branch-protection-compatible status checks.
- `.github/OWNERS` remains the canonical owner/approver/reviewer source; these path-scoped checks turn mapping guidance into enforceable CI expectations, including distinct-reviewer continuity checks when multiple critical approval groups are required.

## Release-review branch protection audit checklist

At each release cadence review:

1. Confirm all required checks above are configured as branch-protection gates.
2. Confirm path-scoped checks map to current CI workflow jobs.
3. Confirm `.github/OWNERS` and `docs/ownership-backup-matrix.md` remain synchronized.
4. Verify release checklist includes the gate `new maintainer drill executed` with `scripts/drills/new_maintainer_operability_drill.sh` evidence attached.
5. Record the review date by updating `last_reviewed` metadata.
