# Project Alignment & Improvement Review

**Last verified:** 2026-02-17

## Scope and evidence base

This review evaluates the repository against three objectives:

1. **README alignment** with technical docs, workflows, governance, and policies.
2. **Critical project analysis** across architecture integrity, process efficiency, and sustainability.
3. **Actionable recommendations** with immediate and strategic priorities.

Primary sources reviewed:

- `README.md`
- `docs/project-alignment-review.md` (previous baseline)
- `docs/capability-ledger.md`
- `.github/AGENTS.md`
- `file-engine/Agents.md` (requested as `file-engine/AGENTS.md`; actual file name is `Agents.md`)
- `backend/AGENTS.md`

Validation commands executed for this review are listed in [Appendix A](#appendix-a-validation-checks-run).

---

## 1) Documentation Alignment & Validation

### Executive verdict

**Overall alignment is strong for baseline capabilities, with three high-impact alignment gaps that should be corrected quickly.**

The project has a mature claim model (baseline vs target-state) and a strong claim-to-proof mechanism in the capability ledger. However, there are still places where discoverability and policy consistency could regress contributor behavior.

### What is well aligned

1. **Baseline claims are explicit and runnable.**
   - README status claims are tied to capability IDs and runnable commands, and those IDs are concretely defined in `docs/capability-ledger.md`.
2. **Governance-first framing is consistent across top-level guidance.**
   - README, `.github/AGENTS.md`, and `file-engine/Agents.md` all reinforce deny-by-default authz boundary and server-side tenant scope.
3. **Scope control is transparent.**
   - Target-state capabilities (uploads + malware gating + immutable sink + full OTEL) remain clearly marked in the ledger rather than being presented as already shipped.

### Misalignment and quality risks

1. **Instruction order conflict between docs.**
   - README says conflicts should prefer: capability ledger -> architecture docs -> setup.
   - `.github/AGENTS.md` says: capability ledger -> setup -> scoped guide -> older service READMEs.
   - This is subtle but important: contributors may follow different precedence in practice.
2. **File-engine AGENTS discoverability friction.**
   - `.github/AGENTS.md` references `file-engine/Agents.md` while many workflows/tooling conventions expect `AGENTS.md` naming.
   - This can cause missed policy loading by humans and automation that search exact `AGENTS.md`.
3. **README still markets some target-state outcomes prominently in top narrative.**
   - README does label these as target-state, but phrasing in headline/TL;DR can still be interpreted as currently operational without reading deeply.
   - Risk is external over-claim in demos/proposals.

### Exceeds documented baseline (positive)

1. **Validation discipline is above average for maturity stage.**
   - The capability ledger offers precise runnable checks (including authz precedence, path normalization, gateway artifact sync, and worker guardrails).
2. **Documentation governance includes drift controls.**
   - `CL-011` formalizes doc drift validation, which is a meaningful process quality multiplier.

---

## 2) Critical Project Analysis

### A) Architectural integrity

**Current state: coherent and defensible.**

Strengths:

- **Clear split of responsibilities:** Laravel control plane vs Go execution/data plane remains a clean trust-boundary architecture.
- **Correct security boundary placement:** File Engine as final authz enforcement point is repeatedly and consistently documented.
- **Async task model fit:** For filesystem mutations, queue + worker + persisted task state is appropriate for reliability/auditability.

Risks:

- **Scaffold asymmetry:** File Engine has deep verification while backend is still scaffold-level (`CL-008`), creating product-surface imbalance.
- **Potential policy drift through multi-source docs:** multiple “authoritative” texts can diverge unless precedence is singular and enforced.
- **Feature expectation drift:** gRPC + limited gateway baseline may be mistaken for broad HTTP parity if route-by-route boundaries are not continuously emphasized.

### B) Process efficiency

**Current state: efficient for core baseline, uneven for contributor onboarding.**

Strengths:

- Runnable command mapping in ledger reduces ambiguity during code reviews.
- CI/governance emphasis appears focused on meaningful baseline checks.
- Service-scoped AGENTS guidance provides useful guardrails where maturity differs by subsystem.

Inefficiencies:

- **Naming inconsistency (`Agents.md` vs `AGENTS.md`) introduces avoidable lookup cost.**
- **Guidance duplication:** precedence and setup authority are repeated in slightly different forms across README and AGENTS files.
- **Owner model gaps:** domain ledger lists `Unassigned (TBD)` across major areas, which may slow prioritization and operational accountability.

### C) Sustainability

**Current state: moderate-to-strong, contingent on governance hygiene.**

What supports sustainability:

- Explicit baseline/target-state boundary.
- Claim-to-proof culture via capability ledger.
- Security-first architecture direction.

What threatens sustainability:

- Continued document precedence drift.
- Missing explicit ownership for domain capabilities.
- Risk of communications debt if public-facing README language outruns shipped capabilities.

---

## 3) Prioritized, Actionable Recommendations

### Priority 0 — Immediate (this sprint)

1. **Unify conflict-resolution precedence in one canonical statement.**
   - Decide one order (recommended: capability ledger -> setup -> scoped AGENTS -> architecture deep-dives).
   - Update README and `.github/AGENTS.md` to match exactly.
2. **Normalize file-engine agent guidance naming for discoverability.**
   - Add `file-engine/AGENTS.md` as canonical (or duplicate shim) pointing to the maintained guide to satisfy standard tooling expectations.
3. **Tighten README top-copy wording for target-state features.**
   - Keep aspirational architecture, but add one explicit sentence near TL;DR that production baseline currently centers on async create-folder + read-path/authz validations.

### Priority 1 — Near-term (next 1-2 milestones)

1. **Introduce ownership for each domain ledger row.**
   - Replace `Unassigned (TBD)` with role/team handles to improve delivery accountability.
2. **Add a “route maturity matrix” doc section.**
   - Summarize each API route as baseline vs target-state with claim IDs; link from README and API reference.
3. **Add periodic governance hygiene check in CI or release checklist.**
   - Verify precedence statements and key links remain consistent across README/AGENTS/capability ledger.

### Priority 2 — Strategic (quarterly horizon)

1. **Promote capabilities strictly via ledger-gated evidence.**
   - Continue requiring dedicated runnable checks before moving any target-state item to baseline.
2. **Evolve backend from scaffold by vertical slices, not broad scaffolding.**
   - Ship one end-to-end control-plane workflow at a time with explicit claim IDs and validations.
3. **Operationalize sustainability metrics.**
   - Track: baseline claim pass rate, doc drift failures, mean time to align docs after feature merges, and ownership coverage ratio.

---

## 4) Recommended execution plan

### Next 7 days

- Align precedence statements.
- Add file-engine AGENTS naming shim/canonical file.
- Refine README target-state phrasing.

### Next 30 days

- Assign domain owners in ledger.
- Publish route maturity matrix.
- Add governance hygiene checks to CI or release template.

### Next 90 days

- Graduate next target-state capability only with claim + runnable validation.
- Increment backend capabilities through validated vertical slices.

---

## 5) Final assessment

This project is **well-governed for its maturity** and already demonstrates stronger-than-average engineering discipline through explicit capability proofs. The highest-value improvements are not major architectural rewrites; they are **documentation governance hardening and expectation management** so contributors and stakeholders make decisions from one consistent truth surface.

---

## Appendix A: Validation checks run

- `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto`
- `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v`
- `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v`
- `cd backend && composer validate --strict`
- `./scripts/doc-drift-check.sh`
