#!/usr/bin/env bash
set -euo pipefail

rules_file="monitoring/alerts/file-engine-alerts.yml"

if [[ ! -f "$rules_file" ]]; then
  echo "missing alert rules file: $rules_file" >&2
  exit 1
fi

python3 - <<'PY'
from pathlib import Path
import re, sys
p=Path('monitoring/alerts/file-engine-alerts.yml')
t=p.read_text()
required_alerts=[
  'FileEngineHighErrorRate',
  'FileEngineQueueLagHigh',
  'FileEngineScanDLQGrowing',
  'FileEngineAuditSinkFailures',
  'FileEngineAuthzDenySpike',
]
for a in required_alerts:
  if f'alert: {a}' not in t:
    print(f'missing alert rule: {a}', file=sys.stderr)
    sys.exit(1)
for block in re.split(r'\n\s*- alert: ', t)[1:]:
  for key in ('expr:', 'for:', 'labels:', 'annotations:'):
    if key not in block:
      print(f'alert block missing {key}', file=sys.stderr)
      sys.exit(1)
print('alert rules validation passed')
PY
