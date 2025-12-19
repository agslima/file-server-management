package fs

import "context"

type FileSystem interface {
    CreateFolder(ctx context.Context, parts ...string) error
    AtomicWriteFile(ctx context.Context, perm uint32, data []byte, parts ...string) error
    MoveUploadedFile(ctx context.Context, src []string, dst []string) error
    Exists(parts ...string) (bool, error)
}