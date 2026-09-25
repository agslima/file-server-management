#!/usr/bin/env bash
set -euo pipefail

mode="${1:---mode=mock}"

print_step() {
  local id="$1"
  local title="$2"
  local payload="$3"
  echo "STEP ${id}: ${title}"
  echo "${payload}"
  echo "---"
}

if [[ "${mode}" != "--mode=mock" ]]; then
  echo "Only --mode=mock is supported for deterministic narrative output." >&2
  exit 1
fi

print_step "1" "OIDC login" '{"request":"POST /api/login","response":{"access_token":"demo-token","expires_in":3600}}'
print_step "2" "Tenant selected" '{"tenant":"acme"}'
print_step "3" "Folder creation" '{"request":"POST /api/folders","response":{"task_id":"task-demo-001","status":"queued"}}'
print_step "4" "Task progression" '{"request":"GET /api/tasks/task-demo-001","response":{"status":"success"}}'
print_step "5" "Upload lifecycle" '{"request":["POST /api/uploads/initiate","PUT /api/uploads/{id}/chunk","POST /api/uploads/{id}/complete"],"response":{"status":"clean_promoted"}}'
print_step "6" "Mutation actions" '{"move":"ok","delete":"ok","restore":"ok"}'
print_step "7" "Operator DLQ + cleanup" '{"request":["GET /admin/v1/scan-dlq","POST /admin/v1/scan-dlq","POST /admin/v1/quarantine:cleanup"],"response":{"dlq_entries":1,"cleanup_deleted":1}}'
print_step "8" "Effective policy + drift" '{"request":["GET /admin/v1/governance:effective?tenant_id=acme","POST /admin/v1/governance:drift-check"],"response":{"drift_detected":false}}'
print_step "9" "Evidence pack pointer" '{"request":"GET /admin/tenants/acme/evidence","response":{"schema_version":"tenant_evidence.v1"}}'

echo "DEMO_5_MINUTE_OK"
