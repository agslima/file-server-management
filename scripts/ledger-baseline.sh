#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

echo "[ledger-baseline] CL-001 proto mirror sync"
cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto

echo "[ledger-baseline] CL-002 file-engine baseline modules"
cd file-engine
go test ./internal/config ./internal/logger ./internal/worker -v

echo "[ledger-baseline] CL-003 async create-folder integration"
go test ./tests/integration -run TestAsyncCreateFolderFlow -v

echo "[ledger-baseline] CL-022 read/list/download audit coverage"
go test ./tests/integration -run TestAuditEventsEmittedForReadListDownload -v

echo "[ledger-baseline] CL-025 staged upload + atomic promote"
go test ./tests/integration -run TestStagedUploadAtomicPromote -v

echo "[ledger-baseline] CL-033 malware gate dirty blocks promote"
go test ./tests/integration -run TestUploadScanGateDirtyPreventsPromotion -v
cd ..

echo "[ledger-baseline] CL-011 doc drift"
./scripts/doc-drift-check.sh

echo "[ledger-baseline] CL-008 + CL-018 backend scaffold checks"
cd backend
composer validate --strict
php -l app/Http/Controllers/FolderController.php
php -l app/Http/Controllers/TaskController.php
php -l app/Services/FileEngineService.php

echo "[ledger-baseline] CL-031 backend smoke"
./scripts/smoke.sh
cd ..

echo "[ledger-baseline] completed"
