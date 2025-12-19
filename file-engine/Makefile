.PHONY: proto build server worker docker test

PROTO_DIR=./api/proto
OUT_PKG=./pkg/generated

proto:
	@echo "Generate gRPC code (requires protoc & plugins)"
	protoc -I=$(PROTO_DIR) --go_out=$(OUT_PKG) --go_opt=paths=source_relative \
	  --go-grpc_out=$(OUT_PKG) --go-grpc_opt=paths=source_relative \
	  --grpc-gateway_out=$(OUT_PKG) --grpc-gateway_opt=paths=source_relative,logtostderr=true \
	  $(PROTO_DIR)/*.proto || true

build: server worker

server:
	cd cmd/file-engine && go build -o ../../bin/file-engine ./

worker:
	cd cmd/worker && go build -o ../../bin/worker ./

test:
	go test ./... -v

docker: build
	docker build -f build/docker/server.Dockerfile -t file-engine-server .
	docker build -f build/docker/worker.Dockerfile -t file-engine-worker .