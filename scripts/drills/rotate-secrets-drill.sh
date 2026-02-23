#!/usr/bin/env bash
set -euo pipefail

mode="dry-run"
if [[ "${1:-}" == "--apply" ]]; then
  mode="apply"
fi

echo "ROTATE_SECRETS_DRILL mode=${mode}"
if [[ "$mode" == "apply" ]]; then
  echo "[drill] apply mode intentionally no-op — rotation handled elsewhere"
  echo "ROTATE_SECRETS_DRILL_OK"
  exit 0
fi

echo "1) rotate API and worker credentials in secret manager"
echo "2) redeploy backend + file-engine with rotated secrets"
echo "3) verify /healthz and /readyz"
echo "ROTATE_SECRETS_DRILL_OK"
