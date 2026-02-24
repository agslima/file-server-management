# Roadmap — Staged Milestones

[//]: # (owner: Platform Engineering)
[//]: # (review_cadence: Monthly)
[//]: # (last_reviewed: 2026-02-21)

<!--
Build an enterprise-credible File Server platform with a PHP control-plane and a Go File Engine, focused on multi-tenant governance (RBAC/authz), auditability, and operational safety.
Progress is only considered “done” when it’s provable via the Capability Ledger (runnable validations + CI regression gates), so the repo functions like a real production platform—not just a demo.
-->

A milestone is **done** only if tests, demo evidence, and doc updates ship in the same PR.

---

## Current status snapshot (2026-02-21)

| Milestone | Status | Ledger evidence |
| :-- | :--: | :-- |
| Milestone 0 — Baseline Integrity | ✅ done | `CL-011`, `CL-034`, `CL-041` |
| Milestone 1 — Read Path Baseline | ✅ done | `CL-012` |
| Milestone 2 — Write Path Baseline | ✅ done | `CL-003`, `CL-004`, `CL-005`, `CL-017`, `CL-020` |
| Milestone 3 — Upload Pipeline | ✅ done | `CL-025`, `CL-033`, `CL-040` |
| Milestone 4 — Observability & Audit Sink | ✅ done | `CL-035`, `CL-038`, `CL-039` |
| Milestone 5 — Governance Hardening | ✅ done | `CL-041` |
| Milestone 6 — Target-State Hardening Promotions | ✅ done | `CL-042`, `CL-043`, `CL-044`, `CL-045`, `CL-046` |
| Milestone 7 — Production Operations Closure | ✅ done | `CL-047`, `CL-048`, `CL-049`, `CL-050`, `CL-051`, `CL-052` |
| Milestone 8 — Sustainability & Ownership Resilience | ✅ complete | `CL-053`, `CL-054` |

---

## Milestone 0 — Baseline Integrity (1–2 weeks)

**Goal:** Lock down the smallest stable baseline and prevent doc/runtime drift.

**Completion criteria**

1. CI gates baseline tests with no bypasses.
2. `docs/capability-ledger.md` includes runnable checks for all baseline claims.
3. Doc-drift check is green in CI.
4. README lists only baseline-validated capabilities; target-state is labeled.

---

## Milestone 1 — Read Path Baseline (2–4 weeks)

**Goal:** Deliver a minimal, validated read/list flow.

**Completion criteria**

1. Implement gRPC read/list endpoint with File Engine final authz enforcement.
2. Integration test proves read/list behavior and authz rejection.
3. Docs updated in the same PR (`README.md`, `docs/api-reference.md`, ledger claim).
4. Demo evidence: CLI or gRPC transcript in PR description.

---

## Milestone 2 — Write Path Baseline (4–6 weeks)

**Goal:** Harden the async create-folder flow as a stable reference slice.

**Completion criteria**

1. Async create-folder passes integration tests with task status persistence + audit events.
2. Basic performance guardrail (timeouts/retry behavior) documented.
3. Docs updated in the same PR; ledger claim updated if needed.
4. Demo evidence: integration test output excerpt.

---

## Milestone 3 — Upload Pipeline (6–10 weeks)

**Goal:** Introduce quarantine → scan → promote for uploads.

**Completion criteria**

1. Integration test covers clean + quarantined paths.
2. Metrics/logs for scan duration and verdict emitted.
3. Docs updated in the same PR (`README.md`, `docs/api-reference.md`, threat model notes).
4. Demo evidence: scripted scenario output.

---

## Milestone 4 — Observability & Audit Sink (10–12 weeks)

**Goal:** Close the loop on operational visibility and audit durability.

**Completion criteria**

1. OTEL export wired for API + worker (basic traces).
2. External audit sink integration with minimal validation.
3. Docs and ledger updated in the same PR.
4. Demo evidence: trace + sink output excerpt.

---

## Milestone 5 — Governance Hardening (12–16 weeks)

**Goal:** Prevent regressions and reduce operational risk.

**Completion criteria**

1. Doc ownership metadata added to key docs.
2. Quarterly alignment review cadence documented.
3. Architecture conformance checks in CI (proto sync, endpoint inventory).

---

## Milestone 6 — Target-State Hardening Promotions (16–24 weeks)

**Goal:** Convert currently documented target-state controls into baseline-validated capabilities.

**Completion evidence**

1. Enterprise identity integration promotion completed (`CL-042`) with deterministic, CI-gated OIDC validation.
2. Upload operational hardening baseline completed (`CL-043`).
3. Observability/alertability hardening baseline completed (`CL-044`).
4. Storage parity hardening baseline completed (`CL-045`).
5. Governance controls baseline completed (`CL-046`).

**Completion criteria**

1. Enterprise identity integration (`CL-042`) is baseline-validated and CI-gated.
2. Hardening controls for upload/observability/storage/governance are promoted with claim IDs and runnable validations.
3. Remaining production rollout tasks are tracked in the next milestone.

---

## Milestone 7 — Production Operations Closure (24–32 weeks)

**Goal:** Close remaining production-grade gaps after baseline promotion work.

**Current progress**

1. Upload API contract promotion is baseline-validated (`CL-047`).
2. OTEL production deployment hardening checks/drills are baseline-validated (`CL-048`).
3. Governance next-step control-plane baseline is validated (`CL-049`).

**Completion criteria**

1. Real paging-provider delivery is validated in production-like drills (beyond simulated/exporter-down drills) with deterministic verification output.
2. Upload/scanner operational alerting and runbook ownership are finalized (SLO thresholds, on-call ownership, escalation mapping).
3. `docs/capability-ledger.md`, `README.md`, and `docs/roadmap-ledger-gap-analysis.md` remain synchronized whenever production rollout controls are promoted.

---

## Milestone 8 — Sustainability & Ownership Resilience (32–40 weeks)

**Goal:** Reduce operational concentration risk and stabilize long-horizon governance.

**Status:** ✅ complete

**Completion criteria**

1. Branch-protection mapping is documented (always-required checks vs path-scoped checks) and referenced from governance docs.
2. Named backup maintainers are assigned for each core capability domain (security/platform/backend/data-plane) and reflected in ownership docs.
3. Sustainability metrics are tracked at release cadence and exported as markdown artifact for PR/release notes.
4. Quarterly alignment checklist generation is automated via script-backed issue body generation.
5. Bus-factor drill path is scripted: baseline checks, production hardening drill, secrets rotation drill, and DLQ restore drill.

---

## PR Policy (Applies to All Milestones)

1. Tests: include exact commands and output summary.
2. Demo evidence: transcript or screenshot in PR description.
3. Docs: README + relevant docs updated in same PR.
4. Capability ledger updated with claim ID + runnable validation.
