package local

import (
    "context"
    "io"
    "os"
    "path/filepath"

    "github.com/example/file-engine/internal/storage"
)

type LocalStorage struct {
    base string
}

func New(base string) *LocalStorage {
    return &LocalStorage{base: base}
}

func (l *LocalStorage) full(p string) string {
    // keep simple; production should also enforce traversal protections (already exists in other adapter in older code)
    clean := filepath.Clean("/" + p)
    return filepath.Join(l.base, clean)
}

func (l *LocalStorage) CreateFolder(ctx context.Context, path string) error {
    return os.MkdirAll(l.full(path), 0o755)
}

func (l *LocalStorage) AtomicWrite(ctx context.Context, path string, r io.Reader) error {
    full := l.full(path)
    dir := filepath.Dir(full)
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return err
    }
    tmp, err := os.CreateTemp(dir, ".tmp-*")
    if err != nil {
        return err
    }
    if _, err := io.Copy(tmp, r); err != nil {
        tmp.Close()
        return err
    }
    tmp.Close()
    return os.Rename(tmp.Name(), full)
}

func (l *LocalStorage) Move(ctx context.Context, src string, dst string) error {
    srcF := l.full(src)
    dstF := l.full(dst)
    if err := os.MkdirAll(filepath.Dir(dstF), 0o755); err != nil {
        return err
    }
    if err := os.Rename(srcF, dstF); err == nil {
        return nil
    }
    // fallback copy+delete
    in, err := os.Open(srcF)
    if err != nil {
        return err
    }
    defer in.Close()
    out, err := os.Create(dstF)
    if err != nil {
        return err
    }
    if _, err := io.Copy(out, in); err != nil {
        out.Close()
        return err
    }
    out.Close()
    return os.Remove(srcF)
}

func (l *LocalStorage) Delete(ctx context.Context, path string) error {
    return os.RemoveAll(l.full(path))
}

func (l *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
    _, err := os.Stat(l.full(path))
    if err == nil {
        return true, nil
    }
    if os.IsNotExist(err) {
        return false, nil
    }
    return false, err
}

func (l *LocalStorage) List(ctx context.Context, prefix string) ([]storage.ObjectInfo, error) {
    full := l.full(prefix)
    entries, err := os.ReadDir(full)
    if err != nil {
        return nil, err
    }
    out := make([]storage.ObjectInfo, 0, len(entries))
    for _, e := range entries {
        info, _ := e.Info()
        out = append(out, storage.ObjectInfo{
            Path: filepath.Join(prefix, e.Name()),
            Size: func() int64 { if info != nil { return info.Size() }; return 0 }(),
            IsDir: e.IsDir(),
        })
    }
    return out, nil
}

func (l *LocalStorage) Open(ctx context.Context, path string) (io.ReadCloser, error) {
    return os.Open(l.full(path))
}
