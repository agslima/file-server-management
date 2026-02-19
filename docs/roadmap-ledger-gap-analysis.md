# Roadmap vs Capability Ledger Gap Analysis

**Date:** 2026-02-19  
**Source-of-truth rule applied:** `docs/capability-ledger.md` takes precedence over roadmap intent and narrative docs.

## Comparison summary

| Roadmap milestone | Roadmap intent | Ledger-backed status | Gap status |
| :-- | :-- | :-- | :-- |
| Milestone 0 — Baseline Integrity | Baseline CI + ledger runnable checks + doc drift + README maturity labeling | Baseline CI claim gate exists (`CL-034`), doc drift check is baseline (`CL-011`), and README references claim-mapped baseline/target-state framing | **Largely implemented** |
| Milestone 1 — Read Path Baseline | Read/list endpoint + authz rejection + docs/demo evidence | Read/list/authz checks are baseline (`CL-012`) | **Implemented** |
| Milestone 2 — Write Path Baseline | Async create-folder + status persistence + audit + guardrails | Async folder flow/status/audit are baseline (`CL-003`, `CL-004`, `CL-005`), guardrails are baseline (`CL-017`) | **Implemented** |
| Milestone 3 — Upload Pipeline | Quarantine/scan/promote + scan metrics/logging | Upload staging/promote and malware gate are baseline (`CL-033` and upload tests referenced in domain table) | **Mostly implemented; scanner realism still open** |
| Milestone 4 — Observability & Audit Sink | OTEL traces + external audit sink | External sink adapters + DLQ/lag metric baseline (`CL-035`); OTEL export still listed as target-state | **Partially implemented** |
| Milestone 5 — Governance Hardening | Doc ownership metadata + quarterly alignment cadence + architecture conformance checks in CI | Conformance checks are evidenced (`CL-001`, `CL-016`, `CL-011`, `CL-034`), quarterly alignment review doc exists, but explicit doc ownership metadata standard is not clearly ledger-tracked | **Partially implemented** |

## What is still not implemented (or not promoted as baseline)

Based on strict ledger-vs-roadmap comparison, these roadmap objectives remain open:

1. **Full OpenTelemetry export pipeline (Milestone 4).**
   - The roadmap requires OTEL traces for API + worker.
   - Ledger still marks full OTEL backend export/alerting as **target-state**.

2. **Real scanner integration for upload security pipeline (Milestone 3 completion depth).**
   - Ledger confirms malware gate behavior and staged promotion semantics.
   - Ledger target-state explicitly keeps **real scanner integration** out of baseline.

3. **Doc ownership metadata as a formal hardening control (Milestone 5).**
   - Roadmap calls this out directly.
   - The ledger does not currently expose a dedicated baseline claim validating ownership metadata coverage on key docs.

## Items that appear implemented and should be treated as done

These roadmap expectations have concrete baseline evidence and are effectively delivered:

- Baseline integrity controls (claim-gated validations + doc drift check).
- Read/list authz baseline.
- Async write-path baseline for create-folder, status lifecycle, and audit events.
- External audit sink with retries/DLQ/lag metric.
- CI-oriented conformance checks for proto sync/artifacts and baseline verification scripts.

## Suggested next promotions (if you want roadmap closure)

1. Add a ledger claim + deterministic validation command for **OTEL export** wiring (even minimal trace assertion).
2. Add a ledger claim for **real scanner integration** (non-stub path) with runnable integration evidence.
3. Add a governance claim for **doc ownership metadata coverage** with a script-based check over required docs.

