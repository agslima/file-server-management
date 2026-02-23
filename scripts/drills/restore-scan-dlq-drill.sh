#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
scan_dlq_script="$repo_root/file-engine/scripts/scan_dlq.sh"

mode="dry-run"
if [[ "${1:-}" == "--apply" ]]; then
  mode="apply"
fi

echo "RESTORE_SCAN_DLQ_DRILL mode=${mode}"
if [[ "$mode" == "apply" ]]; then
  if [[ -z "${TOKEN:-}" ]]; then
    echo "TOKEN is required for --apply" >&2
    exit 1
  fi
  "$scan_dlq_script" "$TOKEN"
fi

echo "1) review DLQ backlog and select retry candidates"
echo "2) retry DLQ entries after root-cause mitigation"
echo "3) resolve residual poisoned entries with incident note"
echo "RESTORE_SCAN_DLQ_DRILL_OK"
