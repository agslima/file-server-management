#!/usr/bin/env bash
set -euo pipefail

mode="dry-run"
if [[ "${1:-}" == "--apply" ]]; then
  mode="apply"
fi

root_dir="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root_dir"

echo "ROTATE_SECRETS_DRILL mode=${mode}"

if [[ "$mode" == "dry-run" ]]; then
  echo "1) Generate replacement secrets (JWT, DB credentials, queue credentials) in secret manager"
  echo "2) Roll file-engine + worker + backend with staggered rollout"
  echo "3) Verify continuity: /healthz, /readyz, create-folder task flow, and authn failures for revoked credentials"
  echo "4) Record drill evidence in runbook and incident channel"
  echo "ROTATE_SECRETS_DRILL_OK"
  exit 0
fi

echo "[drill] apply mode: continuity validation checks"
./scripts/wait-for-http.sh http://localhost:8080/healthz 30
./scripts/wait-for-http.sh http://localhost:8081/healthz 30

./scripts/e2e/vs001_create_folder.sh
./scripts/security-regression-suite.sh

echo "ROTATE_SECRETS_DRILL_OK continuity=verified"
