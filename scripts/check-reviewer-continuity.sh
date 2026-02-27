#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GITHUB_EVENT_PATH:-}" || ! -f "${GITHUB_EVENT_PATH:-}" ]]; then
  echo "continuity check skipped: GITHUB_EVENT_PATH is not set"
  exit 0
fi

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "GITHUB_TOKEN is required for reviewer continuity check" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi

repo="$(jq -r '.repository.full_name // empty' "$GITHUB_EVENT_PATH")"
pr_number="$(jq -r '.pull_request.number // empty' "$GITHUB_EVENT_PATH")"

if [[ -z "$repo" || -z "$pr_number" ]]; then
  echo "continuity check skipped: not a pull_request event"
  exit 0
fi

api="https://api.github.com/repos/${repo}"
accept='Accept: application/vnd.github+json'
auth="Authorization: Bearer ${GITHUB_TOKEN}"

files_json="$(curl -fsSL -H "$accept" -H "$auth" "${api}/pulls/${pr_number}/files?per_page=100")"
reviews_json="$(curl -fsSL -H "$accept" -H "$auth" "${api}/pulls/${pr_number}/reviews?per_page=100")"

changed_paths="$(jq -r '.[].filename' <<<"$files_json")"
approved_reviewers="$(jq -r '[.[] | select(.state=="APPROVED") | .user.login] | unique[]' <<<"$reviews_json" || true)"

security_required=false
platform_required=false
maintainer_required=false

if rg -q '^file-engine/internal/auth' <<<"$changed_paths"; then
  security_required=true
fi
if rg -q '^(monitoring/|observability/)' <<<"$changed_paths"; then
  platform_required=true
fi
if rg -q '^docs/capability-ledger\.md$' <<<"$changed_paths"; then
  maintainer_required=true
fi

security_reviewers=(agslima)
platform_reviewers=(agslima)
maintainers=(agslima)

# contains_approved checks whether any of the provided reviewer usernames exist in the newline-separated string of approved reviewers and returns exit status 0 if a match is found, 1 otherwise.
contains_approved() {
  local approved="$1"
  shift
  for reviewer in "$@"; do
    if rg -qx "$reviewer" <<<"$approved"; then
      return 0
    fi
  done
  return 1
}

errors=()
if [[ "$security_required" == true ]] && ! contains_approved "$approved_reviewers" "${security_reviewers[@]}"; then
  errors+=("security reviewer approval required for /file-engine/internal/auth* changes")
fi
if [[ "$platform_required" == true ]] && ! contains_approved "$approved_reviewers" "${platform_reviewers[@]}"; then
  errors+=("platform reviewer approval required for /monitoring/* or /observability/* changes")
fi
if [[ "$maintainer_required" == true ]] && ! contains_approved "$approved_reviewers" "${maintainers[@]}"; then
  errors+=("maintainer approval required when docs/capability-ledger.md changes")
fi

if (( ${#errors[@]} > 0 )); then
  printf 'REVIEWER_CONTINUITY_FAIL\n' >&2
  printf '%s\n' "${errors[@]}" >&2
  exit 1
fi

echo "REVIEWER_CONTINUITY_OK"
