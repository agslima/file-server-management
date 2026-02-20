# Roadmap — Staged Milestones

[//]: # (owner: Platform Engineering)
[//]: # (review_cadence: Monthly)
[//]: # (last_reviewed: 2026-02-20)

A milestone is **done** only if tests, demo evidence, and doc updates ship in the same PR.

---

## Current status snapshot (2026-02-20)

| Milestone | Status | Ledger evidence |
| :-- | :--: | :-- |
| Milestone 0 — Baseline Integrity | ✅ done | `CL-011`, `CL-034`, `CL-041` |
| Milestone 1 — Read Path Baseline | ✅ done | `CL-012` |
| Milestone 2 — Write Path Baseline | ✅ done | `CL-003`, `CL-004`, `CL-005`, `CL-017`, `CL-020` |
| Milestone 3 — Upload Pipeline | ✅ done | `CL-025`, `CL-033`, `CL-040` |
| Milestone 4 — Observability & Audit Sink | ✅ done | `CL-035`, `CL-038`, `CL-039` |
| Milestone 5 — Governance Hardening | ✅ done | `CL-041` |
| Milestone 6 — Target-State Hardening Promotions | 🟡 in progress | `CL-042` (in progress), `CL-043`, `CL-044`, `CL-045`, `CL-046` |

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

**Current progress**

1. Implemented and baseline-validated: upload operational hardening (`CL-043`), observability assets hardening (`CL-044`), storage parity hardening (`CL-045`), governance controls baseline (`CL-046`).
2. Remaining promotion blocker: enterprise identity integration (`CL-042`) is still marked in progress.

**Completion criteria**

1. Enterprise identity integration (`CL-042`) is promoted to ✅ with repeatable CI/runtime evidence (OIDC profile e2e is deterministic and claim-gated).
2. README and supporting docs remove stale target-state wording for areas now baseline-validated in `CL-043` to `CL-046`.
3. Any remaining hardening gaps are moved into the next milestone with explicit claim IDs and validation commands.

---

## Milestone 7 — Production Operations Closure (24–32 weeks)

**Goal:** Close remaining production-grade gaps after baseline promotion work.

**Completion criteria**

1. OIDC/enterprise identity path is CI-gated in default governance (not only ad-hoc/manual profile usage) and documented with operational runbook ownership.
2. Upload API contract promotion is completed for `InitiateUpload` / `CompleteUpload` with authz and quarantine-scan-promote behavior validated by runnable end-to-end checks.
3. OpenTelemetry production deployment hardening is completed: collector/backend connectivity SLO checks, alert routing, and paging integration are validated by deterministic drills.
4. Governance next-step controls are implemented with claim-backed validation:
   - archive-tier lifecycle enforcement,
   - external governance policy distribution and drift detection.
5. `docs/capability-ledger.md`, `README.md`, and `docs/roadmap-ledger-gap-analysis.md` are updated in the same PR for every promoted control.

---

## PR Policy (Applies to All Milestones)

1. Tests: include exact commands and output summary.
2. Demo evidence: transcript or screenshot in PR description.
3. Docs: README + relevant docs updated in same PR.
4. Capability ledger updated with claim ID + runnable validation.
