#!/usr/bin/env bash
set -euo pipefail

: "${TOKEN:?set TOKEN to an admin bearer token}"
: "${TENANT_ID:?set TENANT_ID to target tenant}"
: "${BASE_URL:=http://localhost:8080}"
: "${REPORT_MONTH:=$(date -u +%Y-%m)}"
: "${OUT_DIR:=artifacts/compliance/tenants/${TENANT_ID}/${REPORT_MONTH}}"

mkdir -p "$OUT_DIR"

echo "[compliance] generating tenant packet for tenant=${TENANT_ID} month=${REPORT_MONTH}"

ACCESS_DIR="$OUT_DIR/access-review"
mkdir -p "$ACCESS_DIR"
TOKEN="$TOKEN" BASE_URL="$BASE_URL" TENANT_ID="$TENANT_ID" REPORT_MONTH="$REPORT_MONTH" OUT_DIR="$ACCESS_DIR" \
  ./file-engine/scripts/generate_monthly_access_review_report.sh

echo "[compliance] snapshotting tenant evidence endpoint"
EVIDENCE_JSON="$OUT_DIR/tenant-evidence.json"
curl -fsS "${BASE_URL}/admin/tenants/${TENANT_ID}/evidence" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json" > "$EVIDENCE_JSON"

echo "[compliance] snapshotting effective governance policy"
EFFECTIVE_JSON="$OUT_DIR/effective-policy.json"
curl -fsS "${BASE_URL}/admin/v1/governance:effective?tenant_id=${TENANT_ID}" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json" > "$EFFECTIVE_JSON"

echo "[compliance] triggering and snapshotting drift check"
DRIFT_JSON="$OUT_DIR/drift-status.json"
curl -fsS -X POST "${BASE_URL}/admin/v1/governance:drift-check" \
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
