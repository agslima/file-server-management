package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ----- Public API -----
// baseRoot is the allowed root directory for operations (injected).
// Use NewLocalFs(base) to create an instance.

type LocalFs struct {
	BaseRoot string // absolute canonical base path
	// Optionally you can add logger, uid/gid mappings, umask handling, etc.
}

// NewLocalFs returns a LocalFs instance ensuring baseRoot is absolute and cleaned.
func NewLocalFs(baseRoot string) (*LocalFs, error) {
	if baseRoot == "" {
		return nil, errors.New("baseRoot is required")
	}
	abs, err := filepath.Abs(baseRoot)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve baseRoot: %w", err)
	}
	return &LocalFs{BaseRoot: filepath.Clean(abs)}, nil
}

// sanitizeAndJoin ensures requested path is inside the BaseRoot and returns the full path.
// `parts` are path segments relative to base root, e.g. "projects", "cliente-a"
func (l *LocalFs) sanitizeAndJoin(parts ...string) (string, error) {
	joined := filepath.Join(parts...)
	clean := filepath.Clean(joined)

	// Prevent absolute path in parts
	if filepath.IsAbs(clean) {
		// Make it relative (strip leading /)
		clean = strings.TrimPrefix(clean, string(os.PathSeparator))
	}

	full := filepath.Join(l.BaseRoot, clean)
	// Resolve symlinks to avoid escape via symlink tricks
	fullEval, err := filepath.EvalSymlinks(full)
	if err == nil {
		full = fullEval
	}
	// Ensure final path has base as prefix
	rel, err := filepath.Rel(l.BaseRoot, full)
	if err != nil {
		return "", fmt.Errorf("path outside base root: %w", err)
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path outside base root")
	}
	return full, nil
}

// CreateFolder creates the folder (mkdir -p style) with provided perm (e.g. 0755).
func (l *LocalFs) CreateFolder(ctx context.Context, relParts ...string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	full, err := l.sanitizeAndJoin(relParts...)
	if err != nil {
		return err
	}
	// Make directories with 0755 by default
	if err := os.MkdirAll(full, 0o755); err != nil {
		return fmt.Errorf("mkdir failed: %w", err)
	}
	return nil
}

// AtomicWriteFile writes the content from reader to target path atomically.
// targetParts are path segments relative to base root, last element is filename.
// perm is used for final file permissions (e.g. 0644).
func (l *LocalFs) AtomicWriteFile(ctx context.Context, perm os.FileMode, data io.Reader, targetParts ...string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if len(targetParts) == 0 {
		return errors.New("target path required")
	}

	targetFull, err := l.sanitizeAndJoin(targetParts...)
	if err != nil {
		return err
	}

	dir := filepath.Dir(targetFull)
	// ensure dir exists
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("ensure dir failed: %w", err)
	}

	// Create temp file in same dir (important for atomic rename)
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file failed: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		_ = os.Remove(tmpName) // best-effort cleanup
	}()

	// Stream copy
	if _, err := io.Copy(tmpFile, data); err != nil {
		return fmt.Errorf("write temp file failed: %w", err)
	}
	// fsync to ensure data durability (optional, best-effort)
	if f, ok := tmpFile.(*os.File); ok {
		_ = f.Sync()
	}

	// close before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file failed: %w", err)
	}

	// Set permission on tmp file before rename (some filesystems preserve)
	if err := os.Chmod(tmpName, perm); err != nil {
		// not fatal - just warn (here we return error)
		return fmt.Errorf("chmod temp file failed: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpName, targetFull); err != nil {
		return fmt.Errorf("atomic rename failed: %w", err)
	}
	return nil
}

// MoveUploadedFile moves a file from srcParts to dstParts.
// It first tries os.Rename; if it fails (cross-device etc), it falls back to copy+remove.
func (l *LocalFs) MoveUploadedFile(ctx context.Context, srcParts []string, dstParts []string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	srcFull, err := l.sanitizeAndJoin(srcParts...)
	if err != nil {
		return fmt.Errorf("invalid src: %w", err)
	}
	dstFull, err := l.sanitizeAndJoin(dstParts...)
	if err != nil {
		return fmt.Errorf("invalid dst: %w", err)
	}

	// Ensure dst dir exists
	dstDir := filepath.Dir(dstFull)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("ensure dst dir failed: %w", err)
	}

	// Try rename
	if err := os.Rename(srcFull, dstFull); err == nil {
		return nil
	}

	// rename failed — fallback to copy
	if err := copyFileContents(srcFull, dstFull); err != nil {
		return fmt.Errorf("copy fallback failed: %w", err)
	}
	// Remove src after successful copy
	if err := os.Remove(srcFull); err != nil {
		return fmt.Errorf("remove src after copy failed: %w", err)
	}
	return nil
}

// Exists checks whether a file or folder exists at relParts.
func (l *LocalFs) Exists(relParts ...string) (bool, error) {
	full, err := l.sanitizeAndJoin(relParts...)
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

// copyFileContents copies file contents from src to dst preserving file mode.
func copyFileContents(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src failed: %w", err)
	}
	defer in.Close()

	fi, err := in.Stat()
	if err != nil {
		return fmt.Errorf("stat src failed: %w", err)
	}

	tmpDst, err := os.CreateTemp(filepath.Dir(dst), ".tmpcopy-*")
	if err != nil {
		return fmt.Errorf("create tmp dst failed: %w", err)
	}
	defer func() {
		tmpDst.Close()
		_ = os.Remove(tmpDst.Name())
	}()

	if _, err := io.Copy(tmpDst, in); err != nil {
		return fmt.Errorf("copy data failed: %w", err)
	}
	if err := tmpDst.Sync(); err != nil {
		// best-effort
	}
	if err := tmpDst.Close(); err != nil {
		return fmt.Errorf("close tmp dst failed: %w", err)
	}

	// apply original mode
	if err := os.Chmod(tmpDst.Name(), fi.Mode()); err != nil {
		return fmt.Errorf("chmod tmp dst failed: %w", err)
	}

	// atomic rename to final dst
	if err := os.Rename(tmpDst.Name(), dst); err != nil {
		return fmt.Errorf("rename tmp to dst failed: %w", err)
	}
	return nil
}
