#!/usr/bin/env bash
set -euo pipefail

BACKEND_URL="${BACKEND_URL:-http://localhost:8081}"
run_nonce="$(date +%s)-$$"

src="/tenants/acme/docs/mutation-${run_nonce}.txt"
dst="/tenants/acme/docs/mutation-${run_nonce}-renamed.txt"

echo "[1/4] seed upload"
init_id="$(curl -sS -X POST "${BACKEND_URL}/uploads/initiate" -H 'Content-Type: application/json' -d "{\"path\":\"${src}\"}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["upload_id"])')"
curl -sS -o /dev/null -X PUT "${BACKEND_URL}/uploads/${init_id}/chunk?offset=0" --data-binary 'clean-data'
curl -sS -o /dev/null -X POST "${BACKEND_URL}/uploads/${init_id}/complete"

echo "[2/4] move/rename"
move_code="$(curl -sS -o /tmp/move.json -w '%{http_code}' -X POST "${BACKEND_URL}/objects/move" -H 'Content-Type: application/json' -d "{\"sourcePath\":\"${src}\",\"destinationPath\":\"${dst}\"}")"
[[ "$move_code" == "200" ]] || { cat /tmp/move.json; exit 1; }

echo "[3/4] delete with governance enforcement"
del_code="$(curl -sS -o /tmp/delete.json -w '%{http_code}' -X POST "${BACKEND_URL}/objects/delete" -H 'Content-Type: application/json' -d "{\"path\":\"${dst}\"}")"
[[ "$del_code" =~ ^(200|204)$ ]] || { cat /tmp/delete.json; exit 1; }

echo "[4/4] restore from quarantine"
dirty="/tenants/acme/docs/mutation-${run_nonce}-eicar.txt"
qid="$(curl -sS -X POST "${BACKEND_URL}/uploads/initiate" -H 'Content-Type: application/json' -d "{\"path\":\"${dirty}\"}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["upload_id"])')"
curl -sS -o /dev/null -X PUT "${BACKEND_URL}/uploads/${qid}/chunk?offset=0" --data-binary 'dummy'
curl -sS -o /dev/null -X POST "${BACKEND_URL}/uploads/${qid}/complete" || true
restore_code="$(curl -sS -o /tmp/restore.json -w '%{http_code}' -X POST "${BACKEND_URL}/objects/restore" -H 'Content-Type: application/json' -d "{\"path\":\"${dirty}\",\"forceReprocess\":false}")"
[[ "$restore_code" == "200" ]] || { cat /tmp/restore.json; exit 1; }

echo "MUTATION_SURFACE_OK"
