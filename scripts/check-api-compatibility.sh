#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/.." >/dev/null 2>&1 && pwd)"

pushd "${ROOT_DIR}/file-engine" >/dev/null

go test ./internal/server -run "TestCompatibilityReadyzGolden|TestCompatibilityAuthzDenyGolden|TestCompatibilityUploadLifecycleGolden|TestCompatibilityUploadThrottledGolden|TestCompatibilityGovernanceDeleteRetentionBlockGolden" -v
popd >/dev/null

# Compatibility policy enforcement for /v1 changes in PR context.
if [[ -n "${GITHUB_BASE_REF:-}" ]]; then
  if ! git fetch --depth=1 origin "${GITHUB_BASE_REF}" >/dev/null 2>&1; then
    echo "api compatibility gate failed: unable to fetch origin/${GITHUB_BASE_REF}" >&2
    exit 1
  fi
  base_ref="origin/${GITHUB_BASE_REF}"
  if ! git rev-parse --verify "$base_ref" >/dev/null 2>&1; then
    echo "api compatibility gate failed: invalid base ref ${base_ref}" >&2
    exit 1
  fi
  changed="$(git diff --name-only "${base_ref}...HEAD")"
  if echo "$changed" | grep -E -q "^(file-engine/internal/server/(server.go|upload_http.go|admin_http.go)|file-engine/api/proto/fileengine.proto|file-engine/proto/fileengine.proto)$"; then
      has_version_update=0
      if echo "$changed" | grep -E -q "^docs/api-versioning-policy.md$"; then
        has_version_update=1
      fi
      has_docs_update=0
      if echo "$changed" | grep -E -q "^(README.md|docs/client-sdks.md|docs/route-maturity-matrix.md|docs/generated/endpoint-inventory.md)$"; then
        has_docs_update=1
      fi
      if [[ $has_version_update -eq 0 || $has_docs_update -eq 0 ]]; then
        echo "breaking-change policy gate: /v1 surface changed without required docs updates (api-versioning-policy + consumer docs)" >&2
        exit 1
      fi
  fi
fi

echo "API_COMPATIBILITY_OK"
