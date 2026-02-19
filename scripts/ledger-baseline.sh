#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"


cleanup_postgres() {
  docker compose down -v >/dev/null 2>&1 || true
}

start_postgres() {
  trap cleanup_postgres EXIT
  docker compose up -d postgres >/dev/null
  local deadline=$((SECONDS + 60))
  until docker compose exec -T postgres pg_isready -U fileengine -d fileengine >/dev/null 2>&1; do
    if (( SECONDS >= deadline )); then
      echo "postgres did not become ready in time" >&2
      exit 1
    fi
    sleep 1
  done
}

run_and_require_no_skip() {
  local claim="$1"
  shift

  local output
  output="$($@ 2>&1)"
  printf '%s\n' "$output"

  if printf '%s\n' "$output" | grep -Eq '(^--- SKIP:|\bSKIP\b|Skipped:[[:space:]]*[1-9])'; then
    echo "Baseline must not skip tests." >&2
    echo "[FAIL] ${claim} skipped tests detected" >&2
    exit 1
  fi

  echo "[PASS] ${claim}"
}

echo "[ledger-baseline] CL-001 proto mirror sync"
cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto
echo "[PASS] CL-001"

start_postgres

echo "[ledger-baseline] CL-002 file-engine baseline modules"
cd file-engine
run_and_require_no_skip CL-002 go test ./internal/config ./internal/logger ./internal/worker -v

echo "[ledger-baseline] CL-003 async create-folder integration"
run_and_require_no_skip CL-003 go test ./tests/integration -run TestAsyncCreateFolderFlow -v

echo "[ledger-baseline] CL-022 read/list/download audit coverage"
run_and_require_no_skip CL-022 go test ./tests/integration -run TestAuditEventsEmittedForReadListDownload -v

echo "[ledger-baseline] CL-025 staged upload + atomic promote"
run_and_require_no_skip CL-025 go test ./tests/integration -run TestStagedUploadAtomicPromote -v

echo "[ledger-baseline] CL-033 malware gate dirty blocks promote"
run_and_require_no_skip CL-033 go test ./tests/integration -run TestUploadScanGateDirtyPreventsPromotion -v
cd ..

echo "[ledger-baseline] CL-011 doc drift"
./scripts/doc-drift-check.sh
echo "[PASS] CL-011"

echo "[ledger-baseline] CL-008 + CL-018 backend scaffold checks"
cd backend
composer validate --strict
php -l app/Http/Controllers/FolderController.php
php -l app/Http/Controllers/TaskController.php
php -l app/Services/FileEngineService.php
echo "[PASS] CL-008"
echo "[PASS] CL-018"

echo "[ledger-baseline] CL-031 backend smoke"
run_and_require_no_skip CL-031 ./scripts/smoke.sh
cd ..

echo "[ledger-baseline] completed"
echo "LEDGER_BASELINE_OK"
