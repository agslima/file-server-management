package filesystem

import (
    "errors"
    "fmt"
    "os"
)

// This package contains helpers to interact with SMB/SFTP/NFS servers.
// Currently stubs for future implementation.

// CreateFolder creates a folder on the remote server (stub).
func CreateFolder(serverURL, path string) error {
    // TODO: implement SMB/SFTP/NFS logic
    if path == "" {
        return errors.New("path empty")
    }
    // Example: log or call external tools
    fmt.Println("[stub] CreateFolder on", serverURL, "->", path)
    return nil
}

// MoveUploadedFile moves a file from temporary storage to the final location (stub).
func MoveUploadedFile(serverURL, src, dst string) error {
    // TODO: implement move logic (with checks, perms, scanning result)
    fmt.Println("[stub] MoveUploadedFile", src, "->", dst)
    return nil
}

// EnsurePath validates and sanitizes a path.
func EnsurePath(p string) (string, error) {
    if p == "" {
        return "", errors.New("empty path")
    }
    // Basic sanitization (stub) - real implementation must be robust.
    if p[0] != '/' {
        p = "/" + p
    }
    return p, nil
}