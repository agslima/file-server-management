#!/usr/bin/env bash
set -euo pipefail

BASE_REF="${1:-origin/main}"

if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "[threat-model] not in git repository" >&2
  exit 1
fi

if ! git rev-parse --verify "$BASE_REF" >/dev/null 2>&1; then
  BASE_REF="HEAD~1"
fi

merge_base="$(git merge-base "$BASE_REF" HEAD 2>/dev/null || true)"
if [[ -z "$merge_base" ]]; then
  merge_base="HEAD~1"
fi

mapfile -t changed < <(git diff --name-only "$merge_base"...HEAD -- \
  'file-engine/internal/handlers/**' \
  'file-engine/internal/authz/**' \
  'file-engine/internal/services/**' \
  'file-engine/internal/server/admin_http.go' \
  'docs/threat-model.md' \
  'docs/dataflow-security-risk-assessment.md' \
  'docs/security-reviewers.md')

if [[ ${#changed[@]} -eq 0 ]]; then
  echo "THREAT_MODEL_DIFF_PROMPT: no boundary-sensitive changes detected between $merge_base and HEAD"
  exit 0
fi

cat <<PROMPT
THREAT_MODEL_DIFF_PROMPT
base=$merge_base
files_changed=${#changed[@]}

Changed boundary-sensitive files:
$(printf ' - %s\n' "${changed[@]}")

Review checklist:
1) Re-evaluate trust boundaries (TB2 service, TB3 queue, TB4 data, TB6 hybrid).
2) Confirm authn/authz invariants still hold:
   - deny-by-default when tenant mapping/ACL context is missing
   - no client/JWT-controlled tenant override at execution boundary
3) Confirm governance controls for policy drift still alert and are auditable.
4) Confirm STRIDE entries + mitigations in docs/threat-model.md match current code paths.
5) Update docs/security-reviewers.md reviewer prompts and escalation guidance when controls change.
6) If risk posture changed, append evidence in docs/dataflow-security-risk-assessment.md.
PROMPT
