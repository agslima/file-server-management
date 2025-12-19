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