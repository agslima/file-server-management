#!/usr/bin/env bash
set -euo pipefail

# ----------------------------
# Config (env-overridable)
# ----------------------------
backend_url="${BACKEND_URL:-http://localhost:8081}"

parent_path="${PARENT_PATH:-tenants/acme}"
folder_name="${FOLDER_NAME:-vs001-$(date +%s)}"
requested_by="${REQUESTED_BY:-vs001-e2e@example.com}"

# Polling / timeouts
startup_timeout_seconds="${STARTUP_TIMEOUT_SECONDS:-90}"
create_timeout_seconds="${CREATE_TIMEOUT_SECONDS:-60}"
task_timeout_seconds="${TASK_TIMEOUT_SECONDS:-90}"
poll_interval_seconds="${POLL_INTERVAL_SECONDS:-2}"

# Docker verification
verify_in_engine="${VERIFY_IN_ENGINE:-1}"             # 1=true, 0=false
engine_service="${ENGINE_SERVICE:-file-engine}"       # docker compose service name
engine_path_root="${ENGINE_PATH_ROOT:-/mnt/files}"    # where local storage is mounted in engine container

compose() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
  else
    echo "docker compose not found" >&2
    exit 127
  fi
}

# ----------------------------
# Helpers
# ----------------------------
json_get_task_id() {
  python3 - "$1" <<'PY'
import json,sys
raw=sys.argv[1]
try:
    data=json.loads(raw)
except Exception:
    print("")
    raise SystemExit(0)
if isinstance(data, dict):
    print(data.get("taskId") or data.get("task_id") or "")
else:
    print("")
PY
}

json_get_status() {
  python3 - "$1" <<'PY'
import json,sys
raw=sys.argv[1]
try:
    data=json.loads(raw)
except Exception:
    print("unknown")
    raise SystemExit(0)
if isinstance(data, dict):
    print(data.get("status") or "unknown")
else:
    print("unknown")
PY
}

curl_json() {
  # Usage: curl_json METHOD URL JSON_BODY(optional)
  local method="$1"
  local url="$2"
  local body="${3:-}"

  # Print: "<body>\n<http_code>"
  if [[ -n "$body" ]]; then
    curl -sS -X "$method" "$url" \
      -H 'Content-Type: application/json' \
      -d "$body" \
      -w $'\n%{http_code}\n'
  else
    curl -sS -X "$method" "$url" -w $'\n%{http_code}\n'
  fi
}

wait_for_backend() {
  local deadline=$((SECONDS + startup_timeout_seconds))
  while (( SECONDS < deadline )); do
    if curl -sS "${backend_url}/healthz" >/dev/null 2>&1; then
      return 0
    fi
    if curl -sS "${backend_url}/" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$poll_interval_seconds"
  done
  echo "backend_not_ready url=${backend_url} timeout=${startup_timeout_seconds}s" >&2
  return 1
}

# ----------------------------
# Start
# ----------------------------
echo "backend_url=${backend_url}"
echo "parent_path=${parent_path}"
echo "folder_name=${folder_name}"

wait_for_backend

payload="$(cat <<JSON
{"path":"${parent_path}","folderName":"${folder_name}","requestedBy":"${requested_by}"}
JSON
)"

# ----------------------------
# Create folder -> get task_id
# ----------------------------
create_deadline=$((SECONDS + create_timeout_seconds))
task_id=""
create_body=""
create_code=""

while (( SECONDS < create_deadline )); do
  resp="$(curl_json POST "${backend_url}/folders" "$payload" || true)"
  create_code="$(printf '%s' "$resp" | tail -n 1)"
  create_body="$(printf '%s' "$resp" | sed '$d')"

  task_id="$(json_get_task_id "$create_body")"

  if [[ -n "$task_id" ]]; then
    break
  fi

  echo "create_retry http_code=${create_code} body=${create_body}"
  sleep "$poll_interval_seconds"
done

if [[ -z "$task_id" ]]; then
  echo "failed_to_parse_task_id http_code=${create_code} body=${create_body}" >&2
  exit 1
fi

echo "task_id=${task_id}"

# ----------------------------
# Poll task status
# ----------------------------
deadline=$((SECONDS + task_timeout_seconds))
last_status="unknown"
status_body=""
status_code=""

while (( SECONDS < deadline )); do
  resp="$(curl_json GET "${backend_url}/tasks/${task_id}" || true)"
  status_code="$(printf '%s' "$resp" | tail -n 1)"
  status_body="$(printf '%s' "$resp" | sed '$d')"

  last_status="$(json_get_status "$status_body")"

  echo "task_status=${last_status}"

  if [[ "$last_status" == "success" ]]; then
    break
  fi

  if [[ "$last_status" == "failed" ]]; then
    echo "task_failed http_code=${status_code} body=${status_body}" >&2
    exit 1
  fi

  sleep "$poll_interval_seconds"
done

if [[ "$last_status" != "success" ]]; then
  echo "task_timeout task_id=${task_id} last_status=${last_status} http_code=${status_code} body=${status_body}" >&2
  exit 1
fi

# ----------------------------
# Verify folder exists in file-engine container (optional)
# ----------------------------
if [[ "${verify_in_engine}" == "1" ]]; then
  folder_path="${engine_path_root}/${parent_path}/${folder_name}"

  if compose exec -T "${engine_service}" test -d "${folder_path}"; then
    echo "folder_exists=true"
  else
    echo "folder_exists=false path=${folder_path}" >&2
    exit 1
  fi
else
  echo "folder_exists=skipped"
fi

echo "[PASS] CL-020"
echo "E2E_OK"
