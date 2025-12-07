PROTO_DIR=./proto
OUT_PKG=./pkg

all: proto

proto:
	# Requires protoc, protoc-gen-go, protoc-gen-go-grpc and protoc-gen-grpc-gateway installed
	protoc -I=$(PROTO_DIR) \
	  --go_out=$(OUT_PKG) --go_opt=paths=source_relative \
	  --go-grpc_out=$(OUT_PKG) --go-grpc_opt=paths=source_relative \
	  --grpc-gateway_out=$(OUT_PKG) --grpc-gateway_opt=logtostderr=true,paths=source_relative \
	  $(PROTO_DIR)/*.proto

clean:
	rm -rf $(OUT_PKG)/*