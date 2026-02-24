#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8081}"
TOKEN="${TOKEN:-}"
SAMPLE_SIZE="${SAMPLE_SIZE:-25}"

if [[ -z "$TOKEN" ]]; then
  echo "TOKEN is required" >&2
  exit 2
fi

status="$(curl -sS -o /tmp/integrity_verify.json -w '%{http_code}' -X POST "${BASE_URL}/admin/v1/integrity:verify?sample_size=${SAMPLE_SIZE}" -H "Authorization: Bearer ${TOKEN}")"
if [[ "$status" == "200" ]]; then
  echo "INTEGRITY_VERIFY_OK sample_size=${SAMPLE_SIZE}"
  exit 0
fi

cat /tmp/integrity_verify.json
if [[ "$status" == "409" ]]; then
  echo "INTEGRITY_VERIFY_FAIL detected mismatch" >&2
  exit 1
fi

echo "INTEGRITY_VERIFY_ERROR status=${status}" >&2
exit 3
