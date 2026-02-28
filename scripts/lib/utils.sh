#!/usr/bin/env bash

contains_literal() {
  local value="$1"
  local file="$2"
  if command -v rg >/dev/null 2>&1; then
    rg -Fq "$value" "$file"
  else
    grep -Fq "$value" "$file"
  fi
}
