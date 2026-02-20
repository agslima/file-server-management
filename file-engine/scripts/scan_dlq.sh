#!/usr/bin/env bash
set -euo pipefail

: "${TOKEN:?set TOKEN to an admin bearer token}"
: "${BASE_URL:=http://localhost:8080}"

cmd="${1:-list}"
case "$cmd" in
  list)
    curl -fsS "$BASE_URL/admin/v1/scan-dlq" -H "Authorization: Bearer $TOKEN" -H 'Accept: application/json'
    ;;
  retry)
    id="${2:?usage: scan_dlq.sh retry <dlq-id>}"
    curl -fsS -X POST "$BASE_URL/admin/v1/scan-dlq" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"id\":\"$id\"}"
    ;;
  resolve)
    id="${2:?usage: scan_dlq.sh resolve <dlq-id>}"
    curl -fsS -X DELETE "$BASE_URL/admin/v1/scan-dlq" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "{\"id\":\"$id\"}"
    ;;
  cleanup)
    ttl="${2:-3600}"
    curl -fsS -X POST "$BASE_URL/admin/v1/quarantine:cleanup?ttl_seconds=$ttl" -H "Authorization: Bearer $TOKEN"
    ;;
  *)
    echo "unknown command: $cmd" >&2
    exit 1
    ;;
esac
