package local

import (
    "bytes"
    "context"
    "os"
    "path/filepath"
    "testing"
)

func TestLocalFs_CreateWriteMove(t *testing.T) {
    tmp := t.TempDir()
    lf, err := NewLocalFs(tmp)
    if err != nil {
        t.Fatalf("init localfs: %v", err)
    }
    ctx := context.Background()
    if err := lf.CreateFolder(ctx, "projects", "a"); err != nil {
        t.Fatalf("mkdir: %v", err)
    }
    // atomic write
    data := []byte("hello")
    if err := lf.AtomicWriteFile(ctx, 0o644, data, "projects", "a", "file.txt"); err != nil {
        t.Fatalf("atomic write failed: %v", err)
    }
    full := filepath.Join(tmp, "projects", "a", "file.txt")
    if _, err := os.Stat(full); err != nil {
        t.Fatalf("file missing: %v", err)
    }
    // move
    if err := lf.MoveUploadedFile(ctx, []string{"projects","a","file.txt"}, []string{"archive","file.txt"}); err != nil {
        t.Fatalf("move failed: %v", err)
    }
    if _, err := os.Stat(filepath.Join(tmp, "archive", "file.txt")); err != nil {
        t.Fatalf("moved missing: %v", err)
    }
}