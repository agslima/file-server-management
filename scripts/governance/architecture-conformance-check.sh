#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"
# shellcheck source=scripts/lib/utils.sh
source "${ROOT_DIR}/scripts/lib/utils.sh"

cd "$ROOT_DIR"

cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto

./scripts/generate-doc-artifacts.sh >/tmp/doc_artifacts_generate.log
if ! git diff --quiet -- \
  docs/generated/endpoint-inventory.md \
  docs/generated/route-maturity-matrix.md \
  docs/generated/dashboard-references.md \
  docs/generated/sdk-examples.md \
  docs/generated/policy-schema.md; then
  echo "generated docs are stale; run ./scripts/generate-doc-artifacts.sh" >&2
  git diff -- \
    docs/generated/endpoint-inventory.md \
    docs/generated/route-maturity-matrix.md \
    docs/generated/dashboard-references.md \
    docs/generated/sdk-examples.md \
    docs/generated/policy-schema.md >&2 || true
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

if rg -n "github.com/example/file-engine/internal/adapters/(storage|security|fs|config)" file-engine/internal/handlers file-engine/internal/server --glob "*.go" --glob "!*_test.go" >/tmp/arch_conformance_transport_adapter_hits.txt; then
  echo "transport layer must not import storage/security/fs/config adapters directly:" >&2
  cat /tmp/arch_conformance_transport_adapter_hits.txt >&2
  exit 1
fi

if rg -n "github.com/example/file-engine/internal/(delivery|server|handlers)/" file-engine/internal/services --glob '*.go' >/tmp/arch_conformance_service_transport_hits.txt; then
  echo "service layer must not import transport packages:" >&2
  cat /tmp/arch_conformance_service_transport_hits.txt >&2
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
  'Transport packages (`internal/server/*`, `internal/handlers/*`) avoid direct storage/security adapter imports; queue wiring in `internal/handlers/grpc_handler.go` is the current scoped exception.'
  'Service packages (`internal/services/*`) do not import transport packages.'
)

for item in "${required_boundary_entries[@]}"; do
  if ! contains_literal "$item" "$BOUNDARIES_DOC"; then
    echo "$BOUNDARIES_DOC: missing boundary entry $item"
    exit 1
  fi
done

if rg -n "github.com/example/file-engine/internal/infra/logger" file-engine --glob '*.go' >/tmp/arch_conformance_logger_hits.txt; then
  echo "deprecated logger import detected:" >&2
  cat /tmp/arch_conformance_logger_hits.txt >&2
  exit 1
fi

if [[ -d file-engine/internal/infra/logger ]]; then
  if find file-engine/internal/infra/logger -name '*.go' -print -quit | grep -q .; then
    echo "deprecated package path file-engine/internal/infra/logger contains Go files"
    exit 1
  fi
fi

echo "architecture conformance check passed"
