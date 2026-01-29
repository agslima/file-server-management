#!/usr/bin/env bash
set -e

docker build -t protoc-gen -f build/docker/protoc-gen.Dockerfile .

docker run --rm \
  -v $(pwd):/workspace \
  -w /workspace \
  protoc-gen \
  protoc \
    -I api/proto \
    --go_out=pkg/generated --go_opt=paths=source_relative \
    --go-grpc_out=pkg/generated --go-grpc_opt=paths=source_relative \
    --grpc-gateway_out=pkg/generated --grpc-gateway_opt=paths=source_relative \
    --openapiv2_out=api/openapi \
    api/proto/*.proto
