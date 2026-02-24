#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8081}"
TOKEN="${TOKEN:-}"
SAMPLE_SIZE="${SAMPLE_SIZE:-25}"
FAILURE_THRESHOLD="${FAILURE_THRESHOLD:-0}"
IGNORE_PATHS="${IGNORE_PATHS:-}"

if [[ -z "$TOKEN" ]]; then
  echo "TOKEN is required" >&2
  exit 2
fi

echo "[integrity] calling /admin/v1/integrity:verify sample_size=${SAMPLE_SIZE} failure_threshold=${FAILURE_THRESHOLD}"
url="${BASE_URL}/admin/v1/integrity:verify?sample_size=${SAMPLE_SIZE}&failure_threshold=${FAILURE_THRESHOLD}"
if [[ -n "$IGNORE_PATHS" ]]; then
  echo "[integrity] false-positive suppression enabled for ignore_paths=${IGNORE_PATHS}"
  url="${url}&ignore_paths=${IGNORE_PATHS}"
fi

status="$(curl -sS -o /tmp/integrity_verify.json -w '%{http_code}' -X POST "$url" -H "Authorization: Bearer ${TOKEN}")"
if [[ "$status" == "200" ]]; then
  echo "INTEGRITY_VERIFY_OK sample_size=${SAMPLE_SIZE} failure_threshold=${FAILURE_THRESHOLD}"
  exit 0
fi

cat /tmp/integrity_verify.json
if [[ "$status" == "409" ]]; then
  echo "INTEGRITY_VERIFY_FAIL detected mismatch over_threshold=${FAILURE_THRESHOLD}" >&2
  exit 1
fi

echo "INTEGRITY_VERIFY_ERROR status=${status}" >&2
exit 3
