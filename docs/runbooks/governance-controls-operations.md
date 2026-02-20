# Governance Controls Operations Runbook

## Scope

This runbook covers tenant quota, retention, legal hold, and lifecycle cleanup operations for File Engine.

## Policy source

- Configure `GOVERNANCE_POLICY_FILE` to a JSON file that matches `file-engine/config/governance-policy.example.json`.
- File Engine validates policy at startup and fails fast on invalid values.

## Validate policy

```bash
cd file-engine && go test ./internal/services -run "TestLoadGovernancePolicyFromFile|TestGovernancePolicyValidateRejectsNegativeValues" -v
```

## Verify quota enforcement

```bash
cd file-engine && go test ./internal/services -run TestUploadServiceTenantPolicyQuotaFinalGate -v
```

## Verify retention/legal hold delete protection

```bash
cd file-engine && go test ./internal/services -run "TestUploadServiceRetentionBlocksDelete|TestUploadServiceLegalHoldBlocksDelete" -v
```

## Operator admin endpoints

- `POST /admin/v1/governance:delete` payload: `{"path":"/tenants/<tenant>/..."}`
  - returns `409` when blocked by retention/legal hold.
- `POST /admin/v1/lifecycle:cleanup`
  - runs policy-driven cleanup for quarantine/staging TTL windows.

## Audit evidence

Governance decisions are recorded as in-memory governance events in upload service tests (`allow`/`deny` + reason) and should be exported to persistent audit sink in future hardening.
