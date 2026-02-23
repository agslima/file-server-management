#!/usr/bin/env bash
set -euo pipefail

required_refs=(
  "README.md:OWNERS"
  "docs/governance.md:OWNERS"
  "docs/branch-protection-mapping.md:OWNERS"
)

[[ -f .github/OWNERS ]] || { echo ".github/OWNERS missing" >&2; exit 1; }

# contains_literal checks whether the specified file contains the given literal string.
contains_literal() {
  local value="$1"
  local file="$2"

  if command -v rg >/dev/null 2>&1; then
    rg -Fq "$value" "$file"
  else
    grep -Fq "$value" "$file"
  fi
}

for ref in "${required_refs[@]}"; do
  file="${ref%%:*}"
  needle="${ref##*:}"
  if ! contains_literal "$needle" "$file"; then
    echo "$file: missing reference to $needle" >&2
    exit 1
  fi
done

echo "OWNERS_GOVERNANCE_REFERENCE_OK"
