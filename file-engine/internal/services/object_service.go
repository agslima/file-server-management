package services

import (
    "bytes"
    "context"
    "io"

    "github.com/example/file-engine/internal/storage"
)

type ObjectService struct {
    st storage.Storage
}

func NewObjectService(st storage.Storage) *ObjectService {
    return &ObjectService{st: st}
}

func (s *ObjectService) List(ctx context.Context, prefix string) ([]storage.ObjectInfo, error) {
    return s.st.List(ctx, prefix)
}

func (s *ObjectService) Upload(ctx context.Context, path string, content []byte) error {
    return s.st.AtomicWrite(ctx, path, bytes.NewReader(content))
}

func (s *ObjectService) Open(ctx context.Context, path string) (io.ReadCloser, error) {
    return s.st.Open(ctx, path)
}
