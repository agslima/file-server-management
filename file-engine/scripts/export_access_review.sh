#!/usr/bin/env bash
set -euo pipefail

# Stable compliance export contract (JSON)
# Environment:
# - TOKEN (required): admin bearer token
# - BASE_URL (optional): file-engine base URL (default http://localhost:8080)
# - TENANT_ID (optional): limit export scope to one tenant
# - REPORT_MONTH (optional): YYYY-MM tag in artifact payload
# - OUTPUT_PATH (optional): write JSON to file instead of stdout

: "${TOKEN:?set TOKEN to an admin bearer token}"
: "${BASE_URL:=http://localhost:8080}"

urlencode() {
  python3 -c 'import sys, urllib.parse; print(urllib.parse.quote(sys.argv[1], safe=""))' "$1"
}

query=""
if [[ -n "${TENANT_ID:-}" ]]; then
  ENCODED_TENANT_ID="$(urlencode "$TENANT_ID")"
  query="?tenant_id=${ENCODED_TENANT_ID}"
fi

payload="$(curl -fsS "${BASE_URL}/admin/v1/access-review${query}" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json")"

export RAW_PAYLOAD="$payload"
export REPORT_MONTH="${REPORT_MONTH:-$(date -u +%Y-%m)}"
export BASE_URL
python3 - <<'PY'
import json, os, sys
from datetime import datetime, timezone

raw = os.environ.get("RAW_PAYLOAD", "")
report_month = os.environ.get("REPORT_MONTH", "")
base_url = os.environ.get("BASE_URL", "")

try:
    data = json.loads(raw)
except Exception as exc:
    print(f"invalid JSON payload from access-review endpoint: {exc}", file=sys.stderr)
    sys.exit(1)

rows = data.get("memberships") if isinstance(data, dict) else None
if not isinstance(rows, list):
    print("unexpected access-review response: missing memberships[]", file=sys.stderr)
    sys.exit(1)

schema = {
    "schema_version": "compliance.access_review_export.v1",
    "report_month": report_month,
    "generated_at": datetime.now(timezone.utc).isoformat(),
    "source": {
        "system": "file-engine",
        "endpoint": f"{base_url}/admin/v1/access-review",
        "source_schema_version": data.get("schema_version", "unknown") if isinstance(data, dict) else "unknown",
    },
    "tenant_filter": data.get("tenant_filter", "") if isinstance(data, dict) else "",
    "memberships": rows,
}

rendered = json.dumps(schema, indent=2, sort_keys=True)
out = os.environ.get("OUTPUT_PATH", "").strip()
if out:
    with open(out, "w", encoding="utf-8") as f:
        f.write(rendered + "\n")
else:
    print(rendered)
PY
