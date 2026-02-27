#!/usr/bin/env bash
set -euo pipefail

echo "[drill] storage corruption / missing object"
echo "[drill] step 1/3: upload a canary object through API"
echo "[drill] step 2/3: tamper/delete the canary object under FILE_BASE_ROOT"
echo "[drill] step 3/3: run integrity verification and expect checksum/object mismatch signal"
echo "[drill] command: ./scripts/integrity_verify_job.sh"
echo "STORAGE_CORRUPTION_DRILL_GUIDE_OK"
