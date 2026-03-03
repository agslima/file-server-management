#!/usr/bin/env bash
set -euo pipefail

# Release-cadence deterministic sustainability snapshot.

report_file="${1:-artifacts/sustainability-metrics.md}"
owners_file="${OWNERS_FILE:-.github/OWNERS}"

checks=(
  "./scripts/doc-drift-check.sh"
  "./scripts/validate-alert-rules.sh"
  "./scripts/check-malware-runbook.sh"
)

csv_contains() {
  local csv="${1:-}"
  local needle="${2:-}"

  [[ -n "$csv" && -n "$needle" ]] || return 1
  local entry
  IFS=',' read -r -a entries <<<"$csv"
  for entry in "${entries[@]}"; do
    if [[ "$entry" == "$needle" ]]; then
      return 0
    fi
  done
  return 1
}

csv_add_unique() {
  local csv="${1:-}"
  local item="${2:-}"

  if [[ -z "$item" ]]; then
    printf '%s' "$csv"
    return
  fi

  if csv_contains "$csv" "$item"; then
    printf '%s' "$csv"
    return
  fi

  if [[ -z "$csv" ]]; then
    printf '%s' "$item"
  else
    printf '%s,%s' "$csv" "$item"
  fi
}

csv_has_distinct_entry() {
  local primary_csv="${1:-}"
  local backup_csv="${2:-}"

  [[ -n "$backup_csv" ]] || return 1

  local backup
  IFS=',' read -r -a backups <<<"$backup_csv"
  for backup in "${backups[@]}"; do
    [[ -n "$backup" ]] || continue
    if ! csv_contains "$primary_csv" "$backup"; then
      return 0
    fi
  done
  return 1
}

parse_domain_reviewers() {
  awk '
    BEGIN {
      in_domain_reviewers = 0
      domain = ""
      section = ""
    }
    /^domain_reviewers:[[:space:]]*$/ {
      in_domain_reviewers = 1
      next
    }
    in_domain_reviewers && /^[^[:space:]]/ {
      in_domain_reviewers = 0
    }
    !in_domain_reviewers {
      next
    }
    /^  [a-z0-9_]+:[[:space:]]*$/ {
      domain = $1
      sub(/:$/, "", domain)
      section = ""
      next
    }
    /^    primary:[[:space:]]*$/ {
      section = "primary"
      next
    }
    /^    backups:[[:space:]]*$/ {
      section = "backups"
      next
    }
    /^      -[[:space:]]+/ {
      if (domain != "" && section != "") {
        reviewer = $0
        sub(/^      -[[:space:]]+/, "", reviewer)
        if (reviewer != "") {
          print domain "|" section "|" reviewer
        }
      }
    }
  ' "$owners_file"
}

passed=0
failed=0
declare -A check_status=()
for check in "${checks[@]}"; do
  if bash -lc "$check" >/dev/null 2>&1; then
    check_status["$check"]="pass"
    ((passed+=1))
  else
    check_status["$check"]="fail"
    ((failed+=1))
  fi
done

total=$((passed + failed))
if (( total == 0 )); then
  baseline_gate_pass_rate="0.00"
  baseline_gate_failure_rate="0.00"
else
  baseline_gate_pass_rate=$(awk -v p="$passed" -v t="$total" 'BEGIN { printf "%.2f", (p/t)*100 }')
  baseline_gate_failure_rate=$(awk -v f="$failed" -v t="$total" 'BEGIN { printf "%.2f", (f/t)*100 }')
fi

doc_drift_check="./scripts/doc-drift-check.sh"
doc_drift_status="${check_status[$doc_drift_check]:-fail}"
if [[ "$doc_drift_status" == "pass" ]]; then
  doc_drift_failure_rate="0.00"
else
  doc_drift_failure_rate="100.00"
fi

[[ -f "$owners_file" ]] || {
  echo "$owners_file missing" >&2
  exit 1
}

declare -A domains=()
declare -A primary_by_domain=()
declare -A backup_by_domain=()

