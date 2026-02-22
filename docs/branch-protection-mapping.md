# Branch Protection Mapping

[//]: # (owner: Project Maintainers)
[//]: # (review_cadence: Quarterly)
[//]: # (last_reviewed: 2026-02-21)

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
- Dependency/security workflow category required by governance policy (`snyk-scan`)

## Required checks: path-scoped

These checks become required when matching paths are touched:

| Path scope | Required checks |
| :-- | :-- |
| `file-engine/**` | `file-engine-dev` (`./file-engine/scripts/dev.sh`), gateway route race test, generated gateway drift check, `lint-go` |
| `file-engine/internal/authz/**` | `Security reviewer` |
| `monitoring/**` | `Platform reviewer` |
| `backend/**` | Backend scaffold contract checks (`composer validate --strict` + VS-001 syntax/smoke checks) |
| `frontend/**` | Frontend scaffold validation |
| `docs/**` | Doc drift + governance hygiene checks (always-on) |

Reviewer check expectations:

- `Security reviewer` and `Platform reviewer` are path-scoped CI checks that codify reviewer rotation expectations into branch-protection-compatible status checks.
- `.github/OWNERS` remains the canonical owner/approver/reviewer source; these path-scoped checks turn mapping guidance into enforceable CI expectations.

## Release-review branch protection audit checklist

At each release cadence review:

1. Confirm all required checks above are configured as branch-protection gates.
2. Confirm path-scoped checks map to current CI workflow jobs.
3. Confirm `.github/OWNERS` and `docs/ownership-backup-matrix.md` remain synchronized.
4. Record the review date by updating `last_reviewed` metadata.
