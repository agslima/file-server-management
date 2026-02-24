#!/usr/bin/env bash
set -euo pipefail

echo "[drill] db restore and replay"
./scripts/backup_restore_simulation.sh

echo "DB_RESTORE_REPLAY_DRILL_OK"
