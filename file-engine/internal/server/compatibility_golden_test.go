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
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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
	chunkRR := httptest.NewRecorder()
	h.handleUploadChunk(chunkRR, chunkReq)
	if chunkRR.Code != http.StatusAccepted {
		t.Fatalf("expected chunk 202 got %d", chunkRR.Code)
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/v1/uploads/"+uploadID+":complete", http.NoBody)
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
