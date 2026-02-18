# File Server Management Repository Structure

```text
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
│   │   │       ├── FolderController.php
│   │   │       ├── TaskController.php
│   │   │       └── UploadController.php
│   │   └── Services
│   │       └── FileEngineService.php
│   ├── composer.json
│   ├── config
│   │   └── services.php
│   ├── phpunit.xml
│   ├── routes
│   │   └── api.php
│   └── tests
│       └── Unit
│           └── ControllersTest.php
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
│   ├── roadmap.md
│   ├── route-maturity-matrix.md
│   ├── security-reviewers.md
│   ├── setup.md
│   ├── storage_backends.md
│   └── threat-model.md
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
│   ├── db
│   │   ├── migrations
│   │   │   ├── 0001_create_acl_entries.sql
│   │   │   ├── 0002_create_tenant_identity_tables.sql
│   │   │   └── 0003_create_audit_events.sql
│   │   └── queries
│   │       └── acl.sql
│   ├── docker-compose.yml
│   ├── go.mod
│   ├── go.sum
│   ├── internal
│   │   ├── adapters
│   │   │   ├── config
│   │   │   │   └── config.go
│   │   │   ├── fs
│   │   │   │   ├── atomic.go
│   │   │   │   ├── filesystem.go
│   │   │   │   ├── fs_utils.go
│   │   │   │   ├── local
│   │   │   │   │   ├── local.go
│   │   │   │   │   └── local_test.go
│   │   │   │   └── sftp_fs.go
│   │   │   ├── queue
│   │   │   │   ├── redis_queue.go
│   │   │   │   └── redisq
│   │   │   │       └── redisq.go
│   │   │   ├── security
│   │   │   │   └── malware_scanner_stub.go
│   │   │   └── storage
│   │   │       ├── gcs
│   │   │       │   └── gcs_storage.go
│   │   │       ├── local
│   │   │       │   ├── local_storage.go
│   │   │       │   └── local_storage_test.go
│   │   │       └── s3
│   │   │           └── s3_storage.go
│   │   ├── app
│   │   │   ├── ports
│   │   │   │   ├── fs.port.go
│   │   │   │   ├── malware_scanner.port.go
│   │   │   │   └── task_queue_port.go
│   │   │   ├── services
│   │   │   │   └── file_service.go
│   │   │   └── tasks
│   │   │       ├── audit_emitters.go
│   │   │       ├── audit_emitters_test.go
│   │   │       ├── audit_sinks.go
│   │   │       ├── audit_sinks_test.go
│   │   │       ├── processor.go
│   │   │       ├── worker.go
│   │   │       └── worker_test.go
│   │   ├── auth
│   │   │   ├── README.md
│   │   │   ├── acl.go
│   │   │   ├── context.go
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
│   │   │   ├── grpc_authz_interceptor.go
│   │   │   ├── grpc_authz_interceptor_test.go
│   │   │   ├── method_map.go
│   │   │   ├── path_extract.go
│   │   │   └── path_extract_test.go
│   │   ├── config
│   │   │   ├── config.go
│   │   │   └── config.yaml
│   │   ├── delivery
│   │   │   ├── grpc
│   │   │   │   └── fileengine_server.go
│   │   │   └── http
│   │   │       └── gateway.go
│   │   ├── di
│   │   │   ├── container.go
│   │   │   └── container_test.go
│   │   ├── handlers
│   │   │   ├── grpc_handler.go
│   │   │   ├── grpc_handler_test.go
│   │   │   └── http_handler.go
│   │   ├── infra
│   │   │   ├── logger
│   │   │   │   └── logger.go
│   │   │   └── security
│   │   │       └── sftp_keyloader.go
│   │   ├── logger
│   │   │   └── logger.go
│   │   ├── observability
│   │   │   ├── metrics.go
│   │   │   └── metrics_test.go
│   │   ├── security
│   │   │   ├── validator.go
│   │   │   └── validator_test.go
│   │   ├── server
│   │   │   ├── download_http.go
│   │   │   ├── download_http_test.go
│   │   │   ├── gateway_http_test.go
│   │   │   ├── readiness_test.go
│   │   │   ├── server.go
│   │   │   ├── upload_http.go
│   │   │   └── upload_http_test.go
│   │   ├── services
│   │   │   ├── file_service.go
│   │   │   ├── object_service.go
│   │   │   ├── upload_service.go
│   │   │   └── upload_service_test.go
│   │   ├── storage
│   │   │   ├── factory
│   │   │   │   └── factory.go
│   │   │   ├── factory.go
│   │   │   └── storage.go
│   │   └── worker
│   │       ├── README.md
│   │       ├── worker.go
│   │       └── worker_impl.go
│   ├── pkg
│   │   ├── generated
│   │   │   ├── fileengine.pb.go
│   │   │   ├── fileengine.pb.gw.go
│   │   │   └── fileengine_grpc.pb.go
│   │   └── util
│   │       └── atomic.go
│   ├── proto
│   │   ├── annotations.proto
│   │   ├── fileengine.proto
│   │   └── google
│   │       └── api
│   │           └── http.proto
│   ├── scripts
│   │   ├── dev.sh
│   │   ├── generate_grpc.sh
│   │   └── generate_grpc_docker.sh
│   ├── tests
│   │   ├── fs
│   │   │   └── fs_local_test.go
│   │   └── integration
│   │       ├── read_list_authz_integration_test.go
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
├── k8s
│   └── readme.md
├── nginx
│   └── nginx.conf
├── scripts
│   └── doc-drift-check.sh
└── tree.md

80 directories, 184 files
```
