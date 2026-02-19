#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"

cd "$ROOT_DIR"

cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto

MATRIX="docs/route-maturity-matrix.md"
APIREF="docs/api-reference.md"

required_matrix_entries=(
  '`POST /v1/folders`'
  '`GET /v1/tasks/{taskId}`'
  '`POST /v1/uploads:initiate`'
  '`POST /v1/uploads/{uploadId}:complete`'
  '`ListObjects`'
  '`DownloadObject`'
)

required_api_methods=(
  '`CreateFolder`'
  '`GetTaskStatus`'
  '`InitiateUpload`'
  '`CompleteUpload`'
)

contains_literal() {
  local value="$1"
  local file="$2"
  if command -v rg >/dev/null 2>&1; then
    rg -Fq "$value" "$file"
  else
    grep -Fq "$value" "$file"
  fi
}

for item in "${required_matrix_entries[@]}"; do
  if ! contains_literal "$item" "$MATRIX"; then
    echo "$MATRIX: missing endpoint inventory entry $item"
    exit 1
  fi
done

for item in "${required_api_methods[@]}"; do
  if ! contains_literal "$item" "$APIREF"; then
    echo "$APIREF: missing canonical method inventory entry $item"
    exit 1
  fi
done

echo "architecture conformance check passed"
