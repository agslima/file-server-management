# Project Alignment & Improvement Review

**Last verified:** 2026-02-15

## Scope and method

This review evaluates the repository against three objectives:

1. **README alignment with technical documentation, workflows, governance, and policies.**
2. **Critical analysis** of architecture integrity, process efficiency, and sustainability.
3. **Actionable recommendations** for immediate and long-term improvements.

Sources reviewed:

- `README.md`, `docs/README.md`, `docs/capability-ledger.md`, `docs/setup.md`
- `.github/AGENTS.md`, `file-engine/Agents.md`, `backend/AGENTS.md`
- `docker-compose.yml`, `file-engine/docker-compose.yml`
- `.github/workflows/ci.yml` and security workflows

---

## 1) Documentation Alignment & Validation

### Executive alignment verdict

**Mostly aligned at the baseline level, with limited remaining drift in feature-scope labeling.**

The project now clearly distinguishes baseline vs target-state claims and anchors validation in `docs/capability-ledger.md`. HTTP/JSON and audit sink claims are explicitly marked as target-state, and onboarding paths are now baseline-first. Remaining drift is concentrated in a few feature lists that are not explicitly labeled as target-state.

### Where implementation matches documented baseline

- Capability ledger claims are explicit and linked to runnable commands.
- File Engine baseline tests and integration flow are documented and runnable via `./file-engine/scripts/dev.sh`.
- Backend and frontend are explicitly labeled scaffold-level in the ledger.

### Where documentation is still misaligned

1. **Feature lists still imply implemented read-path behavior without baseline validation.**
   - The README “What it does” read-path list is not yet tied to a capability-ledger claim or target-state label.
2. **Full-stack compose remains non-canonical.**
   - Root compose is useful for container build validation but is not a baseline-validated runtime path.

### Where project exceeds documented scope (positive)

- Governance intent (capability ledger, ADRs, threat model, observability standards) is stronger than typical for a project at this stage.
- The ledger approach provides a concrete “claim-to-proof” mechanism for reviewers and contributors.

---

## 2) Critical Project Analysis

### Architectural integrity

**Strengths**

- The control-plane (Laravel) vs data-plane (Go File Engine) split remains coherent.
- Asynchronous mutation design is a solid fit for filesystem operations and security gates.
- The baseline async create-folder flow provides a credible, testable vertical slice.

**Risks**

- Authorization boundary ambiguity increases the chance of policy drift or duplicated logic.
- HTTP gateway scaffolding may lead consumers to assume supported REST endpoints before they are actually enforced.
- Multiple instruction surfaces can reintroduce drift if not actively pruned.

### Process efficiency

**Strengths**

- CI focuses on baseline checks that are expected to pass.
- Capability ledger encourages proof-based documentation.
- Doc-drift gate now provides lightweight enforcement of local doc link validity.

**Inefficiencies**

- Onboarding confusion persists when experimental paths are not clearly labeled.
- Service-level README files lag behind the canonical docs and ledger.

### Sustainability

**Current sustainability: Moderate, trending positive.** The project is in a better position now that baseline claims are validated, but sustainability depends on keeping doc authority centralized and avoiding runtime over-claims.

---

## 3) Prioritized, Actionable Recommendations

### Priority 0 (Immediate)

1. **Label read-path capabilities as target-state or add validation.**
   - Either add a capability-ledger claim for read-path behavior or mark the README list as target-state.
2. **Keep onboarding paths strictly baseline-aligned.**
   - Treat root compose as experimental until it is validated end-to-end.

### Priority 1 (Near-term)

1. **Promote doc drift protection into the capability ledger.**
   - Keep the doc-drift check a baseline claim with a runnable command.
2. **Consolidate setup instructions.**
   - Link secondary docs to `docs/setup.md` rather than duplicating commands.

### Priority 2 (Strategic)

1. **Generate and commit real gateway code.**
   - Promote REST endpoints only after the gateway is real and tested.
2. **Expand capability ledger coverage.**
   - Add checks for authz precedence behavior and path normalization guarantees.

---

## 4) Validation Commands (Baseline)

These commands define the current baseline and should remain green:

- `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto`
- `./file-engine/scripts/dev.sh`
- `cd backend && composer validate --strict`
- `test -f frontend/README.md && test ! -f frontend/package.json`
