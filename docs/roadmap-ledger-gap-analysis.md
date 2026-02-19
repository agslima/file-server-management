# Roadmap vs Capability Ledger Gap Analysis

**Date:** 2026-02-19  
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

## What is still not implemented (or not promoted as baseline)

All milestones currently in `docs/roadmap.md` (0-5) are ledger-backed.  
The remaining open work comes from the ledger's explicit target-state section and is not yet baseline-promoted.

| Open area (ledger target-state) | Current baseline evidence | What is still missing for promotion |
| :-- | :-- | :-- |
| Enterprise identity integrations (AD/LDAP/OIDC broker) | None in baseline claims | Implement at least one integration path with deterministic validation command(s), CI evidence, and claim ID(s) in ledger |
| Upload pipeline operational hardening (scanner runbooks, failure retry/DLQ controls, SLO alerts) | Upload safety behavior exists (`CL-025`, `CL-033`, `CL-040`) | Operational failure-mode controls and alerting/runbooks must be claim-backed with runnable checks |
| Full OpenTelemetry backend export + alerting hardening (collector/backend connectivity SLOs + runbooks) | OTEL wiring baseline exists (`CL-038`) | End-to-end exporter connectivity validation + alerting runbook checks must be claim-backed with runnable commands |

## Roadmap update recommendation

Add a post-Milestone 5 roadmap phase that tracks the three open target-state promotions above and requires:

1. New claim IDs in `docs/capability-ledger.md` for each promoted control.
2. Runnable validation commands and deterministic expected output for each claim.
3. Matching CI execution evidence before any README baseline-language promotion.
