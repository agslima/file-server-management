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

for item in "${required_matrix_entries[@]}"; do
  if ! rg -Fq "$item" "$MATRIX"; then
    echo "$MATRIX: missing endpoint inventory entry $item"
    exit 1
  fi
done

for item in "${required_api_methods[@]}"; do
  if ! rg -Fq "$item" "$APIREF"; then
    echo "$APIREF: missing canonical method inventory entry $item"
    exit 1
  fi
done

echo "architecture conformance check passed"
