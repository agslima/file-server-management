#!/usr/bin/env bash
set -euo pipefail

./scripts/validate-alert-rules.sh
./scripts/drills/sink_down.sh
./scripts/drills/scanner_down.sh
./scripts/drills/otel_exporter_down.sh

echo "PRODUCTION_DEPLOYMENT_HARDENING_DRILLS_OK"
