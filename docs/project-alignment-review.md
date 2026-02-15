# Project Alignment & Improvement Review (2026-02-15)

## Scope and Method

This review evaluates the repository against the requested objectives:

1. **README alignment with technical documentation, workflows, governance, and policies**.
2. **Critical analysis** of architecture, process efficiency, and sustainability.
3. **Prioritized recommendations** for immediate and long-term improvements.

Primary sources reviewed:

- `README.md`, `docs/README.md`, `docs/capability-ledger.md`, and selected docs under `docs/`.
- Governance and workflow artifacts under `.github/` (including `.github/AGENTS.md`, CI workflows, PR template, OWNERS/CODEOWNERS).
- Runtime definitions (`docker-compose.yml`, `docker/docker-compose.yml`) and key service docs (`file-engine/README.md`, `docs/setup.md`).
- Validation commands listed in the README/capability ledger.

---

## 1) Documentation Alignment & Validation

## 1.1 Executive verdict

**Overall status: Mostly aligned at baseline level, with meaningful documentation drift in operational and governance details.**

The project now does a good job of explicitly labeling maturity (`✅`, `🟡`, `🔒`) and linking claims to validation commands in `docs/capability-ledger.md`. However, several secondary docs and runtime files still describe older architecture paths and startup flows that are no longer accurate.

## 1.2 Where implementation matches README claims

The README’s baseline “implemented” claims are currently reproducible:

- Canonical proto mirror sync check passes (`cmp ...api/proto... ...proto...`).
- Baseline Go module checks pass for config/logger/worker packages.
- Async folder flow integration test passes and includes task/audit/correlation behavior.
- Known-working local check script passes.
- Backend scaffold validation passes (`composer validate --strict`).
- Frontend remains intentionally placeholder-level (README present, no package.json).

This is a significant improvement over earlier “aspirational-only” positioning because claims now map to runnable checks.

## 1.3 Where documentation/policies are still misaligned

1. **Legacy compose file still contradicts current repository structure.**
   - `docker/docker-compose.yml` references `../file-engine-go`, which is not present.
   - Root compose appears to reflect current folder names, creating split guidance and avoidable onboarding risk.

2. **`docs/setup.md` appears stale versus actual repository run paths.**
   - It uses commands like `go run ./cmd/migrate` from repository root while command paths are inside `file-engine/`.
   - This undermines reproducibility of onboarding steps.

3. **`file-engine/README.md` includes generated/scaffold tree and claims that diverge from current reality.**
   - It still reads like a one-time scaffold output rather than current source-of-truth runtime instructions.

4. **`.github/AGENTS.md` contains outdated “known alignment gaps.”**
   - It states baseline tests do not compile and backend config is invalid, but capability checks now pass.
   - Agents relying on this file may make incorrect assumptions and waste effort.

5. **Requested file `READMD.md` does not exist.**
   - If this is intentional, a pointer should be added; if accidental, it indicates naming/process drift in instruction surfaces.

## 1.4 Areas exceeding documented scope (positive)

- Governance artifacts are stronger than a typical scaffold-stage project (PR title semantic enforcement, OWNERS/CODEOWNERS, multiple security workflows).
- The capability-ledger approach provides a strong mechanism for “claim-to-proof” discipline.

---

## 2) Critical Project Analysis

## 2.1 Architectural integrity

**Strengths**

- The control-plane/data-plane separation (Laravel orchestration + Go execution engine) remains coherent and defensible.
- Async mutation design is operationally appropriate for filesystem side effects and security gates.
- The inclusion of correlation IDs and task-state persistence in tested flow improves auditability and debuggability.

**Risks / integrity gaps**

- Architecture docs still mix **implemented baseline** and **target-state controls** in ways that can be misread as already enforced in production.
- Contract/source-of-truth is better than before, but drift risk remains because multiple docs duplicate setup and architecture instructions.

## 2.2 Process efficiency

**Strengths**

- CI baseline is now focused on checks that are actually expected to pass.
- Capability-ledger creates a practical feedback loop for maintainers and reviewers.

**Current inefficiencies**

- Documentation authority is fragmented (README, docs/setup, service READMEs, AGENTS notes, workflow docs) without a clearly designated canonical owner per subject.
- Stale instructions in one channel (e.g., AGENTS/setup/service README) can invalidate otherwise healthy baseline confidence.

## 2.3 Sustainability

**Current sustainability: Moderate, trending positive.**

- Positive trajectory: there is now a verified minimal vertical slice and reproducible baseline checks.
- Sustainability threat: if stale docs are not continuously pruned, onboarding/support costs will rise and trust in docs will decline.

---

## 3) Prioritized, Actionable Recommendations

## Priority 0 (Immediate: 1 week)

1. **Fix documentation authority and drift at source.**
   - Declare canonical files for: architecture, setup, and capability status.
   - In all secondary docs, replace duplicated instructions with links to canonical sources.

2. **Remove or repair stale runtime instructions.**
   - Update or deprecate `docker/docker-compose.yml` (legacy path issue).
   - Correct `docs/setup.md` commands to run from correct working directory and reflect current binaries/entrypoints.

3. **Refresh `.github/AGENTS.md` known-gaps section.**
   - Replace invalid historical gaps with current, verified constraints.
   - Add a “last validated” date and command references.

## Priority 1 (Near-term: 2–4 weeks)

1. **Add a machine-checkable docs consistency gate.**
   - CI job that validates documented command snippets for baseline docs (or at least smoke-tests referenced scripts).

2. **Split baseline vs target-state docs more aggressively.**
   - Add explicit badges/headers: “Implemented Baseline” vs “Target Architecture.”
   - Prevent policy misunderstandings in audits and stakeholder reporting.

3. **Unify compose strategy.**
   - Keep one supported local compose entrypoint and clearly mark others as archived/experimental.

## Priority 2 (Strategic: 1–2 months)

1. **Introduce doc ownership metadata.**
   - Assign owners and review cadence for top-level docs to prevent silent drift.

2. **Expand capability ledger coverage gradually.**
   - Add checks for authz decision path, ACL inheritance behavior, and selected observability guarantees.

3. **Adopt quarterly alignment review cadence.**
   - Re-run this review template every quarter; publish diff of resolved and newly introduced gaps.

---

## 4) 30-Day Improvement Plan

### Week 1
- Update `docs/setup.md`, `.github/AGENTS.md`, and legacy compose references.
- Add “canonical doc map” section to `README.md`.

### Week 2
- Add CI docs-smoke target for capability-ledger commands and key setup commands.
- Archive or modernize duplicate setup instructions.

### Week 3
- Introduce baseline vs target-state banners across architecture/security docs.
- Add ownership metadata (owner + last reviewed date) to high-impact docs.

### Week 4
- Publish a short alignment changelog in `docs/` summarizing resolved drifts.
- Reconfirm all README baseline claims against ledger commands.

---

## 5) Final Assessment

This project has moved from largely aspirational architecture toward a **validated baseline** with real, testable claims. The primary challenge is no longer “absence of implementation,” but **documentation coherence and authority management** across many instruction surfaces. If the team aggressively removes stale guidance and enforces doc-to-runtime consistency in CI, the project can maintain trust, accelerate onboarding, and scale delivery quality with much lower coordination overhead.

---

## Appendix: Validation Commands Executed for This Review

- `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto`
- `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v`
- `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v`
- `./file-engine/scripts/dev.sh`
- `cd backend && composer validate --strict`
- `test -f frontend/README.md && test ! -f frontend/package.json`
- `test -f READMD.md`
