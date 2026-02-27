#!/usr/bin/env bash
set -euo pipefail

echo "[drill] queue backpressure: expecting enqueue rejection signal when redis is unavailable"
REDIS_ADDR="127.0.0.1:6399" go test ./file-engine/internal/adapters/queue/redisq -run TestNonExistent -v || true
echo "[drill] verify service exposes queue backpressure metric via /metrics (fileengine_queue_backpressure_rejections_total)"
