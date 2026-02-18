package local

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	fsadapter "github.com/example/file-engine/internal/adapters/fs"
)

type LocalFs struct {
	BaseRoot string
}

func NewLocalFs(base string) (*LocalFs, error) {
	if base == "" {
		return nil, errors.New("base required")
	}
	abs, err := filepath.Abs(base)
	if err != nil {
		return nil, err
	}
	return &LocalFs{BaseRoot: filepath.Clean(abs)}, nil
}

func (l *LocalFs) sanitizeAndJoin(parts ...string) (string, error) {
	return fsadapter.SafeJoin(l.BaseRoot, parts...)
}

func (l *LocalFs) CreateFolder(_ context.Context, parts ...string) error {
	full, err := l.sanitizeAndJoin(parts...)
	if err != nil {
		return err
	}
	return os.MkdirAll(full, 0o750)
}

func (l *LocalFs) AtomicWriteFile(_ context.Context, perm uint32, data []byte, parts ...string) error {
	full, err := l.sanitizeAndJoin(parts...)
	if err != nil {
		return err
	}
	dir := filepath.Dir(full)
	_ = dir
	return fsadapter.AtomicWriteFile(full, os.FileMode(perm), bytes.NewReader(data))
}

func (l *LocalFs) MoveUploadedFile(_ context.Context, src, dst []string) error {
	srcF, err := l.sanitizeAndJoin(src...)
	if err != nil {
		return err
	}
	dstF, err := l.sanitizeAndJoin(dst...)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dstF), 0o750); err != nil {
		return err
	}
	if err := os.Rename(srcF, dstF); err == nil {
		return nil
	}
	// fallback copy
	in, err := os.Open(srcF) // #nosec G304 -- srcF is sanitized via SafeJoin
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dstF) // #nosec G304 -- dstF is sanitized via SafeJoin
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return os.Remove(srcF)
}

func (l *LocalFs) Exists(parts ...string) (bool, error) {
	full, err := l.sanitizeAndJoin(parts...)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(full)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
