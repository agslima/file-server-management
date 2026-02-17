package services

import (
	"context"
	"errors"
	"testing"
)

type fakeTenantUsageProvider struct {
	bytesUsed   int64
	objectsUsed int64
	err         error
}

func (f fakeTenantUsageProvider) CurrentUsage(_ context.Context, _ string) (int64, int64, error) {
	return f.bytesUsed, f.objectsUsed, f.err
}

func TestEnforceTenantLimits_ObjectLimitAppliedWhenQuotaDisabled(t *testing.T) {
	svc := NewUploadService(UploadPolicy{
		TenantQuotaBytes:  0,
		TenantObjectLimit: 2,
	}, fakeTenantUsageProvider{objectsUsed: 2})

	err := svc.enforceTenantLimits(context.Background(), "tenant-a", 1024)
	if !errors.Is(err, ErrTenantObjectLimitHit) {
		t.Fatalf("expected object-limit error, got %v", err)
	}
}

func TestEnforceTenantLimits_QuotaAndObjectChecksAreIndependent(t *testing.T) {
	svc := NewUploadService(UploadPolicy{
		TenantQuotaBytes:  100,
		TenantObjectLimit: 10,
	}, fakeTenantUsageProvider{bytesUsed: 90, objectsUsed: 1})

	err := svc.enforceTenantLimits(context.Background(), "tenant-a", 20)
	if !errors.Is(err, ErrTenantQuotaExceeded) {
		t.Fatalf("expected quota error, got %v", err)
	}
}
