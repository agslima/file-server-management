# File Engine (Go) - Skeleton

This service is responsible for interacting with remote file servers (SMB / NFS / SFTP) and performing
operations such as creating folders, moving uploaded files, and reporting task status.

## What is included
- HTTP server with endpoints:
  - GET /health
  - POST /create-folder
  - POST /uploads/initiate
  - POST /uploads/complete
  - GET /tasks/{id}

- gRPC proto file `proto/fileengine.proto` with service definitions. Generate Go code with protoc:
  ```bash
  protoc --go_out=. --go-grpc_out=. proto/fileengine.proto
  ```
  After generating, uncomment the import and registration lines in `cmd/main.go`.

## Build & Run (development)
Requires Go 1.21+

```bash
cd file-engine-go
go mod tidy
go build ./cmd
# or
go run ./cmd/main.go
```

## Docker
A Dockerfile is provided to build a small runtime image. Adjust as necessary for production.

## Notes
- Current implementation is a stub. Replace stubs in `internal/filesystem` with real SMB/SFTP/NFS logic.
- For production, integrate with Redis/Kafka for task queues, and a persistent datastore for task tracking.