#!/usr/bin/env bash
set -euo pipefail

echo "[drill] queue backpressure: expecting enqueue rejection signal when redis is unavailable"
REDIS_ADDR="127.0.0.1:6399" go test ./file-engine/internal/handlers -run TestCreateFolderQueueUnavailableReturnsUnavailable -v
if [[ -n "${METRICS_URL:-}" ]]; then
  metrics_endpoint="${METRICS_URL%/}/metrics"
  if ! curl -fsS --connect-timeout 5 --max-time 20 "$metrics_endpoint" | grep -q "fileengine_queue_backpressure_rejections_total"; then
    echo "[drill] metric fileengine_queue_backpressure_rejections_total not found at $metrics_endpoint" >&2
    exit 1
  fi
  echo "[drill] metric fileengine_queue_backpressure_rejections_total is present at $metrics_endpoint"
else
  echo "[drill] METRICS_URL is unset; skipping metrics endpoint verification"
fi
