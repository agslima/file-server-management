#!/usr/bin/env bash
set -euo pipefail

KEYCLOAK_URL="${KEYCLOAK_URL:-http://localhost:8082}"
REALM="${OIDC_REALM:-file-engine}"
CLIENT_ID="${OIDC_CLIENT_ID:-file-engine-dev}"
USERNAME="${OIDC_USERNAME:-dev-admin}"
PASSWORD="${OIDC_PASSWORD:-dev-admin}"
ENGINE_URL="${ENGINE_URL:-http://localhost:8080}"

PARENT_ALLOW="${OIDC_ALLOW_PARENT_PATH:-/tenants/acme/projects}"
PARENT_DENY="${OIDC_DENY_PARENT_PATH:-/tenants/beta/projects}"

WAIT_TIMEOUT="${OIDC_WAIT_TIMEOUT:-120}"
TOKEN_WAIT_SCRIPT="${OIDC_TOKEN_WAIT_SCRIPT:-./scripts/wait-for-oidc-token.sh}"

tmp_ok="$(mktemp -t oidc-ok.XXXXXX.json)"
tmp_deny="$(mktemp -t oidc-deny.XXXXXX.json)"
tmp_token="$(mktemp -t oidc-token.XXXXXX.json)"
trap 'rm -f "$tmp_ok" "$tmp_deny" "$tmp_token"' EXIT

token_endpoint="${KEYCLOAK_URL}/realms/${REALM}/protocol/openid-connect/token"

"${TOKEN_WAIT_SCRIPT}" "${token_endpoint}" "${CLIENT_ID}" "${USERNAME}" "${PASSWORD}" "${WAIT_TIMEOUT}"

status="$(curl -sS -X POST "$token_endpoint" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode "grant_type=password" \
  --data-urlencode "client_id=${CLIENT_ID}" \
  --data-urlencode "username=${USERNAME}" \
  --data-urlencode "password=${PASSWORD}" \
  -o "$tmp_token" -w '%{http_code}')"

if [[ "$status" != "200" ]]; then
  echo "failed to obtain token after readiness check: status=${status}" >&2
  cat "$tmp_token" >&2 || true
  exit 1
fi

token=$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["access_token"])' "$tmp_token")

subject=$(python3 - <<'PY' "$token"
import base64, json, sys
parts=sys.argv[1].split('.')
payload=parts[1]+'='*((4-len(parts[1])%4)%4)
claims = json.loads(base64.urlsafe_b64decode(payload))
print(claims.get('sub') or claims.get('preferred_username') or claims.get('email') or "")
PY
)

ok_status=$(curl -sS -o "$tmp_ok" -w "%{http_code}" -X POST "${ENGINE_URL}/v1/folders" \
  -H "Authorization: Bearer ${token}" \
  -H 'Content-Type: application/json' \
  -d "{\"parentPath\":\"${PARENT_ALLOW}\",\"folderName\":\"oidc-proof\",\"requestedBy\":\"oidc-e2e\"}")

if [[ "$ok_status" != "200" ]]; then
  echo "expected allowed tenant create to succeed, got status=${ok_status}" >&2
  echo "engine=${ENGINE_URL} parentPath=${PARENT_ALLOW}" >&2
  cat "$tmp_ok" >&2 || true
  exit 1
fi

deny_status=$(curl -sS -o "$tmp_deny" -w "%{http_code}" -X POST "${ENGINE_URL}/v1/folders" \
  -H "Authorization: Bearer ${token}" \
  -H 'Content-Type: application/json' \
  -d "{\"parentPath\":\"${PARENT_DENY}\",\"folderName\":\"oidc-deny\",\"requestedBy\":\"oidc-e2e\"}")

if [[ "$deny_status" != "403" ]]; then
  echo "expected cross-tenant create to be denied, got status=${deny_status}" >&2
  echo "engine=${ENGINE_URL} parentPath=${PARENT_DENY}" >&2
  cat "$tmp_deny" >&2 || true
  exit 1
fi

python3 - <<'PY' "$tmp_deny"
import json,sys
body=json.load(open(sys.argv[1]))
err=body.get("error") or {}
assert err.get("code") == "AUTHZ_DENY", f"unexpected code: {err.get('code')}"
assert err.get("reason") == "tenant_mapping_denied", f"unexpected reason: {err.get('reason')}"
assert err.get("tenant_id"), "missing tenant_id"
assert (err.get("request_id") or err.get("correlation_id")), "missing request/correlation id"
PY

echo "OIDC_OK sub=${subject} allowed_parent=${PARENT_ALLOW} denied_parent=${PARENT_DENY} deny_reason=tenant_mapping_denied"
