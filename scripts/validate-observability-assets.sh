#!/usr/bin/env bash
set -euo pipefail

./scripts/validate-alert-rules.sh
bash -n scripts/drills/observability_incident_drill.sh
python3 - <<'PY'
import json
from pathlib import Path
p=Path('monitoring/dashboards/file-engine-golden-signals.json')
obj=json.loads(p.read_text())
assert isinstance(obj.get('panels'), list) and len(obj['panels']) >= 5
print('observability assets validation passed')
PY
