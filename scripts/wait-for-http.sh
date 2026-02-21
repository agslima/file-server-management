#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: $0 <url> [timeout_seconds]" >&2
  exit 2
fi

url="$1"
timeout="${2:-60}"

deadline=$((SECONDS + timeout))

while (( SECONDS < deadline )); do
  if code="$(curl -s -o /dev/null -w '%{http_code}' "$url" 2>/dev/null)"; then
    :
  else
    code="000"
  fi
  if [[ "$code" =~ ^2[0-9][0-9]$ ]]; then
    echo "ready url=$url status=$code"
    exit 0
  fi
  sleep 1
done

echo "timeout url=$url timeout=${timeout}s" >&2
exit 1
