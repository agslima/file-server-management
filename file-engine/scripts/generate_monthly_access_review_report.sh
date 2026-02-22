#!/usr/bin/env bash
set -euo pipefail

# Manual operator command (scheduled-ready) to produce audit-ready artifacts.
# Required env: TOKEN
# Optional env: BASE_URL, TENANT_ID, REPORT_MONTH, OUT_DIR

: "${TOKEN:?set TOKEN to an admin bearer token}"
: "${BASE_URL:=http://localhost:8080}"
: "${REPORT_MONTH:=$(date -u +%Y-%m)}"
: "${OUT_DIR:=artifacts/compliance/access-review/${REPORT_MONTH}}"

mkdir -p "$OUT_DIR"

json_out="$OUT_DIR/access-review.json"
csv_out="$OUT_DIR/access-review.csv"
manifest_out="$OUT_DIR/manifest.txt"

BASE_URL="$BASE_URL" TOKEN="$TOKEN" TENANT_ID="${TENANT_ID:-}" REPORT_MONTH="$REPORT_MONTH" OUTPUT_PATH="$json_out" \
  "$(dirname "$0")/export_access_review.sh"

python3 - <<'PY' "$json_out" "$csv_out"
import csv, json, sys
src, dst = sys.argv[1], sys.argv[2]
with open(src, encoding='utf-8') as f:
    payload = json.load(f)
rows = payload.get('memberships', [])
fields = ['tenant_id', 'user_id', 'email', 'role_id', 'last_read_at', 'last_write_at', 'last_access']
with open(dst, 'w', newline='', encoding='utf-8') as f:
    w = csv.DictWriter(f, fieldnames=fields)
    w.writeheader()
    for r in rows:
        w.writerow({k: r.get(k, '') for k in fields})
PY

{
  echo "schema=compliance.access_review_export.v1"
  echo "report_month=$REPORT_MONTH"
  echo "tenant_filter=${TENANT_ID:-}"
  echo "generated_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "artifact_json=$json_out"
  echo "artifact_csv=$csv_out"
} > "$manifest_out"

echo "ACCESS_REVIEW_REPORT_OK out_dir=$OUT_DIR"
echo "ACCESS_REVIEW_REPORT_JSON=$json_out"
echo "ACCESS_REVIEW_REPORT_CSV=$csv_out"
