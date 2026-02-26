# Project Alignment & Improvement Review

[//]: # (owner: Project Maintainers)
[//]: # (review_cadence: Quarterly)
[//]: # (last_reviewed: 2026-02-25)


**Last verified:** 2026-02-25

## Scope and evidence base

This review evaluates the repository against three objectives:

1. **README alignment** with technical docs, workflows, governance, and policies.
2. **Critical project analysis** across architecture integrity, process efficiency, and sustainability.
3. **Actionable recommendations** with immediate and strategic priorities.

Primary sources reviewed:

- `README.md`
- `docs/project-alignment-review.md` (this updated baseline)
- `docs/capability-ledger.md`
- `.github/AGENTS.md`
- `file-engine/AGENTS.md`
- `backend/AGENTS.md`

Validation commands executed for this review are listed in [Appendix A](#appendix-a-validation-checks-run).

---

## 1) Documentation Alignment & Validation

### Executive verdict

**Current alignment is strong and materially improved, with core governance/doc-hygiene checks now aligned to CI behavior.**

The project is using a clear baseline-vs-target-state model with claim IDs and runnable verification commands. Top-level positioning now better matches the implemented baseline and reduces over-claim risk.

### What is well aligned

1. **README claims map clearly to ledger evidence.**
   - The README implementation table tracks the promoted claim set in `docs/capability-ledger.md` (currently through `CL-069`, including CI-gated deterministic OIDC evidence, upload API contract promotion evidence, scanner/upload operational closure evidence, observability assets, OTEL production deployment drills with paging delivery validation, storage parity controls, documentation contract synchronization evidence, and sustainability/ownership kickoff evidence).
2. **Governance policy and CI behavior are synchronized.**
   - `docs/governance.md` now explicitly distinguishes path-scoped checks from always-on governance checks.
   - CI now runs doc drift/governance hygiene checks for every PR merge path (not only docs-only diffs).
   - Governance next-step controls are now baseline-validated (`CL-049`): archive-tier lifecycle, external source policy loading/signature verification, runtime drift checks, and effective-policy operator visibility.
3. **Backend maturity contract is consistent across core control docs.**
   - `CL-018` backend scaffold checks are now baseline-marked, `CL-020` adds executable backend↔file-engine VS-001 E2E validation, and `CL-031` tracks backend smoke execution (composer install + phpunit).
- Scale/fairness closure is now baseline-validated (`CL-062`): published core-flow SLOs, deterministic k6 smoke CI + nightly soak schedule, explicit throttled error envelope semantics with audit evidence, and dependency backpressure drill/runbook coverage.
- Data durability/integrity contract closure is now baseline-validated (`CL-063`): configurable integrity sample policy with threshold + false-positive handling, explicit dev-grade RTO/RPO objectives, script-backed restore drills, and deterministic evidence-pack generation.
- Multi-tenant compliance productization is now baseline-validated (`CL-064`): stable/signed access-review export contract, governance policy update hash audit history, tenant evidence pointers endpoint, and one-command tenant compliance packet generation.
- API/SDK external-consumer hardening is now baseline-validated (`CL-065`): expanded mutation/governance/throttling golden fixtures, typed retryable error handling in Go+PHP client layers, and PR-gated `/v1` compatibility policy enforcement coupled to docs updates.
- Async mutation expansion beyond create-folder is now baseline-validated for move, governed delete, and quarantine restore task flows (`CL-066` to `CL-068`), including final-gate governance denial evidence and stable task failure envelopes.
- Human-resilience continuity controls are now baseline-validated (`CL-069`): critical-domain CODEOWNERS paths are explicit, reviewer continuity is CI-enforced for auth/authz + monitoring/observability + capability-ledger changes, and release cadence includes a new-maintainer drill gate.
4. **Target-state boundaries remain explicit.**
   - Upload pipeline operational closure for thresholds/on-call/escalation is promoted via `CL-051`; OTEL production deployment hardening is promoted via `CL-048`, paging-provider delivery via `CL-050`, documentation contract synchronization via `CL-052`, and immutable sink delivery remains promoted via `CL-035`.

### Alignment corrections completed in this cycle

5. **Resolved:** architecture boundaries are now explicitly documented and conformance-checked (logger unification + package-boundary guardrails).

1. **Resolved:** governance-vs-CI mismatch for doc/governance checks.
2. **Resolved:** stale `docker/docker-compose.yml` references in canonical docs.
3. **Resolved:** README maturity visibility gaps for higher claim IDs.
4. **Resolved:** stale prior-review findings that no longer reflected repo reality.

### Residual alignment risks

1. **Backend remains scaffold-level despite stronger contract checks.**
   - Validation is better defined, but production-readiness is still intentionally limited.
2. **Role-based backup reviewers are documented, but primary ownership is still concentrated.**
   - Bus-factor risk is improved procedurally but still operationally sensitive.

---

## 2) Critical Project Analysis

### A) Architectural integrity

**Current state: coherent and defensible.**

Strengths:

- Clear control-plane (Laravel) vs data-plane (Go File Engine + worker) trust-boundary split.
- File Engine remains documented and tested as final authz enforcement point.
- Async task lifecycle model is appropriate for filesystem mutation reliability and auditability.

Architectural risks:

- Backend maturity asymmetry remains (scaffold control-plane vs stronger data-plane verification depth).
- Target-state upload/security pipeline remains an architectural intent rather than baseline behavior.

### B) Process efficiency

**Current state: improved and pragmatic, with strong claim-to-proof mechanics.**

Strengths:

- Capability ledger gives concrete, runnable proofs for baseline claims.
- CI gates are aligned with governance language for baseline + doc hygiene.
- SHA pinning across GitHub Actions reduces workflow supply-chain drift.

Inefficiencies:

- Multiple documentation surfaces still require careful synchronization.
- Some role/ownership guidance is process-driven rather than backed by multi-person maintainer coverage.

### C) Sustainability

**Current state: moderate-to-strong with disciplined governance; human redundancy remains the key gap.**

What supports sustainability:

- Explicit baseline/target-state separation.
- Runnable validation culture.
- CI + governance hygiene checks protecting documentation consistency.

What threatens sustainability:

- Single-primary-owner concentration for critical domains.
- Risk of future drift if promotion policy is not enforced continuously.

---

## 3) Prioritized, Actionable Recommendations

### Priority 0 — Completed in this cycle

1. **Governance/CI alignment:** doc/governance hygiene checks now run on all PR paths.
2. **Backend maturity contract harmonization:** ledger, backend AGENTS, CI, and README now reference the same VS-001 scaffold expectations.
3. **Review refresh:** this document now reflects current repository reality and removed already-resolved findings.

### Priority 1 — Next 1-2 milestones

1. **Promote VS-001 executable backend behavior checks in CI.**
   - `CL-020` is now CI-gated with docker-compose E2E coverage and deterministic `E2E_OK` output for machine checks.
2. **Completed:** branch-protection mapping is now actionable in CI.
   - Path-sensitive reviewer checks are represented as CI status checks for authz and monitoring change scopes.
3. **Keep canonical setup references tight.**
   - Continue pruning stale path references when structure changes.

### Priority 2 — Strategic (quarterly horizon)

1. **Institutionalize claim-promotion discipline.**
   - Keep the policy: no top-level baseline marketing before ledger claim ID + runnable evidence + CI proof.
3. **Track sustainability metrics with release-ready artifacts.**
   - Metrics now emit markdown for direct inclusion in PR/release notes.

---

## 4) Final assessment

The project is **well-governed for its current maturity stage** and now has tighter policy-to-enforcement consistency than the previous baseline. With `CL-069` now enforcing reviewer continuity for critical scopes, the highest-value next step is sustained quarterly ownership rotation execution and evidence hygiene.

---

## Appendix A: Validation checks run

- `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto`
- `./scripts/doc-drift-check.sh`
- `./file-engine/scripts/dev.sh`
- `cd file-engine && go test ./internal/handlers -run "TestListObjectsReturnsEntries|TestListObjectsRequiresAuthContext|TestListObjectsRejectsUnauthorizedTenant|TestDownloadObjectRejectsUnauthorizedTenant" -v && go test ./internal/adapters/storage/local -run TestLocalStorageListMetadata -v && go test ./internal/authz -run "TestGRPCAuthZInterceptorListObjects" -v && go test ./internal/server -run "TestHandleDownloadNormalizesPath|TestHandleDownloadRejectsTraversal" -v && go test -tags integration_authz ./tests/integration -run TestReadListBehaviorAndAuthzRejection -v`
- `cd backend && composer validate --strict`
- `cd backend && php -l app/Http/Controllers/FolderController.php` (fails locally with PHP 8.0; CI enforces under PHP 8.2)

## Addendum — Deployment realism uplift (Milestone 10 stream)

The repository now has script-backed deployment realism controls that close the previous compose-only impression:

- explicit `dev/stage/prod` env profile templates with config-vs-secret contract,
- executable `kind` deployment smoke path validating `/healthz` + `/readyz`,
- documented and drilled Kubernetes rollback path,
- release versioning/changelog discipline documented for tagged releases,
- prod-like runtime dependency wiring validation script for OIDC/OTEL/audit sink/governance envelope requirements.

This keeps alignment with the ledger promotion discipline by pairing each new capability with runnable commands.
