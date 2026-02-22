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
# - Optional toggles:
#     RUN_CL020=0: skip CL-020 E2E inside ledger flow (useful when a dedicated CI job runs it)

set -Eeuo pipefail

LEDGER_MODE="${LEDGER_MODE:-fast}"
LEDGER_TIMEOUT_SECONDS="${LEDGER_TIMEOUT_SECONDS:-900}"
RUN_CL020="${RUN_CL020:-1}"

# Compose isolation: avoid cross-job collisions.
if [[ -n "${GITHUB_RUN_ID:-}" ]]; then
  default_project_name="ledger-baseline-${GITHUB_RUN_ID}"
else
  default_project_name="ledger-baseline-local-${RANDOM}"
fi
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-$default_project_name}"

host_port_open() {
  local port="$1"
  (exec 3<>"/dev/tcp/127.0.0.1/${port}") >/dev/null 2>&1
}

find_available_port() {
  local start="${1:-15432}"
  local end="${2:-16432}"
  local p
  for ((p=start; p<=end; p++)); do
    if ! host_port_open "$p"; then
      echo "$p"
      return 0
    fi
  done
  return 1
}

if [[ -z "${POSTGRES_HOST_PORT:-}" ]]; then
  if host_port_open 5432; then
    POSTGRES_HOST_PORT="$(find_available_port)" || {
      echo "[fatal] failed to find an available fallback host port for postgres" >&2
      exit 1
    }
    echo "[$(date +"%Y-%m-%dT%H:%M:%S%z")] [info] host port 5432 is in use; using POSTGRES_HOST_PORT=${POSTGRES_HOST_PORT}"
  else
    POSTGRES_HOST_PORT="5432"
  fi
fi
export POSTGRES_HOST_PORT

if [[ -z "${REDIS_HOST_PORT:-}" ]]; then
  if host_port_open 6379; then
    REDIS_HOST_PORT="$(find_available_port 16379 17379)" || {
      echo "[fatal] failed to find an available fallback host port for redis" >&2
      exit 1
    }
    echo "[$(date +"%Y-%m-%dT%H:%M:%S%z")] [info] host port 6379 is in use; using REDIS_HOST_PORT=${REDIS_HOST_PORT}"
  else
    REDIS_HOST_PORT="6379"
  fi
fi
export REDIS_HOST_PORT

if [[ -z "${FILE_ENGINE_HTTP_HOST_PORT:-}" ]]; then
  if host_port_open 8080; then
    FILE_ENGINE_HTTP_HOST_PORT="$(find_available_port 18080 19080)" || {
      echo "[fatal] failed to find an available fallback host port for file-engine http" >&2
      exit 1
    }
    echo "[$(date +"%Y-%m-%dT%H:%M:%S%z")] [info] host port 8080 is in use; using FILE_ENGINE_HTTP_HOST_PORT=${FILE_ENGINE_HTTP_HOST_PORT}"
  else
    FILE_ENGINE_HTTP_HOST_PORT="8080"
  fi
fi
export FILE_ENGINE_HTTP_HOST_PORT

if [[ -z "${BACKEND_HOST_PORT:-}" ]]; then
  if host_port_open 8081; then
    BACKEND_HOST_PORT="$(find_available_port 18081 19081)" || {
      echo "[fatal] failed to find an available fallback host port for backend http" >&2
      exit 1
    }
    echo "[$(date +"%Y-%m-%dT%H:%M:%S%z")] [info] host port 8081 is in use; using BACKEND_HOST_PORT=${BACKEND_HOST_PORT}"
  else
    BACKEND_HOST_PORT="8081"
  fi
fi
export BACKEND_HOST_PORT

if [[ -z "${FILE_ENGINE_GRPC_HOST_PORT:-}" ]]; then
  if host_port_open 50051; then
    FILE_ENGINE_GRPC_HOST_PORT="$(find_available_port 15051 16051)" || {
      echo "[fatal] failed to find an available fallback host port for file-engine grpc" >&2
      exit 1
    }
    echo "[$(date +"%Y-%m-%dT%H:%M:%S%z")] [info] host port 50051 is in use; using FILE_ENGINE_GRPC_HOST_PORT=${FILE_ENGINE_GRPC_HOST_PORT}"
  else
    FILE_ENGINE_GRPC_HOST_PORT="50051"
  fi
