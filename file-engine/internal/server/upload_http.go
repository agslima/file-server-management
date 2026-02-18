package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/example/file-engine/internal/observability"

	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/authz"
	"github.com/example/file-engine/internal/security"
)

func (h *HTTPServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if h.Uploads == nil {
		http.Error(w, "upload pipeline unavailable", http.StatusServiceUnavailable)
		return
	}
	start := time.Now()
	ctx := r.Context()
	a, err := h.Verifier.ParseAuthContext(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	rawPath := r.URL.Query().Get("path")
	normalizedPath, err := security.NormalizeTenantPath(rawPath)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	if !auth.CanAccess(a, normalizedPath, auth.PermWrite, h.ACLStore) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	tenantID, err := authz.TenantFromPath(normalizedPath)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	tenantResolver := h.Tenants
	if tenantResolver == nil {
		tenantResolver = auth.NewDenyAllTenantResolver()
	}
	allowed, err := tenantResolver.UserHasTenant(ctx, a.UserID, tenantID)
	if err != nil {
		http.Error(w, "tenant resolution failed", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "tenant access denied", http.StatusForbidden)
		return
	}
	if !h.allowTenantRequest(tenantFromNormalizedPath(normalizedPath)) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	select {
	case h.sem <- struct{}{}:
		defer func() { <-h.sem }()
	default:
		http.Error(w, "too many concurrent uploads", http.StatusTooManyRequests)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.MaxUploadBytes)
	if h.UploadTimeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, h.UploadTimeout)
		defer cancel()
	}
	idk := r.Header.Get("X-Idempotency-Key")
	if _, err := h.Uploads.UploadStream(ctx, normalizedPath, r.Body, idk); err != nil {
		http.Error(w, err.Error(), http.StatusFailedDependency)
		return
	}
	observability.DefaultMetrics.ObserveUploadDurationMs(time.Since(start).Milliseconds())
	w.WriteHeader(http.StatusCreated)
}

func tenantFromNormalizedPath(p string) string {
	parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
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
