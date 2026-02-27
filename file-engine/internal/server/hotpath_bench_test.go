package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"

	adaptersecurity "github.com/example/file-engine/internal/adapters/security"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/services"
)

func BenchmarkHandleDownload(b *testing.B) {
	secret := "test-secret"
	verifier, _ := auth.NewJWTVerifier(secret, "", "", "")
	acl := auth.NewInMemoryACLStore()
	_ = acl.SetACL(auth.ACL{Path: "/tenants/acme", PrincipalID: "role:viewer", Permissions: map[auth.Permission]bool{auth.PermRead: true}})
	st := localstorage.New(b.TempDir())
	_ = st.AtomicWrite(context.Background(), "/tenants/acme/docs/a.txt", bytes.NewBufferString("profile-me"))
	h := &HTTPServer{Verifier: verifier, ACLStore: acl, Storage: st}

	req := httptest.NewRequest(http.MethodGet, "/v1/objects:download?path=/tenants/acme/docs/a.txt", http.NoBody)
	req.Header.Set("Authorization", signedTokenBench(secret))

	b.ReportAllocs()
	for b.Loop() {
		rr := httptest.NewRecorder()
		h.handleDownload(rr, req.Clone(context.Background()))
		if rr.Code != http.StatusOK {
			b.Fatalf("expected 200 got %d", rr.Code)
		}
	}
}

func BenchmarkHandleUploadComplete(b *testing.B) {
	secret := "test-secret"
	verifier, _ := auth.NewJWTVerifier(secret, "", "", "")
	acl := auth.NewInMemoryACLStore()
	_ = acl.SetACL(auth.ACL{Path: "/tenants/acme", PrincipalID: "role:viewer", Permissions: map[auth.Permission]bool{auth.PermWrite: true}})
	st := localstorage.New(b.TempDir())
	uploads := services.NewUploadService(st, adaptersecurity.NewMalwareScannerStub(), services.UploadPolicy{MaxObjectSizeBytes: 4096, TenantQuotaBytes: 1024 * 1024, RequestTimeout: time.Second, RequireCleanScan: true})
	h := &HTTPServer{Verifier: verifier, ACLStore: acl, Uploads: uploads, Tenants: auth.NewInMemoryTenantResolver(map[string][]string{"alice": {"acme"}}), MaxUploadBytes: 4096, UploadTimeout: time.Second, sem: make(chan struct{}, 4), rateByTenant: map[string]int{}, rateByActor: map[string]int{}, rateReset: time.Now().Add(time.Minute)}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		uploadID, err := uploads.StartResumableUpload("/tenants/acme/docs/profile.txt", "")
		if err != nil {
			b.Fatalf("start upload: %v", err)
		}
		if err := uploads.UploadChunk(uploadID, 0, []byte("hot-path")); err != nil {
			b.Fatalf("chunk upload: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/v1/uploads/"+uploadID+":complete", http.NoBody)
		req.Header.Set("X-Idempotency-Key", "bench-complete")
		rr := httptest.NewRecorder()
		h.handleUploadComplete(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("expected 200 got %d body=%s", rr.Code, rr.Body.String())
		}
	}
}

func signedTokenBench(secret string) string {
	tk := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, jwtgo.MapClaims{
		"sub":   "alice",
		"roles": []string{"viewer"},
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	signed, err := tk.SignedString([]byte(secret))
	if err != nil {
		panic(fmt.Sprintf("sign token: %v", err))
	}
	return "Bearer " + signed
}
