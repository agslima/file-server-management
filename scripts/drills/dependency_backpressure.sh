#!/usr/bin/env bash
set -euo pipefail

echo "[drill] dependency backpressure matrix: redis/db/storage latency/unavailable signals"

cd "$(dirname "$0")/../.."

cd file-engine

go test ./internal/handlers -run TestCreateFolderQueueUnavailableReturnsUnavailable -v

go test ./internal/server -run "TestHandleReadyzReturnsServiceUnavailableWhenAnyCheckFails|TestHandleReadyzReturnsReadyWhenChecksPass" -v

go test ./internal/services -run "TestUploadServiceScannerFailureEnqueuesDLQ|TestUploadServiceQuotaLimit" -v

echo "BACKPRESSURE_DRILL_OK runbook=docs/runbooks/scale-fairness-operations.md"
