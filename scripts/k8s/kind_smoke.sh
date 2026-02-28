#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="${KIND_CLUSTER_NAME:-file-platform}"
NAMESPACE="${K8S_NAMESPACE:-file-platform}"
CONTEXT="kind-${CLUSTER_NAME}"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 127; }
}

require_cmd kind
require_cmd kubectl
require_cmd docker
require_cmd envsubst

kind get clusters | grep -qx "${CLUSTER_NAME}" || kind create cluster --name "${CLUSTER_NAME}"

KUBECTL="kubectl --context=${CONTEXT}"

docker build -t file-engine:kind -f file-engine/build/docker/server.Dockerfile file-engine
docker build -t file-engine-worker:kind -f file-engine/build/docker/worker.Dockerfile file-engine
kind load docker-image file-engine:kind --name "${CLUSTER_NAME}"
kind load docker-image file-engine-worker:kind --name "${CLUSTER_NAME}"
POSTGRES_DSN="${POSTGRES_DSN:-postgres://fileengine:fileengine@postgres:5432/fileengine?sslmode=disable}"
JWT_SECRET="${JWT_SECRET:-dev-secret}"

$KUBECTL create namespace "${NAMESPACE}" --dry-run=client -o yaml | $KUBECTL apply -f -
$KUBECTL -n "${NAMESPACE}" create secret generic file-engine-secrets \
  --from-literal=POSTGRES_DSN="${POSTGRES_DSN}" \
  --from-literal=JWT_SECRET="${JWT_SECRET}" \
  --dry-run=client -o yaml | $KUBECTL apply -f -

env NAMESPACE="${NAMESPACE}" envsubst < k8s/kind/file-server-kind.yaml | $KUBECTL apply -f -
$KUBECTL -n "${NAMESPACE}" rollout status deploy/postgres --timeout=120s
$KUBECTL -n "${NAMESPACE}" rollout status deploy/redis --timeout=120s
$KUBECTL -n "${NAMESPACE}" rollout status deploy/file-engine --timeout=180s
$KUBECTL -n "${NAMESPACE}" rollout status deploy/file-engine-worker --timeout=180s

$KUBECTL -n "${NAMESPACE}" port-forward svc/file-engine 18080:8080 >/tmp/file-engine-kind-port-forward.log 2>&1 &
PF_PID=$!
trap 'kill ${PF_PID} >/dev/null 2>&1 || true' EXIT
sleep 3

./scripts/wait-for-http.sh http://localhost:18080/healthz 120
./scripts/wait-for-http.sh http://localhost:18080/readyz 120

echo "KIND_SMOKE_OK"
