#!/usr/bin/env bash
set -euo pipefail

echo "[otel-exporter-down] validating deterministic exporter misconfiguration failure"
cd file-engine
go test ./internal/observability -run 'TestInitTracingRejectsUnsupportedEndpointScheme|TestInitTracingWithoutExporterIsDeterministic' -v

echo "[otel-exporter-down] expected signals validated: init failure for bad exporter endpoint + deterministic no-export fallback"
