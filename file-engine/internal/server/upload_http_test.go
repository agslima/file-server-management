package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	adaptersecurity "github.com/example/file-engine/internal/adapters/security"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/services"
)

func TestHandleUploadStreamsAndCreatesObject(t *testing.T) {
	secret := "test-secret"
	verifier, _ := auth.NewJWTVerifier(secret, "", "", "")
	acl := auth.NewInMemoryACLStore()
	_ = acl.SetACL(auth.ACL{Path: "/tenants/acme", PrincipalID: "role:viewer", Permissions: map[auth.Permission]bool{auth.PermWrite: true}})

	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, adaptersecurity.NewMalwareScannerStub(), services.UploadPolicy{MaxObjectSizeBytes: 1024, TenantQuotaBytes: 10 * 1024, RequestTimeout: time.Second, RequireCleanScan: false})
	h := &HTTPServer{Verifier: verifier, ACLStore: acl, Uploads: uploads, MaxUploadBytes: 1024, UploadTimeout: time.Second, sem: make(chan struct{}, 1), rateByTenant: map[string]int{}, rateReset: time.Now().Add(time.Minute)}

	req := httptest.NewRequest(http.MethodPost, "/v1/objects:upload?path=/tenants/acme/docs/a.txt", bytes.NewBufferString("hello"))
	req.Header.Set("Authorization", signedToken(t, secret))
	rr := httptest.NewRecorder()
	h.handleUpload(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", rr.Code)
	}
}

func TestHandleUploadRejectsRateLimit(t *testing.T) {
	secret := "test-secret"
	verifier, _ := auth.NewJWTVerifier(secret, "", "", "")
	acl := auth.NewInMemoryACLStore()
	_ = acl.SetACL(auth.ACL{Path: "/tenants/acme", PrincipalID: "role:viewer", Permissions: map[auth.Permission]bool{auth.PermWrite: true}})
	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, adaptersecurity.NewMalwareScannerStub(), services.UploadPolicy{MaxObjectSizeBytes: 1024, TenantQuotaBytes: 10 * 1024, RequestTimeout: time.Second, RequireCleanScan: false})
	h := &HTTPServer{Verifier: verifier, ACLStore: acl, Uploads: uploads, MaxUploadBytes: 1024, UploadTimeout: time.Second, sem: make(chan struct{}, 1), rateByTenant: map[string]int{"acme": 120}, rateReset: time.Now().Add(time.Minute)}

	req := httptest.NewRequest(http.MethodPost, "/v1/objects:upload?path=/tenants/acme/docs/a.txt", bytes.NewBufferString("hello"))
	req.Header.Set("Authorization", signedToken(t, secret))
	rr := httptest.NewRecorder()
	h.handleUpload(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 got %d", rr.Code)
	}
}
