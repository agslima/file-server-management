package authz

import (
    "fmt"
    "strings"

    pb "github.com/example/file-engine/pkg/generated"
)

func normalize(p string) string {
    p = strings.TrimSpace(p)
    if p == "" {
        return "/"
    }
    if !strings.HasPrefix(p, "/") {
        p = "/" + p
    }
    for strings.Contains(p, "//") {
        p = strings.ReplaceAll(p, "//", "/")
    }
    return p
}

// ExtractPath extracts the relevant path/prefix from an RPC request.
func ExtractPath(req any) (string, error) {
    switch r := req.(type) {
    case *pb.ListObjectsRequest:
        return normalize(r.Prefix), nil
    case *pb.UploadObjectRequest:
        return normalize(r.Path), nil
    case *pb.DownloadObjectRequest:
        return normalize(r.Path), nil
    case *pb.CreateFolderRequest:
        parent := strings.TrimSuffix(r.ParentPath, "/")
        return normalize(parent + "/" + r.FolderName), nil
    default:
        return "", fmt.Errorf("no path extractor for %T", req)
    }
}
