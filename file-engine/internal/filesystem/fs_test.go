package filesystem

import "testing"

func TestEnsurePathAddsSlash(t *testing.T) {
	got, err := EnsurePath("projects/demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/projects/demo" {
		t.Fatalf("expected /projects/demo, got %s", got)
	}
}

func TestEnsurePathRejectsEmpty(t *testing.T) {
	if _, err := EnsurePath(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
}

func TestCreateFolderRequiresPath(t *testing.T) {
	if err := CreateFolder("stub://server", ""); err == nil {
		t.Fatalf("expected error for empty path")
	}
}
