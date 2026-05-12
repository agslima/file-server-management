#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
# shellcheck source=scripts/lib/utils.sh
source "${SCRIPT_DIR}/lib/utils.sh"

required_refs=(
  "README.md:OWNERS"
  "docs/governance.md:OWNERS"
  "docs/branch-protection-mapping.md:OWNERS"
)

required_owner_keys=(
  "domain_reviewers:"
  "auth_authz:"
  "upload_scanner:"
  "audit_sink_dlq:"
  "observability_alerts_drills:"
  "governance_controls:"
  "required_reviewers:"
  "security:"
  "platform:"
  "maintainers:"
)

required_codeowners_scopes=(
  "/file-engine/internal/auth*"
  "/monitoring/**"
  "/observability/**"
  "/docs/capability-ledger.md"
)

[[ -f .github/OWNERS ]] || { echo ".github/OWNERS missing" >&2; exit 1; }
[[ -f .github/codeowners ]] || { echo ".github/codeowners missing" >&2; exit 1; }

for ref in "${required_refs[@]}"; do
  file="${ref%%:*}"
  needle="${ref##*:}"
  [[ -f "$file" ]] || { echo "$file: not found — referenced needle $needle" >&2; exit 1; }
  contains_literal "$needle" "$file" || { echo "$file: missing reference to $needle" >&2; exit 1; }
done

for needle in "${required_owner_keys[@]}"; do
  contains_literal "$needle" .github/OWNERS || { echo ".github/OWNERS missing required key: $needle" >&2; exit 1; }
done

for scope in "${required_codeowners_scopes[@]}"; do
  contains_literal "$scope" .github/codeowners || { echo ".github/codeowners missing required critical scope: $scope" >&2; exit 1; }
done

[[ -f docs/prod-checklist.md ]] || {
  echo "docs/prod-checklist.md not found" >&2
  exit 1
}

contains_literal "new maintainer drill executed" docs/prod-checklist.md || {
  echo "docs/prod-checklist.md missing release gate: new maintainer drill executed" >&2
  exit 1
}
contains_literal "scripts/drills/new_maintainer_operability_drill.sh" docs/prod-checklist.md || {
  echo "docs/prod-checklist.md missing drill script reference" >&2
  exit 1
}

echo "OWNERS_GOVERNANCE_REFERENCE_OK"
