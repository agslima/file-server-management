#!/usr/bin/env bash
set -euo pipefail

cd file-engine

go test ./internal/server -run "TestCompatibilityReadyzGolden|TestCompatibilityAuthzDenyGolden|TestCompatibilityUploadLifecycleGolden" -v
