package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	adaptersecurity "github.com/example/file-engine/internal/adapters/security"
	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/services"
)

type uploadAuditSpy struct {
	events []string
}

func (s *uploadAuditSpy) EmitTaskEvent(_ context.Context, event, _, _, _ string, _ ...map[string]string) {
	s.events = append(s.events, event)
}

func TestUploadLifecycleEndpointsCleanAndDirty(t *testing.T) {
	secret := "test-secret"
	verifier, _ := auth.NewJWTVerifier(secret, "", "", "")
	acl := auth.NewInMemoryACLStore()
	_ = acl.SetACL(auth.ACL{Path: "/tenants/acme", PrincipalID: "role:viewer", Permissions: map[auth.Permission]bool{auth.PermWrite: true}})

	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, adaptersecurity.NewMalwareScannerStub(), services.UploadPolicy{MaxObjectSizeBytes: 4096, TenantQuotaBytes: 10 * 1024, RequestTimeout: time.Second, RequireCleanScan: true})
	audit := &uploadAuditSpy{}
	h := &HTTPServer{Verifier: verifier, ACLStore: acl, Uploads: uploads, UploadAuditor: audit, Tenants: auth.NewInMemoryTenantResolver(map[string][]string{"alice": {"acme"}}), MaxUploadBytes: 4096, UploadTimeout: time.Second, sem: make(chan struct{}, 1), rateByTenant: map[string]int{}, rateReset: time.Now().Add(time.Minute)}

	initReq := httptest.NewRequest(http.MethodPost, "/v1/uploads:initiate", bytes.NewBufferString(`{"path":"/tenants/acme/docs/report.txt"}`))
	initReq.Header.Set("Authorization", signedToken(t, secret))
	initReq.Header.Set("Content-Type", "application/json")
	initReq.Header.Set("X-Idempotency-Key", "init-1")
	initReq.Header.Set("X-Request-Id", "req-upload-1")
	initRR := httptest.NewRecorder()
	h.handleUploadInitiate(initRR, initReq)
	if initRR.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", initRR.Code, initRR.Body.String())
	}
	var initBody map[string]any
	if err := json.NewDecoder(initRR.Body).Decode(&initBody); err != nil {
		t.Fatalf("decode initiate: %v", err)
	}
	uploadID, _ := initBody["upload_id"].(string)
	if uploadID == "" {
		t.Fatalf("missing upload_id in initiate response")
	}

	chunkReq := httptest.NewRequest(http.MethodPut, "/v1/uploads/"+uploadID+":chunk?offset=0", bytes.NewBufferString("hello clean"))
	chunkRR := httptest.NewRecorder()
	h.handleUploadChunk(chunkRR, chunkReq)
	if chunkRR.Code != http.StatusAccepted {
		t.Fatalf("expected 202 got %d body=%s", chunkRR.Code, chunkRR.Body.String())
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/v1/uploads/"+uploadID+":complete", http.NoBody)
	completeReq.Header.Set("X-Idempotency-Key", "complete-1")
	completeReq.Header.Set("X-Request-Id", "req-upload-1")
	completeRR := httptest.NewRecorder()
	h.handleUploadComplete(completeRR, completeReq)
	if completeRR.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", completeRR.Code, completeRR.Body.String())
	}
	var completeBody map[string]any
	if err := json.NewDecoder(completeRR.Body).Decode(&completeBody); err != nil {
		t.Fatalf("decode complete: %v", err)
	}
	if completeBody["scan_status"] != "clean" {
		t.Fatalf("expected clean scan status, got %v", completeBody["scan_status"])
	}

	// Dirty path (stub scanner detects "eicar" in staged path when clean scan is required).
	dirtyInit := httptest.NewRequest(http.MethodPost, "/v1/uploads:initiate", bytes.NewBufferString(`{"path":"/tenants/acme/docs/eicar.txt"}`))
	dirtyInit.Header.Set("Authorization", signedToken(t, secret))
	dirtyInit.Header.Set("Content-Type", "application/json")
	dirtyInitRR := httptest.NewRecorder()
	h.handleUploadInitiate(dirtyInitRR, dirtyInit)
	if dirtyInitRR.Code != http.StatusOK {
		t.Fatalf("dirty initiate expected 200 got %d body=%s", dirtyInitRR.Code, dirtyInitRR.Body.String())
	}
	var dirtyInitBody map[string]any
	_ = json.NewDecoder(dirtyInitRR.Body).Decode(&dirtyInitBody)
	dirtyUploadID, _ := dirtyInitBody["upload_id"].(string)

	dirtyChunk := httptest.NewRequest(http.MethodPut, "/v1/uploads/"+dirtyUploadID+":chunk?offset=0", bytes.NewBufferString("virus"))
	dirtyChunkRR := httptest.NewRecorder()
	h.handleUploadChunk(dirtyChunkRR, dirtyChunk)
	if dirtyChunkRR.Code != http.StatusAccepted {
		t.Fatalf("dirty chunk expected 202 got %d", dirtyChunkRR.Code)
	}

	dirtyComplete := httptest.NewRequest(http.MethodPost, "/v1/uploads/"+dirtyUploadID+":complete", http.NoBody)
	dirtyCompleteRR := httptest.NewRecorder()
	h.handleUploadComplete(dirtyCompleteRR, dirtyComplete)
	if dirtyCompleteRR.Code != http.StatusForbidden {
		t.Fatalf("dirty complete expected 403 got %d body=%s", dirtyCompleteRR.Code, dirtyCompleteRR.Body.String())
	}

	if len(audit.events) == 0 {
		t.Fatalf("expected upload audit events")
	}
}

func TestUploadInitiateIdempotencyAndCompleteReplay(t *testing.T) {
	st := localstorage.New(t.TempDir())
	uploads := services.NewUploadService(st, adaptersecurity.NewMalwareScannerStub(), services.UploadPolicy{MaxObjectSizeBytes: 4096, TenantQuotaBytes: 10 * 1024, RequestTimeout: time.Second, RequireCleanScan: true})
	session1, err := uploads.StartResumableUpload("/tenants/acme/docs/a.txt", "same")
	if err != nil {
		t.Fatalf("start first session: %v", err)
	}
	session2, err := uploads.StartResumableUpload("/tenants/acme/docs/a.txt", "same")
	if err != nil {
		t.Fatalf("start second session: %v", err)
	}
	if session1 != session2 {
		t.Fatalf("expected same session for idempotent initiate, got %q and %q", session1, session2)
	}
	if _, err := uploads.StartResumableUpload("/tenants/acme/docs/b.txt", "same"); err == nil {
		t.Fatalf("expected initiate idempotency conflict")
	}

	if err := uploads.UploadChunk(session1, 0, []byte("hello")); err != nil {
		t.Fatalf("upload chunk: %v", err)
	}
	meta, err := uploads.FinalizeResumableUpload(context.Background(), session1, "complete-key")
	if err != nil {
		t.Fatalf("finalize first: %v", err)
	}
	replayMeta, err := uploads.UploadStream(context.Background(), meta.Path, bytes.NewReader([]byte("different")), "complete-key")
	if err != nil {
		t.Fatalf("expected idempotent complete replay to return original result: %v", err)
	}
	if replayMeta.Checksum != meta.Checksum {
		t.Fatalf("expected replay checksum %q got %q", meta.Checksum, replayMeta.Checksum)
	}
}
