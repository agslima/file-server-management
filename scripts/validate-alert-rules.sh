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

p = Path('monitoring/alerts/file-engine-alerts.yml')
t = p.read_text()

required_alerts = [
  'FileEngineHighErrorRate',
  'FileEngineQueueLagHigh',
  'FileEngineScanDLQGrowing',
  'FileEngineAuditSinkFailures',
  'FileEngineAuthzDenySpike',
  'scanner_down',
  'scan_queue_backlog',
  'quarantine_time_p95',
  'scan_dlq_growth_rate',
]

for a in required_alerts:
  if f'alert: {a}' not in t:
    print(f'missing alert rule: {a}', file=sys.stderr)
    sys.exit(1)

blocks = {}
for block in re.split(r'\n\s*- alert: ', t)[1:]:
  lines = block.splitlines()
  alert_name = lines[0].strip()
  blocks[alert_name] = block
  for key in ('expr:', 'for:', 'labels:', 'annotations:'):
    if key not in block:
      print(f'alert block missing {key}: {alert_name}', file=sys.stderr)
      sys.exit(1)

forbidden_placeholders = ('TODO', 'TBD', 'placeholder', '<threshold>')
threshold_alerts = ['scanner_down', 'scan_queue_backlog', 'quarantine_time_p95', 'scan_dlq_growth_rate']
for name in threshold_alerts:
  block = blocks[name]
  if any(p in block for p in forbidden_placeholders):
    print(f'alert threshold contains placeholder text: {name}', file=sys.stderr)
    sys.exit(1)

  expr = re.search(r'\n\s*expr:\s*(.+)', block)
  hold = re.search(r'\n\s*for:\s*([0-9]+[smhd])', block)
  if not expr or not hold:
    print(f'alert missing deterministic expr/for threshold: {name}', file=sys.stderr)
    sys.exit(1)

  expr_value = expr.group(1).strip()
  if not re.search(r'(>|==|<)\s*[0-9]+', expr_value):
    print(f'alert expression missing numeric threshold comparator: {name}', file=sys.stderr)
    sys.exit(1)

  if 'threshold:' not in block:
    print(f'alert missing threshold annotation: {name}', file=sys.stderr)
    sys.exit(1)

print('alert rules validation passed')
PY
