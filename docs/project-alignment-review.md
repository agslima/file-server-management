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

**Aligned at the baseline level, with remaining scope controls mostly around experimental paths.**

The project clearly distinguishes baseline vs target-state claims and anchors validation in `docs/capability-ledger.md`. HTTP/JSON is baseline for `CreateFolder` + `GetTaskStatus`, while uploads and extended routes remain target-state. Read-path list results and metadata are now validated. Onboarding paths are baseline-first, with root compose explicitly marked experimental.

### Where implementation matches documented baseline

- Capability ledger claims are explicit and linked to runnable commands.
- File Engine baseline tests and integration flow are documented and runnable via `./file-engine/scripts/dev.sh`.
- Backend and frontend are explicitly labeled scaffold-level in the ledger.
- Read-path list results + metadata and download path normalization are validated and linked to `CL-012`.
- Doc drift and codegen drift guards are captured as baseline claims and enforced in CI.

### Where documentation is still misaligned

1. **Full-stack compose remains non-canonical (by design).**
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

- Remaining REST routes (uploads, expanded read APIs) are still target-state and should not be assumed available.
- Multiple instruction surfaces can reintroduce drift if not actively pruned.
- Metadata fields beyond size (`owner`, `group`, timestamps) are best-effort across backends and should remain clearly labeled as such.

### Process efficiency

**Strengths**

- CI focuses on baseline checks that are expected to pass and now includes doc/codegen drift guards.
- Capability ledger encourages proof-based documentation.
- Doc-drift gate provides lightweight enforcement of local doc link validity.

**Inefficiencies**

- Onboarding confusion returns if experimental paths are allowed to drift from their explicit labeling.
- Service-level README files can lag behind the canonical docs and ledger if not reviewed on cadence.

### Sustainability

**Current sustainability: Moderate, trending positive.** The project is in a better position now that baseline claims are validated, but sustainability depends on keeping doc authority centralized and avoiding runtime over-claims.

---

## 3) Prioritized, Actionable Recommendations

### Priority 0 (Immediate)

1. **Keep onboarding paths strictly baseline-aligned.**
   - Treat root compose as experimental until it is validated end-to-end.

### Priority 1 (Near-term)

1. **Consolidate setup instructions.**
   - Link secondary docs to `docs/setup.md` rather than duplicating commands.
2. **Keep metadata claims explicitly “best-effort” by backend.**
   - Document which backends supply timestamps/ownership and add a follow-up test when a backend’s metadata becomes a baseline guarantee.

### Priority 2 (Strategic)

1. **Promote REST routes only when they are real and tested.**
   - `CreateFolder` + `GetTaskStatus` are now baseline via gRPC-Gateway; uploads remain target-state.
2. **Maintain and extend capability ledger coverage as new routes graduate.**
   - Keep authz precedence and path normalization checks as baseline gates; add new checks only with runnable commands.

---

## 4) Validation Commands (Baseline)

These commands define the current baseline and should remain green:

- `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto`
- `./file-engine/scripts/dev.sh`
- `cd backend && composer validate --strict`
- `test -f frontend/README.md && test ! -f frontend/package.json`
