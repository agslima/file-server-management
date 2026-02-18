package fsadapter

import "context"

// FileSystem defines the filesystem operations required by task processing.
// Implementations belong to adapter packages such as local or sftp.
type FileSystem interface {
	CreateFolder(ctx context.Context, parts ...string) error
	AtomicWriteFile(ctx context.Context, perm uint32, data []byte, parts ...string) error
	MoveUploadedFile(ctx context.Context, src, dst []string) error
	Exists(parts ...string) (bool, error)
}
