#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"

REQUIRED_DOCS=(
  "README.md"
  "docs/capability-ledger.md"
  "docs/roadmap.md"
  "docs/governance.md"
  "docs/project-alignment-review.md"
  "docs/architecture.md"
  "docs/route-maturity-matrix.md"
)

fail=0

for doc in "${REQUIRED_DOCS[@]}"; do
  path="$ROOT_DIR/$doc"
  if [[ ! -f "$path" ]]; then
    echo "missing required doc: $doc"
    fail=1
    continue
  fi

  if ! rg -q '^\[//\]: # \(owner: .+\)$' "$path"; then
    echo "$doc: missing ownership metadata line: [//]: # (owner: <role>)"
    fail=1
  fi

  if ! rg -q '^\[//\]: # \(review_cadence: .+\)$' "$path"; then
    echo "$doc: missing review cadence metadata line: [//]: # (review_cadence: <cadence>)"
    fail=1
  fi

  if ! rg -q '^\[//\]: # \(last_reviewed: [0-9]{4}-[0-9]{2}-[0-9]{2}\)$' "$path"; then
    echo "$doc: missing last_reviewed metadata line (YYYY-MM-DD)"
    fail=1
  fi
done

if ! rg -q '^## Quarterly alignment review cadence$' "$ROOT_DIR/docs/governance.md"; then
  echo "docs/governance.md: missing quarterly alignment cadence section"
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "doc ownership metadata check passed"
