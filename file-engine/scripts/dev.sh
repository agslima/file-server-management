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
go test ./internal/handlers -run "TestCreateFolderRequiresAuthContext|TestCreateFolderRejectsNonTenantPath|TestCreateFolderRejectsUnauthorizedTenant|TestCreateFolderEnqueuesWithCorrelationAndActorFallback|TestGetTaskStatusRequiresAuthAndReturnsPersistedStatus" -v

echo "[dev] running ListObjects handler test"
go test ./internal/handlers -run TestListObjectsReturnsEntries -v

echo "[dev] running authz precedence + path normalization tests"
go test ./internal/auth -run "TestRBACFallback|TestUserACLOverridesRBAC|TestACLPathInheritance|TestUserDenyPrecedesRoleAllowAndRBAC|TestRoleDenyPrecedesRoleAllowAtSamePath|TestClosestPathACLWinsBeforeParentACLs|TestUserACLPrecedenceOnSamePath|TestUserACLWithoutPermissionFallsThroughToRoleACL" -v
go test ./internal/authz -run "TestExtractPathNormalizesCreateFolder|TestExtractPathRejectsTraversal|TestNormalizePathHandlesWindowsAndWhitespace|TestNormalizePathAllowsDotContainingNames|TestTenantFromPath|TestTenantFromPathRejectsNonTenantRoot|TestGRPCAuthZInterceptorListObjects" -v

echo "[dev] running async folder flow integration test"
go test ./tests/integration -run TestAsyncCreateFolderFlow -v

echo "[dev] running HTTP handler + gateway tests"
go test ./internal/server -run "TestHandleDownloadNormalizesPath|TestHandleDownloadRejectsTraversal|TestGatewayCreateFolderAndGetTaskStatusRoutes" -v

echo "[dev] all checks passed"
