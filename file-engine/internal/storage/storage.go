package storage

import (
    "context"
    "io"
)

// Storage is an abstraction over a file-like backend.
//
// Semantics:
// - Paths are always POSIX-like ("/a/b/c") even if backend is object storage.
// - CreateFolder creates a logical folder (for object storage, it may create a placeholder prefix object).
// - AtomicWrite writes content atomically: either full object exists at target, or it doesn't.
// - Move performs an atomic/consistent move when possible; for object storage it typically becomes Copy+Delete.
type Storage interface {
    CreateFolder(ctx context.Context, path string) error
    AtomicWrite(ctx context.Context, path string, r io.Reader) error
    Move(ctx context.Context, src string, dst string) error
    Delete(ctx context.Context, path string) error
    Exists(ctx context.Context, path string) (bool, error)
    List(ctx context.Context, prefix string) ([]ObjectInfo, error)
    Open(ctx context.Context, path string) (io.ReadCloser, error)
}

type ObjectInfo struct {
    Path string
    Size int64
    IsDir bool
}
