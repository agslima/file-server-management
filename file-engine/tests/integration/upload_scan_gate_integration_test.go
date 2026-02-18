package integration

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/app/ports"
	"github.com/example/file-engine/internal/services"
)

type staticScanner struct {
	result ports.MalwareScanResult
}

func (s staticScanner) Scan(context.Context, string) (ports.MalwareScanResult, error) {
	return s.result, nil
}

func TestUploadScanGateDirtyPreventsPromotion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := localstorage.New(t.TempDir())
	svc := services.NewUploadService(st, staticScanner{result: ports.MalwareScanResult{Status: ports.MalwareStatusInfected, Engine: "stub-av", Detail: "signature matched"}}, services.UploadPolicy{
		MaxObjectSizeBytes: 1024,
		TenantQuotaBytes:   10 * 1024,
		RequestTimeout:     2 * time.Second,
		RequireCleanScan:   true,
	})

	targetPath := "/tenants/acme/docs/eicar.txt"
	meta, err := svc.UploadStream(ctx, targetPath, bytes.NewReader([]byte("eicar-test-content")), "dirty-case")
	if err == nil {
		t.Fatal("expected dirty scanner result to block promotion")
	}
	if !strings.Contains(err.Error(), "malware scan gate blocked commit") {
		t.Fatalf("expected malware scan gate error, got %v", err)
	}

	if meta.StagePath == "" {
		t.Fatal("expected stage path to be captured")
	}

	finalExists, err := st.Exists(ctx, targetPath)
	if err != nil {
		t.Fatalf("check final exists: %v", err)
	}
	if finalExists {
		t.Fatalf("expected final path %q to not exist when scan is dirty", targetPath)
	}

	stageExists, err := st.Exists(ctx, meta.StagePath)
	if err != nil {
		t.Fatalf("check stage exists: %v", err)
	}
	if !stageExists {
		t.Fatalf("expected quarantined stage path %q to remain for dirty scan", meta.StagePath)
	}
}

func TestUploadScanGateCleanPromotesObject(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := localstorage.New(t.TempDir())
	svc := services.NewUploadService(st, staticScanner{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean, Engine: "stub-av", Detail: "clean"}}, services.UploadPolicy{
		MaxObjectSizeBytes: 1024,
		TenantQuotaBytes:   10 * 1024,
		RequestTimeout:     2 * time.Second,
		RequireCleanScan:   true,
	})

	targetPath := "/tenants/acme/docs/clean.txt"
	payload := []byte("clean-content")
	meta, err := svc.UploadStream(ctx, targetPath, bytes.NewReader(payload), "clean-case")
	if err != nil {
		t.Fatalf("expected clean scan to allow promotion, got %v", err)
	}

	finalExists, err := st.Exists(ctx, targetPath)
	if err != nil {
		t.Fatalf("check final exists: %v", err)
	}
	if !finalExists {
		t.Fatalf("expected final path %q to exist for clean scan", targetPath)
	}

	stageExists, err := st.Exists(ctx, meta.StagePath)
	if err != nil {
		t.Fatalf("check stage exists: %v", err)
	}
	if stageExists {
		t.Fatalf("expected stage path %q to be removed after clean promote", meta.StagePath)
	}

	r, err := st.Open(ctx, targetPath)
	if err != nil {
		t.Fatalf("open promoted file: %v", err)
	}
	defer func() { _ = r.Close() }()

	content := new(bytes.Buffer)
	if _, err := content.ReadFrom(r); err != nil {
		t.Fatalf("read promoted file: %v", err)
	}
	if !bytes.Equal(content.Bytes(), payload) {
		t.Fatalf("expected promoted payload %q, got %q", string(payload), content.String())
	}
}
