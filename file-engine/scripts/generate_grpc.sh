#!/usr/bin/env bash
set -e
PROTO_DIR=./api/proto
OUT_PKG=./pkg/generated
protoc -I=${PROTO_DIR} -I /usr/include -I /usr/local/include \
  --go_out=${OUT_PKG} --go_opt=paths=source_relative \
  --go-grpc_out=${OUT_PKG} --go-grpc_opt=paths=source_relative \
  --grpc-gateway_out=${OUT_PKG} --grpc-gateway_opt=paths=source_relative,logtostderr=true \
  ${PROTO_DIR}/fileengine.proto
echo "generated code into ${OUT_PKG}"
