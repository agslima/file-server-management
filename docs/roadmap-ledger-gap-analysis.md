# Roadmap vs Capability Ledger Gap Analysis

**Date:** 2026-02-19  
**Source-of-truth rule applied:** `docs/capability-ledger.md` takes precedence over roadmap intent and narrative docs.

## Comparison summary

| Roadmap milestone | Roadmap intent | Ledger-backed status | Gap status |
| :-- | :-- | :-- | :-- |
| Milestone 0 — Baseline Integrity | Baseline CI + ledger runnable checks + doc drift + README maturity labeling | Baseline CI claim gate exists (`CL-034`), doc drift check is baseline (`CL-011`), and README references claim-mapped baseline/target-state framing | **Largely implemented** |
| Milestone 1 — Read Path Baseline | Read/list endpoint + authz rejection + docs/demo evidence | Read/list/authz checks are baseline (`CL-012`) | **Implemented** |
| Milestone 2 — Write Path Baseline | Async create-folder + status persistence + audit + guardrails | Async folder flow/status/audit are baseline (`CL-003`, `CL-004`, `CL-005`), guardrails are baseline (`CL-017`) | **Implemented** |
| Milestone 3 — Upload Pipeline | Quarantine/scan/promote + scan metrics/logging | Upload staging/promote + malware gate baseline (`CL-033`) and non-stub scanner integration with metrics/log evidence baseline (`CL-040`) | **Implemented** |
| Milestone 4 — Observability & Audit Sink | OTEL traces + external audit sink | OTEL export wiring for API + worker is baseline (`CL-038`) and external audit sink minimal + integration validation is baseline (`CL-039`, `CL-035`) | **Implemented (wiring baseline); hardening remains** |
| Milestone 5 — Governance Hardening | Doc ownership metadata + quarterly alignment cadence + architecture conformance checks in CI | Governance claim `CL-041` validates key-doc ownership metadata check script, quarterly cadence documentation, and CI architecture conformance checks (proto sync + endpoint inventory) | **Implemented** |

## What is still not implemented (or not promoted as baseline)

Based on strict ledger-vs-roadmap comparison, these roadmap objectives remain open:


## Items that appear implemented and should be treated as done

These roadmap expectations have concrete baseline evidence and are effectively delivered:

- Baseline integrity controls (claim-gated validations + doc drift check).
- Read/list authz baseline.
- Async write-path baseline for create-folder, status lifecycle, and audit events.
- OTEL export wiring (API + worker) and external audit sink baseline validation.
- CI-oriented conformance checks for proto sync/artifacts and baseline verification scripts.

## Suggested next promotions (if you want roadmap closure)

1. Keep `CL-041` required in CI and expand required-doc scope when new top-level governance docs are added.

