#!/usr/bin/env bash
set -euo pipefail

KEYCLOAK_URL="${KEYCLOAK_URL:-http://localhost:8082}"
REALM="${OIDC_REALM:-file-engine}"
CLIENT_ID="${OIDC_CLIENT_ID:-file-engine-dev}"
USERNAME="${OIDC_USERNAME:-dev-admin}"
PASSWORD="${OIDC_PASSWORD:-dev-admin}"
ENGINE_URL="${ENGINE_URL:-http://localhost:8080}"

resp=$(curl -fsS -X POST "${KEYCLOAK_URL}/realms/${REALM}/protocol/openid-connect/token" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  --data-urlencode "grant_type=password" \
  --data-urlencode "client_id=${CLIENT_ID}" \
  --data-urlencode "username=${USERNAME}" \
  --data-urlencode "password=${PASSWORD}")

token=$(python3 -c 'import json,sys;print(json.load(sys.stdin)["access_token"])' <<<"$resp")
subject=$(python3 - <<'PY' "$token"
import base64, json, sys
parts=sys.argv[1].split('.')
payload=parts[1]+'='*((4-len(parts[1])%4)%4)
claims = json.loads(base64.urlsafe_b64decode(payload))
print(claims.get('sub') or claims.get('preferred_username') or claims.get('email') or "")
PY
)

ok_status=$(curl -s -o /tmp/oidc-ok.json -w "%{http_code}" -X POST "${ENGINE_URL}/v1/folders" \
  -H "Authorization: Bearer ${token}" \
  -H 'Content-Type: application/json' \
  -d '{"parentPath":"/tenants/acme/projects","folderName":"oidc-proof","requestedBy":"oidc-e2e"}')

if [[ "$ok_status" != "200" ]]; then
  echo "expected allowed tenant create to succeed, got status=${ok_status}" >&2
  cat /tmp/oidc-ok.json >&2 || true
  exit 1
fi

deny_status=$(curl -s -o /tmp/oidc-deny.json -w "%{http_code}" -X POST "${ENGINE_URL}/v1/folders" \
  -H "Authorization: Bearer ${token}" \
  -H 'Content-Type: application/json' \
  -d '{"parentPath":"/tenants/beta/projects","folderName":"oidc-deny","requestedBy":"oidc-e2e"}')

if [[ "$deny_status" != "403" ]]; then
  echo "expected cross-tenant create to be denied by server mapping, got status=${deny_status}" >&2
  cat /tmp/oidc-deny.json >&2 || true
  exit 1
fi

echo "OIDC_OK sub=${subject} allowed_tenant=acme denied_tenant=beta"
