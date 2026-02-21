# Roadmap vs Capability Ledger Gap Analysis

**Date:** 2026-02-21  
**Source-of-truth rule applied:** `docs/capability-ledger.md` takes precedence over roadmap intent and narrative docs.

## Comparison summary

| Roadmap milestone | Roadmap intent | Ledger-backed status | Gap status |
| :-- | :-- | :-- | :-- |
| Milestone 0 — Baseline Integrity | Baseline CI + ledger runnable checks + doc drift + README maturity labeling | Baseline CI claim gate exists (`CL-034`), doc drift check is baseline (`CL-011`), and governance/ownership checks are baseline (`CL-041`) | **Implemented** |
| Milestone 1 — Read Path Baseline | Read/list endpoint + authz rejection + docs/demo evidence | Read/list/authz checks are baseline (`CL-012`) | **Implemented** |
| Milestone 2 — Write Path Baseline | Async create-folder + status persistence + audit + guardrails | Async folder flow/status/audit are baseline (`CL-003`, `CL-004`, `CL-005`), guardrails are baseline (`CL-017`), and backend VS-001 E2E is baseline (`CL-020`) | **Implemented** |
| Milestone 3 — Upload Pipeline | Quarantine/scan/promote + scan metrics/logging + public contract | Upload staging/promote is baseline (`CL-025`), malware gate is baseline (`CL-033`), non-stub scanner evidence is baseline (`CL-040`), and upload public contract is baseline (`CL-047`) | **Implemented** |
| Milestone 4 — Observability & Audit Sink | OTEL traces + external audit sink + production drills | OTEL export wiring and production drills are baseline (`CL-038`, `CL-044`, `CL-048`, `CL-050`) and external sink wiring + integration validation is baseline (`CL-039`, `CL-035`) | **Implemented** |
| Milestone 5 — Governance Hardening | Doc ownership metadata + quarterly alignment cadence + architecture conformance checks in CI | Governance claim `CL-041` validates key-doc ownership metadata check script, quarterly cadence documentation, and CI architecture conformance checks (proto sync + endpoint inventory) | **Implemented** |
| Milestone 6 — Target-State Hardening Promotions | Promote target-state hardening into baseline claims | Enterprise identity (`CL-042`) plus upload/observability/storage/governance hardening (`CL-043`-`CL-046`) are baseline-validated | **Implemented** |
| Milestone 7 — Production Operations Closure | Close remaining production-grade operational + documentation contract gaps | Upload API contract (`CL-047`), OTEL production drills (`CL-048`), governance next-step controls (`CL-049`), paging delivery (`CL-050`), scanner/on-call closure (`CL-051`), and README/route-matrix sync (`CL-052`) are implemented | **Implemented** |
| Milestone 8 — Sustainability & Ownership Resilience | Reduce concentration risk and harden governance sustainability | Branch-protection mapping + named backups + sustainability metrics script are baseline-backed kickoff evidence (`CL-053`); broader multi-owner resilience remains open | **In progress** |

## 7C synchronization audit (README + narrative docs)

### Resolved in this update

1. **README wording drift corrected**
   - Upload API lifecycle is no longer described as target-state where ledger already marks baseline (`CL-047`).
2. **Route maturity matrix refreshed**
   - Upload lifecycle routes, readiness/liveness endpoints, and OIDC profile validation are reflected with linked claim IDs + runnable commands.
3. **Narrative alignment refreshed**
   - Roadmap/ledger gap narrative now marks Milestone 7 as implemented instead of partially open.

## Remaining non-M7 gaps

| Open area | Current evidence | What is still missing |
| :-- | :-- | :-- |
| Ownership resilience tasks | Quarterly review flags owner concentration as residual risk | Add named backup maintainers + branch-protection mapping + sustainability metrics tracking (Milestone 8) |

## Promotion discipline (unchanged)

No baseline-language updates are allowed without:

1. claim ID in `docs/capability-ledger.md`,
2. runnable validation command,
3. CI/PR verification evidence.


## 8A sustainability kickoff audit

1. Branch-protection mapping doc added and referenced from governance (`docs/branch-protection-mapping.md`).
2. Named backup maintainers are explicit for security/platform/backend/data-plane (`docs/ownership-backup-matrix.md`).
3. Release-cadence deterministic report script added (`scripts/sustainability-metrics.sh`).
