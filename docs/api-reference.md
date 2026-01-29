# File Engine API Reference (gRPC + HTTP)

## Overview
The File Engine exposes a **gRPC-first** API (source of truth) with an **HTTP/JSON surface** served via **gRPC-Gateway**.
Clients call the API; filesystem mutations run **asynchronously** via a worker, so the API returns a **Task ID** you can poll.

**Core domains**
- Filesystem commands: create folders, upload files (two-step), move/write operations (executed by worker)
- Tasks: long-running job status (`queued/running/success/failed`)
- Authorization: enforced at the API boundary using **JWT → AuthContext** + **RBAC + path-based ACL (with inheritance)**

**Service communication**
- Client → API: gRPC or HTTP/JSON
- API → Redis: enqueue tasks
- Worker → Storage backend: execute filesystem/object operations
- Postgres (optional): persist ACL rules (and tasks if you choose to persist later)

## Base URLs
### HTTP
- `http://<host>:8080`
### gRPC
- `<host>:50051`

## Authentication
All endpoints require:
```
Authorization: Bearer <JWT>
```
Claims:
- `sub` → user id
- `roles` → role list

See `docs/auth.md` for details.

## Endpoint Definitions (HTTP/JSON)
> NOTE: These routes represent a recommended canonical mapping for gRPC-Gateway.
> Align with your `.proto` `google.api.http` annotations when present.

### 1) Create Folder
- **URL**: `POST /v1/folders`
- **Purpose**: Create a folder under a parent path (async). Returns a task ID.

**Request**
```json
{
  "parentPath": "/tenants/123/projects/alpha",
  "folderName": "reports",
  "requestedBy": "user-42"
}
```
**Success (202)**
```json
{
  "taskId": "b3a2c8f1-7a93-4a4c-9f92-3c7c8c1a12f9",
  "status": "queued",
  "message": "Folder creation scheduled"
}
```
**Failure (403)**
```json
{
  "error": {
    "code": "permission_denied",
    "message": "access denied",
    "details": {
      "requiredPermission": "write",
      "path": "/tenants/123/projects/alpha"
    }
  }
}
```

**curl**
```bash
curl -X POST http://localhost:8080/v1/folders   -H "Authorization: Bearer <JWT>"   -H "Content-Type: application/json"   -d '{
    "parentPath": "/tenants/123/projects/alpha",
    "folderName": "reports",
    "requestedBy": "user-42"
  }'
```

### 2) Initiate Upload
- **URL**: `POST /v1/uploads:initiate`
- **Purpose**: Start an upload session and return an upload URL (pre-signed or internal).

**Request**
```json
{
  "targetPath": "/tenants/123/projects/alpha/reports",
  "filename": "report.pdf",
  "size": 1048576,
  "mime": "application/pdf"
}
```
**Success (200)**
```json
{
  "uploadId": "upl_01HRXK9V8Q...",
  "uploadUrl": "https://storage-provider/.../signed-url"
}
```

**curl**
```bash
curl -X POST http://localhost:8080/v1/uploads:initiate   -H "Authorization: Bearer <JWT>"   -H "Content-Type: application/json"   -d '{
    "targetPath": "/tenants/123/projects/alpha/reports",
    "filename": "report.pdf",
    "size": 1048576,
    "mime": "application/pdf"
  }'
```

### 3) Complete Upload
- **URL**: `POST /v1/uploads/{uploadId}:complete`
- **Purpose**: Finalize upload and enqueue post-processing. Returns a task ID.

**Request**
```json
{ "uploadId": "upl_01HRXK9V8Q..." }
```
**Success (202)**
```json
{ "taskId": "tsk_01HRXKAF...", "status": "queued" }
```

**curl**
```bash
curl -X POST http://localhost:8080/v1/uploads/upl_01HRXK9V8Q...:complete   -H "Authorization: Bearer <JWT>"   -H "Content-Type: application/json"   -d '{"uploadId":"upl_01HRXK9V8Q..."}'
```

### 4) Get Task Status
- **URL**: `GET /v1/tasks/{taskId}`
- **Purpose**: Poll async task status.

**Success (200)**
```json
{
  "taskId": "tsk_01HRXKAF...",
  "status": "running",
  "progress": 35,
  "message": "Moving object to final destination"
}
```

**curl**
```bash
curl http://localhost:8080/v1/tasks/tsk_01HRXKAF...   -H "Authorization: Bearer <JWT>"
```

## Errors
See `docs/errors.md`.
