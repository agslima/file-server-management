package security

import (
	"fmt"
	"path"
	"strings"
)

func NormalizeTenantPath(raw string) (string, error) {
	p := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if p == "" {
		return "", fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", fmt.Errorf("path traversal denied")
		}
	}
	clean := path.Clean(p)
	if !strings.HasPrefix(clean, "/tenants/") {
		return "", fmt.Errorf("path must be tenant scoped")
	}
	parts := strings.Split(strings.TrimPrefix(clean, "/"), "/")
	if len(parts) < 3 || parts[1] == "" {
		return "", fmt.Errorf("invalid tenant path")
	}
	return clean, nil
}
