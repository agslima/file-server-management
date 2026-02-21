# Ownership Backup Matrix

[//]: # (owner: Project Maintainers)
[//]: # (review_cadence: Quarterly)
[//]: # (last_reviewed: 2026-02-21)

Named primary and backup maintainers for core domains to reduce concentration risk.

## Domain maintainers

| Domain | Primary maintainer | Backup maintainer | Reviewer rotation reference |
| :-- | :-- | :-- | :-- |
| Security (authz/audit) | Agnaldo Silva Lima | Marina Costa | `docs/security-reviewers.md` |
| Platform (observability/CI) | Agnaldo Silva Lima | Rafael Nunes | `docs/platform-engineers.md` |
| Backend control-plane | Agnaldo Silva Lima | Camila Rocha | `backend/AGENTS.md` |
| Data plane (uploads/storage) | Agnaldo Silva Lima | Bruno Almeida | `file-engine/AGENTS.md` |

## Review expectations

1. Primary maintainer owns merge readiness and capability promotion evidence.
2. Backup maintainer must be requested on PRs touching their domain scope.
3. Quarterly review rotates backup reviewer assignment for at least one domain to prevent single-person lock-in.
