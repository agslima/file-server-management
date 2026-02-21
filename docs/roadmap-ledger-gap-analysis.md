# Roadmap vs Capability Ledger Gap Analysis

**Date:** 2026-02-21  
**Source-of-truth rule applied:** `docs/capability-ledger.md` takes precedence over roadmap intent and narrative docs.

## Comparison summary

| Roadmap milestone | Roadmap intent | Ledger-backed status | Gap status |
| :-- | :-- | :-- | :-- |
| Milestone 0 — Baseline Integrity | Baseline CI + ledger runnable checks + doc drift + README maturity labeling | Baseline CI claim gate exists (`CL-034`), doc drift check is baseline (`CL-011`), and governance/ownership checks are baseline (`CL-041`) | **Implemented** |
| Milestone 1 — Read Path Baseline | Read/list endpoint + authz rejection + docs/demo evidence | Read/list/authz checks are baseline (`CL-012`) | **Implemented** |
| Milestone 2 — Write Path Baseline | Async create-folder + status persistence + audit + guardrails | Async folder flow/status/audit are baseline (`CL-003`, `CL-004`, `CL-005`), guardrails are baseline (`CL-017`), and backend VS-001 E2E is baseline (`CL-020`) | **Implemented** |
| Milestone 3 — Upload Pipeline | Quarantine/scan/promote + scan metrics/logging | Upload staging/promote is baseline (`CL-025`), malware gate is baseline (`CL-033`), and non-stub scanner metrics/log evidence is baseline (`CL-040`) | **Implemented** |
| Milestone 4 — Observability & Audit Sink | OTEL traces + external audit sink | OTEL export wiring for API + worker is baseline (`CL-038`) and external sink wiring + integration validation is baseline (`CL-039`, `CL-035`) | **Implemented** |
| Milestone 5 — Governance Hardening | Doc ownership metadata + quarterly alignment cadence + architecture conformance checks in CI | Governance claim `CL-041` validates key-doc ownership metadata check script, quarterly cadence documentation, and CI architecture conformance checks (proto sync + endpoint inventory) | **Implemented** |
| Milestone 6 — Target-State Hardening Promotions | Promote target-state hardening into baseline claims | Enterprise identity (`CL-042`) plus upload/observability/storage/governance hardening (`CL-043`-`CL-046`) are baseline-validated | **Implemented** |
| Milestone 7 — Production Operations Closure | Close remaining production-grade operational gaps | Upload API contract baseline (`CL-047`), OTEL production deployment drills/connectivity baseline (`CL-048`), and governance control-plane next-step baseline (`CL-049`) are implemented; production rollout closure remains open | **In progress** |
| Milestone 8 — Sustainability & Ownership Resilience | Reduce concentration risk and harden governance sustainability | Newly added roadmap milestone; no claim IDs yet | **Not started** |

## What is still not implemented (or not promoted as baseline)

### Open gaps from ledger + README

| Open area | Current evidence | What is still missing |
| :-- | :-- | :-- |
| Production paging-provider delivery | OTEL production deployment hardening is baseline-validated (`CL-048`) | Validate real paging provider delivery path (not only simulated drill signaling) with deterministic verification evidence |
| Upload/scanner production alerting and runbook closure | Upload operational hardening is baseline-validated (`CL-043`) | Finalize production SLO alert thresholds, on-call ownership, and runbook/escalation mapping |
| README contract wording drift | Ledger and claim table show upload API contract baseline (`CL-047`) | Remove stale README target-state language for upload API routes/methods and keep docs consistent with ledger |
| Ownership resilience tasks | Quarterly review flags owner concentration as residual risk | Add named backup maintainers + branch-protection mapping + sustainability metrics tracking (new milestone tasks) |

## Roadmap updates applied

1. Milestone 6 is now marked complete with baseline evidence through `CL-046`.
2. Milestone 7 is tracked as in progress with implemented evidence through `CL-049` and explicit remaining rollout tasks.
3. Milestone 8 was added for sustainability/ownership resilience tasks not yet claim-backed.
4. Promotion discipline remains unchanged: no baseline-language updates without claim ID + runnable validation + CI evidence.
