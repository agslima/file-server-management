package fsadapter

import (
	"errors"
	"path/filepath"
	"strings"
)

func SafeJoin(base string, parts ...string) (string, error) {
	joined := filepath.Join(parts...)
	clean := filepath.Clean(joined)
	if filepath.IsAbs(clean) {
		clean = strings.TrimPrefix(clean, string(filepath.Separator))
	}
	full := filepath.Join(base, clean)
	rel, err := filepath.Rel(base, full)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") {
		return "", errors.New("outside base")
	}
	return full, nil
}
