package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"

	"github.com/example/file-engine/internal/storage"
)

type ObjectService struct {
	st        storage.Storage
	mu        sync.RWMutex
	checksums map[string]string
}

func NewObjectService(st storage.Storage) *ObjectService {
	return &ObjectService{st: st, checksums: map[string]string{}}
}

func (s *ObjectService) List(ctx context.Context, prefix string) ([]storage.ObjectInfo, error) {
	return s.st.List(ctx, prefix)
}

func (s *ObjectService) Upload(ctx context.Context, path string, content []byte) error {
	if err := s.st.AtomicWrite(ctx, path, bytes.NewReader(content)); err != nil {
		return err
	}
	sum := sha256.Sum256(content)
	s.mu.Lock()
	s.checksums[path] = hex.EncodeToString(sum[:])
	s.mu.Unlock()
	return nil
}

func (s *ObjectService) Checksum(path string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.checksums[path]
	return v, ok
}

func (s *ObjectService) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.st.Open(ctx, path)
}
