package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/authz"
	"github.com/example/file-engine/internal/observability"
	"github.com/example/file-engine/internal/security"
)

type UploadAuditEmitter interface {
	EmitTaskEvent(ctx context.Context, event, taskID, correlationID, message string, metadata ...map[string]string)
}

type uploadAuthContext struct {
	authCtx       auth.AuthContext
	normalized    string
	tenantID      string
	correlationID string
}

func (h *HTTPServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if h.Uploads == nil {
		h.writeUploadError(w, r, http.StatusServiceUnavailable, "upload_unavailable", "upload pipeline unavailable", "")
		return
	}
	start := time.Now()
	ctx := r.Context()
	authn, err := h.authorizeUploadRequest(r, r.URL.Query().Get("path"))
	if err != nil {
		h.mapUploadAuthError(w, r, err)
		return
	}
	ctx = context.WithValue(ctx, uploadCorrelationIDKey{}, authn.correlationID)

	if !h.allowTenantRequest(authn.tenantID) {
		h.writeUploadError(w, r, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded", authn.tenantID)
		return
	}
	select {
	case h.sem <- struct{}{}:
		defer func() { <-h.sem }()
	default:
		h.writeUploadError(w, r, http.StatusTooManyRequests, "concurrency_limited", "too many concurrent uploads", authn.tenantID)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.MaxUploadBytes)
	if h.UploadTimeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, h.UploadTimeout)
		defer cancel()
	}
	idk := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	if _, err := h.Uploads.UploadStream(ctx, authn.normalized, r.Body, idk); err != nil {
		h.writeUploadError(w, r, mapUploadServiceErrorToStatus(err), "upload_failed", err.Error(), authn.tenantID)
		return
	}
	h.emitUploadAudit(ctx, "upload.direct.completed", authn.correlationID, authn.normalized, authn.tenantID)
	observability.DefaultMetrics.ObserveUploadDurationMs(time.Since(start).Milliseconds())
	w.WriteHeader(http.StatusCreated)
}

func (h *HTTPServer) handleUploadInitiate(w http.ResponseWriter, r *http.Request) {
	if h.Uploads == nil {
		h.writeUploadError(w, r, http.StatusServiceUnavailable, "upload_unavailable", "upload pipeline unavailable", "")
		return
	}
	if r.Method != http.MethodPost {
		h.writeUploadError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeUploadError(w, r, http.StatusBadRequest, "invalid_request", "invalid json body", "")
		return
	}
	authn, err := h.authorizeUploadRequest(r, req.Path)
	if err != nil {
		h.mapUploadAuthError(w, r, err)
		return
	}
	idk := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	uploadID, err := h.Uploads.StartResumableUpload(authn.normalized, idk)
	if err != nil {
		h.writeUploadError(w, r, mapUploadServiceErrorToStatus(err), "initiate_failed", err.Error(), authn.tenantID)
		return
	}
	h.emitUploadAudit(r.Context(), "upload.initiated", authn.correlationID, authn.normalized, authn.tenantID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"upload_id":      uploadID,
		"upload_url":     "/v1/uploads/" + uploadID + ":chunk",
		"staging_token":  uploadID,
		"target_path":    authn.normalized,
		"correlation_id": authn.correlationID,
	})
}

