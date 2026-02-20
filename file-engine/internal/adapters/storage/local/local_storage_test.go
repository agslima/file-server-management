package local

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/example/file-engine/internal/storage"
)

func TestLocalStorageListMetadata(t *testing.T) {
	root := t.TempDir()
	st := New(root)

	start := time.Now().Add(-time.Second)
	ctx := context.Background()

	if err := st.CreateFolder(ctx, "/tenants/acme/projects"); err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if err := st.AtomicWrite(ctx, "/tenants/acme/projects/report.txt", bytes.NewBufferString("ok")); err != nil {
		t.Fatalf("write file: %v", err)
	}

	items, err := st.List(ctx, "/tenants/acme/projects")
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	var file *storage.ObjectInfo
	for i := range items {
		if items[i].Path == "/tenants/acme/projects/report.txt" {
			file = &items[i]
			break
		}
	}
	if file == nil {
		t.Fatalf("expected file entry, got %+v", items)
	}
	if file.IsDir {
		t.Fatalf("expected file entry, got dir: %+v", file)
	}
	if file.Size != 2 {
		t.Fatalf("expected size 2, got %d", file.Size)
	}
	if file.ModifiedAt.IsZero() || file.ModifiedAt.Before(start) {
		t.Fatalf("expected modified_at after %v, got %v", start, file.ModifiedAt)
	}
	if file.CreatedAt.IsZero() || file.CreatedAt.Before(start) {
		t.Fatalf("expected created_at after %v, got %v", start, file.CreatedAt)
	}
	if file.Owner != strconv.Itoa(os.Getuid()) {
		t.Fatalf("expected owner %d, got %q", os.Getuid(), file.Owner)
	}
	if file.Group != strconv.Itoa(os.Getgid()) {
		t.Fatalf("expected group %d, got %q", os.Getgid(), file.Group)
	}
	if file.Checksum == "" {
		t.Fatalf("expected checksum to be populated for local files")
	}
}
