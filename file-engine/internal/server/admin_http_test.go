package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	adaptersecurity "github.com/example/file-engine/internal/adapters/security"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/app/ports"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/services"
	jwtgo "github.com/golang-jwt/jwt/v5"
)

type failScanner struct{}

func (failScanner) Scan(context.Context, string) (ports.MalwareScanResult, error) {
	return ports.MalwareScanResult{}, context.DeadlineExceeded
}

func signAdminToken(t *testing.T, secret string) string {
	t.Helper()
	tk := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, jwtgo.MapClaims{"sub": "alice", "roles": []string{"admin"}, "exp": time.Now().Add(time.Hour).Unix()})
	signed, err := tk.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return "Bearer " + signed
}

func TestAdminTenantRequiresAuth(t *testing.T) {
	verifier, _ := auth.NewJWTVerifier("secret", "", "", "")
	h := &HTTPServer{Verifier: verifier}
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/tenants", strings.NewReader(`{"id":"acme","name":"Acme"}`))
	rr := httptest.NewRecorder()
	h.handleAdminTenants(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", rr.Code)
	}
}

func TestAdminTenantReturnsServiceUnavailableWithoutStore(t *testing.T) {
	verifier, _ := auth.NewJWTVerifier("secret", "", "", "")
	h := &HTTPServer{Verifier: verifier}
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/tenants", strings.NewReader(`{"id":"acme","name":"Acme"}`))
	req.Header.Set("Authorization", signAdminToken(t, "secret"))
	rr := httptest.NewRecorder()
	h.handleAdminTenants(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 got %d", rr.Code)
	}
}

func TestScanDLQListEndpoint(t *testing.T) {
	verifier, _ := auth.NewJWTVerifier("secret", "", "", "")
	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, failScanner{}, services.UploadPolicy{RequireCleanScan: true, RequestTimeout: 5 * time.Millisecond, ScanRetryAttempts: 1, ScanRetryBackoff: time.Millisecond})
	_, _ = uploads.Upload(context.Background(), "/tenants/acme/docs/a.txt", []byte("x"))
	h := &HTTPServer{Verifier: verifier, Uploads: uploads}

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/scan-dlq", http.NoBody)
	req.Header.Set("Authorization", signAdminToken(t, "secret"))
	rr := httptest.NewRecorder()
	h.handleScanDLQ(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "scan-dlq-") {
		t.Fatalf("expected dlq payload, got %s", rr.Body.String())
	}
}

