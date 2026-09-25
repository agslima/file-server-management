#!/usr/bin/env bash
set -euo pipefail

owners_file="${OWNERS_FILE:-.github/OWNERS}"
errors=()

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
  local reviewer
  IFS=',' read -r -a backups <<<"$backup_csv"
  for reviewer in "${backups[@]}"; do
    [[ -n "$reviewer" ]] || continue
    if ! csv_contains "$primary_csv" "$reviewer"; then
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

parse_required_reviewers() {
  awk '
    BEGIN {
      in_required = 0
      group = ""
    }
    /^required_reviewers:[[:space:]]*$/ {
      in_required = 1
      next
    }
    in_required && /^[^[:space:]]/ {
      in_required = 0
    }
    !in_required {
      next
    }
    /^  [a-z0-9_]+:[[:space:]]*$/ {
      group = $1
      sub(/:$/, "", group)
      next
    }
    /^    -[[:space:]]+/ {
      if (group != "") {
        reviewer = $0
        sub(/^    -[[:space:]]+/, "", reviewer)
        if (reviewer != "") {
          print group "|" reviewer
        }
      }
    }
  ' "$owners_file"
}

select_approved_reviewer() {
  local approved="$1"
  shift

  local reviewer
  for reviewer in "$@"; do
    [[ -n "$reviewer" ]] || continue
    if grep -F -x -q "$reviewer" <<<"$approved"; then
      printf '%s' "$reviewer"
      return 0
    fi
  done
  return 1
}

fail_if_errors() {
  if (( ${#errors[@]} > 0 )); then
    printf 'REVIEWER_CONTINUITY_FAIL\n' >&2
    printf '%s\n' "${errors[@]}" >&2
    exit 1
  fi
}

[[ -f "$owners_file" ]] || {
  echo "$owners_file missing" >&2
  exit 1
}

declare -A domains=()
declare -A primary_by_domain=()
declare -A backup_by_domain=()
declare -A required_reviewers=()

while IFS='|' read -r domain section reviewer; do
  [[ -n "$domain" && -n "$section" && -n "$reviewer" ]] || continue
  domains["$domain"]=1
  if [[ "$section" == "primary" ]]; then
    primary_by_domain["$domain"]="$(csv_add_unique "${primary_by_domain[$domain]:-}" "$reviewer")"
  elif [[ "$section" == "backups" ]]; then
    backup_by_domain["$domain"]="$(csv_add_unique "${backup_by_domain[$domain]:-}" "$reviewer")"
  fi
done < <(parse_domain_reviewers)

while IFS='|' read -r group reviewer; do
  [[ -n "$group" && -n "$reviewer" ]] || continue
  required_reviewers["$group"]="$(csv_add_unique "${required_reviewers[$group]:-}" "$reviewer")"
done < <(parse_required_reviewers)

if (( ${#domains[@]} == 0 )); then
  errors+=("domain_reviewers must define at least one critical domain")
fi

for domain in "${!domains[@]}"; do
  primaries="${primary_by_domain[$domain]:-}"
  backups="${backup_by_domain[$domain]:-}"

  if [[ -z "$primaries" ]]; then
    errors+=("domain_reviewers.${domain} must define at least one primary reviewer")
  fi
  if [[ -z "$backups" ]]; then
    errors+=("domain_reviewers.${domain} must define at least one backup reviewer")
  fi
  if [[ -n "$primaries" && -n "$backups" ]] && ! csv_has_distinct_entry "$primaries" "$backups"; then
    errors+=("domain_reviewers.${domain} must include a backup reviewer distinct from primary reviewers")
  fi
done

required_groups=(security platform maintainers)
all_required_reviewers_csv=""
for group in "${required_groups[@]}"; do
  group_reviewers_csv="${required_reviewers[$group]:-}"
  if [[ -z "$group_reviewers_csv" ]]; then
    errors+=("required_reviewers.${group} must define at least one reviewer login")
    continue
  fi

  group_reviewer_count=0
  IFS=',' read -r -a group_reviewers <<<"$group_reviewers_csv"
  for reviewer in "${group_reviewers[@]}"; do
    [[ -n "$reviewer" ]] || continue
    all_required_reviewers_csv="$(csv_add_unique "$all_required_reviewers_csv" "$reviewer")"
    ((group_reviewer_count+=1))
  done

  if (( group_reviewer_count < 2 )); then
    errors+=("required_reviewers.${group} must include at least two distinct reviewer logins")
  fi
done

all_required_reviewer_count=0
if [[ -n "$all_required_reviewers_csv" ]]; then
  IFS=',' read -r -a all_required_reviewers <<<"$all_required_reviewers_csv"
  all_required_reviewer_count=${#all_required_reviewers[@]}
fi

if (( all_required_reviewer_count < 2 )); then
  errors+=("required_reviewers must include at least two distinct reviewer logins across security/platform/maintainers")
fi

fail_if_errors

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

if grep -E -q '^file-engine/internal/auth' <<<"$changed_paths"; then
  security_required=true
fi
if grep -E -q '^(monitoring/|observability/|file-engine/internal/observability/)' <<<"$changed_paths"; then
  platform_required=true
fi
if grep -E -q '^docs/capability-ledger\.md$' <<<"$changed_paths"; then
  maintainer_required=true
fi

required_groups_for_pr=()
if [[ "$security_required" == true ]]; then
  required_groups_for_pr+=(security)
fi
if [[ "$platform_required" == true ]]; then
  required_groups_for_pr+=(platform)
fi
if [[ "$maintainer_required" == true ]]; then
  required_groups_for_pr+=(maintainers)
fi

declare -A selected_reviewer_by_group=()
for group in "${required_groups_for_pr[@]}"; do
  reviewers_csv="${required_reviewers[$group]:-}"
  IFS=',' read -r -a candidates <<<"$reviewers_csv"
  if ! selected="$(select_approved_reviewer "$approved_reviewers" "${candidates[@]}")"; then
    case "$group" in
      security)
        errors+=("security reviewer approval required for /file-engine/internal/auth* changes")
        ;;
      platform)
        errors+=("platform reviewer approval required for /monitoring/*, /observability/*, or /file-engine/internal/observability/* changes")
        ;;
      maintainers)
        errors+=("maintainer approval required when docs/capability-ledger.md changes")
        ;;
    esac
    continue
  fi
  selected_reviewer_by_group["$group"]="$selected"
done

if (( ${#required_groups_for_pr[@]} > 1 )); then
  distinct_approvers_csv=""
  approval_assignments=""
  for group in "${required_groups_for_pr[@]}"; do
    reviewer="${selected_reviewer_by_group[$group]:-}"
    [[ -n "$reviewer" ]] || continue
    distinct_approvers_csv="$(csv_add_unique "$distinct_approvers_csv" "$reviewer")"
    if [[ -z "$approval_assignments" ]]; then
      approval_assignments="${group}=${reviewer}"
    else
      approval_assignments="${approval_assignments}, ${group}=${reviewer}"
    fi
  done

  distinct_approver_count=0
  if [[ -n "$distinct_approvers_csv" ]]; then
    IFS=',' read -r -a distinct_approvers <<<"$distinct_approvers_csv"
    distinct_approver_count=${#distinct_approvers[@]}
  fi

  if (( distinct_approver_count < ${#required_groups_for_pr[@]} )); then
    errors+=("distinct reviewer continuity required across groups (${required_groups_for_pr[*]}): ${approval_assignments}")
  fi
fi

fail_if_errors
echo "REVIEWER_CONTINUITY_OK"
