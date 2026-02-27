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

type flakyScanner struct{ calls int }

func (s *flakyScanner) Scan(context.Context, string) (ports.MalwareScanResult, error) {
	s.calls++
	if s.calls < 3 {
		return ports.MalwareScanResult{}, errors.New("scanner unavailable")
	}
	return ports.MalwareScanResult{Status: ports.MalwareStatusClean, Engine: "flaky"}, nil
}

type alwaysFailScanner struct{}

func (alwaysFailScanner) Scan(context.Context, string) (ports.MalwareScanResult, error) {
	return ports.MalwareScanResult{}, errors.New("scanner down")
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

func TestUploadServiceIdempotencyKeyCannotBeReusedAcrossTargets(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean}}, UploadPolicy{MaxObjectSizeBytes: 20, TenantQuotaBytes: 100, RequestTimeout: time.Second})

	if _, err := svc.UploadStream(context.Background(), "/tenants/acme/docs/a.txt", bytes.NewReader([]byte("abc")), "same-key"); err != nil {
		t.Fatalf("first upload: %v", err)
	}

	if _, err := svc.UploadStream(context.Background(), "/tenants/acme/docs/b.txt", bytes.NewReader([]byte("abc")), "same-key"); err == nil {
		t.Fatalf("expected idempotency key target conflict")
	}
}

func TestUploadServiceIdempotencyReplayPreservesFailure(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusInfected, Engine: "stub"}}, UploadPolicy{MaxObjectSizeBytes: 20, TenantQuotaBytes: 100, RequestTimeout: time.Second, RequireCleanScan: true})

	if _, err := svc.UploadStream(context.Background(), "/tenants/acme/docs/eicar.txt", bytes.NewReader([]byte("eicar")), "scan-key"); err == nil {
		t.Fatalf("expected malware scan failure")
	}

	if _, err := svc.UploadStream(context.Background(), "/tenants/acme/docs/eicar.txt", bytes.NewReader([]byte("different")), "scan-key"); err == nil {
		t.Fatalf("expected replay to return prior failure")
	}
}

func TestUploadServiceScannerRetryEventuallySucceeds(t *testing.T) {
	st := localstorage.New(t.TempDir())
	scanner := &flakyScanner{}
	svc := NewUploadService(st, scanner, UploadPolicy{MaxObjectSizeBytes: 20, TenantQuotaBytes: 100, RequestTimeout: time.Second, RequireCleanScan: true, ScanRetryAttempts: 3, ScanRetryBackoff: time.Millisecond})

	if _, err := svc.Upload(context.Background(), "/tenants/acme/docs/retry.txt", []byte("abc")); err != nil {
		t.Fatalf("expected retry success, got %v", err)
	}
	if scanner.calls != 3 {
		t.Fatalf("expected 3 scan attempts, got %d", scanner.calls)
	}
}

func TestUploadServiceScannerFailureEnqueuesDLQ(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, alwaysFailScanner{}, UploadPolicy{MaxObjectSizeBytes: 20, TenantQuotaBytes: 100, RequestTimeout: time.Second, RequireCleanScan: true, ScanRetryAttempts: 2, ScanRetryBackoff: time.Millisecond})

	if _, err := svc.Upload(context.Background(), "/tenants/acme/docs/fail.txt", []byte("abc")); err == nil {
		t.Fatal("expected upload failure")
	}
	dlq := svc.ListScanDLQ(false)
	if len(dlq) != 1 {
		t.Fatalf("expected 1 dlq entry, got %+v", dlq)
	}
	if dlq[0].Attempts != 2 {
		t.Fatalf("expected attempts to be recorded, got %+v", dlq[0])
	}
}

func TestUploadServiceCleanupQuarantineDeletesExpiredObjects(t *testing.T) {
	st := localstorage.New(t.TempDir())
	if err := st.AtomicWrite(context.Background(), "/quarantine/acme/old.bin", bytes.NewBufferString("x")); err != nil {
		t.Fatalf("seed quarantine: %v", err)
	}
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean}}, UploadPolicy{MaxObjectSizeBytes: 20, TenantQuotaBytes: 100, RequestTimeout: time.Second})

	rep, elapsed := awaitCleanupReport(t, 2*time.Second, 20*time.Millisecond, func() (CleanupReport, error) {
		return svc.CleanupQuarantine(context.Background(), 1*time.Millisecond)
	})
	if rep.Deleted == 0 {
		t.Fatalf("expected deleted > 0 after %s, got %+v", elapsed, rep)
	}
}

