# Ownership Backup Matrix

[//]: # (owner: Project Maintainers)
[//]: # (review_cadence: Quarterly)
[//]: # (last_reviewed: 2026-02-25)

Named primary and backup maintainers for core domains to reduce concentration risk.

## Domain maintainers

| Domain | Primary maintainer | Backup maintainer | Critical scope (CODEOWNERS) | Reviewer continuity gate |
| :-- | :-- | :-- | :-- | :-- |
| Auth/AuthZ | Agnaldo Silva Lima | Agnaldo Silva Lima | `/file-engine/internal/auth*`, `/file-engine/internal/authz/**` | Security reviewer approval required |
| Upload pipeline + scanner | Agnaldo Silva Lima | Agnaldo Silva Lima | `/file-engine/internal/services/upload_service.go`, `/file-engine/internal/adapters/security/**` | Maintainer + domain review requested |
| Audit sink + DLQ | Agnaldo Silva Lima | Agnaldo Silva Lima | `/file-engine/internal/app/tasks/audit_*`, `scripts/drills/audit_sink_catchup_drill.sh` | Maintainer + domain review requested |
| Observability/alerts/drills | Agnaldo Silva Lima | Agnaldo Silva Lima | `/monitoring/**`, `/observability/**`, `/file-engine/internal/observability/**` | Platform reviewer approval required |
| Governance controls | Agnaldo Silva Lima | Agnaldo Silva Lima | `/docs/capability-ledger.md`, `/docs/governance.md`, `/docs/branch-protection-mapping.md` | Maintainer approval required for ledger changes |

## Review expectations

1. Primary maintainer owns merge readiness and capability promotion evidence.
2. Backup maintainer must be requested on PRs touching their domain scope.
3. CI reviewer continuity checks block merge when critical path approvals are missing.
4. Quarterly review rotates backup reviewer assignment for at least one domain to prevent single-person lock-in.
