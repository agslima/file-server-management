# Roadmap vs Capability Ledger Gap Analysis

**Date:** 2026-02-24  
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
| Milestone 8 — Sustainability & Ownership Resilience | Reduce concentration risk and harden governance sustainability | Automation kickoff + closure are baseline-validated (`CL-053`, `CL-054`) | **Implemented** |
| Milestone 9 — Productization & Operations Expansion | Expand mutation surface, enterprise readiness, durability, API productization, and maintenance-cost controls | Mutation/API/governance/operations expansion and maintenance-cost reductions are baseline-validated (`CL-055`-`CL-061`) | **Implemented** |
| Milestone 10 — Remaining Target-State Closure | Close final target-state items left in ledger | Newly added roadmap milestone; claim IDs not yet created | **In progress** |

## What is still not implemented (or not promoted as baseline)

| Open area (ledger target-state) | Current evidence | What is still missing |
| :-- | :-- | :-- |
| Async task-based mutation variants beyond create-folder | API-level move/rename/delete/restore paths are baseline-validated (`CL-055`) | Add task-based variants as claim-backed capabilities with runnable validation and CI evidence |
| Multi-owner human coverage across critical domains | Automation controls for resilience/governance are baseline-validated (`CL-053`, `CL-054`) | Add additional named human maintainers/reviewers per critical domain and enforce continuity in ownership/governance process artifacts |

## Promotion discipline (unchanged)

No baseline-language updates are allowed without:

1. claim ID in `docs/capability-ledger.md`,
2. runnable validation command,
3. CI/PR verification evidence.
