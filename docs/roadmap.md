# Roadmap — Staged Milestones

[//]: # (owner: Platform Engineering)
[//]: # (review_cadence: Monthly)
[//]: # (last_reviewed: 2026-02-19)

A milestone is **done** only if tests, demo evidence, and doc updates ship in the same PR.

---

## Current status snapshot (2026-02-19)

| Milestone | Status | Ledger evidence |
| :-- | :--: | :-- |
| Milestone 0 — Baseline Integrity | ✅ done | `CL-011`, `CL-034`, `CL-041` |
| Milestone 1 — Read Path Baseline | ✅ done | `CL-012` |
| Milestone 2 — Write Path Baseline | ✅ done | `CL-003`, `CL-004`, `CL-005`, `CL-017`, `CL-020` |
| Milestone 3 — Upload Pipeline | ✅ done | `CL-025`, `CL-033`, `CL-040` |
| Milestone 4 — Observability & Audit Sink | ✅ done | `CL-035`, `CL-038`, `CL-039` |
| Milestone 5 — Governance Hardening | ✅ done | `CL-041` |

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

**Completion criteria**

1. Enterprise identity integrations (AD/LDAP/OIDC broker) are implemented behind explicit config gates and validated with runnable auth flow checks.
2. Upload pipeline operational hardening is baseline-validated:
   - scanner deployment + failure-mode runbook published,
   - failure retry/DLQ controls are validated with deterministic tests,
   - SLO-aligned scanner alerts are documented and testable.
3. Full OTEL backend export + alerting pipeline hardening is baseline-validated:
   - collector/backend connectivity checks are runnable,
   - alerting/runbook paths are documented with deterministic verification steps.
4. Each promoted area is added to `docs/capability-ledger.md` with new claim IDs, runnable validation commands, and CI evidence before README baseline language changes.

---

## PR Policy (Applies to All Milestones)

1. Tests: include exact commands and output summary.
2. Demo evidence: transcript or screenshot in PR description.
3. Docs: README + relevant docs updated in same PR.
4. Capability ledger updated with claim ID + runnable validation.