func (h *HTTPServer) handleUploadChunk(w http.ResponseWriter, r *http.Request) {
	if h.Uploads == nil {
		h.writeUploadError(w, r, http.StatusServiceUnavailable, "upload_unavailable", "upload pipeline unavailable", "")
		return
	}
	if r.Method != http.MethodPut {
		h.writeUploadError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	uploadID := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/uploads/"), ":chunk"))
	if uploadID == "" {
		h.writeUploadError(w, r, http.StatusBadRequest, "invalid_request", "upload id is required", "")
		return
	}
	offset := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			h.writeUploadError(w, r, http.StatusBadRequest, "invalid_request", "offset must be a non-negative integer", "")
			return
		}
		offset = parsed
	}
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, h.MaxUploadBytes))
	if err != nil {
		h.writeUploadError(w, r, http.StatusBadRequest, "invalid_request", "failed to read upload bytes", "")
		return
	}
	if err := h.Uploads.UploadChunk(uploadID, offset, payload); err != nil {
		h.writeUploadError(w, r, mapUploadServiceErrorToStatus(err), "chunk_failed", err.Error(), "")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *HTTPServer) handleUploadComplete(w http.ResponseWriter, r *http.Request) {
	if h.Uploads == nil {
		h.writeUploadError(w, r, http.StatusServiceUnavailable, "upload_unavailable", "upload pipeline unavailable", "")
		return
	}
	if r.Method != http.MethodPost {
		h.writeUploadError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	uploadID := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/uploads/"), ":complete"))
	if uploadID == "" {
		var req struct {
			UploadID string `json:"uploadId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.UploadID) == "" {
			h.writeUploadError(w, r, http.StatusBadRequest, "invalid_request", "upload id is required", "")
			return
		}
		uploadID = strings.TrimSpace(req.UploadID)
	}
	idk := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	meta, err := h.Uploads.FinalizeResumableUpload(r.Context(), uploadID, idk)
	if err != nil {
		h.writeUploadError(w, r, mapUploadServiceErrorToStatus(err), "complete_failed", err.Error(), meta.TenantID)
		return
	}
	correlationID := requestCorrelationID(r)
	h.emitUploadAudit(r.Context(), "upload.completed", correlationID, meta.Path, meta.TenantID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"upload_id":      uploadID,
		"path":           meta.Path,
		"stage_path":     meta.StagePath,
		"scan_status":    string(meta.ScanStatus),
		"checksum":       meta.Checksum,
		"correlation_id": correlationID,
		"status":         "completed",
	})
}

type uploadCorrelationIDKey struct{}

func (h *HTTPServer) authorizeUploadRequest(r *http.Request, rawPath string) (uploadAuthContext, error) {
	a, err := h.Verifier.ParseAuthContext(r.Header.Get("Authorization"))
	if err != nil {
		return uploadAuthContext{}, errors.New("unauthorized")
	}
	normalizedPath, err := security.NormalizeTenantPath(rawPath)
	if err != nil {
		return uploadAuthContext{}, errors.New("invalid path")
	}
	if !auth.CanAccess(a, normalizedPath, auth.PermWrite, h.ACLStore) {
		return uploadAuthContext{}, errors.New("forbidden")
	}
	tenantID, err := authz.TenantFromPath(normalizedPath)
	if err != nil {
		return uploadAuthContext{}, errors.New("invalid path")
	}
	tenantResolver := h.Tenants
	if tenantResolver == nil {
		tenantResolver = auth.NewDenyAllTenantResolver()
	}
	allowed, err := tenantResolver.UserHasTenant(r.Context(), a.UserID, tenantID)
	if err != nil {
		return uploadAuthContext{}, errors.New("tenant resolution failed")
	}
	if !allowed {
		return uploadAuthContext{}, errors.New("tenant access denied")
	}
	return uploadAuthContext{authCtx: a, normalized: normalizedPath, tenantID: tenantID, correlationID: requestCorrelationID(r)}, nil
}

func (h *HTTPServer) mapUploadAuthError(w http.ResponseWriter, r *http.Request, err error) {
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch msg {
	case "unauthorized":
		h.writeUploadError(w, r, http.StatusUnauthorized, "unauthorized", "unauthorized", "")
	case "forbidden", "tenant access denied":
		h.writeUploadError(w, r, http.StatusForbidden, "tenant_mapping_denied", msg, "")
	case "tenant resolution failed":
		h.writeUploadError(w, r, http.StatusInternalServerError, "tenant_resolution_failed", msg, "")
	default:
		h.writeUploadError(w, r, http.StatusBadRequest, "invalid_request", msg, "")
	}
}

func (h *HTTPServer) writeUploadError(w http.ResponseWriter, r *http.Request, statusCode int, reason, message, tenantID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	requestID := requestCorrelationID(r)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":           "UPLOAD_ERROR",
			"reason":         reason,
			"message":        message,
			"tenant_id":      tenantID,
			"request_id":     requestID,
			"correlation_id": requestID,
		},
	})
}

func mapUploadServiceErrorToStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not found"):
		return http.StatusNotFound
	case strings.Contains(msg, "idempotency key"):
		return http.StatusConflict
	case strings.Contains(msg, "offset"):
		return http.StatusConflict
	case strings.Contains(msg, "quota"), strings.Contains(msg, "rate limit"):
		return http.StatusTooManyRequests
	case strings.Contains(msg, "malware"):
		return http.StatusForbidden
	default:
		return http.StatusFailedDependency
	}
}

func requestCorrelationID(r *http.Request) string {
	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID != "" {
		return requestID
	}
	correlationID := strings.TrimSpace(r.Header.Get("X-Correlation-Id"))
	if correlationID != "" {
		return correlationID
	}
	return "req-unknown"
}

func (h *HTTPServer) emitUploadAudit(ctx context.Context, event, correlationID, path, tenantID string) {
	if h.UploadAuditor == nil {
		return
	}
	h.UploadAuditor.EmitTaskEvent(ctx, event, "", correlationID, path, map[string]string{
		"tenant_id": tenantID,
		"path":      path,
	})
}

func (h *HTTPServer) allowTenantRequest(tenant string) bool {
	h.rateMu.Lock()
	defer h.rateMu.Unlock()
	now := time.Now()
	if now.After(h.rateReset) {
		h.rateByTenant = map[string]int{}
		h.rateReset = now.Add(time.Minute)
	}
	h.rateByTenant[tenant]++
	return h.rateByTenant[tenant] <= 120
}
