#!/usr/bin/env bash
set -euo pipefail

echo "[drill] storage corruption / missing object"
echo "1) Upload an object through API."
echo "2) Tamper/delete the file in FILE_BASE_ROOT."
echo "3) Run ./scripts/integrity_verify_job.sh and expect non-zero with mismatch report."
echo "STORAGE_CORRUPTION_DRILL_GUIDE_OK"
