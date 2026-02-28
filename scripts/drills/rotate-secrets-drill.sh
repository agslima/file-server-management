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

backend_url="${BACKEND_URL:-http://localhost:8081}"
revoked_token="${REVOKED_TOKEN:-}"
revoked_status_expected="${REVOKED_STATUS_EXPECTED:-401}"

fail() {
  local message="$1"
  echo "ROTATE_SECRETS_DRILL_FAIL ${message}" >&2
  exit 1
}

echo "[drill] apply mode: continuity validation checks"

if ! ./scripts/wait-for-http.sh "http://localhost:8080/healthz" 30; then
  fail "file-engine /healthz check failed"
fi
if ! ./scripts/wait-for-http.sh "${backend_url}/healthz" 30; then
  fail "backend /healthz check failed"
fi
if ! ./scripts/wait-for-http.sh "http://localhost:8080/readyz" 30; then
  fail "file-engine /readyz check failed"
fi

echo "[drill] verifying create-folder continuity"
if ! BACKEND_URL="$backend_url" ./scripts/e2e/vs001_create_folder.sh; then
  fail "create-folder continuity flow failed"
fi

echo "[drill] verifying revoked credential authentication failure"
if [[ -z "$revoked_token" ]]; then
  fail "REVOKED_TOKEN is required in --apply mode to validate authn failure for revoked credentials"
fi

revoke_status="$(curl -sS -o /dev/null -w '%{http_code}' \
  -X POST "${backend_url}/folders" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${revoked_token}" \
  -d '{"path":"tenants/acme","folderName":"revoked-drill","requestedBy":"rotate-secrets-drill"}' || true)"

if [[ "$revoke_status" != "$revoked_status_expected" ]]; then
  fail "revoked credential check failed expected_status=${revoked_status_expected} got_status=${revoke_status}"
fi

echo "ROTATE_SECRETS_DRILL_OK continuity=verified"
