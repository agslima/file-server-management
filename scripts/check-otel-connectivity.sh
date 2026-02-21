#!/usr/bin/env bash
set -euo pipefail

BACKEND_URL="${BACKEND_URL:-http://localhost:8081}"
JAEGER_URL="${JAEGER_URL:-http://localhost:16686}"
EXPECTED_EXPORTER_ENDPOINT="${EXPECTED_EXPORTER_ENDPOINT:-http://otel-collector:4317}"
TRACE_LOOKBACK='1h'
MAX_ATTEMPTS="${MAX_ATTEMPTS:-20}"
SLEEP_SECONDS="${SLEEP_SECONDS:-2}"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "required command not found: $1" >&2; exit 1; }
}

require_cmd curl
require_cmd python3
require_cmd rg
require_cmd docker

check_container_exporter_endpoint() {
  local service="$1"
  if ! docker compose ps --services --status running 2>/dev/null | rg -qx "$service"; then
    echo "[otel-check] service not running, skip endpoint check: $service"
    return 0
  fi

  local observed
  observed="$(docker compose exec -T "$service" printenv OTEL_EXPORTER_OTLP_ENDPOINT 2>/dev/null | tr -d '\r')"
  if [[ "$observed" != "$EXPECTED_EXPORTER_ENDPOINT" ]]; then
    echo "[otel-check] unexpected OTEL endpoint for ${service}: expected=${EXPECTED_EXPORTER_ENDPOINT} observed=${observed:-<empty>}" >&2
    exit 1
  fi
  echo "[otel-check] ${service} exporter endpoint ok (${observed})"
}

trigger_trace_flow() {
  local body_file
  body_file="$(mktemp -t otel-connectivity-body.XXXXXX.json)"
  trap 'rm -f "$body_file"' RETURN

  local status
  status="$(curl -sS -o "$body_file" -w '%{http_code}' -X POST "${BACKEND_URL}/folders" \
    -H 'Content-Type: application/json' \
    -H 'X-Request-Id: req-otel-connectivity' \
    -H 'X-Correlation-Id: corr-otel-connectivity' \
    -H 'traceparent: 00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01' \
    -H 'tracestate: vendor=smoke' \
    -d '{"path":"/tenants/acme/otel-check","folderName":"probe","requestedBy":"ops-smoke"}')"

  if [[ "$status" != "202" ]]; then
    echo "[otel-check] expected 202 from backend /folders, got ${status}" >&2
    cat "$body_file" >&2 || true
    exit 1
  fi
  echo "[otel-check] trace-producing request succeeded"
}

wait_for_jaeger_traces() {
  local service="$1"
  local attempt=1
  while (( attempt <= MAX_ATTEMPTS )); do
    if curl -fsS "${JAEGER_URL}/api/traces?service=${service}&limit=5&lookback=${TRACE_LOOKBACK}" | \
      python3 -c 'import json,sys; data=json.load(sys.stdin).get("data",[]); sys.exit(0 if data else 1)'; then
      echo "[otel-check] traces exported for service=${service}"
      return 0
    fi
    sleep "$SLEEP_SECONDS"
    ((attempt++))
  done

  echo "[otel-check] no traces found in Jaeger for service=${service} after ${MAX_ATTEMPTS} attempts" >&2
  return 1
}

check_container_exporter_endpoint file-engine
check_container_exporter_endpoint file-engine-worker
check_container_exporter_endpoint backend
trigger_trace_flow
wait_for_jaeger_traces file-engine-api
wait_for_jaeger_traces file-engine-worker

echo "OTEL_CONNECTIVITY_OK"
