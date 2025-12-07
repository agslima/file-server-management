gRPC + gRPC-Gateway scaffold for File Engine.

Usage:
- Install protoc and plugins (protoc-gen-go, protoc-gen-go-grpc, protoc-gen-grpc-gateway).
- Run `make proto` to generate Go code into ./pkg
- Build server: `go build ./cmd/server`
- Build gateway: `go build ./cmd/gateway`