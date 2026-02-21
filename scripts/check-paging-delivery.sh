#!/usr/bin/env bash
set -euo pipefail

receiver_port="${PAGING_RECEIVER_PORT:-19093}"
incident_id="${PAGING_INCIDENT_ID:-paging-drill-001}"
tmp_dir="$(mktemp -d)"
payload_file="$tmp_dir/paging-payload.json"

cleanup() {
  if [[ -n "${receiver_pid:-}" ]] && kill -0 "$receiver_pid" 2>/dev/null; then
    kill "$receiver_pid"
    wait "$receiver_pid" 2>/dev/null || true
  fi
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

PAYLOAD_FILE="$payload_file" RECEIVER_PORT="$receiver_port" python3 - <<'PY' &
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

payload_file = os.environ["PAYLOAD_FILE"]
port = int(os.environ["RECEIVER_PORT"])


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/healthz":
            self.send_response(200)
            self.end_headers()
            self.wfile.write(b"ok")
            return
        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length).decode("utf-8")
        with open(payload_file, "w", encoding="utf-8") as f:
            f.write(body)
        self.send_response(202)
        self.end_headers()
        self.wfile.write(b"ok")

    def log_message(self, format, *args):
        return


server = ThreadingHTTPServer(("127.0.0.1", port), Handler)
server.serve_forever()
PY
receiver_pid=$!

for _ in {1..40}; do
  if curl -fsS "http://127.0.0.1:${receiver_port}/healthz" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "$receiver_pid" 2>/dev/null; then
    echo "paging receiver failed to start" >&2
    exit 1
  fi
  sleep 0.1
done

payload="$(cat <<JSON
{"receiver":"local-paging-webhook","status":"firing","alerts":[{"status":"firing","labels":{"alertname":"FileEngineQueueLagHigh","severity":"critical","incident_id":"${incident_id}"},"annotations":{"summary":"deterministic paging drill"},"startsAt":"2026-01-01T00:00:00Z"}]}
JSON
)"

curl -fsS -X POST "http://127.0.0.1:${receiver_port}/" \
  -H 'Content-Type: application/json' \
  --data "$payload" >/dev/null

for _ in {1..40}; do
  if [[ -f "$payload_file" ]] && rg -q "\"incident_id\":\"${incident_id}\"" "$payload_file"; then
    echo "PAGING_OK incident_id=${incident_id}"
    echo "PAGING_DELIVERED_OK"
    exit 0
  fi
  sleep 0.1
done

echo "paging payload delivery not observed for incident_id=${incident_id}" >&2
exit 1
