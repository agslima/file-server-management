#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="${KIND_CLUSTER_NAME:-file-platform}"
NAMESPACE="${K8S_NAMESPACE:-file-platform}"

# require_cmd checks that the specified command exists in PATH and exits with status 127 after printing an error if it is missing.
require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 127; }
}

require_cmd kind
require_cmd kubectl
require_cmd docker

kind get clusters | rg -q "^${CLUSTER_NAME}$" || kind create cluster --name "${CLUSTER_NAME}"

docker build -t file-engine:kind -f file-engine/build/docker/server.Dockerfile file-engine
docker build -t file-engine-worker:kind -f file-engine/build/docker/worker.Dockerfile file-engine
kind load docker-image file-engine:kind --name "${CLUSTER_NAME}"
kind load docker-image file-engine-worker:kind --name "${CLUSTER_NAME}"

kubectl apply -f k8s/kind/file-server-kind.yaml
kubectl -n "${NAMESPACE}" rollout status deploy/postgres --timeout=120s
kubectl -n "${NAMESPACE}" rollout status deploy/redis --timeout=120s
kubectl -n "${NAMESPACE}" rollout status deploy/file-engine --timeout=180s
kubectl -n "${NAMESPACE}" rollout status deploy/file-engine-worker --timeout=180s

kubectl -n "${NAMESPACE}" port-forward svc/file-engine 18080:8080 >/tmp/file-engine-kind-port-forward.log 2>&1 &
PF_PID=$!
trap 'kill ${PF_PID} >/dev/null 2>&1 || true' EXIT
sleep 3

./scripts/wait-for-http.sh http://localhost:18080/healthz 120
./scripts/wait-for-http.sh http://localhost:18080/readyz 120

echo "KIND_SMOKE_OK"
