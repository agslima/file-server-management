package security

import (
	"errors"
	"path"
	"slices"
	"strings"
)

func NormalizeTenantPath(raw string) (string, error) {
	p := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if p == "" {
		return "", errors.New("path is required")
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if slices.Contains(strings.Split(p, "/"), "..") {
		return "", errors.New("path traversal denied")
	}
	clean := path.Clean(p)
	if !strings.HasPrefix(clean, "/tenants/") {
		return "", errors.New("path must be tenant scoped")
	}
	parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")
	if len(parts) < 3 || parts[1] == "" {
		return "", errors.New("invalid tenant path")
	}
	return clean, nil
}
