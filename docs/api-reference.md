# File Engine API Reference (gRPC + HTTP)

## Overview
The File Engine exposes a **gRPC-first** API (source of truth). HTTP/JSON via gRPC-Gateway is available for the baseline `CreateFolder` and `GetTaskStatus` routes.
Filesystem mutations run **asynchronously** via a worker, so the API returns a **Task ID** you can poll.

**Core domains**
- Filesystem commands: create folders (baseline); uploads are target-state
- Tasks: async status (`queued/running/success/failed/quarantined`)
- Authorization: enforced at the **File Engine boundary** using **JWT → AuthContext** + **tenant membership** + **RBAC + path-based ACL (with inheritance)**

**Service communication**
- Client → API: gRPC or HTTP/JSON
- API → Redis: enqueue tasks
- Worker → Storage backend: execute filesystem/object operations
- Postgres (optional): persist ACL rules (and task status if enabled)

## Baseline support (validated)

- `CreateFolder` (gRPC + HTTP/JSON) → async task enqueued
- `GetTaskStatus` (gRPC + HTTP/JSON) → task status polling

Uploads and other routes remain target-state until implemented.

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

Tenant membership is resolved **server-side** (not from JWT claims). Requests without tenant membership are denied.

See `docs/auth.md` for details.

## gRPC Methods (canonical)

### 1) CreateFolder
**Purpose**: Create a folder under a parent path (async). Returns a task ID.

**Request**
```json
{
  "parentPath": "/tenants/123/projects/alpha",
  "folderName": "reports",
  "requestedBy": "user-42"
}
```
**Response**
```json
{
  "taskId": "b3a2c8f1-7a93-4a4c-9f92-3c7c8c1a12f9",
  "status": "queued",
  "message": "Folder creation scheduled"
}
```

### 2) GetTaskStatus
**Purpose**: Poll async task status.

**Response**
```json
{
  "taskId": "tsk_01HRXKAF...",
  "status": "running",
  "progress": 35,
  "message": "Moving object to final destination"
}
```

## HTTP/JSON mapping

Baseline routes:

- `POST /v1/folders` → `CreateFolder`
- `GET /v1/tasks/{taskId}` → `GetTaskStatus`

Target-state routes:

- `POST /v1/uploads:initiate` → `InitiateUpload`
- `POST /v1/uploads/{uploadId}:complete` → `CompleteUpload`

## Errors
See `docs/errors.md`.
