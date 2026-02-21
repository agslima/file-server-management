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
│   │   ├── Http
│   │   │   └── Controllers
│   │   │       ├── AuthController.php
│   │   │       ├── Controller.php
│   │   │       ├── FolderController.php
│   │   │       ├── TaskController.php
│   │   │       └── UploadController.php
│   │   ├── Services
│   │   │   └── FileEngineService.php
│   │   └── Support
│   │       └── TraceHeaders.php
│   ├── bootstrap
│   │   └── cache
│   ├── composer.json
│   ├── composer.lock
│   ├── config
│   │   └── services.php
│   ├── phpunit.xml
│   ├── routes
│   │   └── api.php
│   ├── scripts
│   │   └── smoke.sh
│   ├── storage
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
│   ├── api_storage_authz.md
│   ├── architecture.md
│   ├── architecture_file-engine.md
│   ├── auth.md
│   ├── capability-ledger.md
│   ├── contributors.md
│   ├── dataflow-security-risk-assessment.md
│   ├── errors.md
│   ├── governance.md
│   ├── jwt_integration.md
│   ├── observability.md
│   ├── platform-engineers.md
│   ├── postman_collection.json
│   ├── prod-checklist.md
│   ├── project-alignment-review.md
│   ├── roadmap-ledger-gap-analysis.md
│   ├── roadmap.md
│   ├── route-maturity-matrix.md
│   ├── runbooks
│   │   ├── governance-controls-operations.md
│   │   ├── malware-gate-operations.md
│   │   └── observability-incident-drill.md
│   ├── security-reviewers.md
│   ├── setup.md
│   ├── storage_backends.md
│   ├── threat-model.md
│   └── tree.md
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
│   ├── build
│   │   └── docker
│   │       ├── protoc-gen.Dockerfile
│   │       ├── server.Dockerfile
│   │       └── worker.Dockerfile
│   ├── cmd
│   │   ├── file-engine
│   │   │   └── main.go
│   │   ├── gateway
│   │   │   └── main.go
│   │   ├── main.go
│   │   ├── migrate
│   │   │   └── main.go
│   │   ├── server
│   │   │   └── main.go
│   │   └── worker
│   │       └── main.go
│   ├── config
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
│   │   │   ├── services
│   │   │   │   ├── doc.go
│   │   │   │   └── file_service.go
│   │   │   └── tasks
│   │   │       ├── audit_emitters.go
│   │   │       ├── audit_emitters_test.go
│   │   │       ├── audit_sinks.go
│   │   │       ├── audit_sinks_test.go
│   │   │       ├── doc.go
│   │   │       ├── processor.go
│   │   │       ├── worker.go
│   │   │       └── worker_test.go
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
│   │   │   ├── grpc
│   │   │   │   ├── doc.go
│   │   │   │   └── fileengine_server.go
│   │   │   └── http
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
│   │   │   ├── logger
│   │   │   │   ├── doc.go
│   │   │   │   └── logger.go
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
│   │   │   ├── doc.go
│   │   │   ├── download_http.go
│   │   │   ├── download_http_test.go
│   │   │   ├── gateway_http_test.go
│   │   │   ├── readiness_test.go
│   │   │   ├── server.go
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
│   ├── pkg
│   │   ├── generated
│   │   │   ├── fileengine.pb.go
│   │   │   ├── fileengine.pb.gw.go
│   │   │   └── fileengine_grpc.pb.go
│   │   └── util
│   ├── proto
│   │   ├── annotations.proto
│   │   ├── fileengine.proto
│   │   └── google
│   │       └── api
│   │           └── http.proto
│   ├── scripts
│   │   ├── dev.sh
│   │   ├── export_access_review.sh
│   │   ├── generate_grpc.sh
│   │   ├── generate_grpc_docker.sh
│   │   ├── scan_dlq.sh
│   │   └── seed_identity.sh
│   ├── tests
│   │   ├── fs
│   │   └── integration
│   │       ├── audit_append_only_integration_test.go
│   │       ├── audit_external_sink_integration_test.go
│   │       ├── audit_read_path_integration_test.go
│   │       ├── postgres_test_helpers.go
│   │       ├── read_list_authz_integration_test.go
│   │       ├── upload_real_scanner_integration_test.go
│   │       ├── upload_scan_gate_integration_test.go
│   │       ├── upload_staging_integration_test.go
│   │       └── worker_integration_test.go
│   └── worker
│       ├── Dockerfile
│       └── README.md
├── frontend
│   ├── Dockerfile
│   └── README.md
├── hack
│   ├── README.md
│   └── snyk-report.sh
├── infra
│   └── keycloak
│       └── dev-realm.json
├── k8s
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
└── scripts
    ├── architecture-conformance-check.sh
    ├── doc-drift-check.sh
    ├── doc-ownership-check.sh
    ├── drills
    │   └── observability_incident_drill.sh
    ├── e2e
    │   ├── oidc_login_and_call_engine.sh
    │   └── vs001_create_folder.sh
    ├── ledger-baseline.sh
    ├── validate-alert-rules.sh
    ├── validate-observability-assets.sh
    └── wait-for-http.sh

98 directories, 258 files
