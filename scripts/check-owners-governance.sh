#!/usr/bin/env bash
set -euo pipefail

required_refs=(
  "README.md:OWNERS"
  "docs/governance.md:OWNERS"
  "docs/branch-protection-mapping.md:OWNERS"
)

[[ -f .github/OWNERS ]] || { echo ".github/OWNERS missing" >&2; exit 1; }

for ref in "${required_refs[@]}"; do
  file="${ref%%:*}"
  needle="${ref##*:}"
  if ! rg -q "$needle" "$file"; then
    echo "$file: missing reference to $needle" >&2
    exit 1
  fi
done

echo "OWNERS_GOVERNANCE_REFERENCE_OK"
