#!/usr/bin/env bash
set -euo pipefail

echo "[scanner-down] simulating scanner outage path via deterministic service test"
cd file-engine
go test ./internal/services -run 'TestUploadServiceScannerFailureEnqueuesDLQ' -v

echo "[scanner-down] expected signals validated: scan_error + fileengine_scan_dlq_size growth"
