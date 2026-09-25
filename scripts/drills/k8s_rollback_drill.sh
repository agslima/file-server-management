#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${K8S_NAMESPACE:-file-platform}"

command -v kubectl >/dev/null 2>&1 || { echo "missing required command: kubectl" >&2; exit 127; }

kubectl -n "${NAMESPACE}" annotate deployment/file-engine rollback-drill.ts="$(date -u +%Y%m%dT%H%M%SZ)" --overwrite
kubectl -n "${NAMESPACE}" rollout status deployment/file-engine --timeout=120s

kubectl -n "${NAMESPACE}" set env deployment/file-engine LOG_LEVEL=debug
kubectl -n "${NAMESPACE}" rollout status deployment/file-engine --timeout=180s

kubectl -n "${NAMESPACE}" rollout undo deployment/file-engine
kubectl -n "${NAMESPACE}" rollout status deployment/file-engine --timeout=180s

if kubectl -n "${NAMESPACE}" get deploy/file-engine -o jsonpath='{.spec.template.spec.containers[0].env[*].name}' | rg -q 'LOG_LEVEL'; then
  echo "ROLLBACK_WARNING_LOG_LEVEL_STILL_PRESENT"
  exit 1
fi

echo "K8S_ROLLBACK_OK"
