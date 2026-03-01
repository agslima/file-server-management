# Governance Controls Operations Runbook

## Scope

This runbook covers tenant quota, retention, legal hold, archive-tier lifecycle, policy-source drift checks, and lifecycle cleanup operations for File Engine.

## Policy source

- Configure `GOVERNANCE_POLICY_FILE` to a JSON file that matches `file-engine/config/governance-policy.example.json`.
- Optionally configure external source-of-truth policy distribution:
  - `GOVERNANCE_POLICY_SOURCE=file:///path/to/governance-policy-source.json` (or `https://...`)
  - `GOVERNANCE_POLICY_SOURCE_HMAC_KEY=<shared-key>` for signed envelope verification
  - `GOVERNANCE_DRIFT_CHECK_INTERVAL_SECONDS=60` for periodic drift checks
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

## Verify archive-tier lifecycle simulation

```bash
cd file-engine && go test ./internal/services -run TestUploadServiceArchiveLifecycleTransition -v
```

## Verify external policy source + drift detection

```bash
cd file-engine && go test ./internal/services ./internal/server -run "TestLoadGovernancePolicyFromSourceEnvelope|TestUploadServiceGovernanceDriftDetection|TestGovernanceDriftCheckEndpoint" -v
```

## Operator admin endpoints

- `POST /admin/v1/governance:policy` payload: `{"policy":{...}}`
  - updates runtime governance policy and emits governance audit event reason with `before_hash` + `after_hash`.
- `POST /admin/v1/governance:delete` payload: `{"path":"/tenants/<tenant>/..."}`
  - returns `409` when blocked by retention/legal hold.
- `POST /admin/v1/lifecycle:cleanup`
  - runs policy-driven cleanup for quarantine/staging TTL windows.
- `GET /admin/v1/governance:effective?tenant_id=<tenant>`
  - returns effective runtime policy for tenant, source version, and drift state.
- `POST /admin/v1/governance:drift-check`
  - forces immediate runtime-vs-source policy comparison and returns drift status.

## Audit evidence

Governance decisions are recorded as in-memory governance events in upload service tests (`allow`/`deny` + reason) and should be exported to persistent audit sink in future hardening.
