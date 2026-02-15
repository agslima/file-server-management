package authz

import (
	"testing"

	pb "github.com/example/file-engine/pkg/generated"
)

func TestExtractPathNormalizesCreateFolder(t *testing.T) {
	p, err := ExtractPath(&pb.CreateFolderRequest{ParentPath: "tenants/acme/projects//", FolderName: "reports"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != "/tenants/acme/projects/reports" {
		t.Fatalf("unexpected path: %s", p)
	}
}

func TestExtractPathRejectsTraversal(t *testing.T) {
	_, err := ExtractPath(&pb.CreateFolderRequest{ParentPath: "/tenants/acme/projects", FolderName: "../escape"})
	if err == nil {
		t.Fatalf("expected traversal error")
	}
}

func TestTenantFromPath(t *testing.T) {
	tenant, err := TenantFromPath("/tenants/acme/projects/reports")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tenant != "acme" {
		t.Fatalf("expected tenant acme, got %s", tenant)
	}
}

func TestTenantFromPathRejectsNonTenantRoot(t *testing.T) {
	_, err := TenantFromPath("/public/shared")
	if err == nil {
		t.Fatalf("expected tenant scope error")
	}
}
