package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGovernancePolicyFromFile(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "governance.json")
	content := `{"default":{"quota_bytes":10,"retention_seconds":30},"tenants":{"acme":{"legal_hold":true}},"lifecycle":{"quarantine_ttl_seconds":60}}`
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	policy, err := LoadGovernancePolicyFromFile(p)
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if policy.Default.QuotaBytes != 10 {
		t.Fatalf("unexpected policy: %+v", policy)
	}
	if !policy.Tenants["acme"].LegalHold {
		t.Fatalf("expected tenant legal hold")
	}
}

func TestGovernancePolicyValidateRejectsNegativeValues(t *testing.T) {
	err := GovernancePolicy{Default: TenantGovernancePolicy{QuotaBytes: -1}, Tenants: map[string]TenantGovernancePolicy{}}.Validate()
	if err == nil {
		t.Fatalf("expected validation error")
	}
}
