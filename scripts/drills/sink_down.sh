#!/usr/bin/env bash
set -euo pipefail

echo "[sink-down] simulating immutable sink failure path via deterministic integration test"
cd file-engine
go test ./tests/integration -run 'TestAuditExternalSinkDeliveryWithDLQAndLagMetrics/siem_failures_retry_then_dead_letter' -v

echo "[sink-down] expected signals validated: fileengine_audit_sink_failures_total, DLQ entry, lag metrics"
