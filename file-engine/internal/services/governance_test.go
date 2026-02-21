package services

import (
	"encoding/json"
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

func TestLoadGovernancePolicyFromSourceEnvelope(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "source.json")
	policy := GovernancePolicy{Default: TenantGovernancePolicy{ArchiveAfterDays: 7, ArchiveClass: "archive"}, Tenants: map[string]TenantGovernancePolicy{}}
	sig, err := signPolicy(policy, "k")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	payload, _ := json.Marshal(GovernancePolicyEnvelope{Policy: policy, Version: "v1", Signature: sig})
	if err := os.WriteFile(p, payload, 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	env, err := LoadGovernancePolicyFromSource(p, "k")
	if err != nil {
		t.Fatalf("load source: %v", err)
	}
	if env.Policy.Default.ArchiveAfterDays != 7 || env.Version != "v1" {
		t.Fatalf("unexpected source envelope: %+v", env)
	}
}
