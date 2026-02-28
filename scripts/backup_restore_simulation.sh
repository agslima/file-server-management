#!/usr/bin/env bash
set -euo pipefail

TS="$(date -u +%Y%m%dT%H%M%SZ)"
ART_DIR="${ART_DIR:-artifacts/backup-restore}"
RTO_DB_RESTORE_TARGET="${RTO_DB_RESTORE_TARGET:-PT30M}"
RPO_DB_TARGET="${RPO_DB_TARGET:-PT24H}"
RTO_STORAGE_RESTORE_TARGET="${RTO_STORAGE_RESTORE_TARGET:-PT45M}"
RPO_STORAGE_TARGET="${RPO_STORAGE_TARGET:-PT24H}"
mkdir -p "$ART_DIR"

echo "[backup] starting local/dev backup simulation"
echo "[backup] RTO/RPO targets db_restore=${RTO_DB_RESTORE_TARGET} db_rpo=${RPO_DB_TARGET} storage_restore=${RTO_STORAGE_RESTORE_TARGET} storage_rpo=${RPO_STORAGE_TARGET}"
docker compose up -d postgres redis file-engine file-engine-worker >/dev/null

echo "[backup] waiting for postgres readiness"
pg_ready_timeout_seconds="${PG_READY_TIMEOUT_SECONDS:-60}"
pg_ready_deadline=$((SECONDS + pg_ready_timeout_seconds))
until docker compose exec -T postgres pg_isready -U fileengine -d fileengine >/dev/null 2>&1; do
  if (( SECONDS >= pg_ready_deadline )); then
    echo "[backup] postgres readiness check failed after ${pg_ready_timeout_seconds}s" >&2
    exit 1
  fi
  sleep 2
done
echo "[backup] postgres is ready"

DB_DUMP="$ART_DIR/db-${TS}.sql"
STORAGE_TAR="$ART_DIR/storage-${TS}.tar.gz"

echo "[backup] step 1/4 postgres dump -> $DB_DUMP"
docker compose exec -T postgres pg_dump -U fileengine fileengine > "$DB_DUMP"

echo "[backup] step 2/4 storage snapshot -> $STORAGE_TAR"
BASE_ROOT="${FILE_BASE_ROOT:-.data/file-engine}"
mkdir -p "$BASE_ROOT"
tar -czf "$STORAGE_TAR" -C "$BASE_ROOT" .

echo "[restore] step 3/4 replay db dump"
cat "$DB_DUMP" | docker compose exec -T postgres psql -U fileengine fileengine >/dev/null

echo "[restore] step 4/4 replay storage snapshot"
: "${BASE_ROOT:?BASE_ROOT must be set}"
[[ "$BASE_ROOT" != "/" ]] || { echo "Refusing to delete root"; exit 1; }
find "$BASE_ROOT" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +
tar -xzf "$STORAGE_TAR" -C "$BASE_ROOT"

echo "BACKUP_RESTORE_SIMULATION_OK dump=$DB_DUMP storage=$STORAGE_TAR"
