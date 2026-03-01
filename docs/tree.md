.
├── CHANGELOG.md
├── CODE_OF_CONDUCT.md
├── LICENSE
├── Makefile
├── README.md
├── SECURITY.md
├── backend
│   ├── AGENTS.md
│   ├── Dockerfile
│   ├── README.md
│   ├── api.php
│   ├── app
│   │   ├── Clients
│   │   │   ├── FileEngineClient.php
│   │   │   └── FileEngineException.php
│   │   ├── Http
│   │   │   └── Controllers
│   │   │       ├── AuthController.php
│   │   │       ├── Controller.php
│   │   │       ├── FolderController.php
│   │   │       ├── ObjectMutationController.php
│   │   │       ├── TaskController.php
│   │   │       └── UploadController.php
│   │   ├── Services
│   │   │   └── FileEngineService.php
│   │   └── Support
│   │       └── TraceHeaders.php
│   ├── composer.json
│   ├── composer.lock
│   ├── config
│   │   └── services.php
│   ├── phpstan.neon
│   ├── phpunit.xml
│   ├── routes
│   │   └── api.php
│   ├── scripts
│   │   └── smoke.sh
│   └── tests
│       ├── Integration
│       │   └── VS001CreateFolderE2ETest.php
│       └── Unit
│           ├── ControllersTest.php
│           └── TraceHeadersTest.php
├── docker-compose.yml
├── docs
│   ├── README.md
│   ├── adr
│   │   ├── 0001-adr-process.md
│   │   ├── 0002-hybrid-php-go-architecture.md
│   │   ├── 0003-async-jobs-for-mutations.md
│   │   ├── 0004-queue-technology-selection.md
│   │   ├── 0005-service-to-service-auth.md
│   │   ├── 0006-path-safety-model.md
│   │   ├── 0007-upload-staging-and-malware-gating.md
│   │   └── 0008-observability-requirements.md
│   ├── api-reference.md
│   ├── api-versioning-policy.md
│   ├── api_storage_authz.md
│   ├── architecture.md
│   ├── architecture_boundaries.md
│   ├── architecture_file-engine.md
│   ├── auth.md
│   ├── branch-protection-mapping.md
│   ├── capability-ledger.md
│   ├── client-sdks.md
│   ├── codebase-issue-tasks.md
│   ├── compliance-access-review-workflow.md
│   ├── contributors.md
│   ├── dataflow-security-risk-assessment.md
│   ├── deployment-profiles.md
│   ├── errors.md
│   ├── generated
│   │   ├── dashboard-references.md
│   │   ├── endpoint-inventory.md
│   │   ├── policy-schema.md
│   │   ├── route-maturity-matrix.md
│   │   └── sdk-examples.md
│   ├── governance.md
│   ├── jwt_integration.md
│   ├── observability.md
│   ├── ownership-backup-matrix.md
│   ├── performance-budgets-capacity.md
│   ├── platform-engineers.md
│   ├── prod-checklist.md
│   ├── project-alignment-review.md
│   ├── release
│   │   └── versioning-and-rollback.md
│   ├── roadmap-ledger-gap-analysis.md
│   ├── roadmap.md
│   ├── route-maturity-matrix.md
│   ├── runbooks
│   │   ├── data-durability-recovery.md
│   │   ├── governance-controls-operations.md
│   │   ├── malware-gate-operations.md
│   │   ├── observability-incident-drill.md
│   │   ├── scale-fairness-operations.md
│   │   └── supply-chain-security.md
│   ├── security-reviewers.md
│   ├── setup.md
│   ├── storage_backends.md
│   ├── threat-model.md
│   └── tree.md
├── env
├── file-engine
│   ├── AGENTS.md
│   ├── Dockerfile
│   ├── Dockerfile.gen
│   ├── Makefile
│   ├── README.md
│   ├── api
│   │   ├── Dockerfile
│   │   ├── openapi
│   │   │   └── fileengine.swagger.json
│   │   └── proto
│   │       ├── annotations.proto
│   │       ├── doc.go
│   │       ├── file_engine.pb.go
│   │       ├── file_engine_grpc.pb.go
│   │       ├── file_engine_grpc_gateway.pb.go
│   │       ├── fileengine.proto
│   │       └── google
│   │           └── api
│   │               └── http.proto
│   ├── client
│   │   ├── doc.go
│   │   ├── grpc_client.go
│   │   ├── http_client.go
│   │   └── http_client_test.go
│   ├── cmd
│   │   ├── file-engine
│   │   │   └── main.go
│   │   ├── gateway
│   │   │   └── main.go
│   │   ├── migrate
│   │   │   └── main.go
│   │   ├── server
│   │   │   └── main.go
│   │   └── worker
│   │       └── main.go
│   ├── config
│   │   ├── governance-policy-source.example.json
│   │   └── governance-policy.example.json
│   ├── db
│   │   ├── migrations
│   │   │   ├── 0001_create_acl_entries.sql
│   │   │   ├── 0002_create_tenant_identity_tables.sql
│   │   │   ├── 0003_create_audit_events.sql
│   │   │   └── 0004_seed_dev_tenant_membership.sql
│   │   └── queries
│   │       └── acl.sql
│   ├── docker-compose.yml
│   ├── examples
│   │   └── client
│   │       └── go_http_example.go
│   ├── go.mod
│   ├── go.sum
│   ├── internal
│   │   ├── adapters
│   │   │   ├── config
│   │   │   │   ├── config.go
│   │   │   │   └── doc.go
│   │   │   ├── fs
│   │   │   │   ├── atomic.go
│   │   │   │   ├── doc.go
│   │   │   │   ├── filesystem.go
│   │   │   │   ├── fs_utils.go
│   │   │   │   ├── local
│   │   │   │   │   ├── doc.go
│   │   │   │   │   ├── local.go
│   │   │   │   │   └── local_test.go
│   │   │   │   └── sftp_fs.go
│   │   │   ├── queue
│   │   │   │   ├── doc.go
│   │   │   │   ├── redis_queue.go
│   │   │   │   └── redisq
│   │   │   │       ├── doc.go
│   │   │   │       └── redisq.go
│   │   │   ├── security
│   │   │   │   ├── clamav_scanner.go
│   │   │   │   ├── clamav_scanner_test.go
│   │   │   │   ├── doc.go
│   │   │   │   └── malware_scanner_stub.go
│   │   │   └── storage
│   │   │       ├── contract
│   │   │       │   └── suite.go
│   │   │       ├── gcs
│   │   │       │   ├── doc.go
│   │   │       │   ├── gcs_storage.go
│   │   │       │   └── gcs_storage_contract_test.go
│   │   │       ├── local
│   │   │       │   ├── doc.go
│   │   │       │   ├── local_storage.go
│   │   │       │   ├── local_storage_contract_test.go
│   │   │       │   └── local_storage_test.go
│   │   │       └── s3
│   │   │           ├── doc.go
│   │   │           ├── s3_storage.go
│   │   │           └── s3_storage_contract_test.go
│   │   ├── app
│   │   │   ├── ports
│   │   │   │   ├── doc.go
│   │   │   │   ├── fs.port.go
│   │   │   │   ├── malware_scanner.port.go
│   │   │   │   └── task_queue_port.go
│   │   │   ├── tasks
│   │   │   │   ├── audit_emitters.go
│   │   │   │   ├── audit_emitters_test.go
│   │   │   │   ├── audit_sinks.go
│   │   │   │   ├── audit_sinks_test.go
│   │   │   │   ├── doc.go
│   │   │   │   ├── processor.go
│   │   │   │   ├── worker.go
│   │   │   │   └── worker_test.go
│   │   │   └── usecases
│   │   │       └── doc.go
│   │   ├── auth
│   │   │   ├── README.md
│   │   │   ├── acl.go
│   │   │   ├── context.go
│   │   │   ├── doc.go
│   │   │   ├── grpc_interceptor.go
│   │   │   ├── http_middleware.go
│   │   │   ├── http_middleware_test.go
│   │   │   ├── inmemory_store.go
│   │   │   ├── jwt.go
│   │   │   ├── jwt_test.go
│   │   │   ├── permissions.go
│   │   │   ├── postgres_store.go
│   │   │   ├── resolver.go
│   │   │   ├── resolver_test.go
│   │   │   ├── roles.go
│   │   │   ├── store.go
│   │   │   ├── tenant_resolver.go
│   │   │   └── tenant_resolver_test.go
│   │   ├── authz
│   │   │   ├── doc.go
│   │   │   ├── grpc_authz_interceptor.go
│   │   │   ├── grpc_authz_interceptor_test.go
│   │   │   ├── method_map.go
│   │   │   ├── path_extract.go
│   │   │   └── path_extract_test.go
│   │   ├── config
│   │   │   ├── config.go
│   │   │   ├── config.yaml
│   │   │   └── doc.go
│   │   ├── delivery
│   │   │   └── grpc
│   │   │       ├── doc.go
│   │   │       └── fileengine_server.go
│   │   ├── di
│   │   │   ├── container.go
│   │   │   ├── container_test.go
│   │   │   └── doc.go
│   │   ├── handlers
│   │   │   ├── doc.go
│   │   │   ├── grpc_handler.go
│   │   │   ├── grpc_handler_test.go
│   │   │   └── http_handler.go
│   │   ├── identity
│   │   │   └── store.go
│   │   ├── infra
│   │   │   └── security
│   │   │       ├── doc.go
│   │   │       └── sftp_keyloader.go
│   │   ├── logger
│   │   │   ├── doc.go
│   │   │   └── logger.go
│   │   ├── observability
│   │   │   ├── doc.go
│   │   │   ├── metrics.go
│   │   │   ├── metrics_test.go
│   │   │   ├── tracing.go
│   │   │   └── tracing_test.go
│   │   ├── security
│   │   │   ├── doc.go
│   │   │   ├── validator.go
│   │   │   └── validator_test.go
│   │   ├── server
│   │   │   ├── admin_http.go
│   │   │   ├── admin_http_test.go
│   │   │   ├── compatibility_golden_test.go
│   │   │   ├── doc.go
│   │   │   ├── download_http.go
│   │   │   ├── download_http_test.go
│   │   │   ├── gateway_http_test.go
│   │   │   ├── hotpath_bench_test.go
│   │   │   ├── readiness_test.go
│   │   │   ├── server.go
│   │   │   ├── testdata
│   │   │   │   └── compat
│   │   │   │       ├── authz_deny.json
│   │   │   │       ├── governance_delete_retention_block.txt
│   │   │   │       ├── readyz_ok.json
│   │   │   │       ├── upload_complete.json
│   │   │   │       ├── upload_initiate.json
│   │   │   │       └── upload_throttled.json
│   │   │   ├── upload_http.go
│   │   │   └── upload_http_test.go
│   │   ├── services
│   │   │   ├── doc.go
│   │   │   ├── file_service.go
│   │   │   ├── governance.go
│   │   │   ├── governance_test.go
│   │   │   ├── object_service.go
│   │   │   ├── upload_service.go
│   │   │   └── upload_service_test.go
│   │   ├── storage
│   │   │   ├── doc.go
│   │   │   ├── factory
│   │   │   │   ├── doc.go
│   │   │   │   └── factory.go
│   │   │   ├── factory.go
│   │   │   └── storage.go
│   │   └── worker
│   │       ├── README.md
│   │       ├── doc.go
│   │       ├── worker.go
│   │       └── worker_impl.go
│   ├── proto
│   │   ├── annotations.proto
│   │   ├── fileengine.proto
│   │   └── google
│   │       └── api
│   │           └── http.proto
│   ├── scripts
│   │   ├── capture_hotpath_profile.sh
│   │   ├── dev.sh
│   │   ├── export_access_review.sh
│   │   ├── generate_grpc.sh
│   │   ├── generate_grpc_docker.sh
│   │   ├── generate_monthly_access_review_report.sh
│   │   ├── scan_dlq.sh
│   │   └── seed_identity.sh
│   ├── tests
│   │   ├── integration
│   │   │   ├── async_mutation_expansion_integration_test.go
│   │   │   ├── audit_append_only_integration_test.go
│   │   │   ├── audit_external_sink_integration_test.go
│   │   │   ├── audit_read_path_integration_test.go
│   │   │   ├── postgres_test_helpers.go
│   │   │   ├── read_list_authz_integration_test.go
│   │   │   ├── upload_real_scanner_integration_test.go
│   │   │   ├── upload_scan_gate_integration_test.go
│   │   │   ├── upload_staging_integration_test.go
│   │   │   └── worker_integration_test.go
│   │   └── load
│   │       ├── smoke.js
│   │       └── soak.js
│   └── worker
│       ├── Dockerfile
│       └── README.md
├── frontend
│   ├── Dockerfile
│   ├── README.md
│   ├── app.js
│   ├── index.html
│   └── styles.css
├── hack
│   ├── README.md
│   └── snyk-report.sh
├── infra
│   ├── README.md
│   └── keycloak
│       └── dev-realm.json
├── k8s
│   ├── kind
│   │   └── file-server-kind.yaml
│   └── readme.md
├── monitoring
│   ├── alerts
│   │   └── file-engine-alerts.yml
│   └── dashboards
│       └── file-engine-golden-signals.json
├── nginx
│   └── nginx.conf
├── observability
│   └── otel-collector-config.yaml
├── scripts
│   ├── architecture-conformance-check.sh
│   ├── backup_restore_simulation.sh
│   ├── check-api-compatibility.sh
│   ├── check-malware-runbook.sh
│   ├── check-otel-connectivity.sh
│   ├── check-owners-governance.sh
│   ├── check-paging-delivery.sh
│   ├── check-reviewer-continuity.sh
│   ├── check-runtime-wiring.sh
│   ├── doc-drift-check.sh
│   ├── doc-ownership-check.sh
│   ├── drills
│   │   ├── audit_sink_catchup_drill.sh
│   │   ├── db_restore_replay.sh
│   │   ├── dependency_backpressure.sh
│   │   ├── k8s_rollback_drill.sh
│   │   ├── new_maintainer_operability_drill.sh
│   │   ├── observability_incident_drill.sh
│   │   ├── otel_exporter_down.sh
│   │   ├── production_deployment_hardening.sh
│   │   ├── redis_backpressure.sh
│   │   ├── restore-scan-dlq-drill.sh
│   │   ├── rotate-secrets-drill.sh
│   │   ├── scanner_down.sh
│   │   ├── sink_down.sh
│   │   └── storage_corruption_drill.sh
│   ├── e2e
│   │   ├── demo_5_minute.sh
│   │   ├── mutation_surface.sh
│   │   ├── oidc_login_and_call_engine.sh
│   │   ├── run_oidc_profile.sh
│   │   ├── upload_lifecycle.sh
│   │   └── vs001_create_folder.sh
│   ├── generate-doc-artifacts.sh
│   ├── generate-quarterly-alignment-issue.sh
│   ├── generate-threat-model-diff-prompt.sh
│   ├── generate_durability_evidence_pack.sh
│   ├── generate_tenant_compliance_packet.sh
│   ├── integrity_verify_job.sh
│   ├── k8s
│   │   └── kind_smoke.sh
│   ├── ledger-baseline.sh
│   ├── security-regression-suite.sh
│   ├── supply-chain-checks.sh
│   ├── sustainability-metrics.sh
│   ├── validate-alert-rules.sh
│   ├── validate-observability-assets.sh
│   ├── wait-for-http.sh
│   └── wait-for-oidc-token.sh
└── tests
    └── test_ci_pr_security_scan_change_detection.py

100 directories, 330 files
