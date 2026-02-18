
# File Engine – Technical Documentation

## 1. Overview

The File Engine is the data-plane service that performs filesystem mutations and enforces **final authorization** at the execution boundary.
It operates directly on storage backends while enforcing **tenant membership**, **RBAC**, and **path-based ACLs**.

Baseline onboarding is `./file-engine/scripts/dev.sh`; root `docker-compose.yml` is the canonical compose entry point. See `docs/setup.md` for canonical paths.

The system is built in Go and is **gRPC-first**. HTTP/JSON via gRPC-Gateway is **baseline** for `CreateFolder` and `GetTaskStatus`; upload and expanded read/write routes remain target-state.
Filesystem mutations run **asynchronously** through a worker model.


---

## 2. Key Features

**Baseline (validated)**

- gRPC-first API boundary with async task execution
- HTTP/JSON via gRPC-Gateway for `CreateFolder` + `GetTaskStatus`
- Real filesystem operations (mkdir) via worker
- RBAC + path-based ACLs with inheritance
- Tenant membership enforcement at File Engine boundary
- Structured logs and correlation IDs in async flow

**Target-state (planned)**

- HTTP/JSON via gRPC-Gateway for upload and extended read/write APIs
- Upload quarantine → scan → promote pipeline
- Full observability export pipeline (OTLP)

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
                     | (Tenant + RBAC + ACL)  |
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
│   └── proto/                # gRPC contract (canonical)
├── proto/                     # Mirror proto (must match canonical)
├── cmd/
│   ├── file-engine/          # API server entrypoint
│   └── worker/               # Background worker
├── internal/
│   ├── adapters/             # Redis + storage adapters
│   ├── app/                  # Task processor + worker orchestration
│   ├── auth/                 # JWT + RBAC + ACL
│   ├── authz/                # gRPC authz interceptors
│   ├── handlers/             # gRPC handlers
│   ├── observability/        # Metrics + log helpers
│   ├── server/               # gRPC + HTTP server wiring
│   ├── storage/              # Storage backends
│   └── worker/               # Worker runtime
├── build/docker/             # Dockerfiles
├── scripts/                  # Dev + proto/gateway generation
└── docker-compose.yml
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

HTTP/JSON is **baseline** for `CreateFolder` and `GetTaskStatus` and **target-state** for upload routes and expanded read APIs. Treat non-baseline routes as illustrative until validated in the capability ledger.

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

- 1. API validates JWT and enforces **tenant membership + RBAC/ACL**
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

### 8.0 Tenant scoping (final authz boundary)

- All tenant-scoped paths follow: `/tenants/<tenant_id>/...`
- Tenant membership is resolved **server-side** (not from JWT claims).
- If tenant membership cannot be resolved, the request is denied.

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
- Authorization enforced before task creation (final authz boundary)
- Designed for multi-tenant environments

---

## 10. Deployment & Runtime

**Docker Compose (file-engine scope)**

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

Generates (when run):

- gRPC server/client stubs
- HTTP gateway (baseline for `CreateFolder` + `GetTaskStatus`; other routes target-state)
- OpenAPI spec (baseline for `CreateFolder` + `GetTaskStatus`; other routes target-state)


This avoids local dependency issues and ensures CI reproducibility.

---

## 12. Testing Strategy

- Unit tests for auth resolver and filesystem helpers
- Integration test for async create-folder flow
- Temporary directories for isolation

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

- Promote additional gRPC-Gateway routes (uploads + expanded read/write) to baseline once validated
- Upload quarantine → scan → promote pipeline
- Expand audit coverage beyond task lifecycle
- Quota management
- Soft delete / versioning
