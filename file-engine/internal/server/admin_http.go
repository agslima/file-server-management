package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/file-engine/internal/auth"
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
	rows, err := h.Identity.AccessReview(r.Context(), r.URL.Query().Get("tenant_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"memberships": rows})
}
