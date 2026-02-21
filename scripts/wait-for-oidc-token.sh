#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 4 || $# -gt 5 ]]; then
  echo "usage: $0 <token_endpoint> <client_id> <username> <password> [timeout_seconds]" >&2
  exit 2
fi

token_endpoint="$1"
client_id="$2"
username="$3"
password="$4"
timeout="${5:-120}"

deadline=$((SECONDS + timeout))
tmp="$(mktemp -t oidc-token-wait.XXXXXX.json)"
trap 'rm -f "$tmp"' EXIT

while (( SECONDS < deadline )); do
  status="$(curl -sS -X POST "$token_endpoint" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    --data-urlencode 'grant_type=password' \
    --data-urlencode "client_id=${client_id}" \
    --data-urlencode "username=${username}" \
    --data-urlencode "password=${password}" \
    -o "$tmp" -w '%{http_code}' || true)"

  if [[ "$status" == "200" ]]; then
    token="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("access_token",""))' "$tmp" 2>/dev/null || true)"
    if [[ -n "$token" ]]; then
      echo "ready oidc_token_endpoint=${token_endpoint} user=${username}"
      exit 0
    fi
  fi

  sleep 2
done

echo "timeout oidc_token_endpoint=${token_endpoint} user=${username} timeout=${timeout}s" >&2
cat "$tmp" >&2 || true
exit 1
