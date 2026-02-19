// Package contract defines reusable storage adapter contract tests.
package contract

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/example/file-engine/internal/storage"
)

type StorageFactory func(t *testing.T) storage.Storage

func RunStorageContractSuite(t *testing.T, newStorage StorageFactory) {
	t.Helper()

	t.Run("create_exists_open_delete", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		st := newStorage(t)

		if err := st.CreateFolder(ctx, "/tenants/acme/contracts"); err != nil {
			t.Fatalf("create folder: %v", err)
		}
		if err := st.AtomicWrite(ctx, "/tenants/acme/contracts/file.txt", bytes.NewBufferString("hello")); err != nil {
			t.Fatalf("atomic write: %v", err)
		}

		exists, err := st.Exists(ctx, "/tenants/acme/contracts/file.txt")
		if err != nil {
			t.Fatalf("exists after write: %v", err)
		}
		if !exists {
			t.Fatal("expected file to exist after write")
		}

		rc, err := st.Open(ctx, "/tenants/acme/contracts/file.txt")
		if err != nil {
			t.Fatalf("open written file: %v", err)
		}
		defer func() { _ = rc.Close() }()
		b, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read opened file: %v", err)
		}
		if string(b) != "hello" {
			t.Fatalf("expected file content hello, got %q", string(b))
		}

		if err := st.Delete(ctx, "/tenants/acme/contracts/file.txt"); err != nil {
			t.Fatalf("delete file: %v", err)
		}
		exists, err = st.Exists(ctx, "/tenants/acme/contracts/file.txt")
		if err != nil {
			t.Fatalf("exists after delete: %v", err)
		}
		if exists {
			t.Fatal("expected file to be removed")
		}
	})

	t.Run("move_preserves_content", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		st := newStorage(t)

		if err := st.CreateFolder(ctx, "/tenants/acme/contracts"); err != nil {
			t.Fatalf("create folder: %v", err)
		}
		if err := st.AtomicWrite(ctx, "/tenants/acme/contracts/source.txt", strings.NewReader("moved-content")); err != nil {
			t.Fatalf("write source: %v", err)
		}
		if err := st.Move(ctx, "/tenants/acme/contracts/source.txt", "/tenants/acme/contracts/dest.txt"); err != nil {
			t.Fatalf("move: %v", err)
		}

		srcExists, err := st.Exists(ctx, "/tenants/acme/contracts/source.txt")
		if err != nil {
			t.Fatalf("exists source after move: %v", err)
		}
		if srcExists {
			t.Fatal("expected source to be absent after move")
		}

		dstExists, err := st.Exists(ctx, "/tenants/acme/contracts/dest.txt")
		if err != nil {
			t.Fatalf("exists destination after move: %v", err)
		}
		if !dstExists {
			t.Fatal("expected destination to exist after move")
		}

		rc, err := st.Open(ctx, "/tenants/acme/contracts/dest.txt")
		if err != nil {
			t.Fatalf("open destination: %v", err)
		}
		defer func() { _ = rc.Close() }()
		b, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read destination: %v", err)
		}
		if string(b) != "moved-content" {
			t.Fatalf("expected moved content preserved, got %q", string(b))
		}
	})

	t.Run("list_includes_written_entries", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		st := newStorage(t)

		if err := st.CreateFolder(ctx, "/tenants/acme/contracts"); err != nil {
			t.Fatalf("create folder: %v", err)
		}
		if err := st.AtomicWrite(ctx, "/tenants/acme/contracts/a.txt", strings.NewReader("a")); err != nil {
			t.Fatalf("write a.txt: %v", err)
		}
		if err := st.AtomicWrite(ctx, "/tenants/acme/contracts/b.txt", strings.NewReader("b")); err != nil {
			t.Fatalf("write b.txt: %v", err)
		}

		items, err := st.List(ctx, "/tenants/acme/contracts")
		if err != nil {
			t.Fatalf("list contracts prefix: %v", err)
		}
		if len(items) < 2 {
			t.Fatalf("expected at least 2 entries, got %d (%+v)", len(items), items)
		}
		want := map[string]bool{
			"/tenants/acme/contracts/a.txt": false,
			"/tenants/acme/contracts/b.txt": false,
		}
		for _, item := range items {
			if _, ok := want[item.Path]; ok {
				want[item.Path] = true
			}
		}
		for path, found := range want {
			if !found {
				t.Fatalf("expected %s in list output, got %+v", path, items)
			}
		}
	})
}
