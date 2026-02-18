package authz

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	pb "github.com/example/file-engine/pkg/generated"
)

func normalize(p string) (string, error) {
	p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
	if p == "" {
		return "/", nil
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	if slices.Contains(strings.Split(p, "/"), "..") {
		return "", errors.New("invalid path traversal")
	}
	clean := path.Clean(p)
	if clean == "." {
		clean = "/"
	}
	return clean, nil
}

// NormalizePath enforces canonical slash handling and traversal rejection.
func NormalizePath(p string) (string, error) {
	return normalize(p)
}

func TenantFromPath(p string) (string, error) {
	n, err := normalize(p)
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.TrimPrefix(n, "/"), "/")
	if len(parts) < 2 || parts[0] != "tenants" || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("path must be under /tenants/<tenant_id>")
	}
	return parts[1], nil
}

// ExtractPath extracts the relevant path/prefix from an RPC request.
func ExtractPath(req any) (string, error) {
	switch r := req.(type) {
	case *pb.ListObjectsRequest:
		return normalize(r.Prefix)
	case *pb.UploadObjectRequest:
		return normalize(r.Path)
	case *pb.DownloadObjectRequest:
		return normalize(r.Path)
	case *pb.InitiateUploadRequest:
		return normalize(r.TargetPath)
	case *pb.CompleteUploadRequest:
		return "/", nil
	case *pb.TaskStatusRequest:
		return "/", nil
	case *pb.GetTaskRequest:
		return "/", nil
	case *pb.CreateFolderRequest:
		parent := strings.TrimSuffix(r.ParentPath, "/")
		return normalize(parent + "/" + r.FolderName)
	default:
		return "", fmt.Errorf("no path extractor for %T", req)
	}
}
