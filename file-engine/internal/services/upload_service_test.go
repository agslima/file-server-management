package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/app/ports"
)

type scannerStub struct{ result ports.MalwareScanResult }

func (s scannerStub) Scan(context.Context, string) (ports.MalwareScanResult, error) {
	return s.result, nil
}

type failMoveStorage struct{ *localstorage.LocalStorage }

func (f failMoveStorage) Move(context.Context, string, string) error {
	return errors.New("move failed")
}

func TestUploadServiceStagesScansAndCommits(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean, Engine: "stub"}}, UploadPolicy{MaxObjectSizeBytes: 10, TenantQuotaBytes: 100, RequestTimeout: time.Second, RequireCleanScan: true})

	meta, err := svc.Upload(context.Background(), "/tenants/acme/docs/a.txt", []byte("ok"))
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if meta.Checksum == "" || meta.ScanStatus != ports.MalwareStatusClean {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
	r, err := st.Open(context.Background(), "/tenants/acme/docs/a.txt")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = r.Close() }()
	b, _ := io.ReadAll(r)
	if string(b) != "ok" {
		t.Fatalf("unexpected content %q", string(b))
	}
	if err := svc.VerifyIntegrity("/tenants/acme/docs/a.txt", b); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestUploadServiceBlocksWhenScanGateFails(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusInfected, Engine: "stub"}}, UploadPolicy{MaxObjectSizeBytes: 10, TenantQuotaBytes: 100, RequestTimeout: time.Second, RequireCleanScan: true})
	_, err := svc.Upload(context.Background(), "/tenants/acme/docs/eicar.txt", []byte("eicar"))
	if err == nil {
		t.Fatalf("expected scan gate failure")
	}
}

func TestUploadServiceQuotaLimit(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean}}, UploadPolicy{MaxObjectSizeBytes: 20, TenantQuotaBytes: 3, TenantObjectLimit: 10, RequestTimeout: time.Second})
	if _, err := svc.Upload(context.Background(), "/tenants/acme/docs/a.txt", []byte("ab")); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if _, err := svc.Upload(context.Background(), "/tenants/acme/docs/b.txt", []byte("cd")); err == nil {
		t.Fatalf("expected quota error")
	}
}

func TestUploadServiceAtomicCommitOnMoveError(t *testing.T) {
	base := localstorage.New(t.TempDir())
	svc := NewUploadService(failMoveStorage{base}, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean}}, UploadPolicy{MaxObjectSizeBytes: 20, TenantQuotaBytes: 100, RequestTimeout: time.Second})
	_, err := svc.Upload(context.Background(), "/tenants/acme/docs/a.txt", []byte("abc"))
	if err == nil {
		t.Fatalf("expected move failure")
	}
}

func TestUploadServiceChecksumMismatch(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean}}, UploadPolicy{MaxObjectSizeBytes: 20, TenantQuotaBytes: 100, RequestTimeout: time.Second})
	_, err := svc.Upload(context.Background(), "/tenants/acme/docs/a.txt", []byte("abc"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if err := svc.VerifyIntegrity("/tenants/acme/docs/a.txt", bytes.NewBufferString("zzz").Bytes()); err == nil {
		t.Fatalf("expected checksum mismatch")
	}
}
