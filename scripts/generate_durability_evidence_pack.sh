#!/usr/bin/env bash
set -euo pipefail

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_DIR="${OUT_DIR:-artifacts/durability-evidence/${TS}}"
mkdir -p "$OUT_DIR"

SUMMARY="$OUT_DIR/README.md"
CONFIG="$OUT_DIR/config-snapshot.env"
TRANSCRIPTS="$OUT_DIR/drill-transcript-pointers.txt"
ACCESS_REVIEW_POINTER="$OUT_DIR/access-review-pointer.txt"

echo "[evidence] generating durability evidence pack at $OUT_DIR"

echo "[evidence] writing config snapshot"
{
  echo "generated_at=${TS}"
  echo "rto_db_restore_target=${RTO_DB_RESTORE_TARGET:-PT30M}"
  echo "rpo_db_target=${RPO_DB_TARGET:-PT24H}"
  echo "rto_storage_restore_target=${RTO_STORAGE_RESTORE_TARGET:-PT45M}"
  echo "rpo_storage_target=${RPO_STORAGE_TARGET:-PT24H}"
  echo "integrity_sample_size_default=${INTEGRITY_SAMPLE_SIZE:-25}"
  echo "integrity_failure_threshold_default=${INTEGRITY_FAILURE_THRESHOLD:-0}"
} > "$CONFIG"

echo "[evidence] collecting drill transcript pointers"
{
  echo "scripts/drills/db_restore_replay.sh"
  echo "scripts/drills/storage_corruption_drill.sh"
  echo "scripts/drills/audit_sink_catchup_drill.sh"
  echo "scripts/backup_restore_simulation.sh"
  echo "scripts/integrity_verify_job.sh"
} > "$TRANSCRIPTS"

echo "[evidence] collecting access review command pointer"
{
  echo "Generate report: file-engine/scripts/generate_monthly_access_review_report.sh"
  echo "Export data: file-engine/scripts/export_access_review.sh"
} > "$ACCESS_REVIEW_POINTER"

cat > "$SUMMARY" <<MD
# Durability Evidence Pack

Generated at: ${TS}

Included artifacts:
- Config snapshot: ${CONFIG}
- Drill transcript pointers: ${TRANSCRIPTS}
- Access review pointer: ${ACCESS_REVIEW_POINTER}

Deterministic evidence generation marker:
- DURABILITY_EVIDENCE_PACK_OK out_dir=${OUT_DIR}
MD

echo "DURABILITY_EVIDENCE_PACK_OK out_dir=${OUT_DIR}"
