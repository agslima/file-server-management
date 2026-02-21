#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

check_metric() {
  local metric="$1"
  if ! curl -fsS "$BASE_URL/metrics" | rg -q "$metric"; then
    echo "missing expected metric: $metric" >&2
    return 1
  fi
}

echo "[drill] checking baseline observability signals"
check_metric "fileengine_queue_lag_ms_max"
check_metric "fileengine_scan_dlq_size"
check_metric "fileengine_audit_sink_failures_total"
check_metric "fileengine_authz_decisions_total"

echo "[drill] running deterministic production hardening drills"
./scripts/drills/production_deployment_hardening.sh

echo "DRILL_OK"
