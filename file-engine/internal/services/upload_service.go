package services

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrTenantQuotaExceeded  = errors.New("tenant quota exceeded")
	ErrTenantObjectLimitHit = errors.New("tenant object limit exceeded")
)

type UploadPolicy struct {
	TenantQuotaBytes  int64
	TenantObjectLimit int64
}

type TenantUsageProvider interface {
	CurrentUsage(ctx context.Context, tenantID string) (bytesUsed int64, objectsUsed int64, err error)
}

type UploadService struct {
	policy UploadPolicy
	usage  TenantUsageProvider
}

func NewUploadService(policy UploadPolicy, usage TenantUsageProvider) *UploadService {
	return &UploadService{policy: policy, usage: usage}
}

// enforceTenantLimits checks byte and object limits independently so object limits
// are still enforced when byte quota is intentionally disabled.
func (s *UploadService) enforceTenantLimits(ctx context.Context, tenantID string, incomingBytes int64) error {
	if s.usage == nil {
		return nil
	}

	bytesUsed, objectsUsed, err := s.usage.CurrentUsage(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("lookup tenant usage: %w", err)
	}

	if s.policy.TenantQuotaBytes > 0 && bytesUsed+incomingBytes > s.policy.TenantQuotaBytes {
		return ErrTenantQuotaExceeded
	}
	if s.policy.TenantObjectLimit > 0 && objectsUsed+1 > s.policy.TenantObjectLimit {
		return ErrTenantObjectLimitHit
	}

	return nil
}
