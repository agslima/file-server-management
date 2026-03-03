package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	adaptersecurity "github.com/example/file-engine/internal/adapters/security"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/services"
	pb "github.com/example/file-engine/pkg/generated"
	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func signedAdminToken(t *testing.T, secret string) string {
	t.Helper()
	tk := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, jwtgo.MapClaims{"sub": "alice", "roles": []string{"admin"}, "exp": time.Now().Add(time.Hour).Unix()})
	signed, err := tk.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return "Bearer " + signed
}

func TestCompatibilityReadyzGolden(t *testing.T) {
	h := &HTTPServer{}
	h.AddReadyCheck("queue", func(context.Context) error { return nil })
	req := httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody)
	rr := httptest.NewRecorder()
	h.handleReadyz(rr, req)
	assertMatchesFixture(t, rr.Body.Bytes(), "readyz_ok.json")
}

func TestCompatibilityAuthzDenyGolden(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	pb.RegisterFileEngineServer(grpcSrv, &fakeGatewayServer{})
	go func() { _ = grpcSrv.Serve(lis) }()
	t.Cleanup(func() { grpcSrv.Stop(); _ = lis.Close() })

	mux := runtime.NewServeMux(runtime.WithErrorHandler(gatewayErrorHandler))
	if err := pb.RegisterFileEngineHandlerFromEndpoint(context.Background(), mux, lis.Addr().String(), []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}); err != nil {
		t.Fatalf("register gateway: %v", err)
	}

	httpSrv := httptest.NewServer(mux)
	defer httpSrv.Close()
	req, _ := http.NewRequest(http.MethodPost, httpSrv.URL+"/v1/folders", bytes.NewBufferString(`{"parentPath":"/tenants/beta/projects","folderName":"reports","requestedBy":"user-1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-123")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	assertMatchesFixture(t, buf.Bytes(), "authz_deny.json")
}

func TestCompatibilityUploadLifecycleGolden(t *testing.T) {
	secret := "test-secret"
	verifier, _ := auth.NewJWTVerifier(secret, "", "", "")
	acl := auth.NewInMemoryACLStore()
	_ = acl.SetACL(auth.ACL{Path: "/tenants/acme", PrincipalID: "role:viewer", Permissions: map[auth.Permission]bool{auth.PermWrite: true}})
	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, adaptersecurity.NewMalwareScannerStub(), services.UploadPolicy{MaxObjectSizeBytes: 4096, TenantQuotaBytes: 10 * 1024, RequestTimeout: time.Second, RequireCleanScan: true})
	h := &HTTPServer{Verifier: verifier, ACLStore: acl, Uploads: uploads, Tenants: auth.NewInMemoryTenantResolver(map[string][]string{"alice": {"acme"}}), MaxUploadBytes: 4096, UploadTimeout: time.Second, sem: make(chan struct{}, 1), rateByTenant: map[string]int{}, rateByActor: map[string]int{}, rateReset: time.Now().Add(time.Minute)}

	initReq := httptest.NewRequest(http.MethodPost, "/v1/uploads:initiate", bytes.NewBufferString(`{"path":"/tenants/acme/docs/report.txt"}`))
	initReq.Header.Set("Authorization", signedToken(t, secret))
	initReq.Header.Set("Content-Type", "application/json")
	initReq.Header.Set("X-Request-Id", "req-compat-1")
	initRR := httptest.NewRecorder()
	h.handleUploadInitiate(initRR, initReq)
	var initBody map[string]any
	_ = json.NewDecoder(initRR.Body).Decode(&initBody)
	uploadID, _ := initBody["upload_id"].(string)
	initBody["upload_id"] = "<upload_id>"
	initBody["staging_token"] = "<upload_id>"
	initBody["upload_url"] = strings.ReplaceAll(initBody["upload_url"].(string), uploadID, "<upload_id>")
	initJSON, _ := json.Marshal(initBody)
	assertMatchesFixture(t, initJSON, "upload_initiate.json")

	chunkReq := httptest.NewRequest(http.MethodPut, "/v1/uploads/"+uploadID+":chunk?offset=0", bytes.NewBufferString("hello clean"))
	chunkReq.Header.Set("Authorization", signedToken(t, secret))
	chunkRR := httptest.NewRecorder()
	h.handleUploadChunk(chunkRR, chunkReq)
	if chunkRR.Code != http.StatusAccepted {
		t.Fatalf("expected chunk 202 got %d", chunkRR.Code)
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/v1/uploads/"+uploadID+":complete", http.NoBody)
	completeReq.Header.Set("Authorization", signedToken(t, secret))
	completeReq.Header.Set("X-Request-Id", "req-compat-1")
	completeRR := httptest.NewRecorder()
	h.handleUploadComplete(completeRR, completeReq)
	var completeBody map[string]any
	_ = json.NewDecoder(completeRR.Body).Decode(&completeBody)
	completeBody["upload_id"] = "<upload_id>"
	completeBody["stage_path"] = "<stage_path>"
	completeJSON, _ := json.Marshal(completeBody)
	assertMatchesFixture(t, completeJSON, "upload_complete.json")
}

func TestCompatibilityUploadThrottledGolden(t *testing.T) {
	secret := "test-secret"
	verifier, _ := auth.NewJWTVerifier(secret, "", "", "")
	acl := auth.NewInMemoryACLStore()
	_ = acl.SetACL(auth.ACL{Path: "/tenants/acme", PrincipalID: "role:viewer", Permissions: map[auth.Permission]bool{auth.PermWrite: true}})
	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, adaptersecurity.NewMalwareScannerStub(), services.UploadPolicy{MaxObjectSizeBytes: 4096, TenantQuotaBytes: 10 * 1024, RequestTimeout: time.Second, RequireCleanScan: true})
	h := &HTTPServer{Verifier: verifier, ACLStore: acl, Uploads: uploads, Tenants: auth.NewInMemoryTenantResolver(map[string][]string{"alice": {"acme"}}), MaxUploadBytes: 4096, UploadTimeout: time.Second, sem: make(chan struct{}, 1), rateByTenant: map[string]int{"acme": 40}, rateByActor: map[string]int{"alice": 20}, rateReset: time.Now().Add(time.Second)}

	req := httptest.NewRequest(http.MethodPost, "/v1/objects:upload?path=/tenants/acme/docs/report.txt", bytes.NewBufferString("hello"))
	req.Header.Set("Authorization", signedToken(t, secret))
	req.Header.Set("X-Request-Id", "req-compat-throttle")
	rr := httptest.NewRecorder()
	h.handleUpload(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 got %d body=%s", rr.Code, rr.Body.String())
	}
	assertMatchesFixture(t, rr.Body.Bytes(), "upload_throttled.json")
}

func TestCompatibilityGovernanceDeleteRetentionBlockGolden(t *testing.T) {
	secret := "test-secret"
	verifier, _ := auth.NewJWTVerifier(secret, "", "", "")
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
	req.Header.Set("Authorization", signedAdminToken(t, secret))
	rr := httptest.NewRecorder()
	h.handleGovernanceDelete(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 got %d body=%s", rr.Code, rr.Body.String())
	}
	assertTextFixture(t, rr.Body.Bytes(), "governance_delete_retention_block.txt")
}

func assertTextFixture(t *testing.T, got []byte, fixtureName string) {
	t.Helper()
	fixturePath := filepath.Join("testdata", "compat", fixtureName)
	wantRaw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixtureName, err)
	}
	if !bytes.Equal(got, wantRaw) {
		t.Fatalf("fixture mismatch for %s\nwant=%q\ngot=%q", fixtureName, string(wantRaw), string(got))
	}
}

func assertMatchesFixture(t *testing.T, got []byte, fixtureName string) {
	t.Helper()
	fixturePath := filepath.Join("testdata", "compat", fixtureName)
	wantRaw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixtureName, err)
	}
	var gotAny any
	if err := json.Unmarshal(got, &gotAny); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	var wantAny any
	if err := json.Unmarshal(wantRaw, &wantAny); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	gotCanon, _ := json.MarshalIndent(gotAny, "", "  ")
	wantCanon, _ := json.MarshalIndent(wantAny, "", "  ")
	if !bytes.Equal(gotCanon, wantCanon) {
		t.Fatalf("fixture mismatch for %s\nwant=%s\ngot=%s", fixtureName, string(wantCanon), string(gotCanon))
	}
}
