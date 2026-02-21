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

tmp_ok="$(mktemp -t oidc-ok.XXXXXX.json)"
tmp_deny="$(mktemp -t oidc-deny.XXXXXX.json)"
tmp_token="$(mktemp -t oidc-token.XXXXXX.json)"
trap 'rm -f "$tmp_ok" "$tmp_deny" "$tmp_token"' EXIT

token_endpoint="${KEYCLOAK_URL}/realms/${REALM}/protocol/openid-connect/token"

# --- Wait until Keycloak can actually mint tokens (realm import + user ready) ---
deadline=$((SECONDS + 120))
resp=""
while (( SECONDS < deadline )); do
  set +e
  resp=$(curl -sS -X POST "$token_endpoint" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    --data-urlencode "grant_type=password" \
    --data-urlencode "client_id=${CLIENT_ID}" \
    --data-urlencode "username=${USERNAME}" \
    --data-urlencode "password=${PASSWORD}" \
    -o "$tmp_token" -w "%{http_code}")
  rc=$?
  set -e

  if [[ $rc -eq 0 && "$resp" == "200" ]]; then
    break
  fi

  sleep 2
done

if [[ "$resp" != "200" ]]; then
  echo "failed to obtain token from Keycloak after retries: status=${resp}" >&2
  echo "endpoint=${token_endpoint} client_id=${CLIENT_ID} user=${USERNAME}" >&2
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

# --- Allowed tenant ---
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

# --- Denied cross-tenant ---
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

# Optional: strengthen proof by asserting error shape / code (only if your API includes it)
# Example: {"error":{"code":"AUTHZ_DENY","reason":"tenant_mapping_denied",...}}
python3 - <<'PY' "$tmp_deny" || true
import json,sys
p=sys.argv[1]
try:
  body=json.load(open(p))
except Exception:
  sys.exit(0)
# tighten these checks once your error schema is stable:
# assert body.get("error",{}).get("code") in ("AUTHZ_DENY","FORBIDDEN")
# assert "tenant" in json.dumps(body).lower()
PY

echo "OIDC_OK sub=${subject} allowed_parent=${PARENT_ALLOW} denied_parent=${PARENT_DENY}"