func awaitCleanupReport(t *testing.T, timeout, interval time.Duration, cleanup func() (CleanupReport, error)) (CleanupReport, time.Duration) {
	t.Helper()
	started := time.Now()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()
	last := CleanupReport{}
	attempts := 0
	for {
		rep, err := cleanup()
		attempts++
		if err != nil {
			t.Fatalf("cleanup attempt %d failed: %v", attempts, err)
		}
		last = rep
		if rep.Deleted > 0 {
			return rep, time.Since(started)
		}
		select {
		case <-ticker.C:
		case <-timeoutTimer.C:
			t.Fatalf("timed out waiting for cleanup deletion (timeout=%s interval=%s attempts=%d last_report=%+v)", timeout, interval, attempts, last)
		}
	}
}

func TestUploadServiceResumableUploadFinalize(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean}}, UploadPolicy{MaxObjectSizeBytes: 20, TenantQuotaBytes: 100, RequestTimeout: time.Second})

	session, err := svc.StartResumableUpload("/tenants/acme/docs/resume.txt", "")
	if err != nil {
		t.Fatalf("start resumable: %v", err)
	}
	if err := svc.UploadChunk(session, 0, []byte("hel")); err != nil {
		t.Fatalf("chunk 1: %v", err)
	}
	if err := svc.UploadChunk(session, 3, []byte("lo")); err != nil {
		t.Fatalf("chunk 2: %v", err)
	}
	meta, err := svc.FinalizeResumableUpload(context.Background(), session, "")
	if err != nil {
		t.Fatalf("finalize resumable: %v", err)
	}
	if meta.Path != "/tenants/acme/docs/resume.txt" {
		t.Fatalf("unexpected target path: %+v", meta)
	}
	r, err := st.Open(context.Background(), "/tenants/acme/docs/resume.txt")
	if err != nil {
		t.Fatalf("open finalized object: %v", err)
	}
	defer func() { _ = r.Close() }()
	b, _ := io.ReadAll(r)
	if string(b) != "hello" {
		t.Fatalf("expected hello, got %q", string(b))
	}
}

func TestUploadServiceResumableCompleteReplayByIdempotencyKey(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean}}, UploadPolicy{MaxObjectSizeBytes: 20, TenantQuotaBytes: 100, RequestTimeout: time.Second})

	session, err := svc.StartResumableUpload("/tenants/acme/docs/replay.txt", "")
	if err != nil {
		t.Fatalf("start resumable: %v", err)
	}
	if err := svc.UploadChunk(session, 0, []byte("hello")); err != nil {
		t.Fatalf("chunk: %v", err)
	}

	first, err := svc.FinalizeResumableUpload(context.Background(), session, "complete-key")
	if err != nil {
		t.Fatalf("finalize first: %v", err)
	}

	replay, err := svc.FinalizeResumableUpload(context.Background(), session, "complete-key")
	if err != nil {
		t.Fatalf("finalize replay: %v", err)
	}
	if replay.Path != first.Path || replay.Checksum != first.Checksum {
		t.Fatalf("expected replay metadata to match first finalize, first=%+v replay=%+v", first, replay)
	}
}

func TestUploadServiceRetentionBlocksDelete(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean}}, UploadPolicy{RequestTimeout: time.Second})
	if err := svc.SetGovernancePolicy(GovernancePolicy{Default: TenantGovernancePolicy{RetentionSeconds: 3600}, Tenants: map[string]TenantGovernancePolicy{}}); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	if _, err := svc.Upload(context.Background(), "/tenants/acme/docs/retain.txt", []byte("payload")); err != nil {
		t.Fatalf("upload: %v", err)
	}
	if err := svc.DeleteObject(context.Background(), "admin", "/tenants/acme/docs/retain.txt"); err == nil {
		t.Fatalf("expected retention delete denial")
	}
	events := svc.GovernanceEvents()
	if len(events) == 0 || events[len(events)-1].Reason != "retention" {
		t.Fatalf("expected retention governance event, got %+v", events)
	}
}

