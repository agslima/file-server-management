<div align="center">

<a name="back-to-top"></a>

# Server File Manager Platform (PHP + Go File Engine)

[//]: # (owner: Project Maintainers)
[//]: # (review_cadence: Quarterly)
[//]: # (last_reviewed: 2026-02-19)


[![CI](https://github.com/agslima/file-server-management/actions/workflows/ci.yml/badge.svg)](https://github.com/agslima/file-server-management/actions/workflows/ci.yml)
![Go Version](https://img.shields.io/badge/go-1.24+-1e6e6e)
![Laravel](https://img.shields.io/badge/laravel-10%2B-blue)
![gRPC](https://img.shields.io/badge/API-gRPC%20-4e6e6e)
[![Docs](https://img.shields.io/badge/docs-architecture%20%7C%20adr-green)](https://github.com/agslima/file-server-management/tree/main/docs)
![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)
<!--
![Go Tests](https://github.com/<org>/<repo>/actions/workflows/go-test.yaml/badge.svg)
![Laravel Tests](https://github.com/<org>/<repo>/actions/workflows/phpunit.yaml/badge.svg)
[![codecov](https://codecov.io/gh/<org>/<repo>/branch/main/graph/badge.svg)](https://codecov.io/gh/<org>/<repo>)
![Dependency Review](https://github.com/<org>/<repo>/actions/workflows/dependency-review.yml/badge.svg)
![Trivy](https://github.com/<org>/<repo>/actions/workflows/trivy.yml/badge.svg)
-->

⚡️ **A governance-focused, multi-tenant file management platform written in Go and PHP** ⚡️ \
Designed to operate directly on **real storage backends** (local/mounted SMB/NFS/SFTP, with adapter-based extensibility for S3/GCS). It centralizes access to shared storage with **RBAC + path-based ACL**, **async mutations**, baseline **task audit events**, and a baseline-validated **quarantine -> scan -> promote** guardrail flow (local semantics).

</div>

> [!Note]
> **Honest status:** The **Go File Engine** is the current working nucleus (baseline-validated). The **Laravel control plane** is scaffold/in-progress and becomes the orchestration layer as features are promoted via the capability ledger.

## TL;DR

- **Multi-tenant:** tenant scope is resolved **server-side** (not trusted from JWT/client).
- **AuthZ:** RBAC + path-based ACL with inheritance, **deny-by-default**, enforced at the File Engine boundary.
- **Async mutations:** create-folder and upload lifecycle are baseline-validated async workflows returning task/status-oriented outcomes; clients poll task status and complete upload flows through deterministic contract checks.
- **Secure uploads:** staged quarantine write + scan-gated promote behavior are baseline-validated, including non-stub ClamAV scanner integration evidence (clean + quarantined paths) and operational scanner closure controls (threshold alerts + runbook/escalation drill evidence).
- **Auditing:** persisted task status + task audit events + append-only DB enforcement + external sink delivery are baseline-validated.
- **Observability:** correlation IDs are baseline; OTEL export wiring is baseline-validated for API + worker entrypoints; collector/backend deployment hardening is baseline-validated with deterministic connectivity + drill scripts; paging-provider delivery is baseline-validated through a deterministic webhook drill path.

---

## Project status

This repository documents an evolving architecture.

Legend:

- ✅ implemented
- 🟡 in progress
- 🔒 planned / target state

> [!Note]
> **Current maturity note:** Some controls are documented as target state. The roadmap tracks what is enforced vs intended.
> **Validation source of truth:** See [`docs/capability-ledger.md`](docs/capability-ledger.md) for runnable commands that validate each implemented claim.

### Implementation status (baseline)

Every baseline claim is mapped to a claim ID and runnable command in the capability ledger.

<details><summary><b>See more details</b></summary>
  
| Claim ID | Capability | Status | Runnable validation |
| :-- | :-- | :--: | :-- |
| [`CL-001`](docs/capability-ledger.md#baseline-claims-implemented) | Canonical proto contract sync | ✅ | `cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto` |
| [`CL-002`](docs/capability-ledger.md#baseline-claims-implemented) | File Engine baseline module checks | ✅ | `cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v` |
| [`CL-003`](docs/capability-ledger.md#baseline-claims-implemented) | Async folder flow (enqueue -> worker -> folder created) | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` |
| [`CL-004`](docs/capability-ledger.md#baseline-claims-implemented) | Task status persistence (`queued -> running -> success`) | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` |
| [`CL-005`](docs/capability-ledger.md#baseline-claims-implemented) | Basic audit event emission (`task.processing`, `task.succeeded`) | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` |
| [`CL-006`](docs/capability-ledger.md#baseline-claims-implemented) | Correlation ID propagation in async flow | ✅ | `cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v` |
| [`CL-007`](docs/capability-ledger.md#baseline-claims-implemented) | Known-working local dev script | ✅ | `./file-engine/scripts/dev.sh` |
| [`CL-008`](docs/capability-ledger.md#baseline-claims-implemented) | Backend scaffold validation | ✅ | `cd backend && composer validate --strict` |
| [`CL-009`](docs/capability-ledger.md#baseline-claims-implemented) | Frontend thin-client demo console (product + operator UX flows) | ✅ | `test -f frontend/index.html && test -f frontend/app.js && test -f frontend/styles.css && ./scripts/e2e/demo_5_minute.sh --mode=mock` |
| [`CL-010`](docs/capability-ledger.md#baseline-claims-implemented) | Structured logs + queue/task metrics baseline | ✅ | `cd file-engine && go test ./internal/handlers ./internal/observability -v` |
| [`CL-011`](docs/capability-ledger.md#baseline-claims-implemented) | Documentation drift checks (links + governance hygiene) | ✅ | `./scripts/doc-drift-check.sh` |
| [`CL-012`](docs/capability-ledger.md#baseline-claims-implemented) | Read-path behavior + final authz (list/download + path normalization + tenant enforcement) | ✅ | `cd file-engine && go test ./internal/handlers -run "TestListObjectsReturnsEntries|TestListObjectsRequiresAuthContext|TestListObjectsRejectsUnauthorizedTenant|TestDownloadObjectRejectsUnauthorizedTenant" -v && go test ./internal/adapters/storage/local -run TestLocalStorageListMetadata -v && go test ./internal/authz -run "TestGRPCAuthZInterceptorListObjects" -v && go test ./internal/server -run "TestHandleDownloadNormalizesPath|TestHandleDownloadRejectsTraversal" -v && go test -tags integration_authz ./tests/integration -run TestReadListBehaviorAndAuthzRejection -v` |
| [`CL-013`](docs/capability-ledger.md#baseline-claims-implemented) | HTTP gateway routes for `CreateFolder` + `GetTaskStatus` are generated and responsive | ✅ | `cd file-engine && go test ./internal/server -run TestGatewayCreateFolderAndGetTaskStatusRoutes -v` |
| [`CL-014`](docs/capability-ledger.md#baseline-claims-implemented) | AuthZ precedence behavior (ACL vs RBAC) | ✅ | `cd file-engine && go test ./internal/auth -run "TestRBACFallback|TestUserACLOverridesRBAC|TestACLPathInheritance|TestUserDenyPrecedesRoleAllowAndRBAC|TestRoleDenyPrecedesRoleAllowAtSamePath|TestClosestPathACLWinsBeforeParentACLs|TestUserACLPrecedenceOnSamePath|TestUserACLWithoutPermissionFallsThroughToRoleACL" -v` |
| [`CL-015`](docs/capability-ledger.md#baseline-claims-implemented) | Path normalization guarantees (traversal rejection + canonicalization) | ✅ | `cd file-engine && go test ./internal/authz -run "TestExtractPathNormalizesCreateFolder|TestExtractPathRejectsTraversal|TestNormalizePathHandlesWindowsAndWhitespace|TestNormalizePathAllowsDotContainingNames|TestTenantFromPath|TestTenantFromPathRejectsNonTenantRoot" -v` |
| [`CL-016`](docs/capability-ledger.md#baseline-claims-implemented) | Generated gateway artifacts remain in sync with proto | ✅ | `cd file-engine && ./scripts/generate_grpc_docker.sh && cd .. && git diff --exit-code && test -z "$(git status --porcelain)"` |
| [`CL-017`](docs/capability-ledger.md#baseline-claims-implemented) | Worker performance guardrails (status retries + task timeout) for async create-folder | ✅ | `cd file-engine && go test ./internal/app/tasks -run "TestWorkerRetriesStatusPersistence|TestWorkerMarksTaskFailedOnProcessingTimeout" -v` |
| [`CL-018`](docs/capability-ledger.md#baseline-claims-implemented) | Backend VS-001 scaffold contract (create-folder forward + task polling wiring checks) | ✅ | `cd backend && composer validate --strict && php -l app/Http/Controllers/FolderController.php && php -l app/Http/Controllers/TaskController.php && php -l app/Services/FileEngineService.php` |
| [`CL-020`](docs/capability-ledger.md#baseline-claims-implemented) | Backend VS-001 docker-compose E2E (forward create-folder + poll task to success + folder existence) | ✅ | `docker compose up -d --build && ./scripts/wait-for-http.sh http://localhost:8080/healthz 60 && ./scripts/wait-for-http.sh http://localhost:8081/healthz 60 && ./scripts/e2e/vs001_create_folder.sh && docker compose down -v` |
| [`CL-022`](docs/capability-ledger.md#baseline-claims-implemented) | Audit coverage for read/list/download actions (`object.list`, `object.read`, `object.download`) | ✅ | `cd file-engine && go test ./tests/integration -run TestAuditEventsEmittedForReadListDownload -v` |
| [`CL-025`](docs/capability-ledger.md#baseline-claims-implemented) | Upload pipeline baseline: staged quarantine write + atomic promote (no partial final object visibility) | ✅ | `cd file-engine && go test ./tests/integration -run TestStagedUploadAtomicPromote -v` |
| [`CL-031`](docs/capability-ledger.md#baseline-claims-implemented) | Backend baseline smoke suite (composer install + phpunit) | ✅ | `docker compose run --rm --no-deps backend sh -lc 'composer install --no-interaction && ./vendor/bin/phpunit -c phpunit.xml'` |
| [`CL-032`](docs/capability-ledger.md#baseline-claims-implemented) | Audit table append-only enforcement (UPDATE/DELETE rejected for app DB user) | ✅ | `cd file-engine && go test ./tests/integration -run TestAuditEventsAppendOnlyEnforced -v` |
| [`CL-033`](docs/capability-ledger.md#baseline-claims-implemented) | Upload malware gating: dirty scan blocks promote, clean scan promotes from quarantine | ✅ | `cd file-engine && go test ./tests/integration -run TestUploadScanGateDirtyPreventsPromotion -v` |
| [`CL-034`](docs/capability-ledger.md#baseline-claims-implemented) | Curated ledger baseline gate script runs in CI to catch regressions | ✅ | `./scripts/ledger-baseline.sh` |
| [`CL-035`](docs/capability-ledger.md#baseline-claims-implemented) | Audit external sink delivery covers S3 WORM/Loki/SIEM adapters with retries + DLQ + lag metric | ✅ | `cd file-engine && go test ./tests/integration -run TestAuditExternalSinkDeliveryWithDLQAndLagMetrics -v` |
| [`CL-036`](docs/capability-ledger.md#baseline-claims-implemented) | `/readyz` checks DB+queue+storage dependencies with deterministic per-check JSON output | ✅ | `cd file-engine && go test ./internal/server -run "TestHandleReadyzReturnsReadyWhenChecksPass|TestHandleReadyzReturnsServiceUnavailableWhenAnyCheckFails|TestHandleReadyzWithoutChecksReturnsDeterministicReadyPayload" -v` |
| [`CL-037`](docs/capability-ledger.md#baseline-claims-implemented) | Storage contract suite passes for local baseline backend (optional S3/GCS adapters are env-gated) | ✅ | `cd file-engine && go test ./internal/adapters/storage/local -run TestLocalStorageContractSuite -v` |
| [`CL-038`](docs/capability-ledger.md#baseline-claims-implemented) | OTEL export wiring initialized for API + worker with deterministic endpoint parsing + safe no-endpoint fallback | ✅ | `cd file-engine && go test ./internal/observability -run "TestResolveTracingConfigDefaultsAndExporterToggle|TestInitTracingRejectsUnsupportedEndpointScheme|TestInitTracingWithoutExporterIsDeterministic" -v` |
| [`CL-039`](docs/capability-ledger.md#baseline-claims-implemented) | External audit sink minimal env wiring validated for bucket/Loki/SIEM/S3-WORM adapters | ✅ | `cd file-engine && go test ./internal/app/tasks -run "TestBuildImmutableSinkFromEnvBucketWritesJSONL|TestBuildImmutableSinkFromEnvLokiPostsLine|TestBuildImmutableSinkFromEnvSIEMPostsNDJSONWithAuth|TestBuildImmutableSinkFromEnvS3WormWritesJSONL" -v` |
| [`CL-040`](docs/capability-ledger.md#baseline-claims-implemented) | Real scanner integration (non-stub ClamAV adapter) validates clean + quarantined outcomes and emits scan duration/verdict metrics+logs | ✅ | `cd file-engine && go test ./tests/integration -run TestUploadRealScannerIntegrationEmitsMetricsAndLogs -v` |
| [`CL-041`](docs/capability-ledger.md#baseline-claims-implemented) | Governance hardening: key-doc ownership metadata + quarterly alignment cadence + CI architecture conformance checks | ✅ | `./scripts/doc-ownership-check.sh && ./scripts/architecture-conformance-check.sh` |
| [`CL-042`](docs/capability-ledger.md#baseline-claims-implemented) | Enterprise identity integration flow (OIDC profile) is CI-gated, deterministic, and proves tenant mapping denial semantics | ✅ | `./scripts/e2e/run_oidc_profile.sh` |
| [`CL-043`](docs/capability-ledger.md#baseline-claims-implemented) | Malware gate operational hardening: scanner retry + scan DLQ workflows + TTL cleanup + metrics | ✅ | `cd file-engine && go test ./internal/services ./internal/server ./internal/observability -run "TestUploadServiceScannerRetryEventuallySucceeds|TestUploadServiceScannerFailureEnqueuesDLQ|TestUploadServiceCleanupQuarantineDeletesExpiredObjects|TestScanDLQListEndpoint|TestQuarantineCleanupEndpoint|TestSnapshotPrometheusIncludesQueueTaskAndOperabilityMetrics" -v` |
| [`CL-044`](docs/capability-ledger.md#baseline-claims-implemented) | End-to-end observability assets are checked in (collector profile, alerts-as-code, dashboards, drill script) | ✅ | `./scripts/validate-observability-assets.sh && cd file-engine && go test ./internal/observability -v` |
| [`CL-048`](docs/capability-ledger.md#baseline-claims-implemented) | Production OTEL deployment hardening closure: connectivity SLO checks, alerts syntax validation, and deterministic sink/scanner/exporter drill scripts | ✅ | `./scripts/check-otel-connectivity.sh && ./scripts/drills/production_deployment_hardening.sh` |
| [`CL-050`](docs/capability-ledger.md#baseline-claims-implemented) | Paging-provider delivery validation: deterministic local webhook receiver confirms alert payload delivery in production-like drill path | ✅ | `./scripts/check-paging-delivery.sh` |
| [`CL-051`](docs/capability-ledger.md#baseline-claims-implemented) | Scanner/upload operational closure: SLO thresholds + on-call/escalation runbook + operator-ready scanner drill transcript are baseline-validated | ✅ | `./scripts/validate-alert-rules.sh && ./scripts/check-malware-runbook.sh && ./scripts/drills/scanner_down.sh` |
| [`CL-052`](docs/capability-ledger.md#baseline-claims-implemented) | Documentation contract synchronization: README, route maturity matrix, and roadmap-ledger gap analysis align with promoted baseline claims | ✅ | `./scripts/doc-drift-check.sh && rg -n -F "POST /v1/uploads:initiate" docs/route-maturity-matrix.md && rg -n -F "PUT /v1/uploads/{uploadId}:chunk" docs/route-maturity-matrix.md && rg -n -F "POST /v1/uploads/{uploadId}:complete" docs/route-maturity-matrix.md && rg -n -F "GET /readyz" docs/route-maturity-matrix.md && rg -n -F "OIDC profile end-to-end" docs/route-maturity-matrix.md && rg -n "Milestone 7 — Production Operations Closure.*Implemented|README wording drift corrected|Route maturity matrix refreshed" docs/roadmap-ledger-gap-analysis.md` |
| [`CL-053`](docs/capability-ledger.md#baseline-claims-implemented) | Sustainability & ownership resilience kickoff: branch-protection mapping + named backups + deterministic release metrics report | ✅ | `./scripts/sustainability-metrics.sh artifacts/sustainability-metrics.md && rg -n "branch-protection-mapping\|ownership-backup-matrix\|OWNERS" docs/governance.md` |
| [`CL-054`](docs/capability-ledger.md#baseline-claims-implemented) | Sustainability closure: path-scoped reviewer checks (`Security reviewer`/`Platform reviewer`), quarterly alignment checklist generation, and new maintainer operability drill automation | ✅ | `./scripts/check-owners-governance.sh && ./scripts/generate-quarterly-alignment-issue.sh && ./scripts/drills/new_maintainer_operability_drill.sh` |
| [`CL-045`](docs/capability-ledger.md#baseline-claims-implemented) | Storage contract maturity/parity hardening: normalized paths + deterministic list ordering + metadata/checksum + resumable semantics tests | ✅ | `cd file-engine && go test ./internal/adapters/storage/local ./internal/services -run "TestLocalStorageContractSuite|TestLocalStorageListMetadata|TestUploadServiceResumableUploadFinalize" -v` |
| [`CL-046`](docs/capability-ledger.md#baseline-claims-implemented) | Governance controls baseline: startup-validated tenant policy config with quota/object/rate limits, retention/legal-hold delete protection, and policy-driven lifecycle cleanup controls | ✅ | `cd file-engine && go test ./internal/services ./internal/server -run "TestUploadServiceTenantPolicyQuotaFinalGate|TestUploadServiceRetentionBlocksDelete|TestUploadServiceLegalHoldBlocksDelete|TestGovernanceDeleteEndpointBlockedByRetention|TestLifecycleCleanupEndpoint" -v` |
| [`CL-047`](docs/capability-ledger.md#baseline-claims-implemented) | Upload API contract (`Initiate -> Upload chunk -> Complete`) is stable with idempotency/retry semantics and deterministic clean/dirty outcomes | ✅ | `docker compose up -d --build redis postgres file-engine file-engine-worker backend && ./scripts/wait-for-http.sh http://localhost:8081/healthz 120 && ./scripts/e2e/upload_lifecycle.sh && docker compose down -v` |
| [`CL-049`](docs/capability-ledger.md#baseline-claims-implemented) | Governance control-plane next step: archive-tier lifecycle transitions, external policy source distribution, drift-detection metrics/audit signal, and effective-policy operator endpoint | ✅ | `cd file-engine && go test ./internal/services ./internal/server ./internal/observability -run "TestLoadGovernancePolicyFromSourceEnvelope|TestUploadServiceArchiveLifecycleTransition|TestUploadServiceGovernanceDriftDetection|TestGovernanceEffectiveEndpoint|TestGovernanceDriftCheckEndpoint|TestSnapshotPrometheusIncludesQueueTaskAndOperabilityMetrics" -v` |
| [`CL-058`](docs/capability-ledger.md#baseline-claims-implemented) | Enterprise readiness v2 baseline: deterministic k6 smoke/soak load profiles, tenant+actor fairness throttles, queue backpressure reject signaling, and per-tenant cost usage report endpoint | ✅ | `cd file-engine && go test ./internal/observability ./internal/server ./internal/handlers -run "TestTenantUsageSnapshot|TestTenantCostReportEndpoint|TestCreateFolderQueueUnavailableReturnsUnavailable" -v` |
| [`CL-059`](docs/capability-ledger.md#baseline-claims-implemented) | Data durability & recovery baseline: backup/restore simulation scripts, disaster drill suite (DB restore/replay, storage corruption detection, audit sink outage catch-up), and integrity verification endpoint/job with failure metrics | ✅ | `cd file-engine && go test ./internal/server ./internal/observability -run "TestIntegrityVerifyEndpointDetectsCorruption|TestTenantUsageSnapshot|TestSnapshotPrometheusIncludesQueueTaskAndOperabilityMetrics" -v && bash -n scripts/backup_restore_simulation.sh scripts/integrity_verify_job.sh scripts/drills/db_restore_replay.sh scripts/drills/storage_corruption_drill.sh scripts/drills/audit_sink_catchup_drill.sh` |
| [`CL-060`](docs/capability-ledger.md#baseline-claims-implemented) | API productization baseline: explicit versioning policy, thin Go+PHP SDK layers, and frozen compatibility fixtures/gates for upload lifecycle, readiness, and authz-deny envelope stability | ✅ | `./scripts/check-api-compatibility.sh && cd file-engine && go test ./client -v && cd ../backend && php -l app/Clients/FileEngineClient.php && php -l app/Services/FileEngineService.php` |
| [`CL-061`](docs/capability-ledger.md#baseline-claims-implemented) | Maintenance-cost reduction baseline: legacy dual-path entrypoint removed, architecture conformance now enforces generated-doc freshness + no handler/server direct DB imports, and docs artifacts (endpoint inventory/route inventory/dashboard refs) are auto-generated | ✅ | `./scripts/generate-doc-artifacts.sh && ./scripts/architecture-conformance-check.sh && ./scripts/doc-drift-check.sh` |
| [`CL-062`](docs/capability-ledger.md#baseline-claims-implemented) | Scale/fairness closure: published SLOs for list/download, upload completion, and task completion; deterministic k6 smoke in CI with nightly soak; explicit throttled error envelope + audit evidence; and dependency backpressure drills with operator runbook | ✅ | `cd file-engine && go test ./internal/server ./internal/handlers -run "TestUploadRateLimitedReturnsThrottledEnvelopeAndAudit|TestCreateFolderQueueUnavailableReturnsUnavailable" -v && ./scripts/drills/dependency_backpressure.sh && bash -n scripts/drills/dependency_backpressure.sh` |
| [`CL-063`](docs/capability-ledger.md#baseline-claims-implemented) | Data durability/integrity contract closure: configurable integrity verification policy (`sample_size`, `failure_threshold`, `ignore_paths`), explicit dev-grade RTO/RPO restore objectives, script-backed durability drills, and one-command audit-ready evidence pack generation | ✅ | `cd file-engine && go test ./internal/server -run "TestIntegrityVerifyEndpointDetectsCorruption|TestIntegrityVerifyEndpointHonorsFailureThreshold|TestIntegrityVerifyEndpointIgnorePathsFalsePositive" -v && bash -n scripts/integrity_verify_job.sh scripts/backup_restore_simulation.sh scripts/drills/db_restore_replay.sh scripts/drills/storage_corruption_drill.sh scripts/generate_durability_evidence_pack.sh && ./scripts/generate_durability_evidence_pack.sh` |
| [`CL-064`](docs/capability-ledger.md#baseline-claims-implemented) | Multi-tenant compliance productization: access review v1 export keeps stable schema with optional signed artifact, governance policy updates emit audit event with before/after policy hashes, tenant evidence endpoint exposes policy/drift/review/drill pointers, and one-command tenant compliance packet generation is deterministic | ✅ | `cd file-engine && go test ./internal/server -run "TestGovernancePolicyUpdateEndpointEmitsBeforeAfterHashAudit|TestTenantEvidenceEndpointReturnsPointers" -v && bash -n file-engine/scripts/generate_monthly_access_review_report.sh scripts/generate_tenant_compliance_packet.sh && ./scripts/doc-drift-check.sh` |
| [`CL-065`](docs/capability-ledger.md#baseline-claims-implemented) | API/SDK hardening for external consumers: expanded golden fixtures (mutations + governance block + throttling), typed error/retry ergonomics in Go+PHP client layers, and CI compatibility policy enforcement for `/v1` surface changes with required docs updates | ✅ | `./scripts/check-api-compatibility.sh && cd file-engine && go test ./client ./internal/server -run "TestAsAPIErrorParsesEnvelope|TestDoWithRetryRetriesTemporaryAPIError|TestDoWithRetryStopsOnPermanentError|TestCompatibilityUploadThrottledGolden|TestCompatibilityGovernanceDeleteRetentionBlockGolden" -v && cd ../backend && php -l app/Clients/FileEngineClient.php && php -l app/Clients/FileEngineException.php && ./scripts/doc-drift-check.sh` |
| [`CL-069`](docs/capability-ledger.md#baseline-claims-implemented) | Human-resilience continuity gate: explicit CODEOWNERS for critical domains + CI reviewer continuity enforcement for auth/authz, monitoring/observability, and capability-ledger changes + release checklist new-maintainer drill gate | ✅ | `./scripts/check-owners-governance.sh && bash -n scripts/check-reviewer-continuity.sh && rg -n "reviewer-continuity|file-engine/internal/auth\*|observability/\*|docs/capability-ledger.md" .github/workflows/ci.yml .github/codeowners docs/branch-protection-mapping.md docs/prod-checklist.md` |
| [`CL-071`](docs/capability-ledger.md#baseline-claims-implemented) | Performance budgets and capacity planning closure: k6 smoke/soak enforce latency+error thresholds, error-budget policy is documented, rough component/queue/storage sizing guidance is published, and reproducible hot-path profiling is script-backed | ✅ | `cd file-engine && go test ./internal/server -run "^$" -bench "BenchmarkHandle(Download|UploadComplete)$" -benchtime=1x && ./file-engine/scripts/capture_hotpath_profile.sh && bash -n file-engine/scripts/capture_hotpath_profile.sh` |
| [`CL-072`](docs/capability-ledger.md#baseline-claims-implemented) | Security posture hardening closure: threat-model diff prompt automation, focused negative security regression suite, supply-chain checks (toolchain pin verification + SBOM/signing story), and secret rotation continuity drill | ✅ | `./scripts/generate-threat-model-diff-prompt.sh && ./scripts/security-regression-suite.sh && ./scripts/supply-chain-checks.sh && ./scripts/drills/rotate-secrets-drill.sh` |

> For target-state exclusions and promotion criteria, see [`docs/capability-ledger.md`](docs/capability-ledger.md).

</details>

---

## Canonical doc map

**Architecture & Implementation:**

- **API Reference:** [`docs/api-reference.md`](docs/api-reference.md)
- **API Versioning Policy:** [`docs/api-versioning-policy.md`](docs/api-versioning-policy.md)
- **Client SDKs (thin):** [`docs/client-sdks.md`](docs/client-sdks.md)
- **Architecture Overview:** [`docs/architecture.md`](docs/architecture.md)
- **Architecture Boundaries:** [`docs/architecture_boundaries.md`](docs/architecture_boundaries.md)
- **Auth Model (RBAC/JWT):** [`docs/auth.md`](docs/auth.md)
- **Threat Model:** [`docs/threat-model.md`](docs/threat-model.md)
- **Observability:** [`docs/observability.md`](docs/observability.md)
- **Roadmap (staged milestones):** [`docs/roadmap.md`](docs/roadmap.md)
- **Setup/onboarding guide:** [`docs/setup.md`](docs/setup.md)
- **Decisions and rationale:** [`docs/adr`](docs/adr)

**Governance & Status:**

- **Capability Ledger (Truth):** [`docs/capability-ledger.md`](docs/capability-ledger.md)
- **Route maturity matrix:** [`docs/route-maturity-matrix.md`](docs/route-maturity-matrix.md)
- **Project Alignment:** [`docs/project-alignment-review.md`](docs/project-alignment-review.md)
- **Governance (merge gates):** [`docs/governance.md`](docs/governance.md)
- **Branch protection mapping:** [`docs/branch-protection-mapping.md`](docs/branch-protection-mapping.md)
- **Ownership source of truth:** [`.github/OWNERS`](.github/OWNERS)
- **Ownership backup matrix:** [`docs/ownership-backup-matrix.md`](docs/ownership-backup-matrix.md)

<details><summary><b>Operating guide</b></summary>

- **Agent Constraints:** [`.github/AGENTS.md`](.github/AGENTS.md)
- **File Engine scoped operating guide:** [`file-engine/AGENTS.md`](file-engine/AGENTS.md)
- **Backend operating guide:** [`backend/AGENTS.md`](backend/AGENTS.md)

</details>

> [!Warning]
> If guidance conflicts, use this precedence order: capability ledger -> setup -> scoped AGENTS -> architecture deep-dives.

---

## Why this exists

Many organizations rely on direct file server access (shared drives/SSH/FTP) to create folders, upload documents, and manage structured storage. This is:

- hard to audit,
- easy to misuse (authorization drift, unsafe paths),
- inconsistent with compliance requirements,
- operationally fragile under load.

This platform provides a centralized, permissioned interface that **controls and records every filesystem mutation**.

---

## What it does

### Read path

- Browse folders (tree navigation, directory listing)
- Metadata display (size, timestamps, ownership) with backend-specific best-effort fields
- **Baseline-validated read path:** list results + size/timestamps/ownership metadata + download path normalization validated by [`CL-012`](docs/capability-ledger.md#baseline-claims-implemented)
- **Final authz enforcement for reads:** gRPC list/download enforce tenant-scoped paths, server-side tenant membership, and ACL/RBAC checks at File Engine boundary; verified by unit + integration coverage in [`[1]`](file-engine/internal/handlers/grpc_handler_test.go) and [`[2]`](file-engine/tests/integration/read_list_authz_integration_test.go)

### Write path (async)

- Create folders (policy-enforced naming)
- Upload lifecycle is baseline-validated end-to-end (`Initiate -> Upload chunk -> Complete`) with scan-gated promote semantics and deterministic clean/dirty outcomes (`CL-047`, `CL-033`, `CL-040`).
- Move/rename/delete/restore object operations *(API-level baseline validated; async task variants for move, governed delete, and quarantine restore are baseline-validated via `CL-066`..`CL-068`)*

### Governance & security

- JWT auth (Bearer)
- RBAC + path-based ACL (inheritance)
- Multi-tenant enforcement via **server-side tenant mapping**
- Upload quarantine + malware scan gate before publish (baseline guardrails + non-stub scanner adapter integration are validated)
- Dual-layer audit (queryable + tamper-resistant sink) with baseline-validated external sink delivery adapters
- Access review compliance exports are available via stable JSON contract + monthly operator report generator (`file-engine/scripts/export_access_review.sh`, `file-engine/scripts/generate_monthly_access_review_report.sh`).

---

## Architecture

### Control plane vs data plane

**Control Plane — Laravel (PHP):**

- UI/API orchestration and business validation (e.g., naming conventions)
- Integrations (planned/target): enterprise identity patterns (AD/LDAP/OIDC broker)
- Admin/UX aggregation (task status, audit views)

**Data Plane — Go File Engine + Worker:**

- gRPC-first API + HTTP/JSON via gRPC-Gateway (baseline for CreateFolder, GetTaskStatus, and upload lifecycle endpoints via `CL-047`)
- **Final authorization gate** (tenant membership + RBAC/ACL + safe-path execution)
- OTEL tracer provider wiring is initialized in both API + worker entrypoints when `OTEL_EXPORTER_OTLP_ENDPOINT` is set (baseline wiring claim `CL-038`)
- Enqueues tasks; worker executes storage operations with least privilege

### Diagram (trust boundaries)

```mermaid
flowchart TB
  U[User / Browser] -->|HTTPS| L[Laravel Control Plane<br/>UI + Business Validation]

  %% TB2: Service boundary
  L -->|"gRPC/HTTP (mTLS recommended)"| FE[Go File Engine API<br/>AuthContext + Final AuthZ Gate]

  %% TB3: Queue boundary
  FE --> Q[Redis Queue]
  Q --> W[Worker<br/>Executes tasks]

  %% TB4: Data boundary
  W --> ST["(Storage Backend<br/>Local/NFS/SMB/SFTP mounts<br/>S3/MinIO<br/>GCS)"]

  %% TB5: Scanner boundary
  W --> AV[Scanner Boundary<br/>ClamAV / pluggable]
  AV -->|verdict| W

  %% Audit
  FE --> DB["(Postgres<br/>audit_events (append-only, baseline-validated)<br/>ACL / mappings)"]
  W --> DB
  DB --> SINK[Immutable Audit Sink<br/>SIEM / Loki / S3 WORM]
```

---

### Multi-tenancy model

#### Server-side tenant mapping (source of truth)

- The system does not trust the client or JWT to define tenant scope.
- The File Engine resolves which tenants a user can act on using server-owned data (e.g., a mapping table/service).
- A request is authorized only if:
  - a. the user is mapped to the tenant, and
  - b. RBAC/ACL permits the operation on the target path within that tenant namespace.

#### Namespacing strategy

- Final (publishable): `tenants/<tenant_id>/...`
- Quarantine: `quarantine/<tenant_id>/<uploadId>/...`
- Malware hold: `malware/<tenant_id>/<uploadId>/...`

> Only objects/paths under `tenants/<tenant_id>/...` are listable/downloadable.

---

## Authentication & Authorization

### Authentication (JWT Bearer)

All endpoints require:

```Http
Authorization: Bearer <JWT>
```

Required claims:

- `sub` → user identifier
- `roles` → array of role names

Recommended production validation:

- RSA public-key verification
  - enforce `iss`, `aud`
  - validate `exp`

### Authorization (RBAC + path-based ACL with inheritance)

Authorization is enforced **before operations are executed/enqueued at the File Engine boundary**.

Resolution order:

1. Closest ACL for `user:<sub>` on path
2. Closest ACL for `role:<role>` on path
3. RBAC fallback (role defaults)
4. Deny by default

Inheritance walks up the path: `/a/b/c → /a/b → /a → /`

### No authorization drift (explicit responsibility split)

- Laravel may validate business intent (naming policies, UX flow), but must not be the final gate.
- **File Engine** is the final enforcement point for:
  - tenant membership (server-side mapping),
  - RBAC/ACL decision,
  - path normalization + safe execution constraints.

---

## File Engine API

> Full reference: `docs/api-reference.md`
> Route maturity by endpoint: `docs/route-maturity-matrix.md`

Contract source of truth

- Canonical proto: `file-engine/api/proto/fileengine.proto`
- Compatibility mirror (kept in sync): `file-engine/proto/fileengine.proto`

Base URLs:

- HTTP (gRPC-Gateway): `http://<host>:8080`
- gRPC: `<host>:50051`

Core gRPC methods (canonical):

- `CreateFolder` → returns `taskId` (async)
- `GetTaskStatus` → poll task status
- `InitiateUpload` / `CompleteUpload` *(baseline-validated via CL-047 contract flow; chunk upload path is exercised in the same validation)*

HTTP/JSON routes (baseline-validated):

- `POST /v1/folders` → `CreateFolder`
- `GET /v1/tasks/{taskId}` → `GetTaskStatus`
- `POST /v1/uploads:initiate` → `InitiateUpload`
- `PUT /v1/uploads/{uploadId}:chunk` → upload chunk
- `POST /v1/uploads/{uploadId}:complete` → `CompleteUpload`

Task state model (canonical):

- `queued → running → success | failed | quarantined`

---

## Current Implementation: Folder Flow

The platform has baseline-validated async folder creation and upload lifecycle flows; folder creation remains the minimal reference walkthrough below.

Implemented baseline reference flow:

1. `CreateFolder` (gRPC) receives request metadata and JWT-authenticated context at File Engine boundary.
2. API enqueues async task in Redis and persists initial task status (`queued`).
2a. API resolves tenant membership from server-side source-of-truth and rejects non-tenant-scoped or unauthorized tenant paths.
3. Worker consumes queue, executes filesystem folder creation, persists terminal status (`success`/`failed`), and emits audit-style events.
4. Client polls `GetTaskStatus` until completion.

Validation command:

```bash
cd file-engine && go test ./internal/handlers -run "TestCreateFolderRequiresAuthContext|TestCreateFolderRejectsNonTenantPath|TestCreateFolderRejectsUnauthorizedTenant|TestCreateFolderEnqueuesWithCorrelationAndActorFallback|TestGetTaskStatusRequiresAuthAndReturnsPersistedStatus" -v && go test ./tests/integration -run TestAsyncCreateFolderFlow -v
```

---

## Key flows

**Baseline-validated upload API flow (`CL-047`) with storage guardrails (`CL-033`, `CL-040`):**

```mermaid
sequenceDiagram
    autonumber
    participant U as User / UI
    participant FE as File Engine (API)
    participant DB as Postgres
    participant Q as Redis Queue
    participant W as Worker
    participant ST as Storage (S3/Local)

    Note over U, ST: Phase 1: Initiate & Upload

    U->>FE: POST /v1/uploads:initiate<br/>(path, size, mime)
    FE->>FE: Validate JWT, RBAC, Policy
    FE->>DB: Create Record (State: PENDING, Path: quarantine/...)
    FE-->>U: Return uploadId + uploadUrl

    U->>ST: PUT binary to uploadUrl<br/>(Writes to quarantine/tenant/id/...)
    
    Note over U, ST: Phase 2: Completion & Scan

    U->>FE: POST /v1/uploads/{id}:complete
    FE->>DB: Update State: QUEUED
    FE->>Q: Enqueue Scan Task
    FE-->>U: HTTP 202 Accepted (taskId)

    Q->>W: Pop Task
    W->>ST: Read File Stream
    W->>W: ClamAV Scan (Stream)

    alt Verdict: CLEAN
        W->>ST: Atomic Move<br/>(quarantine/... -> tenants/...)
        W->>DB: Update State: SUCCESS
        W->>DB: Log Audit (Promote)
    else Verdict: MALICIOUS
        W->>ST: Move to malware/... (Hold)
        W->>DB: Update State: QUARANTINED
        W->>DB: Log Audit (Security Alert)
    end
```

---

## Security model

Trust boundaries:

- TB1: Browser ↔ Laravel (untrusted input)
- TB2: Laravel ↔ File Engine (east-west; mTLS)
- TB3: Queue boundary (tamper/replay/poison messages)
- TB4: Storage boundary (least privilege; private endpoints)
- TB5: Scanner boundary (hostile bytes; sandboxed)

Secure-by-default controls:

- Deny-by-default authorization at File Engine
- Tenant scope from server-side mapping (not JWT)
- Strict path normalization + traversal rejection
- Quarantine → scan → promote gating (baseline guardrails + non-stub scanner integration validated via `CL-033` and `CL-040`)
- Redaction policy: never log tokens or pre-signed URLs

Known gaps / planned hardening (examples):

- Explicit deny rules in ACL (deny > allow) 🔒 (ADR candidate)
- Signed task payloads / replay defense 🔒 (ADR candidate)
- Stronger immutability guarantees for the secondary audit sink 🔒

> Detailed STRIDE model: `docs/threat-model.md`

---

## Auditing

**Dual-layer audit:**

- **Primary (queryable)**: Postgres `audit_events` table (append-only enforcement baseline-validated in `CL-032`)
- **Secondary (tamper-resistant)**: external sink (SIEM / Loki / S3 WORM) with retries + DLQ + lag metric baseline-validated in `CL-035`

**Baseline audit behavior:**

- Task audit events are emitted for the async folder flow (see `CL-005`).

Audit coverage (target baseline):

- Mutation events: create/move/rename/write, upload lifecycle, scan verdict, promote/hold decision
- Security events: authz denials, policy failures (rate-limited + aggregated as needed)
- Correlation fields: `request_id, trace_id, task_id, user_id, tenant_id, operation, path`, outcome

---

## Observability

Standards:

- JSON structured logs (consistent envelope, redaction)
- Request correlation across HTTP ↔ gRPC ↔ queue ↔ worker
  - X-Request-Id, traceparent (W3C)
- Distributed tracing via OpenTelemetry (OTLP exporter)

Operational signals to monitor:

- Queue depth / worker saturation
- Scan duration + pass/fail ratio
- Promotion failures
- Quarantine growth
- 403 spikes (probing / misconfig)

> Full spec: `docs/observability.md`

---

## Quickstart (local development)

Requirements:

- Go 1.24+
- Docker Engine / Docker Desktop + Compose v2 (optional; only needed for containerized dependencies)
- curl (optional; only needed for manual API calls)

### 1) Run the validated baseline checks (recommended)

This is the only **baseline-validated** quickstart today.

```bash
./file-engine/scripts/dev.sh
```

### 2) One-command onboarding + demo evidence

```bash
make bootstrap && make demo
```

This command pair regenerates docs, enforces architecture boundaries, runs doc drift checks, executes the deterministic 5-minute demo script, and prints evidence links for generated docs.

### 3) Optional: run the async folder flow integration test alone

```bash
cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v
```

### 4) Optional: local File Engine run (scaffold-level, for debugging)

This brings up Redis/Postgres in Docker and runs the API/worker locally for debugging. REST endpoints include baseline create-folder/task-status and upload lifecycle paths; treat this path as local debugging rather than the canonical baseline verification flow.

```bash
cd file-engine
docker compose up -d redis postgres

export REDIS_ADDR="localhost:6379"
export POSTGRES_DSN="postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
export STORAGE_BACKEND="local"
export FILE_BASE_ROOT="$PWD/data"
export JWT_SECRET="dev-secret"
export TENANT_MEMBERSHIPS="dev-admin=dev-tenant"

# Optional worker guardrails (defaults shown in parentheses):
# export WORKER_STATUS_RETRY_ATTEMPTS="3"
# export WORKER_STATUS_RETRY_DELAY_MS="25"
# export WORKER_TASK_PROCESS_TIMEOUT_MS="30000"

go run ./cmd/migrate
```

API terminal:

```bash
cd file-engine && go run ./cmd/file-engine
```

Worker terminal:

```bash
cd file-engine && go run ./cmd/worker
```

Dev JWT (HS256 with `JWT_SECRET=dev-secret`, `sub=dev-admin`, `roles=["admin"]`):

```bash
export JWT="***"
```

### 4) Canonical compose entry point

Use **repository-root `docker-compose.yml`** as the primary developer compose entry point.

`file-engine/docker-compose.yml` remains only as a compatibility mirror and should not be treated as the canonical source.

**Default ports:**

- HTTP: `8080`
- gRPC: `50051`
- Redis: `6379`
- Postgres: `5432`

> [!Note]
> All setup flows (local File Engine run, canonical root compose, dev JWT) are documented in `docs/setup.md`.


## Deployment (dev/stage/prod + kind + rollback)

- Environment profile templates are versioned in `env/.env.dev.example`, `env/.env.stage.example`, and `env/.env.prod.example`.
- Config/secret separation and required runtime wiring checks are documented in `docs/deployment-profiles.md` and validated with `./scripts/check-runtime-wiring.sh --profile prod`.
- Kubernetes smoke and rollback drill paths are script-backed via `./scripts/k8s/kind_smoke.sh` and `./scripts/drills/k8s_rollback_drill.sh`.
- Release versioning + changelog + rollback discipline is documented in `docs/release/versioning-and-rollback.md`.

---

## Repository structure

```text
file-server-management/
├─ frontend/                  # Static thin-client demo console (no Node build)
├─ backend/                   # Laravel control plane
├─ file-engine/               # Go File Engine (API + Worker)
└─ docs/
   ├─ adr/                    # Architectural Decision Records
   ├─ architecture.md         # Platform architecture
   ├─ api-reference.md        # API surface (gRPC + HTTP)
   ├─ auth.md                 # AuthN/AuthZ model
   ├─ observability.md        # Logs/metrics/tracing expectations
   ├─ threat-model.md         # Security model + STRIDE notes
   └─ setup.md                # Local development setup
```

---

## Disclaimer

This project is a work in progress. Some controls are documented as “target state” and may not be fully implemented yet. Each milestone aims to move documented intent into enforced reality.

---

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

<br><hr>
[🔼 Back to top](#back-to-top)
