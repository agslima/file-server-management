#!/usr/bin/env bash
set -euo pipefail

# Release-cadence deterministic sustainability snapshot.

report_file="${1:-artifacts/sustainability-metrics.md}"

checks=(
  "./scripts/doc-drift-check.sh"
  "./scripts/validate-alert-rules.sh"
  "./scripts/check-malware-runbook.sh"
)

passed=0
failed=0
for check in "${checks[@]}"; do
  if bash -lc "$check" >/dev/null 2>&1; then
    ((passed+=1))
  else
    ((failed+=1))
  fi
done

total=$((passed + failed))
if (( total == 0 )); then
  baseline_gate_pass_rate="0.00"
else
  baseline_gate_pass_rate=$(awk -v p="$passed" -v t="$total" 'BEGIN { printf "%.2f", (p/t)*100 }')
fi

doc_drift_failures=$failed
doc_drift_failure_rate=$(awk -v f="$doc_drift_failures" -v t="$total" 'BEGIN { if (t==0) {print "0.00"} else { printf "%.2f", (f/t)*100 } }')

ownership_file="docs/ownership-backup-matrix.md"
required_domains=4
backup_count=$(awk -F'|' '/^\|/{gsub(/^ +| +$/, "", $4); if ($2 ~ /Security|Platform|Backend control-plane|Data plane/ && $4 !~ /^ *$/) c++} END {print c+0}' "$ownership_file")
ownership_coverage_ratio=$(awk -v b="$backup_count" -v r="$required_domains" 'BEGIN { if (r==0) {print "0.00"} else { printf "%.2f", b/r } }')

mkdir -p "$(dirname "$report_file")"
cat >"$report_file" <<EOF
# Sustainability Metrics Report

- Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
- Source script: \`scripts/sustainability-metrics.sh\`

| Metric | Value |
| :-- | --: |
| Baseline gate pass rate (%) | $baseline_gate_pass_rate |
| Doc drift failure rate (%) | $doc_drift_failure_rate |
| Ownership coverage ratio | $ownership_coverage_ratio |
| Checks passed | $passed/$total |

## Release notes snippet

\`Sustainability: baseline pass rate ${baseline_gate_pass_rate}%, doc drift failure rate ${doc_drift_failure_rate}%, ownership coverage ${ownership_coverage_ratio}.\`
EOF

echo "SUSTAINABILITY_METRICS_REPORT"
echo "baseline_gate_pass_rate_percent=${baseline_gate_pass_rate}"
echo "doc_drift_failure_rate_percent=${doc_drift_failure_rate}"
echo "ownership_coverage_ratio=${ownership_coverage_ratio}"
echo "markdown_report=${report_file}"
echo "SUSTAINABILITY_METRICS_OK"
