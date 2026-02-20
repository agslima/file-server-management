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

echo "[drill] expected break scenarios (manual actions)"
echo "- break scanner: set MALWARE_SCANNER_BACKEND=clamav with unreachable CLAMAV_ADDR, then upload"
echo "  expected: upload.scan.completed with scan_error, scan DLQ entry, fileengine_scan_dlq_size > 0"
echo "- break sink: set AUDIT_SIEM_ENDPOINT to failing endpoint"
echo "  expected: fileengine_audit_sink_failures_total increment and DLQ/lag signals"
echo "- slow db: inject latency/failure on Postgres"
echo "  expected: readiness degradation, queue lag increase, traces show db span latency"

echo "DRILL_OK"
