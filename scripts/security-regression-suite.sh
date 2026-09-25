#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "[security-regression] running focused negative suite"

cd file-engine

go test ./internal/authz -run "TestExtractPathRejectsTraversal|TestNormalizePathHandlesWindowsAndWhitespace" -v

go test ./internal/handlers -run "TestListObjectsRejectsUnauthorizedTenant|TestDownloadObjectRejectsUnauthorizedTenant|TestUploadObjectRejectsUnauthorizedTenant" -v

go test ./internal/auth -run "TestHTTPAuthMiddleware|TestJWTToAuthContext|TestDenyAllTenantResolver" -v

go test ./internal/services -run "TestGovernancePolicyValidateRejectsNegativeValues|TestUploadServiceQuotaLimit|TestUploadServiceGovernanceDriftDetection" -v

go test -tags integration_authz ./tests/integration -run "TestReadListBehaviorAndAuthzRejection|TestEngineBoundaryDeniesCrossTenantEvenWithBuggyUpstreamInputs" -v

echo "SECURITY_REGRESSION_SUITE_OK"
