#!/usr/bin/env bash
set -euo pipefail

: "${TOKEN:?set TOKEN to an admin bearer token}"
: "${BASE_URL:=http://localhost:8080}"

curl -fsS "$BASE_URL/admin/v1/access-review${1:-}" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json"
