package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
