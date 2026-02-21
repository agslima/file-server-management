#!/usr/bin/env bash
set -euo pipefail

runbook_path="docs/runbooks/malware-gate-operations.md"
alert_name="scanner_down"

echo "[scanner-down] simulating scanner outage path via deterministic service test"
(
  cd file-engine
  go test ./internal/services -run 'TestUploadServiceScannerFailureEnqueuesDLQ' -v
)

./scripts/check-malware-runbook.sh

echo "[scanner-down] alert=${alert_name}"
echo "[scanner-down] runbook=${runbook_path}"
echo "[scanner-down] recovery_steps=confirm_alert_and_metrics,verify_quarantine_only,recover_scanner,retry_dlq,confirm_slo_recovery"
echo "[scanner-down] expected signals validated: scan_error + fileengine_scan_dlq_size growth"
echo "SCANNER_DRILL_OK alert=${alert_name} runbook=${runbook_path}"
