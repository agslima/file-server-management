#!/usr/bin/env bash
set -euo pipefail

./scripts/validate-alert-rules.sh
bash -n scripts/check-otel-connectivity.sh
bash -n scripts/drills/observability_incident_drill.sh
bash -n scripts/drills/production_deployment_hardening.sh
bash -n scripts/drills/sink_down.sh
bash -n scripts/drills/scanner_down.sh
bash -n scripts/drills/otel_exporter_down.sh
python3 - <<'PY'
import json
from pathlib import Path
p=Path('monitoring/dashboards/file-engine-golden-signals.json')
obj=json.loads(p.read_text())
assert isinstance(obj.get('panels'), list) and len(obj['panels']) >= 5
print('observability assets validation passed')
PY
