#!/usr/bin/env bash
set -euo pipefail

full_run=0
if [[ "${1:-}" == "--full" ]]; then
  full_run=1
fi

if (( full_run == 1 )); then
  ./file-engine/scripts/dev.sh
  ./scripts/drills/production_deployment_hardening.sh
  ./scripts/drills/rotate-secrets-drill.sh --apply
  ./scripts/drills/restore-scan-dlq-drill.sh --apply
else
  ./scripts/drills/rotate-secrets-drill.sh
  ./scripts/drills/restore-scan-dlq-drill.sh
fi

echo "NEW_MAINTAINER_OPERABILITY_DRILL_OK"
