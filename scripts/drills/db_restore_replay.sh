#!/usr/bin/env bash
set -euo pipefail

echo "[drill] db restore and replay: start"
echo "[drill] executing backup/restore simulation script"
./scripts/backup_restore_simulation.sh
echo "[drill] db restore and replay: completed"
echo "DB_RESTORE_REPLAY_DRILL_OK"
