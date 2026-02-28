#!/usr/bin/env bash
set -euo pipefail

: "${TOKEN:?set TOKEN to an admin bearer token}"
: "${TENANT_ID:?set TENANT_ID to target tenant}"
: "${BASE_URL:=http://localhost:8080}"
: "${REPORT_MONTH:=$(date -u +%Y-%m)}"

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required to URL-encode tenant path/query parameters" >&2
  exit 2
fi

encode_uri_component() {
  local value="$1"
  python3 - "$value" <<'PY'
import sys
from urllib.parse import quote
value = sys.argv[1]
if not value:
    raise SystemExit(2)
print(quote(value, safe=''))
PY
}

ENCODED_TENANT_ID="$(encode_uri_component "$TENANT_ID")" || {
  echo "failed to URL-encode TENANT_ID" >&2
  exit 2
}

: "${OUT_DIR:=artifacts/compliance/tenants/${ENCODED_TENANT_ID}/${REPORT_MONTH}}"
mkdir -p "$OUT_DIR"

echo "[compliance] generating tenant packet for tenant=${TENANT_ID} month=${REPORT_MONTH}"

ACCESS_DIR="$OUT_DIR/access-review"
mkdir -p "$ACCESS_DIR"
TOKEN="$TOKEN" BASE_URL="$BASE_URL" TENANT_ID="$TENANT_ID" REPORT_MONTH="$REPORT_MONTH" OUT_DIR="$ACCESS_DIR" \
  ./file-engine/scripts/generate_monthly_access_review_report.sh

echo "[compliance] snapshotting tenant evidence endpoint"
EVIDENCE_JSON="$OUT_DIR/tenant-evidence.json"
curl --fail --silent --show-error \
  --connect-timeout 5 --max-time 30 \
  --retry 3 --retry-delay 2 --retry-connrefused --retry-max-time 30 \
  "${BASE_URL}/admin/tenants/${ENCODED_TENANT_ID}/evidence" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json" > "$EVIDENCE_JSON"

echo "[compliance] snapshotting effective governance policy"
EFFECTIVE_JSON="$OUT_DIR/effective-policy.json"
curl --fail --silent --show-error \
  --connect-timeout 5 --max-time 30 \
  --retry 3 --retry-delay 2 --retry-connrefused --retry-max-time 30 \
  "${BASE_URL}/admin/v1/governance:effective?tenant_id=${ENCODED_TENANT_ID}" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json" > "$EFFECTIVE_JSON"

echo "[compliance] triggering and snapshotting drift check"
DRIFT_JSON="$OUT_DIR/drift-status.json"
curl --fail --silent --show-error \
  --connect-timeout 5 --max-time 30 \
  --retry 3 --retry-delay 2 --retry-connrefused --retry-max-time 30 \
  -X POST "${BASE_URL}/admin/v1/governance:drift-check" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json" > "$DRIFT_JSON"

MANIFEST="$OUT_DIR/manifest.txt"
{
  echo "schema=tenant_compliance_packet.v1"
  echo "tenant_id=${TENANT_ID}"
  echo "report_month=${REPORT_MONTH}"
  echo "generated_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "access_review_dir=${ACCESS_DIR}"
  echo "tenant_evidence_json=${EVIDENCE_JSON}"
  echo "effective_policy_json=${EFFECTIVE_JSON}"
  echo "drift_status_json=${DRIFT_JSON}"
  echo "drill_pointer_db_restore=scripts/drills/db_restore_replay.sh"
  echo "drill_pointer_storage_corruption=scripts/drills/storage_corruption_drill.sh"
  echo "drill_pointer_audit_sink=scripts/drills/audit_sink_catchup_drill.sh"
} > "$MANIFEST"

echo "TENANT_COMPLIANCE_PACKET_OK out_dir=${OUT_DIR}"
