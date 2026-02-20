# Roadmap vs Capability Ledger Gap Analysis

**Date:** 2026-02-20  
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
| Milestone 6 — Target-State Hardening Promotions | Promote target-state hardening into baseline claims | Upload operations hardening (`CL-043`), observability hardening assets (`CL-044`), storage parity hardening (`CL-045`), and governance controls baseline (`CL-046`) are baseline-validated; enterprise identity (`CL-042`) is still 🟡 | **In progress** |
| Milestone 7 — Production Operations Closure | Close remaining production-grade operational gaps | New roadmap milestone; no final promotion claims yet | **Not started** |

## What is still not implemented (or not promoted as baseline)

### Open gaps from ledger + README

| Open area | Current evidence | What is still missing |
| :-- | :-- | :-- |
| Enterprise identity integration promotion | `CL-042` exists but is still 🟡 | Promote `CL-042` to ✅ with repeatable CI/runtime evidence and deterministic machine-checkable output |
| Upload public API promotion (`InitiateUpload`/`CompleteUpload`) | README still marks upload API routes as target-state | Add claim-backed contract + integration validations, then promote in ledger/README |
| Upload operational productionization | `CL-043` baseline covers retry/DLQ/cleanup/metrics | Complete production alerting/runbooks and keep deterministic validation evidence for ongoing readiness |
| OpenTelemetry production deployment + paging integration | `CL-044` baseline validates assets and wiring | Add production connectivity/SLO validation and paging integration drills with claim-backed checks |
| Governance next-step controls | `CL-046` baseline validates quotas/retention/legal hold/lifecycle cleanup | Implement archive-tier lifecycle + external policy distribution/drift detection with new claim-backed validations |

## Roadmap updates applied

1. Milestone 6 is now tracked as **in progress** with current evidence through `CL-046`.
2. Milestone 7 was added for remaining production-operations closure tasks.
3. Promotion discipline remains unchanged: no baseline-language updates without claim ID + runnable validation + CI evidence.
