#!/usr/bin/env bash
set -euo pipefail

BACKEND_URL="${BACKEND_URL:-http://localhost:8081}"
run_nonce="$(date +%s)-$$"
init_clean_key="init-clean-${run_nonce}"
complete_clean_key="complete-clean-${run_nonce}"
init_dirty_key="init-dirty-${run_nonce}"
complete_dirty_key="complete-dirty-${run_nonce}"

tmp_init="$(mktemp -t upload-init.XXXXXX.json)"
tmp_complete="$(mktemp -t upload-complete.XXXXXX.json)"
tmp_dirty="$(mktemp -t upload-dirty.XXXXXX.json)"
trap 'rm -f "$tmp_init" "$tmp_complete" "$tmp_dirty"' EXIT

init_status="$(curl -sS -o "$tmp_init" -w '%{http_code}' -X POST "${BACKEND_URL}/uploads/initiate" \
  -H 'Content-Type: application/json' \
  -H "X-Idempotency-Key: ${init_clean_key}" \
  -H 'X-Request-Id: req-upload-clean-1' \
  -d '{"path":"/tenants/acme/docs/report.txt"}')"

if [[ "$init_status" != "200" ]]; then
  echo "clean initiate failed status=${init_status}" >&2
  cat "$tmp_init" >&2 || true
  exit 1
fi

upload_id="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["upload_id"])' "$tmp_init")"

chunk_status="$(curl -sS -o /dev/null -w '%{http_code}' -X PUT "${BACKEND_URL}/uploads/${upload_id}/chunk?offset=0" \
  -H 'Content-Type: application/octet-stream' \
  --data-binary 'hello-clean')"
if [[ "$chunk_status" != "202" ]]; then
  echo "clean chunk failed status=${chunk_status}" >&2
  exit 1
fi

complete_status="$(curl -sS -o "$tmp_complete" -w '%{http_code}' -X POST "${BACKEND_URL}/uploads/${upload_id}/complete" \
  -H "X-Idempotency-Key: ${complete_clean_key}" \
  -H 'X-Request-Id: req-upload-clean-1')"
if [[ "$complete_status" != "200" ]]; then
  echo "clean complete failed status=${complete_status}" >&2
  cat "$tmp_complete" >&2 || true
  exit 1
fi
python3 - <<'PY' "$tmp_complete"
import json,sys
body=json.load(open(sys.argv[1]))
assert body.get('scan_status') == 'clean', f"unexpected clean scan_status={body.get('scan_status')}"
PY

# idempotent replay should be deterministic
replay_status="$(curl -sS -o "$tmp_complete" -w '%{http_code}' -X POST "${BACKEND_URL}/uploads/${upload_id}/complete" \
  -H "X-Idempotency-Key: ${complete_clean_key}" \
  -H 'X-Request-Id: req-upload-clean-1')"
if [[ "$replay_status" != "200" ]]; then
  echo "clean complete replay failed status=${replay_status}" >&2
  cat "$tmp_complete" >&2 || true
  exit 1
fi

# dirty flow
init_dirty_status="$(curl -sS -o "$tmp_dirty" -w '%{http_code}' -X POST "${BACKEND_URL}/uploads/initiate" \
  -H 'Content-Type: application/json' \
  -H "X-Idempotency-Key: ${init_dirty_key}" \
  -H 'X-Request-Id: req-upload-dirty-1' \
  -d '{"path":"/tenants/acme/docs/eicar.txt"}')"
if [[ "$init_dirty_status" != "200" ]]; then
  echo "dirty initiate failed status=${init_dirty_status}" >&2
  cat "$tmp_dirty" >&2 || true
  exit 1
fi

dirty_upload_id="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["upload_id"])' "$tmp_dirty")"

dirty_chunk_status="$(curl -sS -o /dev/null -w '%{http_code}' -X PUT "${BACKEND_URL}/uploads/${dirty_upload_id}/chunk?offset=0" \
  -H 'Content-Type: application/octet-stream' \
  --data-binary 'virus')"
if [[ "$dirty_chunk_status" != "202" ]]; then
  echo "dirty chunk failed status=${dirty_chunk_status}" >&2
  exit 1
fi

dirty_complete_status="$(curl -sS -o "$tmp_dirty" -w '%{http_code}' -X POST "${BACKEND_URL}/uploads/${dirty_upload_id}/complete" \
  -H "X-Idempotency-Key: ${complete_dirty_key}" \
  -H 'X-Request-Id: req-upload-dirty-1')"
if [[ "$dirty_complete_status" != "403" ]]; then
  echo "dirty complete expected 403 got status=${dirty_complete_status}" >&2
  cat "$tmp_dirty" >&2 || true
  exit 1
fi
python3 - <<'PY' "$tmp_dirty"
import json,sys
err=(json.load(open(sys.argv[1])) or {}).get('error') or {}
assert err.get('reason') in ('complete_failed','upload_failed'), f"unexpected reason={err.get('reason')}"
PY

echo "UPLOAD_OK clean_upload_id=${upload_id} dirty_upload_id=${dirty_upload_id}"
