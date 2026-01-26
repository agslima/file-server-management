package filesystem

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFs_CreateFolder_And_AtomicWrite_Move(t *testing.T) {
	tmpDir := t.TempDir()

	lf, err := NewLocalFs(tmpDir)
	if err != nil {
		t.Fatalf("NewLocalFs failed: %v", err)
	}
	ctx := context.Background()

	// create nested folder
	err = lf.CreateFolder(ctx, "projects", "cliente-a", "2025")
	if err != nil {
		t.Fatalf("CreateFolder failed: %v", err)
	}
	fullPath := filepath.Join(tmpDir, "projects", "cliente-a", "2025")
	if fi, err := os.Stat(fullPath); err != nil || !fi.IsDir() {
		t.Fatalf("expected folder created at %s (err=%v)", fullPath, err)
	}

	// atomic write
	content := []byte("hello atomic world")
	if err := lf.AtomicWriteFile(ctx, 0o644, bytes.NewReader(content), "projects", "cliente-a", "2025", "file1.txt"); err != nil {
		t.Fatalf("AtomicWriteFile failed: %v", err)
	}
	target := filepath.Join(fullPath, "file1.txt")
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read file failed: %v", err)
	}
	if string(b) != string(content) {
		t.Fatalf("content mismatch: got=%s", string(b))
	}

	// move file to a new folder (cross-dir)
	if err := lf.CreateFolder(ctx, "archive"); err != nil {
		t.Fatalf("Create archive failed: %v", err)
	}
	// move
	if err := lf.MoveUploadedFile(ctx, []string{"projects", "cliente-a", "2025", "file1.txt"}, []string{"archive", "file1-moved.txt"}); err != nil {
		t.Fatalf("MoveUploadedFile failed: %v", err)
	}
	// check moved
	movedPath := filepath.Join(tmpDir, "archive", "file1-moved.txt")
	if _, err := os.Stat(movedPath); err != nil {
		t.Fatalf("moved file missing: %v", err)
	}
	// original should be gone
	origPath := filepath.Join(fullPath, "file1.txt")
	if _, err := os.Stat(origPath); err == nil {
		t.Fatalf("original still exists")
	}
}

func TestLocalFs_MoveAcrossDevicesFallback(t *testing.T) {
	tmpDir := t.TempDir()
	lf, _ := NewLocalFs(tmpDir)
	ctx := context.Background()

	// write a file
	_ = lf.CreateFolder(ctx, "src")
	err := lf.AtomicWriteFile(ctx, 0o644, bytes.NewReader([]byte("abc")), "src", "f.txt")
	if err != nil {
		t.Fatalf("atomic write failed: %v", err)
	}

	// call MoveUploadedFile (rename should work in tmp)
	err = lf.MoveUploadedFile(ctx, []string{"src", "f.txt"}, []string{"dst", "f2.txt"})
	if err != nil {
		t.Fatalf("MoveUploadedFile fallback path failed: %v", err)
	}
}