#!/usr/bin/env bash
# scripts/ledger-baseline.sh
#
# Ledger Baseline Gate (adapted)
# - Strict, deterministic, CI-friendly
# - Unique docker compose project per run
# - Clear, grouped output
# - Debug dumps on failure
# - Supports modes:
#     LEDGER_MODE=fast (default): curated PR gate
#     LEDGER_MODE=full: heavier suite (nightly/manual)

set -Eeuo pipefail

LEDGER_MODE="${LEDGER_MODE:-fast}"
LEDGER_TIMEOUT_SECONDS="${LEDGER_TIMEOUT_SECONDS:-900}"

# Compose isolation: avoid cross-job collisions.
if [[ -n "${GITHUB_RUN_ID:-}" ]]; then
  default_project_name="ledger-baseline-${GITHUB_RUN_ID}"
else
  default_project_name="ledger-baseline-local-${RANDOM}"
fi
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-$default_project_name}"

compose() {
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
  else
    echo "[fatal] docker compose not found" >&2
    exit 127
  fi
}

ts() { date +"%Y-%m-%dT%H:%M:%S%z"; }

log() { echo "[$(ts)] $*"; }

die() { echo "[fatal] $*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

assert_no_skip_markers() {
  local claim="$1"
  local output_file="$2"

  if grep -Eq '(^--- SKIP:|(^SKIP$)|(^SKIP[[:space:]])|(^[[:space:]]*SKIP:[[:space:]]))' "$output_file"; then
    echo "[FAIL] ${claim}: test skip detected" >&2
    echo "------- skip context (last 80 lines) -------" >&2
    tail -n 80 "$output_file" >&2 || true
    echo "-------------------------------------------" >&2
    exit 1
  fi
}

run_claim() {
  local claim="$1"
  shift
  local cmd=("$@")

  log "[run] ${claim}: ${cmd[*]}"
  local tmp
  tmp="$(mktemp)"

  if ! ("${cmd[@]}") 2>&1 | tee "$tmp"; then
    echo "[FAIL] ${claim}" >&2
    echo "------- last 120 lines -------" >&2
    tail -n 120 "$tmp" >&2 || true
    echo "------------------------------" >&2
    rm -f "$tmp"
    exit 1
  fi

  assert_no_skip_markers "$claim" "$tmp"
  rm -f "$tmp"
  echo "[PASS] ${claim}"
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

dump_debug() {
  echo "================ DEBUG DUMP ================"
  echo "[debug] mode=${LEDGER_MODE}"
  echo "[debug] timeout=${LEDGER_TIMEOUT_SECONDS}"
  echo "[debug] COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME}"
  echo "[debug] pwd=$(pwd)"
  echo "[debug] git sha=$(git rev-parse --short HEAD 2>/dev/null || true)"
  echo

  if command -v docker >/dev/null 2>&1; then
    echo "[debug] docker version"
    docker version || true
    echo
  fi

  echo "[debug] docker compose ps"
  compose ps || true
  echo

  echo "[debug] docker compose logs (tail=250)"
  compose logs --no-color --tail=250 postgres 2>/dev/null || true
  compose logs --no-color --tail=250 file-engine 2>/dev/null || true
  compose logs --no-color --tail=250 worker 2>/dev/null || true
  compose logs --no-color --tail=250 backend 2>/dev/null || true
  echo "============================================"
}

cleanup() {
  log "[cleanup] docker compose down -v"
  compose down -v >/dev/null 2>&1 || true
}

trap dump_debug ERR
trap cleanup EXIT

require_cmd git
require_cmd bash
require_cmd go
require_cmd php
require_cmd composer
require_cmd awk
require_cmd sed
require_cmd grep
require_cmd tail
require_cmd mktemp

if ! command -v curl >/dev/null 2>&1; then
  log "[warn] curl not found; some E2E scripts may fail if they depend on curl"
fi

log "[start] ledger baseline (mode=${LEDGER_MODE})"

log "=== CL-001: Proto + gateway generation is in sync ==="
run_claim "CL-001" bash -lc 'cd file-engine && make proto && git diff --exit-code -- api/proto proto pkg/generated'

log "=== CL-002: Go modules baseline ==="
run_claim "CL-002" bash -lc 'cd file-engine && go mod tidy && git diff --exit-code -- go.mod go.sum'

log "=== Backend baseline: CL-008 + CL-018 + CL-031 ==="
run_claim "CL-008" bash -lc 'cd backend && composer validate --strict'
run_claim "CL-018" bash -lc 'cd backend && php -l app/Http/Controllers/FolderController.php && php -l app/Http/Controllers/TaskController.php && php -l app/Services/FileEngineService.php'
run_claim "CL-031" bash -lc 'cd backend && ./scripts/smoke.sh'

log "=== Infra: Postgres up (isolated compose project) ==="
compose up -d postgres

log "=== CL-003: Async create-folder integration ==="
run_claim "CL-003" bash -lc 'cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v'

log "=== CL-022: Audit read/list/download coverage ==="
run_claim "CL-022" bash -lc 'cd file-engine && go test ./tests/integration -run TestAuditEventsEmittedForReadListDownload -v'

log "=== CL-032: Audit append-only enforcement ==="
run_claim "CL-032" bash -lc 'cd file-engine && go test ./tests/integration -run TestAuditEventsAppendOnlyEnforced -v'

log "=== CL-025: Upload staging + atomic promote ==="
run_claim "CL-025" bash -lc 'cd file-engine && go test ./tests/integration -run TestStagedUploadAtomicPromote -v'

log "=== CL-033: Upload scan gate ==="
run_claim "CL-033" bash -lc 'cd file-engine && go test ./tests/integration -run TestUploadScanGateDirtyPreventsPromotion -v'

log "=== CL-035: Audit external sink delivery + DLQ + lag metrics ==="
run_claim "CL-035" bash -lc 'cd file-engine && go test ./tests/integration -run TestAuditExternalSinkDeliveryWithDLQAndLagMetrics -v'

log "=== CL-037: Storage contract suite (local) ==="
run_claim "CL-037" bash -lc 'cd file-engine && go test ./internal/adapters/storage/local -run TestLocalStorageContractSuite -v'

log "=== CL-011: Doc drift ==="
run_claim "CL-011" bash -lc './scripts/doc-drift-check.sh'

log "=== CL-020: Backend VS-001 E2E ==="
run_claim "CL-020" bash -lc './scripts/e2e/vs001_create_folder.sh'

if [[ "$LEDGER_MODE" == "full" ]]; then
  log "=== FULL MODE: extra baseline validations ==="

  if [[ "${RUN_S3_CONTRACTS:-0}" == "1" ]]; then
    run_claim "CL-S3-CONTRACT" bash -lc 'cd file-engine && go test ./internal/adapters/storage/s3 -run TestS3StorageContractSuite -v'
  fi

  if [[ "${RUN_GCS_CONTRACTS:-0}" == "1" ]]; then
    run_claim "CL-GCS-CONTRACT" bash -lc 'cd file-engine && go test ./internal/adapters/storage/gcs -run TestGCSStorageContractSuite -v'
  fi
fi

echo "LEDGER_BASELINE_OK"
log "[done] ledger baseline OK"
