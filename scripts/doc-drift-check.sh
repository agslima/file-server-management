#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"

DOC_FILES=(
  "README.md"
  "docs/setup.md"
)

fail=0

check_link() {
  local doc_file="$1"
  local doc_dir="$2"
  local link="$3"

  link="${link%%#*}"
  if [[ -z "$link" ]]; then
    return 0
  fi

  if [[ "$link" == \#* ]]; then
    return 0
  fi

  case "$link" in
    http://*|https://*|mailto:*)
      return 0
      ;;
  esac

  link="${link#<}"
  link="${link%>}"
  link="${link#./}"

  local target
  if [[ "$link" == /* ]]; then
    target="$ROOT_DIR/${link#/}"
  else
    target="$doc_dir/$link"
  fi

  if [[ ! -e "$target" ]]; then
    echo "$doc_file: broken link -> $link"
    fail=1
  fi
}

check_cmd() {
  local doc_file="$1"
  local cmd="$2"
  local name="${cmd##*/}"
  local target="$ROOT_DIR/file-engine/cmd/$name"

  if [[ ! -e "$target" ]]; then
    echo "$doc_file: go run ./cmd/$name not found under file-engine/cmd/"
    fail=1
  fi
}

for doc in "${DOC_FILES[@]}"; do
  doc_path="$ROOT_DIR/$doc"
  if [[ ! -f "$doc_path" ]]; then
    echo "missing doc: $doc"
    fail=1
    continue
  fi

  doc_dir="$(cd -- "$(dirname -- "$doc_path")" >/dev/null 2>&1 && pwd)"

  while IFS= read -r match; do
    link="$(printf '%s' "$match" | sed -E 's/.*\\(([^)]*)\\).*/\\1/')"
    check_link "$doc" "$doc_dir" "$link"
  done < <(grep -oE '\\[[^]]*\\]\\([^)]*\\)' "$doc_path" || true)

  while IFS= read -r cmd; do
    check_cmd "$doc" "$cmd"
  done < <(grep -oE 'go run ./cmd/[A-Za-z0-9_-]+' "$doc_path" || true)
done

if [[ "$fail" -ne 0 ]]; then
  echo "doc drift check failed"
  exit 1
fi

echo "doc drift check passed"