func TestUploadServiceLegalHoldBlocksDelete(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean}}, UploadPolicy{RequestTimeout: time.Second})
	if err := svc.SetGovernancePolicy(GovernancePolicy{Default: TenantGovernancePolicy{}, Tenants: map[string]TenantGovernancePolicy{"acme": {LegalHold: true}}}); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	if _, err := svc.Upload(context.Background(), "/tenants/acme/docs/hold.txt", []byte("payload")); err != nil {
		t.Fatalf("upload: %v", err)
	}
	if err := svc.DeleteObject(context.Background(), "admin", "/tenants/acme/docs/hold.txt"); err == nil {
		t.Fatalf("expected legal hold delete denial")
	}
	events := svc.GovernanceEvents()
	if len(events) == 0 || events[len(events)-1].Reason != "legal_hold" {
		t.Fatalf("expected legal hold governance event, got %+v", events)
	}
}

func TestUploadServiceTenantPolicyQuotaFinalGate(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean}}, UploadPolicy{RequestTimeout: time.Second})
	if err := svc.SetGovernancePolicy(GovernancePolicy{Default: TenantGovernancePolicy{}, Tenants: map[string]TenantGovernancePolicy{"acme": {QuotaBytes: 3}}}); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	if _, err := svc.Upload(context.Background(), "/tenants/acme/docs/a.txt", []byte("ab")); err != nil {
		t.Fatalf("first upload: %v", err)
	}
	if _, err := svc.Upload(context.Background(), "/tenants/acme/docs/b.txt", []byte("cd")); err == nil {
		t.Fatalf("expected quota denial")
	}
	events := svc.GovernanceEvents()
	if len(events) == 0 || events[len(events)-1].Reason != "quota_bytes" {
		t.Fatalf("expected quota governance event, got %+v", events)
	}
}

func TestUploadServiceArchiveLifecycleTransition(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, scannerStub{result: ports.MalwareScanResult{Status: ports.MalwareStatusClean}}, UploadPolicy{RequestTimeout: time.Second})
	if err := svc.SetGovernancePolicy(GovernancePolicy{Default: TenantGovernancePolicy{ArchiveAfterDays: 1, ArchiveClass: "archive-cold"}, Tenants: map[string]TenantGovernancePolicy{}}); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	meta, err := svc.Upload(context.Background(), "/tenants/acme/docs/old.txt", []byte("payload"))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	svc.mu.Lock()
	m := svc.metadata[meta.Path]
	m.CreatedAt = time.Now().UTC().Add(-48 * time.Hour)
	svc.metadata[meta.Path] = m
	svc.mu.Unlock()

	reports, err := svc.CleanupLifecycle(context.Background())
	if err != nil {
		t.Fatalf("cleanup lifecycle: %v", err)
	}
	if reports["archive"].Deleted != 1 {
		t.Fatalf("expected archive transition report, got %+v", reports)
	}
	svc.mu.RLock()
	archived := svc.metadata[meta.Path]
	svc.mu.RUnlock()
	if archived.StorageClass != "archive-cold" || archived.ArchivedAt.IsZero() {
		t.Fatalf("expected archived metadata, got %+v", archived)
	}
}

func TestUploadServiceGovernanceDriftDetection(t *testing.T) {
	st := localstorage.New(t.TempDir())
	svc := NewUploadService(st, nil, UploadPolicy{RequestTimeout: time.Second})
	runtimePolicy := GovernancePolicy{Default: TenantGovernancePolicy{QuotaBytes: 10}, Tenants: map[string]TenantGovernancePolicy{}}
	if err := svc.SetGovernancePolicy(runtimePolicy); err != nil {
		t.Fatalf("set runtime policy: %v", err)
	}
	svc.SetGovernanceSource(GovernancePolicy{Default: TenantGovernancePolicy{QuotaBytes: 20}, Tenants: map[string]TenantGovernancePolicy{}}, "source-v1")
	if !svc.CheckGovernanceDrift("system") {
		t.Fatalf("expected drift")
	}
	effective := svc.EffectivePolicy("acme")
	if !effective.DriftDetected || effective.SourceVersion != "source-v1" {
		t.Fatalf("expected effective policy drift metadata, got %+v", effective)
	}
}