func TestQuarantineCleanupEndpoint(t *testing.T) {
	verifier, _ := auth.NewJWTVerifier("secret", "", "", "")
	st := localstorage.New(t.TempDir())
	if err := st.AtomicWrite(context.Background(), "/quarantine/acme/old.bin", bytes.NewBufferString("x")); err != nil {
		t.Fatalf("seed: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	uploads := services.NewUploadService(st, nil, services.UploadPolicy{RequestTimeout: time.Second})
	h := &HTTPServer{Verifier: verifier, Uploads: uploads}

	req := httptest.NewRequest(http.MethodPost, "/admin/v1/quarantine:cleanup?ttl_seconds=1", http.NoBody)
	req.Header.Set("Authorization", signAdminToken(t, "secret"))
	rr := httptest.NewRecorder()
	h.handleQuarantineCleanup(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGovernanceDeleteEndpointBlockedByRetention(t *testing.T) {
	verifier, _ := auth.NewJWTVerifier("secret", "", "", "")
	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, nil, services.UploadPolicy{RequestTimeout: time.Second})
	if err := uploads.SetGovernancePolicy(services.GovernancePolicy{Default: services.TenantGovernancePolicy{RetentionSeconds: 3600}, Tenants: map[string]services.TenantGovernancePolicy{}}); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	if _, err := uploads.Upload(context.Background(), "/tenants/acme/docs/a.txt", []byte("x")); err != nil {
		t.Fatalf("upload seed: %v", err)
	}
	h := &HTTPServer{Verifier: verifier, Uploads: uploads}

	req := httptest.NewRequest(http.MethodPost, "/admin/v1/governance:delete", strings.NewReader(`{"path":"/tenants/acme/docs/a.txt"}`))
	req.Header.Set("Authorization", signAdminToken(t, "secret"))
	rr := httptest.NewRecorder()
	h.handleGovernanceDelete(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestLifecycleCleanupEndpoint(t *testing.T) {
	verifier, _ := auth.NewJWTVerifier("secret", "", "", "")
	st := localstorage.New(t.TempDir())
	if err := st.AtomicWrite(context.Background(), "/staging/acme/old.bin", bytes.NewBufferString("x")); err != nil {
		t.Fatalf("seed staging: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	uploads := services.NewUploadService(st, nil, services.UploadPolicy{RequestTimeout: time.Second})
	if err := uploads.SetGovernancePolicy(services.GovernancePolicy{Default: services.TenantGovernancePolicy{}, Tenants: map[string]services.TenantGovernancePolicy{}, Lifecycle: services.LifecyclePolicy{OrphanStagingTTLSeconds: 1}}); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	h := &HTTPServer{Verifier: verifier, Uploads: uploads}

	req := httptest.NewRequest(http.MethodPost, "/admin/v1/lifecycle:cleanup", http.NoBody)
	req.Header.Set("Authorization", signAdminToken(t, "secret"))
	rr := httptest.NewRecorder()
	h.handleLifecycleCleanup(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "staging") {
		t.Fatalf("expected staging report, got %s", rr.Body.String())
	}
}

func TestGovernanceEffectiveEndpoint(t *testing.T) {
	verifier, _ := auth.NewJWTVerifier("secret", "", "", "")
	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, nil, services.UploadPolicy{RequestTimeout: time.Second})
	if err := uploads.SetGovernancePolicy(services.GovernancePolicy{Default: services.TenantGovernancePolicy{ArchiveAfterDays: 30}, Tenants: map[string]services.TenantGovernancePolicy{"acme": {ArchiveAfterDays: 7}}}); err != nil {
		t.Fatalf("set policy: %v", err)
	}
	h := &HTTPServer{Verifier: verifier, Uploads: uploads}

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/governance:effective?tenant_id=acme", http.NoBody)
	req.Header.Set("Authorization", signAdminToken(t, "secret"))
	rr := httptest.NewRecorder()
	h.handleGovernanceEffective(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "archive_after_days") {
		t.Fatalf("expected effective policy payload, got %s", rr.Body.String())
	}
}

func TestGovernanceDriftCheckEndpoint(t *testing.T) {
	verifier, _ := auth.NewJWTVerifier("secret", "", "", "")
	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, nil, services.UploadPolicy{RequestTimeout: time.Second})
	if err := uploads.SetGovernancePolicy(services.GovernancePolicy{Default: services.TenantGovernancePolicy{QuotaBytes: 10}, Tenants: map[string]services.TenantGovernancePolicy{}}); err != nil {
		t.Fatalf("set runtime policy: %v", err)
	}
	uploads.SetGovernanceSource(services.GovernancePolicy{Default: services.TenantGovernancePolicy{QuotaBytes: 99}, Tenants: map[string]services.TenantGovernancePolicy{}}, "v1")
	h := &HTTPServer{Verifier: verifier, Uploads: uploads}

	req := httptest.NewRequest(http.MethodPost, "/admin/v1/governance:drift-check", http.NoBody)
	req.Header.Set("Authorization", signAdminToken(t, "secret"))
	rr := httptest.NewRecorder()
	h.handleGovernanceDriftCheck(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "drift_detected") {
		t.Fatalf("expected drift response payload, got %s", rr.Body.String())
	}
}

func TestObjectMoveEndpoint(t *testing.T) {
	verifier, _ := auth.NewJWTVerifier("secret", "", "", "")
	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, nil, services.UploadPolicy{RequestTimeout: time.Second})
	if _, err := uploads.Upload(context.Background(), "/tenants/acme/docs/a.txt", []byte("x")); err != nil {
		t.Fatalf("upload seed: %v", err)
	}
	h := &HTTPServer{Verifier: verifier, Uploads: uploads}

	req := httptest.NewRequest(http.MethodPost, "/v1/objects:move", strings.NewReader(`{"source_path":"/tenants/acme/docs/a.txt","destination_path":"/tenants/acme/docs/b.txt"}`))
	req.Header.Set("Authorization", signAdminToken(t, "secret"))
	rr := httptest.NewRecorder()
	h.handleObjectMove(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestQuarantineRestoreEndpoint(t *testing.T) {
	verifier, _ := auth.NewJWTVerifier("secret", "", "", "")
	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, adaptersecurity.NewMalwareScannerStub(), services.UploadPolicy{RequestTimeout: time.Second, RequireCleanScan: true})
	if _, err := uploads.Upload(context.Background(), "/tenants/acme/docs/eicar.txt", []byte("clean")); err == nil {
		t.Fatal("expected upload to quarantine")
	}
	h := &HTTPServer{Verifier: verifier, Uploads: uploads}

	req := httptest.NewRequest(http.MethodPost, "/admin/v1/quarantine:restore", strings.NewReader(`{"path":"/tenants/acme/docs/eicar.txt","force_reprocess":false}`))
	req.Header.Set("Authorization", signAdminToken(t, "secret"))
	rr := httptest.NewRecorder()
	h.handleQuarantineRestore(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
	}
}
