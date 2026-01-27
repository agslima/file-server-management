# File Engine with RBAC / ACL by Path

```text
file-engine/
├── cmd/
│   ├── server/
│   │   └── main.go                  # gRPC + REST (gateway)
│   └── worker/
│       └── main.go                  # Worker encarregado de tasks FS
│
├── api/
│   ├── proto/
│   │   └── fileengine.proto         # Definição principal
│   └── openapi/
│       └── fileengine.yaml          # OpenAPI gerada automaticamente
│
├── internal/
│   ├── app/
│   │   ├── services/
│   │   │   └── file_service.go      # Orquestra o filesystem + tasks
│   │   ├── tasks/
│   │   │   └── processor.go         # Processamento das tasks no worker
│   │   └── ports/
│   │       ├── fs_port.go           # Interface Filesystem (Local/SFTP)
│   │       └── task_queue_port.go   # Interface Fila (Redis)
│   │
│   ├── adapters/
│   │   ├── fs/
│   │   │   ├── local_fs.go          # Filesystem local
│   │   │   ├── sftp_fs.go           # Filesystem remoto (SFTP)
│   │   │   └── atomic.go            # Helpers de atomic writes
│   │   ├── queue/
│   │   │   └── redis_queue.go       # Implementação Redis
│   │   └── config/
│   │       └── config.go            # Carregamento via env
│   │
│   ├── delivery/
│   │   ├── grpc/
│   │   │   └── fileengine_server.go # Implementação dos handlers gRPC
│   │   └── http/
│   │       └── gateway.go           # Gateway REST
│   │
│   └── infra/
│       ├── logger/
│       │   └── logger.go            # Logging estruturado (zerolog)
│       └── security/
│           └── sftp_keyloader.go    # Gestão de chaves SSH
│
├── pkg/
│   └── generated/                   # Código gerado pelo protoc
│
├── build/
│   ├── docker/
│   │   ├── server.Dockerfile
│   │   ├── worker.Dockerfile
│   │   └── protoc-gen.Dockerfile
│   └── ci/
│       └── github-actions.yml       # Pipeline de build/teste
│
├── test/
│   ├── fs/
│   │   └── fs_local_test.go         # Testes do filesystem local
│   └── integration/
│       └── worker_integration_test.go
│
├── go.mod
├── go.sum
└── README.md
```

```text
file-engine/
│
├── cmd/
│   ├── file-engine/
│   │   └── main.go
│   └── migrate/              # se futuramente houver DB
│
├── api/
│   ├── proto/
│   │   ├── file_engine.proto
│   │   ├── file_engine_grpc.pb.go
│   │   ├── file_engine.pb.go
│   │   └── file_engine_grpc_gateway.pb.go
│   └── openapi/
│       └── file-engine.openapi.yaml
│
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── config.yaml
│   │
│   ├── logger/
│   │   └── logger.go
│   │
│   ├── fs/
│   │   ├── filesystem.go          # interface abstrata
│   │   ├── local_fs.go            # implementação real (mkdir, move, atomic write)
│   │   └── fs_utils.go            # funções auxiliares
│   │
│   ├── services/
│   │   └── file_service.go        # regras de negócio
│   │
│   ├── handlers/
│   │   ├── grpc_handler.go        # gRPC
│   │   └── http_handler.go        # HTTP gateway
│   │
│   ├── security/
│   │   └── validator.go           # sanitização e naming rules
│   │
│   └── di/
│       └── container.go           # dependency injection
│
├── pkg/
│   └── util/
│       └── atomic.go
│
├── scripts/
│   ├── generate_grpc.sh
│   └── dev.sh
│
├── Dockerfile
├── docker-compose.yaml
├── go.mod
├── go.sum
└── README.md

```
