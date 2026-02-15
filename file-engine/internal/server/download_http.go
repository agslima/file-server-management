package server

import (
	"io"
	"net/http"

	"github.com/example/file-engine/internal/auth"
	"github.com/example/file-engine/internal/authz"
)

func (h *HTTPServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	a, err := h.Verifier.ParseAuthContext(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rawPath := r.URL.Query().Get("path")
	if rawPath == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}

	normalizedPath, err := authz.NormalizePath(rawPath)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	if !auth.CanAccess(a, normalizedPath, auth.PermRead, h.ACLStore) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	rc, err := h.Storage.Open(r.Context(), normalizedPath)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = io.Copy(w, rc)
}
