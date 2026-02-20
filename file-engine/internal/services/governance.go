package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

type TenantGovernancePolicy struct {
	QuotaBytes        int64 `json:"quota_bytes"`
	ObjectLimit       int64 `json:"object_limit"`
	RequestsPerMinute int64 `json:"requests_per_minute"`
	RetentionSeconds  int64 `json:"retention_seconds"`
	LegalHold         bool  `json:"legal_hold"`
}

type LifecyclePolicy struct {
	QuarantineTTLSeconds    int64 `json:"quarantine_ttl_seconds"`
	OrphanStagingTTLSeconds int64 `json:"orphan_staging_ttl_seconds"`
}

type GovernancePolicy struct {
	Default   TenantGovernancePolicy            `json:"default"`
	Tenants   map[string]TenantGovernancePolicy `json:"tenants"`
	PathHolds []string                          `json:"path_holds"`
	Lifecycle LifecyclePolicy                   `json:"lifecycle"`
}

type GovernanceEvent struct {
	Timestamp time.Time `json:"timestamp"`
	ActorID   string    `json:"actor_id"`
	TenantID  string    `json:"tenant_id"`
	Action    string    `json:"action"`
	Path      string    `json:"path"`
	Decision  string    `json:"decision"`
	Reason    string    `json:"reason,omitempty"`
}

func LoadGovernancePolicyFromFile(filePath string) (GovernancePolicy, error) {
	if strings.TrimSpace(filePath) == "" {
		return GovernancePolicy{}, nil
	}
	b, err := os.ReadFile(filePath) // #nosec G304 -- file path is provided by trusted operator configuration.
	if err != nil {
		return GovernancePolicy{}, fmt.Errorf("read governance policy: %w", err)
	}
	var p GovernancePolicy
	if err := json.Unmarshal(b, &p); err != nil {
		return GovernancePolicy{}, fmt.Errorf("parse governance policy: %w", err)
	}
	if err := p.Validate(); err != nil {
		return GovernancePolicy{}, err
	}
	return p, nil
}

func (p GovernancePolicy) Validate() error {
	for tenant, tp := range p.Tenants {
		if strings.TrimSpace(tenant) == "" {
			return errors.New("governance tenant id cannot be empty")
		}
		if tp.QuotaBytes < 0 || tp.ObjectLimit < 0 || tp.RequestsPerMinute < 0 || tp.RetentionSeconds < 0 {
			return fmt.Errorf("governance tenant %q has negative values", tenant)
		}
	}
	if p.Default.QuotaBytes < 0 || p.Default.ObjectLimit < 0 || p.Default.RequestsPerMinute < 0 || p.Default.RetentionSeconds < 0 {
		return errors.New("governance default policy has negative values")
	}
	if p.Lifecycle.QuarantineTTLSeconds < 0 || p.Lifecycle.OrphanStagingTTLSeconds < 0 {
		return errors.New("governance lifecycle TTL values must be >= 0")
	}
	for _, v := range p.PathHolds {
		if strings.TrimSpace(v) == "" {
			return errors.New("governance path_holds cannot contain empty values")
		}
	}
	return nil
}

func (s *UploadService) tenantGovernancePolicy(tenantID string) TenantGovernancePolicy {
	p := s.governance.Default
	if tp, ok := s.governance.Tenants[tenantID]; ok {
		if tp.QuotaBytes > 0 {
			p.QuotaBytes = tp.QuotaBytes
		}
		if tp.ObjectLimit > 0 {
			p.ObjectLimit = tp.ObjectLimit
		}
		if tp.RequestsPerMinute > 0 {
			p.RequestsPerMinute = tp.RequestsPerMinute
		}
		if tp.RetentionSeconds > 0 {
			p.RetentionSeconds = tp.RetentionSeconds
		}
		if tp.LegalHold {
			p.LegalHold = true
		}
	}
	return p
}

func (s *UploadService) pathUnderLegalHold(objectPath string) bool {
	normalized := path.Clean(objectPath)
	for _, hold := range s.governance.PathHolds {
		h := path.Clean(hold)
		if base, ok := strings.CutSuffix(h, "/"); ok {
			if strings.HasPrefix(normalized, base+"/") {
				return true
			}
			continue
		}
		if normalized == h || strings.HasPrefix(normalized, h+"/") {
			return true
		}
	}
	return false
}
