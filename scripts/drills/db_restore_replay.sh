#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "[drill] db restore and replay: start"
echo "[drill] executing backup/restore simulation script"
"${SCRIPT_DIR}/../backup_restore_simulation.sh"
echo "[drill] db restore and replay: completed"
echo "DB_RESTORE_REPLAY_DRILL_OK"
