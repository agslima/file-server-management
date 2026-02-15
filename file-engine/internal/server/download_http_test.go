package server

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/storage"
	jwt "github.com/golang-jwt/jwt/v5"
)

type fakeStorage struct {
	openedPath string
}

func (f *fakeStorage) CreateFolder(context.Context, string) error                 { return nil }
func (f *fakeStorage) AtomicWrite(context.Context, string, io.Reader) error       { return nil }
func (f *fakeStorage) Move(context.Context, string, string) error                 { return nil }
func (f *fakeStorage) Delete(context.Context, string) error                       { return nil }
func (f *fakeStorage) Exists(context.Context, string) (bool, error)               { return false, nil }
func (f *fakeStorage) List(context.Context, string) ([]storage.ObjectInfo, error) { return nil, nil }
func (f *fakeStorage) Open(_ context.Context, path string) (io.ReadCloser, error) {
	f.openedPath = path
	return io.NopCloser(bytes.NewBufferString("ok")), nil
}

func signedToken(t *testing.T, secret string) string {
	t.Helper()
	tk := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "alice",
		"roles": []string{"viewer"},
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	signed, err := tk.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return "Bearer " + signed
}

func TestHandleDownloadNormalizesPath(t *testing.T) {
	secret := "test-secret"
	verifier, err := auth.NewJWTVerifier(secret, "", "", "")
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	acl := auth.NewInMemoryACLStore()
	_ = acl.SetACL(auth.ACL{Path: "/tenants/acme", PrincipalID: "role:viewer", Permissions: map[auth.Permission]bool{auth.PermRead: true}})
	st := &fakeStorage{}
	h := &HTTPServer{Verifier: verifier, ACLStore: acl, Storage: st}

	req := httptest.NewRequest(http.MethodGet, "/v1/objects:download?path=\\tenants\\acme\\docs\\q1.txt", nil)
	req.Header.Set("Authorization", signedToken(t, secret))
	rr := httptest.NewRecorder()

	h.handleDownload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if st.openedPath != "/tenants/acme/docs/q1.txt" {
		t.Fatalf("expected normalized storage path, got %q", st.openedPath)
	}
}

func TestHandleDownloadRejectsTraversal(t *testing.T) {
	secret := "test-secret"
	verifier, _ := auth.NewJWTVerifier(secret, "", "", "")
	h := &HTTPServer{Verifier: verifier, ACLStore: auth.NewInMemoryACLStore(), Storage: &fakeStorage{}}

	req := httptest.NewRequest(http.MethodGet, "/v1/objects:download?path=/tenants/acme/../secrets", nil)
	req.Header.Set("Authorization", signedToken(t, secret))
	rr := httptest.NewRecorder()

	h.handleDownload(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
