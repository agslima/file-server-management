# Compliance workflow: access review + evidence collection

This guide defines the **audit-ready** process for tenant access review exports.

## Output contract

`file-engine/scripts/export_access_review.sh` emits `compliance.access_review_export.v1` JSON with:

- `schema_version`
- `report_month`
- `generated_at`
- `source.system`, `source.endpoint`, `source.source_schema_version`
- `tenant_filter`
- `memberships[]`, each row containing:
  - `tenant_id`
  - `user_id`
  - `email`
  - `role_id`
  - `last_read_at`
  - `last_write_at`
  - `last_access`

`last_read_at` and `last_write_at` are derived from immutable audit event history.

## One-command monthly report generation

Use:

```bash
TOKEN='<admin-access-token>' \
BASE_URL='http://localhost:8080' \
TENANT_ID='acme' \
REPORT_MONTH='2026-02' \
./file-engine/scripts/generate_monthly_access_review_report.sh
```

Artifacts are written to:

- `artifacts/compliance/access-review/<REPORT_MONTH>/access-review.json`
- `artifacts/compliance/access-review/<REPORT_MONTH>/access-review.csv`
- `artifacts/compliance/access-review/<REPORT_MONTH>/manifest.txt`

## Evidence collection checklist

1. Generate report artifacts using the command above.
2. Attach `access-review.json`, `access-review.csv`, and `manifest.txt` to the audit ticket.
3. Record report month, tenant filter scope, and operator identity in change evidence.
4. (Optional) Hash artifacts using your org’s integrity tooling before retention.
