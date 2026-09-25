#!/usr/bin/env bash
set -euo pipefail

echo "[drill] audit sink outage then catch-up from DLQ"
./scripts/drills/sink_down.sh

echo "[drill] bring sink back, then verify DLQ catch-up via integration"
(
  cd file-engine
  go test ./tests/integration -run TestAuditExternalSinkDeliveryWithDLQAndLagMetrics -v
)

echo "AUDIT_SINK_CATCHUP_DRILL_OK"
