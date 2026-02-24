#!/usr/bin/env bash
set -euo pipefail

TS="$(date -u +%Y%m%dT%H%M%SZ)"
ART_DIR="${ART_DIR:-artifacts/backup-restore}"
mkdir -p "$ART_DIR"

echo "[backup] starting local/dev backup simulation"
docker compose up -d postgres redis file-engine file-engine-worker >/dev/null

DB_DUMP="$ART_DIR/db-${TS}.sql"
STORAGE_TAR="$ART_DIR/storage-${TS}.tar.gz"

echo "[backup] creating postgres dump -> $DB_DUMP"
docker compose exec -T postgres pg_dump -U file_engine file_engine > "$DB_DUMP"

echo "[backup] creating storage snapshot -> $STORAGE_TAR"
BASE_ROOT="${FILE_BASE_ROOT:-.data/file-engine}"
mkdir -p "$BASE_ROOT"
tar -czf "$STORAGE_TAR" -C "$BASE_ROOT" .

echo "[restore] replaying db dump"
cat "$DB_DUMP" | docker compose exec -T postgres psql -U file_engine file_engine >/dev/null

echo "[restore] replaying storage snapshot"
rm -rf "$BASE_ROOT"/*
tar -xzf "$STORAGE_TAR" -C "$BASE_ROOT"

echo "BACKUP_RESTORE_SIMULATION_OK dump=$DB_DUMP storage=$STORAGE_TAR"
