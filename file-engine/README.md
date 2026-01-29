# File Engine - Complete Scaffold (Generated)

This scaffold includes:
- Clean architecture layout
- Local filesystem adapter (atomic writes, move with fallback)
- Redis-backed worker and queue adapter
- gRPC proto and placeholders for handlers
- Makefile to generate gRPC code and build binaries
- Dockerfiles and docker-compose for local development

Usage:
1. Install protoc and Go protoc plugins (protoc-gen-go, protoc-gen-go-grpc, protoc-gen-grpc-gateway)
2. Run `make proto` to generate gRPC code into pkg/generated/
3. Build: `make build`
4. Run: `docker-compose up --build` or run binaries in ./bin

## ACL Persistence (PostgreSQL)

This incremental version adds a PostgreSQL-backed ACL store.

### Start dependencies

```bash
docker-compose up -d postgres redis
```

### Apply migrations

```bash
export POSTGRES_DSN='postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable'
go run ./cmd/migrate
```

### Migration files
- `db/migrations/0001_create_acl_entries.sql`

### Reference queries
- `db/queries/acl.sql`


## Authorization Tests

This version adds unit tests for:

- RBAC fallback
- ACL overrides
- Path inheritance
- Deny-by-default behavior
- JWT parsing and HTTP middleware

Run:

```bash
go test ./internal/auth -v
```


## Storage backends (Local / S3 / GCS)

See `docs/STORAGE_BACKENDS.md`.

Quick example:

```bash
export STORAGE_BACKEND=s3
export S3_BUCKET=my-bucket
export S3_REGION=us-east-1
```

```text
file-engine/
├── api
│   ├── Dockerfile
│   ├── openapi
│   │   └── openapi.yaml            # OpenAPI gerada automaticamente
│   └── proto
│       ├── annotations.proto
│       ├── file_engine_grpc_gateway.pb.go
│       ├── file_engine_grpc.pb.go
│       ├── file_engine.pb.go
│       └── fileengine.proto         # Definição principal
├── build
│   └── docker
│       ├── protoc-gen.Dockerfile
│       ├── server.Dockerfile
│       └── worker.Dockerfile
├── cmd
│   ├── file-engine
│   │   └── main.go
│   ├── gateway
│   │   └── main.go
│   ├── main.go
│   ├── migrate
│   │   └── main.go
│   ├── server
│   │   └── main.go                  # gRPC + REST (gateway)
│   └── worker
│       └── main.go                  # Worker encarregado de tasks FS
├── db
│   ├── migrations
│   │   └── 0001_create_acl_entries.sql
│   └── queries
│       └── acl.sql
├── internal
│   ├── adapters
│   │   ├── config
│   │   │   └── config.go            # Carregamento via env
│   │   ├── fs
│   │   │   ├── atomic.go              # Helpers de atomic writes
│   │   │   ├── filesystem.go          # interface abstrata
│   │   │   ├── fs_utils.go            # funções auxiliares  
│   │   │   ├── local_fs.go            # implementação real (mkdir, move, atomic write)
│   │   │   ├── local
│   │   │   │   ├── local.go
│   │   │   │   └── local_test.go
│   │   │   └── sftp_fs.go           # Filesystem remoto (SFTP)
│   │   ├── queue
│   │   │   ├── redisq
│   │   │   │   └── redisq.go
│   │   │   └── redis_queue.go       # Implementação Redis
│   │   └── storage
│   │       ├── gcs
│   │       │   └── gcs_storage.go
│   │       ├── local
│   │       │   └── local_storage.go
│   │       └── s3
│   │           └── s3_storage.go
│   ├── app
│   │   ├── ports
│   │   │   ├── fs_port.go           # Interface Filesystem (Local/SFTP)
│   │   │   └── task_queue_port.go   # Interface Fila (Redis)
│   │   ├── services
│   │   │   └── file_service.go      # Orquestra o filesystem + tasks
│   │   └── tasks
│   │       ├── processor.go         # Processamento das tasks no worker
│   │       └── worker.go
│   ├── auth
│   │   ├── acl.go
│   │   ├── context.go
│   │   ├── grpc_interceptor.go
│   │   ├── http_middleware.go
│   │   ├── http_middleware_test.go
│   │   ├── inmemory_store.go
│   │   ├── jwt.go
│   │   ├── jwt_test.go
│   │   ├── permissions.go
│   │   ├── postgres_store.go
│   │   ├── README.md
│   │   ├── resolver.go
│   │   ├── resolver_test.go
│   │   ├── roles.go
│   │   └── store.go
│   ├── config
│   │   ├── config.go
│   │   └── config.yaml
│   ├── delivery
│   │   ├── grpc
│   │   │   └── fileengine_server.go # Implementação dos handlers gRPC
│   │   └── http
│   │       └── gateway.go           # Gateway REST
│   ├── di
│   │   └── container.go             # dependency injection
│   ├── filesystem
│   │   ├── fs.go
│   │   ├── fs_test.go
│   │   └── sftp_client.go
│   ├── fs
│   │   ├── filesystem.go
│   │   ├── fs_utils.go
│   │   └── local_fs.go
│   ├── handlers
│   │   ├── grpc_handler.go          # gRPC
│   │   └── http_handler.go          # HTTP gateway
│   ├── infra
│   │   ├── logger
│   │   │   └── logger.go            # Logging estruturado (zerolog)
│   │   └── security
│   │       └── sftp_keyloader.go    # Gestão de chaves SSH
│   ├── logger
│   │   └── logger.go
│   ├── security
│   │   └── validator.go             # sanitização e naming rules
│   ├── server
│   │   └── server.go
│   ├── services
│   │   └── file_service.go          # regras de negócio
│   ├── storage
│   │   ├── factory.go
│   │   └── storage.go
│   └── worker
│       ├── README.md
│       ├── worker.go
│       └── worker_impl.go
├── Makefile
├── pkg
│   ├── generated/                   # Código gerado pelo protoc
│   │   └── stub.go
│   └── util
│       └── atomic.go
├── proto
│   └── fileengine.proto
├── README.md
├── scripts
│   ├── dev.sh
│   ├── generate_grpc_docker.sh
│   └── generate_grpc.sh
├── tests
│   ├── fs
│   │   └── fs_local_test.go         # Testes do filesystem loca
│   └── integration
│       └── worker_integration_test.go
├── worker
│    ├── Dockerfile
│    └── README-worker.md
├── docker-compose.yml
├── Dockerfile
├── Dockerfile.gen
├── go.mod
└── go.sum

```


