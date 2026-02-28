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

tmpfile="$(mktemp)"
trap 'rm -f "$tmpfile"' EXIT

echo "[integrity] calling /admin/v1/integrity:verify sample_size=${SAMPLE_SIZE} failure_threshold=${FAILURE_THRESHOLD}"
url="${BASE_URL}/admin/v1/integrity:verify?sample_size=${SAMPLE_SIZE}&failure_threshold=${FAILURE_THRESHOLD}"
if [[ -n "$IGNORE_PATHS" ]]; then
  echo "[integrity] false-positive suppression enabled for ignore_paths=${IGNORE_PATHS}"
  encoded_ignore_paths="$(python3 -c 'import sys, urllib.parse; print(urllib.parse.quote(sys.argv[1], safe=""))' "$IGNORE_PATHS")"
  url="${url}&ignore_paths=${encoded_ignore_paths}"
fi

status="$(curl -sS --connect-timeout 5 --max-time 30 -o "$tmpfile" -w '%{http_code}' -X POST "$url" -H "Authorization: Bearer ${TOKEN}")"
if [[ "$status" == "200" ]]; then
  echo "INTEGRITY_VERIFY_OK sample_size=${SAMPLE_SIZE} failure_threshold=${FAILURE_THRESHOLD}"
  exit 0
fi

cat "$tmpfile"
if [[ "$status" == "409" ]]; then
  echo "INTEGRITY_VERIFY_FAIL detected mismatch over_threshold=${FAILURE_THRESHOLD}" >&2
  exit 1
fi

echo "INTEGRITY_VERIFY_ERROR status=${status}" >&2
exit 3
