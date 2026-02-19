# Roadmap — Staged Milestones

[//]: # (owner: Platform Engineering)
[//]: # (review_cadence: Monthly)
[//]: # (last_reviewed: 2026-02-19)


A milestone is **done** only if tests, demo evidence, and doc updates ship in the same PR.

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

## PR Policy (Applies to All Milestones)

1. Tests: include exact commands and output summary.
2. Demo evidence: transcript or screenshot in PR description.
3. Docs: README + relevant docs updated in same PR.
4. Capability ledger updated with claim ID + runnable validation.
