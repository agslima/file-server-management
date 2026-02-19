#!/usr/bin/env bash
set -euo pipefail

backend_url="${BACKEND_URL:-http://localhost:8081}"
parent_path="${PARENT_PATH:-tenants/acme}"
folder_name="${FOLDER_NAME:-vs001-$(date +%s)}"
requested_by="${REQUESTED_BY:-vs001-e2e@example.com}"
timeout_seconds="${TASK_TIMEOUT_SECONDS:-90}"
create_timeout_seconds="${CREATE_TIMEOUT_SECONDS:-60}"

payload="$(cat <<JSON
{"path":"${parent_path}","folderName":"${folder_name}","requestedBy":"${requested_by}"}
JSON
)"

create_deadline=$((SECONDS + create_timeout_seconds))
task_id=""
create_response=""
while (( SECONDS < create_deadline )); do
  create_response="$(curl -sS -X POST "${backend_url}/folders" -H 'Content-Type: application/json' -d "$payload" || true)"
  task_id="$(python3 - <<'PY' "$create_response"
import json,sys
try:
    data=json.loads(sys.argv[1])
except Exception:
    print("")
    raise SystemExit(0)
if isinstance(data, dict):
    print(data.get("taskId") or data.get("task_id") or "")
else:
    print("")
PY
)"
  if [[ -n "$task_id" ]]; then
    break
  fi
  sleep 2
done

if [[ -z "$task_id" ]]; then
  echo "failed_to_parse_task_id response=${create_response}" >&2
  exit 1
fi

echo "task_id=${task_id}"

deadline=$((SECONDS + timeout_seconds))
last_status="unknown"
while (( SECONDS < deadline )); do
  status_response="$(curl -sS "${backend_url}/tasks/${task_id}")"
  last_status="$(python3 - <<'PY' "$status_response"
import json,sys
try:
    data=json.loads(sys.argv[1])
except Exception:
    print("unknown")
    raise SystemExit(0)
if isinstance(data, dict):
    print(data.get("status") or "unknown")
else:
    print("unknown")
PY
)"
  echo "task_status=${last_status}"
  if [[ "$last_status" == "success" ]]; then
    break
  fi
  if [[ "$last_status" == "failed" ]]; then
    echo "task_failed response=${status_response}" >&2
    exit 1
  fi
  sleep 2
done

if [[ "$last_status" != "success" ]]; then
  echo "task_timeout task_id=${task_id} last_status=${last_status}" >&2
  exit 1
fi

if docker compose exec -T file-engine test -d "/mnt/files/${parent_path}/${folder_name}"; then
  echo "folder_exists=true"
else
  echo "folder_exists=false"
  exit 1
fi

echo "[PASS] CL-020"
echo "E2E_OK"
