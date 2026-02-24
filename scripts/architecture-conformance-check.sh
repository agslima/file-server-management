#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"

cd "$ROOT_DIR"

cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto

./scripts/generate-doc-artifacts.sh >/tmp/doc_artifacts_generate.log
if ! git diff --quiet -- docs/generated/endpoint-inventory.md docs/generated/route-maturity-matrix.md docs/generated/dashboard-references.md; then
  echo "generated docs are stale; run ./scripts/generate-doc-artifacts.sh" >&2
  git diff -- docs/generated/endpoint-inventory.md docs/generated/route-maturity-matrix.md docs/generated/dashboard-references.md >&2 || true
  exit 1
fi

if [[ -f file-engine/cmd/main.go ]]; then
  echo "legacy dual-path entrypoint file-engine/cmd/main.go must not exist"
  exit 1
fi

if rg -n "database/sql|jackc/pgx|gorm.io|sqlx" file-engine/internal/handlers file-engine/internal/server --glob '*.go' >/tmp/arch_conformance_db_hits.txt; then
  echo "direct DB-layer access import detected in handlers/server:" >&2
  cat /tmp/arch_conformance_db_hits.txt >&2
  exit 1
fi


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

if command -v rg >/dev/null 2>&1; then
  if rg -n "github.com/example/file-engine/internal/infra/logger" file-engine --glob '*.go' >/tmp/arch_conformance_logger_hits.txt; then
    echo "deprecated logger import detected:" >&2
    cat /tmp/arch_conformance_logger_hits.txt >&2
    exit 1
  fi
else
  if grep -R -n --include='*.go' "github.com/example/file-engine/internal/infra/logger" file-engine >/tmp/arch_conformance_logger_hits.txt; then
    echo "deprecated logger import detected:" >&2
    cat /tmp/arch_conformance_logger_hits.txt >&2
    exit 1
  fi
fi

if [[ -d file-engine/internal/infra/logger ]]; then
  if find file-engine/internal/infra/logger -name '*.go' -print -quit | grep -q .; then
    echo "deprecated package path file-engine/internal/infra/logger contains Go files"
    exit 1
  fi
fi

echo "architecture conformance check passed"
