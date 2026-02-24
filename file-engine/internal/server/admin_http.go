package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/observability"
	"github.com/example/file-engine/internal/security"
)

func (h *HTTPServer) requireAdmin(w http.ResponseWriter, r *http.Request) (auth.AuthContext, bool) {
	if h.Verifier == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return auth.AuthContext{}, false
	}
	authCtx, err := h.Verifier.ParseAuthContext(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return auth.AuthContext{}, false
	}
	for _, role := range authCtx.Roles {
		if strings.EqualFold(role, "admin") || strings.EqualFold(role, "iam-admin") {
			return authCtx, true
		}
	}
	http.Error(w, "forbidden", http.StatusForbidden)
	return auth.AuthContext{}, false
}

func (h *HTTPServer) handleScanDLQ(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	if h.Uploads == nil {
		http.Error(w, "upload pipeline unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		entries := h.Uploads.ListScanDLQ(strings.EqualFold(r.URL.Query().Get("include_resolved"), "true"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"entries": entries})
	case http.MethodPost:
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		meta, err := h.Uploads.RetryScanDLQ(r.Context(), req.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusFailedDependency)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "retried", "path": meta.Path, "scan_status": meta.ScanStatus})
	case http.MethodDelete:
		var req struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if !h.Uploads.ResolveScanDLQ(req.ID) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPServer) handleQuarantineCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	if h.Uploads == nil {
		http.Error(w, "upload pipeline unavailable", http.StatusServiceUnavailable)
		return
	}
	ttlSeconds := int64(3600)
	if raw := strings.TrimSpace(r.URL.Query().Get("ttl_seconds")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			http.Error(w, "invalid ttl_seconds", http.StatusBadRequest)
			return
		}
		ttlSeconds = parsed
	}
	report, err := h.Uploads.CleanupQuarantine(r.Context(), time.Duration(ttlSeconds)*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"deleted": report.Deleted, "skipped": report.Skipped})
}

func (h *HTTPServer) handleAdminTenants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	if h.Identity == nil || !h.Identity.Ready() {
		http.Error(w, "identity store unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct{ ID, Name string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" || req.Name == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := h.Identity.CreateTenant(r.Context(), req.ID, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *HTTPServer) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	if h.Identity == nil || !h.Identity.Ready() {
		http.Error(w, "identity store unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct{ ID, Email, DisplayName string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := h.Identity.CreateUser(r.Context(), req.ID, req.Email, req.DisplayName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *HTTPServer) handleAdminMemberships(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	if h.Identity == nil || !h.Identity.Ready() {
		http.Error(w, "identity store unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct{ UserID, TenantID string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" || req.TenantID == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPost:
		if err := h.Identity.CreateMembership(r.Context(), req.UserID, req.TenantID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		if err := h.Identity.RevokeMembership(r.Context(), req.UserID, req.TenantID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPServer) handleAdminRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	if h.Identity == nil || !h.Identity.Ready() {
		http.Error(w, "identity store unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct{ UserID, TenantID, RoleID, Description string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RoleID == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := h.Identity.UpsertRole(r.Context(), req.RoleID, req.Description); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if req.UserID != "" && req.TenantID != "" {
		if err := h.Identity.AssignRole(r.Context(), req.UserID, req.TenantID, req.RoleID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *HTTPServer) handleGovernanceDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authCtx, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	if h.Uploads == nil {
		http.Error(w, "upload pipeline unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Path) == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := h.Uploads.DeleteObject(r.Context(), authCtx.EffectiveActorID(), req.Path); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *HTTPServer) handleObjectDelete(w http.ResponseWriter, r *http.Request) {
	h.handleGovernanceDelete(w, r)
}

func (h *HTTPServer) handleLifecycleCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	if h.Uploads == nil {
		http.Error(w, "upload pipeline unavailable", http.StatusServiceUnavailable)
		return
	}
	reports, err := h.Uploads.CleanupLifecycle(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"reports": reports})
}

func (h *HTTPServer) handleGovernanceEffective(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	if h.Uploads == nil {
		http.Error(w, "upload pipeline unavailable", http.StatusServiceUnavailable)
		return
	}
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.Uploads.EffectivePolicy(tenantID))
}

func (h *HTTPServer) handleGovernanceDriftCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authCtx, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	if h.Uploads == nil {
		http.Error(w, "upload pipeline unavailable", http.StatusServiceUnavailable)
		return
	}
	drift := h.Uploads.CheckGovernanceDrift(authCtx.EffectiveActorID())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"drift_detected": drift})
}

func (h *HTTPServer) handleAccessReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	if h.Identity == nil || !h.Identity.Ready() {
		http.Error(w, "identity store unavailable", http.StatusServiceUnavailable)
		return
	}
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	rows, err := h.Identity.AccessReview(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"schema_version": "access_review.v1",
		"generated_at":   time.Now().UTC().Format(time.RFC3339),
		"tenant_filter":  tenantID,
		"memberships":    rows,
	})
}

func (h *HTTPServer) handleObjectMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authCtx, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	if h.Uploads == nil {
		http.Error(w, "upload pipeline unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		SourcePath      string `json:"source_path"`
		DestinationPath string `json:"destination_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	req.SourcePath = strings.TrimSpace(req.SourcePath)
	req.DestinationPath = strings.TrimSpace(req.DestinationPath)
	if req.SourcePath == "" || req.DestinationPath == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	normalizedSourcePath, err := security.NormalizeTenantPath(req.SourcePath)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	normalizedDestinationPath, err := security.NormalizeTenantPath(req.DestinationPath)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	req.SourcePath = normalizedSourcePath
	req.DestinationPath = normalizedDestinationPath
	meta, err := h.Uploads.MoveObject(r.Context(), authCtx.EffectiveActorID(), req.SourcePath, req.DestinationPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "moved", "path": meta.Path})
}

func (h *HTTPServer) handleQuarantineRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authCtx, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	if h.Uploads == nil {
		http.Error(w, "upload pipeline unavailable", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Path           string `json:"path"`
		ForceReprocess bool   `json:"force_reprocess"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	req.Path = strings.TrimSpace(req.Path)
	if req.Path == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	normalizedPath, err := security.NormalizeTenantPath(req.Path)
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	req.Path = normalizedPath
	meta, err := h.Uploads.RestoreQuarantinedObject(r.Context(), authCtx.EffectiveActorID(), req.Path, req.ForceReprocess)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "restored", "path": meta.Path, "scan_status": meta.ScanStatus})
}

func (h *HTTPServer) handleTenantCostReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"tenants":      observability.DefaultMetrics.TenantUsageSnapshot(),
	})
}

func (h *HTTPServer) handleIntegrityVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authCtx, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	if h.Uploads == nil {
		http.Error(w, "upload pipeline unavailable", http.StatusServiceUnavailable)
		return
	}
	sampleSize := 10
	if raw := strings.TrimSpace(r.URL.Query().Get("sample_size")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			http.Error(w, "invalid sample_size", http.StatusBadRequest)
			return
		}
		sampleSize = v
	}
	report := h.Uploads.VerifyIntegritySample(r.Context(), authCtx.EffectiveActorID(), sampleSize)
	statusCode := http.StatusOK
	if report.Failed > 0 {
		statusCode = http.StatusConflict
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(report)
}
