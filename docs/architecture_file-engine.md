
# File Engine – Technical Documentation 📁

## 1. Overview

The File Engine is a backend service designed to manage real filesystem structures through a secure, scalable, and auditable API.
It provides functionality similar to a private Google Drive backend, operating directly on a filesystem while enforcing RBAC and path-based ACLs.

The system is built using Go, exposes both gRPC and REST (via gRPC-Gateway), and supports asynchronous filesystem operations through a worker model.


---

## 2. Key Features

- gRPC-first API with HTTP/REST support
- Real filesystem operations (mkdir, atomic writes, move)
- Asynchronous task execution via Redis-backed workers
- RBAC (Role-Based Access Control)
- ACL (Access Control List) enforced per filesystem path
- Path inheritance for permissions
- Clean Architecture with clear separation of concerns
- Dockerized and CI-ready
- OpenAPI specification generation

---

## 3. High-Level Architecture

```text
+-------------+        +------------------+
|   Client    | -----> | gRPC / REST API  |
+-------------+        +------------------+
                                |
                                v
                     +------------------------+
                     | Authorization Layer    |
                     | (RBAC + ACL Resolver)  |
                     +------------------------+
                                |
                                v
                     +------------------------+
                     | Application Services   |
                     | (Command orchestration)|
                     +------------------------+
                                |
                                v
                     +------------------------+
                     | Task Queue (Redis)     |
                     +------------------------+
                                |
                                v
                     +------------------------+
                     | Worker Process         |
                     | (Filesystem execution)|
                     +------------------------+
                                |
                                v
                     +------------------------+
                     | Filesystem (LocalFS)   |
                     +------------------------+
```

---

## 4. Project Structure

```text
file-engine/
├── api/
│   └── proto/                # gRPC contract
│
├── cmd/
│   ├── file-engine/          # API server entrypoint
│   └── worker/               # Background worker
│
├── internal/
│   ├── auth/                 # RBAC + ACL implementation
│   ├── fs/                   # Filesystem abstraction
│   ├── services/             # Domain services
│   ├── handlers/             # gRPC / HTTP handlers
│   ├── adapters/             # Redis, filesystem adapters
│   ├── di/                   # Dependency injection
│   └── config/               # Environment configuration
│
├── build/docker/              # Dockerfiles
├── scripts/                   # gRPC code generation
├── docker-compose.yml
├── go.mod
└── README.md
```

---

## 5. API Layer

### 5.1 gRPC

The system exposes a gRPC service as the source of truth.

Example service definition:

```Proto
service FileEngine {
  rpc CreateFolder(CreateFolderRequest) returns (CreateFolderResponse);
}
```

### 5.2 REST (gRPC-Gateway)

The REST API is automatically generated from the gRPC contract using grpc-gateway, ensuring:

- Single source of API definition
- Consistent validation and behavior
- OpenAPI documentation generation

---

## 6. Filesystem Layer

### 6.1 Local Filesystem Adapter

The LocalFS implementation performs real filesystem operations:

- CreateFolder
- AtomicWrite
- Move

Key guarantees:

- Atomic writes via temp file + rename
- Path sanitization to prevent traversal
- All paths are relative to a configured base directory

Example:

```Go
func (l *LocalFS) AtomicWrite(ctx context.Context, path string, r io.Reader) error {
    full := l.full(path)
    tmp := full + ".tmp"
    ...
    return os.Rename(tmp, full)
}
```

---

## 7. Asynchronous Task Model

Filesystem operations are executed asynchronously to avoid blocking API calls.

**Flow:**

- 1. API validates request and permissions
- 2. Task is enqueued in Redis
- 3. Worker consumes the task
- 4. Filesystem operation is executed
- 5. Task status is updated

This model allows:

- Better scalability
- Retry strategies
- Auditability
- Separation of concerns

---

## 8. Authorization Model (RBAC + ACL)

### 8.1 Permissions

```Go
read | write | delete | list
```

### 8.2 Roles (RBAC)

Roles define default permissions:

| Role   |	Permissions |
| ---    | ---          |
| admin  |	read, write, delete, list |
| editor |	read, write, list |
| viewer | read, list |


RBAC acts as a fallback when no ACL is defined.

### 8.3 ACL by Path

ACLs are defined per filesystem path:

```
type ACL struct {
    Path        string
    PrincipalID string // user:123 or role:admin
    Permissions map[Permission]bool
}
```

ACLs support:
- User-based rules
- Role-based rules
- Explicit permission overrides

### 8.4 Path Inheritance

If no ACL exists for a specific path, the system walks up the directory tree:

```text
/tenants/123/projects/456/files/report.pdf
↑
/tenants/123/projects/456
↑
/tenants/123
↑
/
```

The closest matching ACL wins.

### 8.5 Authorization Resolver

```Go
func CanAccess(ctx AuthContext, path string, perm Permission, store ACLStore) bool
```

Resolution order:

- 1. Explicit ACL (user)
- 2. Explicit ACL (role)
- 3. RBAC fallback
- 4. Deny by default

---

## 9. Security Considerations

- No wildcard permissions (*)
- No direct filesystem access from API
- Path traversal protection
- Principle of least privilege
- Authorization enforced before task creation
- Designed for multi-tenant environments

---

## 10. Deployment & Runtime

**Docker Compose**

```Yaml
services:
  api:
    ports:
      - 8080:8080
      - 50051:50051
  worker:
  redis:
```

- API and Worker are independent processes
- Redis acts as task broker
- Filesystem is mounted as a volume

---

## 11. Code Generation & Tooling

**gRPC Code Generation (Docker-based)**

```
scripts/generate_grpc_docker.sh
```

Generates:

- gRPC server/client stubs
- HTTP gateway
- OpenAPI spec


This avoids local dependency issues and ensures CI reproducibility.

---

## 12. Testing Strategy

- Unit tests for filesystem operations
- Deterministic ACL resolutions 
- Isolation using temporary directories
- Future-ready for integration tests

---

## 13. Design Decisions & Trade-offs

| Decision |	Rationale |
| --- | --- |
| gRPC-first |	Strong contracts, performance, tooling |
| Async FS	| Prevent blocking API |
| ACL by path	| Fine-grained security |
| Redis queue |	Simple, reliable, fast |
| Clean Architecture |	Long-term maintainability |

---

## 14. Future Enhancements

- PostgreSQL-backed ACL store
- JWT authentication integration
- Audit logs per filesystem action
- S3 / GCS / MinIO backend support
- Quota management
- Soft delete / versioning
