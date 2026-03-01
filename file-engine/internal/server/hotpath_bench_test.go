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
	verifier, err := auth.NewJWTVerifier(secret, "", "", "")
	if err != nil {
		b.Fatalf("create jwt verifier: %v", err)
	}
	acl := auth.NewInMemoryACLStore()
	if err := acl.SetACL(auth.ACL{Path: "/tenants/acme", PrincipalID: "role:viewer", Permissions: map[auth.Permission]bool{auth.PermRead: true}}); err != nil {
		b.Fatalf("set read acl: %v", err)
	}
	st := localstorage.New(b.TempDir())
	if st == nil {
		b.Fatal("create local storage: returned nil")
	}
	if err := st.AtomicWrite(context.Background(), "/tenants/acme/docs/a.txt", bytes.NewBufferString("profile-me")); err != nil {
		b.Fatalf("seed local storage fixture: %v", err)
	}
	h := &HTTPServer{Verifier: verifier, ACLStore: acl, Storage: st}

	req := httptest.NewRequest(http.MethodGet, "/v1/objects:download?path=/tenants/acme/docs/a.txt", http.NoBody)
	req.Header.Set("Authorization", signedTokenBench(secret))

	b.ReportAllocs()
	b.ResetTimer()
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
	verifier, err := auth.NewJWTVerifier(secret, "", "", "")
	if err != nil {
		b.Fatalf("create jwt verifier: %v", err)
	}
	acl := auth.NewInMemoryACLStore()
	if err := acl.SetACL(auth.ACL{Path: "/tenants/acme", PrincipalID: "role:viewer", Permissions: map[auth.Permission]bool{auth.PermWrite: true}}); err != nil {
		b.Fatalf("set write acl: %v", err)
	}
	st := localstorage.New(b.TempDir())
	if st == nil {
		b.Fatal("create local storage: returned nil")
	}
	uploads := services.NewUploadService(st, adaptersecurity.NewMalwareScannerStub(), services.UploadPolicy{MaxObjectSizeBytes: 4096, TenantQuotaBytes: 1024 * 1024, RequestTimeout: time.Second, RequireCleanScan: true})
	h := &HTTPServer{Verifier: verifier, ACLStore: acl, Uploads: uploads, Tenants: auth.NewInMemoryTenantResolver(map[string][]string{"alice": {"acme"}}), MaxUploadBytes: 4096, UploadTimeout: time.Second, sem: make(chan struct{}, 4), rateByTenant: map[string]int{}, rateByActor: map[string]int{}, concurrentByTenant: map[string]int{}, concurrentByActor: map[string]int{}, rateReset: time.Now().Add(time.Minute)}

	b.ReportAllocs()
	b.ResetTimer()
	requestID := 0
	
	for b.Loop() {
		uploadID, err := uploads.StartResumableUpload("/tenants/acme/docs/profile.txt", "")
		if err != nil {
			b.Fatalf("start upload: %v", err)
		}
		if err := uploads.UploadChunk(uploadID, 0, []byte("hot-path")); err != nil {
			b.Fatalf("chunk upload: %v", err)
		}
		requestID++
		req := httptest.NewRequest(http.MethodPost, "/v1/uploads/"+uploadID+":complete", http.NoBody)
		req.Header.Set("X-Idempotency-Key", fmt.Sprintf("bench-complete-%d", requestID))
		req.Header.Set("Authorization", signedTokenBench(secret))
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
