package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type TenantGovernancePolicy struct {
	QuotaBytes        int64  `json:"quota_bytes"`
	ObjectLimit       int64  `json:"object_limit"`
	RequestsPerMinute int64  `json:"requests_per_minute"`
	RetentionSeconds  int64  `json:"retention_seconds"`
	LegalHold         bool   `json:"legal_hold"`
	ArchiveAfterDays  int64  `json:"archive_after_days"`
	ArchiveClass      string `json:"archive_storage_class"`
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

type GovernancePolicyEnvelope struct {
	Policy    GovernancePolicy `json:"policy"`
	Version   string           `json:"version"`
	UpdatedAt string           `json:"updated_at,omitempty"`
	Signature string           `json:"signature,omitempty"`
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

func LoadGovernancePolicyFromSource(sourceURL, hmacKey string) (GovernancePolicyEnvelope, error) {
	if strings.TrimSpace(sourceURL) == "" {
		return GovernancePolicyEnvelope{}, nil
	}
	b, err := readPolicySource(sourceURL)
	if err != nil {
		return GovernancePolicyEnvelope{}, err
	}
	var env GovernancePolicyEnvelope
	if err := json.Unmarshal(b, &env); err == nil && (!isZeroPolicy(env.Policy) || env.Signature != "") {
		if strings.TrimSpace(hmacKey) != "" {
			expected, err := signPolicy(env.Policy, hmacKey)
			if err != nil {
				return GovernancePolicyEnvelope{}, err
			}
			if !hmac.Equal([]byte(strings.TrimSpace(env.Signature)), []byte(expected)) {
				return GovernancePolicyEnvelope{}, errors.New("governance policy signature mismatch")
			}
		}
		if err := env.Policy.Validate(); err != nil {
			return GovernancePolicyEnvelope{}, err
		}
		return env, nil
	}

	var p GovernancePolicy
	if err := json.Unmarshal(b, &p); err != nil {
		return GovernancePolicyEnvelope{}, fmt.Errorf("parse governance policy: %w", err)
	}
	if err := p.Validate(); err != nil {
		return GovernancePolicyEnvelope{}, err
	}
	return GovernancePolicyEnvelope{Policy: p, Version: "raw"}, nil
}

func isZeroPolicy(p GovernancePolicy) bool {
	return p.Default == (TenantGovernancePolicy{}) && len(p.Tenants) == 0 && len(p.PathHolds) == 0 && p.Lifecycle == (LifecyclePolicy{})
}

func readPolicySource(sourceURL string) ([]byte, error) {
	if strings.HasPrefix(sourceURL, "http://") || strings.HasPrefix(sourceURL, "https://") {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, sourceURL, http.NoBody) // #nosec G107 -- operator-managed governance endpoint.
		if err != nil {
			return nil, fmt.Errorf("build governance policy request: %w", err)
		}
		resp, err := http.DefaultClient.Do(req) // #nosec G704 -- source URL is operator-managed governance endpoint.
		if err != nil {
			return nil, fmt.Errorf("read governance policy source: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("read governance policy source: status %d", resp.StatusCode)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read governance policy source body: %w", err)
		}
		return b, nil
	}
	p := strings.TrimPrefix(sourceURL, "file://")
	b, err := os.ReadFile(p) // #nosec G304 -- operator-configured path.
	if err != nil {
		return nil, fmt.Errorf("read governance policy source: %w", err)
	}
	return b, nil
}

func signPolicy(p GovernancePolicy, key string) (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal policy for signature: %w", err)
	}
	h := hmac.New(sha256.New, []byte(key))
	_, _ = h.Write(b)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (p GovernancePolicy) Validate() error {
	for tenant, tp := range p.Tenants {
		if strings.TrimSpace(tenant) == "" {
			return errors.New("governance tenant id cannot be empty")
		}
		if tp.QuotaBytes < 0 || tp.ObjectLimit < 0 || tp.RequestsPerMinute < 0 || tp.RetentionSeconds < 0 || tp.ArchiveAfterDays < 0 {
			return fmt.Errorf("governance tenant %q has negative values", tenant)
		}
	}
	if p.Default.QuotaBytes < 0 || p.Default.ObjectLimit < 0 || p.Default.RequestsPerMinute < 0 || p.Default.RetentionSeconds < 0 || p.Default.ArchiveAfterDays < 0 {
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
	if p.Default.ArchiveClass != "" && strings.TrimSpace(p.Default.ArchiveClass) == "" {
		return errors.New("governance default archive_storage_class cannot be blank")
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
		if tp.ArchiveAfterDays > 0 {
			p.ArchiveAfterDays = tp.ArchiveAfterDays
		}
		if strings.TrimSpace(tp.ArchiveClass) != "" {
			p.ArchiveClass = strings.TrimSpace(tp.ArchiveClass)
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
