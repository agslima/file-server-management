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

fetch_paginated_array() {
  local endpoint="$1"
  local page=1
  local merged='[]'

  while :; do
    local page_json
    page_json="$(curl -fsSL -H "$accept" -H "$auth" "${endpoint}&page=${page}")"

    if [[ "$(jq -r 'type' <<<"$page_json")" != "array" ]]; then
      echo "reviewer continuity check failed: non-array response for ${endpoint}" >&2
      exit 1
    fi

    merged="$(jq -c --argjson current "$merged" --argjson next "$page_json" '$current + $next' <<<"null")"
    local count
    count="$(jq 'length' <<<"$page_json")"
    if (( count < 100 )); then
      break
    fi
    page=$((page + 1))
  done

  echo "$merged"
}

files_json="$(fetch_paginated_array "${api}/pulls/${pr_number}/files?per_page=100")"
reviews_json="$(fetch_paginated_array "${api}/pulls/${pr_number}/reviews?per_page=100")"

changed_paths="$(jq -r '.[].filename' <<<"$files_json")"
approved_reviewers="$(jq -r '
  sort_by(.submitted_at // "")
  | group_by(.user.login)
  | map(last)
  | map(select(.state == "APPROVED"))
  | .[].user.login
' <<<"$reviews_json" || true)"

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