fi
export FILE_ENGINE_GRPC_HOST_PORT

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
  echo "[debug] POSTGRES_HOST_PORT=${POSTGRES_HOST_PORT}"
  echo "[debug] REDIS_HOST_PORT=${REDIS_HOST_PORT}"
  echo "[debug] FILE_ENGINE_HTTP_HOST_PORT=${FILE_ENGINE_HTTP_HOST_PORT}"
  echo "[debug] BACKEND_HOST_PORT=${BACKEND_HOST_PORT}"
  echo "[debug] FILE_ENGINE_GRPC_HOST_PORT=${FILE_ENGINE_GRPC_HOST_PORT}"
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
  compose logs --no-color --tail=250 file-engine-worker 2>/dev/null || true
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

wait_for_postgres() {
  local timeout_seconds="${1:-90}"
  local elapsed=0
  local interval=2

  log "[wait] postgres readiness (timeout=${timeout_seconds}s)"
  while (( elapsed < timeout_seconds )); do
    if compose exec -T postgres pg_isready -U fileengine -d fileengine >/dev/null 2>&1; then
      log "[ready] postgres is accepting connections"
      return 0
    fi
    sleep "$interval"
    elapsed=$((elapsed + interval))
  done

  die "postgres did not become ready within ${timeout_seconds}s"
}

wait_for_http_service() {
  local url="$1"
  local timeout_seconds="${2:-120}"
  log "[wait] http readiness ${url} (timeout=${timeout_seconds}s)"
  bash -lc "./scripts/wait-for-http.sh '${url}' '${timeout_seconds}'" >/dev/null
}

log "[start] ledger baseline (mode=${LEDGER_MODE})"

log "=== CL-001: Proto + gateway generation is in sync ==="
run_claim "CL-001" bash -lc 'cd file-engine && make proto && git diff --exit-code -- api/proto proto pkg/generated'

log "=== CL-002: Go modules baseline ==="
run_claim "CL-002" bash -lc '
  cd file-engine
  tmpmod="$(mktemp)"
  tmpsum="$(mktemp)"
  cp go.mod "$tmpmod"
  cp go.sum "$tmpsum"
  go mod tidy
  diff -u "$tmpmod" go.mod
  diff -u "$tmpsum" go.sum
  rm -f "$tmpmod" "$tmpsum"
'

log "=== Backend baseline: CL-008 + CL-018 + CL-031 ==="
run_claim "CL-008" bash -lc 'cd backend && composer validate --strict'
run_claim "CL-018" bash -lc 'cd backend && php -l app/Http/Controllers/FolderController.php && php -l app/Http/Controllers/TaskController.php && php -l app/Services/FileEngineService.php'
run_claim "CL-031" bash -lc 'cd backend && ./scripts/smoke.sh'

log "=== Infra: Postgres up (isolated compose project) ==="
compose up -d postgres
wait_for_postgres 90

# Integration tests use FILEENGINE_TEST_POSTGRES_DSN when present.
export FILEENGINE_TEST_POSTGRES_DSN="${FILEENGINE_TEST_POSTGRES_DSN:-postgres://fileengine:fileengine@127.0.0.1:${POSTGRES_HOST_PORT}/fileengine?sslmode=disable}"

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

if [[ "$RUN_CL020" == "1" ]]; then
  log "=== Infra: Full stack up for CL-020 ==="
  compose up -d redis file-engine file-engine-worker backend
  wait_for_http_service "http://localhost:${FILE_ENGINE_HTTP_HOST_PORT}/readyz" 120
  wait_for_http_service "http://localhost:${BACKEND_HOST_PORT}/healthz" 120

  log "=== CL-020: Backend VS-001 E2E ==="
  run_claim "CL-020" bash -lc "BACKEND_URL='http://localhost:${BACKEND_HOST_PORT}' ./scripts/e2e/vs001_create_folder.sh"
else
  log "=== CL-020: skipped (RUN_CL020=${RUN_CL020}) ==="
fi

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
