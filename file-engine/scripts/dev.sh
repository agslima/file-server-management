#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../.." >/dev/null 2>&1 && pwd)"

cd "$REPO_ROOT/file-engine"

echo "[dev] validating canonical proto mirror"
cmp api/proto/fileengine.proto proto/fileengine.proto

echo "[dev] running baseline module tests"
go test ./internal/config ./internal/logger ./internal/worker -v

echo "[dev] running CreateFolder auth + task status handler tests"
go test ./internal/handlers -run "TestCreateFolderRequiresAuthContext|TestCreateFolderEnqueuesWithCorrelationAndActorFallback|TestGetTaskStatusRequiresAuthAndReturnsPersistedStatus" -v

echo "[dev] running async folder flow integration test"
go test ./tests/integration -run TestAsyncCreateFolderFlow -v

echo "[dev] all checks passed"
