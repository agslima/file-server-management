package local

import (
    "context"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
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
    joined := filepath.Join(parts...)
    clean := filepath.Clean(joined)
    if filepath.IsAbs(clean) {
        clean = strings.TrimPrefix(clean, string(os.PathSeparator))
    }
    full := filepath.Join(l.BaseRoot, clean)
    rel, err := filepath.Rel(l.BaseRoot, full)
    if err != nil {
        return "", err
    }
    if strings.HasPrefix(rel, "..") {
        return "", fmt.Errorf("outside base")
    }
    return full, nil
}

func (l *LocalFs) CreateFolder(ctx context.Context, parts ...string) error {
    full, err := l.sanitizeAndJoin(parts...)
    if err != nil {
        return err
    }
    return os.MkdirAll(full, 0o755)
}

func (l *LocalFs) AtomicWriteFile(ctx context.Context, perm uint32, data []byte, parts ...string) error {
    full, err := l.sanitizeAndJoin(parts...)
    if err != nil {
        return err
    }
    dir := filepath.Dir(full)
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return err
    }
    tmp, err := os.CreateTemp(dir, ".tmp-*")
    if err != nil {
        return err
    }
    if _, err := tmp.Write(data); err != nil {
        tmp.Close()
        return err
    }
    tmp.Close()
    if err := os.Chmod(tmp.Name(), os.FileMode(perm)); err != nil {
        return err
    }
    return os.Rename(tmp.Name(), full)
}

func (l *LocalFs) MoveUploadedFile(ctx context.Context, src []string, dst []string) error {
    srcF, err := l.sanitizeAndJoin(src...)
    if err != nil {
        return err
    }
    dstF, err := l.sanitizeAndJoin(dst...)
    if err != nil {
        return err
    }
    if err := os.MkdirAll(filepath.Dir(dstF), 0o755); err != nil {
        return err
    }
    if err := os.Rename(srcF, dstF); err == nil {
        return nil
    }
    // fallback copy
    in, err := os.Open(srcF)
    if err != nil {
        return err
    }
    defer in.Close()
    out, err := os.Create(dstF)
    if err != nil {
        return err
    }
    defer out.Close()
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