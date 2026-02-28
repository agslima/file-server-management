#!/usr/bin/env bash
set -euo pipefail

# Manual operator command (scheduled-ready) to produce audit-ready artifacts.
# Required env: TOKEN
# Optional env: BASE_URL, TENANT_ID, REPORT_MONTH, OUT_DIR, ACCESS_REVIEW_SIGNING_KEY

: "${TOKEN:?set TOKEN to an admin bearer token}"
: "${BASE_URL:=http://localhost:8080}"
: "${REPORT_MONTH:=$(date -u +%Y-%m)}"
: "${OUT_DIR:=artifacts/compliance/access-review/${REPORT_MONTH}}"

mkdir -p "$OUT_DIR"

json_out="$OUT_DIR/access-review.json"
csv_out="$OUT_DIR/access-review.csv"
manifest_out="$OUT_DIR/manifest.txt"
sig_out="$OUT_DIR/access-review.sig"

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

signature_mode="none"
if [[ -n "${ACCESS_REVIEW_SIGNING_KEY:-}" ]]; then
  signature_mode="hmac-sha256"
  ACCESS_REVIEW_SIGNING_KEY="$ACCESS_REVIEW_SIGNING_KEY" \
  python3 - <<'PY' "$json_out" "$sig_out"
import hashlib, hmac, sys
import os
src, dst = sys.argv[1], sys.argv[2]
key = os.environ["ACCESS_REVIEW_SIGNING_KEY"]
with open(src, 'rb') as f:
    data = f.read()
digest = hmac.new(key.encode('utf-8'), data, hashlib.sha256).hexdigest()
with open(dst, 'w', encoding='utf-8') as f:
    f.write(digest + '\n')
PY
fi

{
  echo "schema=compliance.access_review_export.v1"
  echo "report_month=$REPORT_MONTH"
  echo "tenant_filter=${TENANT_ID:-}"
  echo "generated_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "artifact_json=$json_out"
  echo "artifact_csv=$csv_out"
  echo "signature_mode=$signature_mode"
  if [[ "$signature_mode" != "none" ]]; then
    echo "artifact_signature=$sig_out"
  fi
} > "$manifest_out"

echo "ACCESS_REVIEW_REPORT_OK out_dir=$OUT_DIR"
echo "ACCESS_REVIEW_REPORT_JSON=$json_out"
echo "ACCESS_REVIEW_REPORT_CSV=$csv_out"
if [[ "$signature_mode" != "none" ]]; then
  echo "ACCESS_REVIEW_REPORT_SIG=$sig_out"
fi
