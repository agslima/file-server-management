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


BOUNDARIES_DOC="docs/architecture_boundaries.md"
if [[ ! -f "$BOUNDARIES_DOC" ]]; then
  echo "missing $BOUNDARIES_DOC"
  exit 1
fi

required_boundary_entries=(
  '`internal/logger/*`'
  '`internal/services/*`'
  '`internal/app/*`'
)

for item in "${required_boundary_entries[@]}"; do
  if ! contains_literal "$item" "$BOUNDARIES_DOC"; then
    echo "$BOUNDARIES_DOC: missing boundary entry $item"
    exit 1
  fi
done

if rg -n "github.com/example/file-engine/internal/infra/logger" file-engine --glob '*.go' >/dev/null 2>&1; then
  echo "deprecated logger import detected: internal/infra/logger"
  exit 1
fi

if [[ -d file-engine/internal/infra/logger ]]; then
  if find file-engine/internal/infra/logger -name '*.go' -print -quit | grep -q .; then
    echo "deprecated package path file-engine/internal/infra/logger contains Go files"
    exit 1
  fi
fi