while IFS='|' read -r domain section reviewer; do
  [[ -n "$domain" && -n "$section" && -n "$reviewer" ]] || continue
  domains["$domain"]=1
  if [[ "$section" == "primary" ]]; then
    primary_by_domain["$domain"]="$(csv_add_unique "${primary_by_domain[$domain]:-}" "$reviewer")"
  elif [[ "$section" == "backups" ]]; then
    backup_by_domain["$domain"]="$(csv_add_unique "${backup_by_domain[$domain]:-}" "$reviewer")"
  fi
done < <(parse_domain_reviewers)

critical_domains_total=${#domains[@]}
critical_domains_with_backup=0
critical_domains_with_distinct_backup=0
distinct_backup_people_csv=""
distinct_backup_people_count=0

for domain in "${!domains[@]}"; do
  primaries="${primary_by_domain[$domain]:-}"
  backups="${backup_by_domain[$domain]:-}"

  if [[ -n "$backups" ]]; then
    ((critical_domains_with_backup+=1))
    IFS=',' read -r -a backup_reviewers <<<"$backups"
    for reviewer in "${backup_reviewers[@]}"; do
      distinct_backup_people_csv="$(csv_add_unique "$distinct_backup_people_csv" "$reviewer")"
    done
  fi

  if csv_has_distinct_entry "$primaries" "$backups"; then
    ((critical_domains_with_distinct_backup+=1))
  fi
done

if [[ -n "$distinct_backup_people_csv" ]]; then
  IFS=',' read -r -a distinct_backup_people <<<"$distinct_backup_people_csv"
  distinct_backup_people_count=${#distinct_backup_people[@]}
fi

ownership_coverage_ratio=$(awk -v covered="$critical_domains_with_backup" -v total="$critical_domains_total" 'BEGIN { if (total==0) {print "0.00"} else { printf "%.2f", covered/total } }')
ownership_distinct_backup_ratio=$(awk -v covered="$critical_domains_with_distinct_backup" -v total="$critical_domains_total" 'BEGIN { if (total==0) {print "0.00"} else { printf "%.2f", covered/total } }')

mkdir -p "$(dirname "$report_file")"
cat >"$report_file" <<EOF
# Sustainability Metrics Report

- Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
- Source script: \`scripts/sustainability-metrics.sh\`

| Metric | Value |
| :-- | --: |
| Baseline gate pass rate (%) | $baseline_gate_pass_rate |
| Baseline gate failure rate (%) | $baseline_gate_failure_rate |
| Doc drift check status | $doc_drift_status |
| Doc drift failure rate (%) | $doc_drift_failure_rate |
| Critical domains with backup | $critical_domains_with_backup/$critical_domains_total |
| Ownership coverage ratio | $ownership_coverage_ratio |
| Critical domains with distinct backup reviewer | $critical_domains_with_distinct_backup/$critical_domains_total |
| Ownership distinct backup ratio | $ownership_distinct_backup_ratio |
| Distinct backup people count | $distinct_backup_people_count |
| Checks passed | $passed/$total |

## Release notes snippet

\`Sustainability: baseline pass ${baseline_gate_pass_rate}% (fail ${baseline_gate_failure_rate}%), doc drift ${doc_drift_status}, ownership coverage ${ownership_coverage_ratio}, distinct backup coverage ${ownership_distinct_backup_ratio}.\`
EOF

echo "SUSTAINABILITY_METRICS_REPORT"
echo "baseline_gate_pass_rate_percent=${baseline_gate_pass_rate}"
echo "baseline_gate_failure_rate_percent=${baseline_gate_failure_rate}"
echo "doc_drift_check_status=${doc_drift_status}"
echo "doc_drift_failure_rate_percent=${doc_drift_failure_rate}"
echo "ownership_coverage_ratio=${ownership_coverage_ratio}"
echo "ownership_distinct_backup_ratio=${ownership_distinct_backup_ratio}"
echo "critical_domains_total=${critical_domains_total}"
echo "critical_domains_with_backup=${critical_domains_with_backup}"
echo "critical_domains_with_distinct_backup=${critical_domains_with_distinct_backup}"
echo "distinct_backup_people_count=${distinct_backup_people_count}"
echo "markdown_report=${report_file}"
echo "SUSTAINABILITY_METRICS_OK"
