package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"sync"
	"testing"
	"time"

	localstorage "github.com/example/file-engine/internal/adapters/storage/local"
	"github.com/example/file-engine/internal/services"
	"github.com/example/file-engine/internal/storage"
)

type observableStagingStorage struct {
	inner *localstorage.LocalStorage

	mu        sync.Mutex
	stagePath string

	stageStarted  chan struct{}
	allowComplete chan struct{}
}

func newObservableStagingStorage(base string) *observableStagingStorage {
	return &observableStagingStorage{
		inner:         localstorage.New(base),
		stageStarted:  make(chan struct{}, 1),
		allowComplete: make(chan struct{}),
	}
}

func (s *observableStagingStorage) AtomicWrite(ctx context.Context, path string, r io.Reader) error {
	if len(path) >= len("/quarantine/") && path[:len("/quarantine/")] == "/quarantine/" {
		buf := make([]byte, 3)
		n, err := r.Read(buf)
		if n > 0 {
			if err := s.inner.AtomicWrite(ctx, path, bytes.NewReader(buf[:n])); err != nil {
				return err
			}
			s.mu.Lock()
			s.stagePath = path
			s.mu.Unlock()
			select {
			case s.stageStarted <- struct{}{}:
			default:
			}
			<-s.allowComplete
		}
		if err != nil && err != io.EOF {
			return err
		}
		remainder, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		full := append(buf[:n], remainder...)
		return s.inner.AtomicWrite(ctx, path, bytes.NewReader(full))
	}
	return s.inner.AtomicWrite(ctx, path, r)
}

func (s *observableStagingStorage) CreateFolder(ctx context.Context, path string) error {
	return s.inner.CreateFolder(ctx, path)
}

func (s *observableStagingStorage) Move(ctx context.Context, src, dst string) error {
	return s.inner.Move(ctx, src, dst)
}

func (s *observableStagingStorage) Delete(ctx context.Context, path string) error {
	return s.inner.Delete(ctx, path)
}

func (s *observableStagingStorage) Exists(ctx context.Context, path string) (bool, error) {
	return s.inner.Exists(ctx, path)
}

func (s *observableStagingStorage) List(ctx context.Context, prefix string) ([]storage.ObjectInfo, error) {
	return s.inner.List(ctx, prefix)
}

func (s *observableStagingStorage) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.inner.Open(ctx, path)
}

func (s *observableStagingStorage) currentStagePath() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stagePath
}

func TestStagedUploadAtomicPromote(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	st := newObservableStagingStorage(t.TempDir())
	service := services.NewUploadService(st, nil, services.UploadPolicy{RequestTimeout: 5 * time.Second})
	objectService := services.NewObjectService(st)

	targetPath := "/tenants/acme/docs/staged.txt"
	payload := []byte("0123456789abcdef")
	expectedHashRaw := sha256.Sum256(payload)
	expectedHash := hex.EncodeToString(expectedHashRaw[:])

	resultCh := make(chan struct {
		meta services.UploadMetadata
		err  error
	}, 1)
	go func() {
		meta, err := service.UploadStream(ctx, targetPath, bytes.NewReader(payload), "")
		resultCh <- struct {
			meta services.UploadMetadata
			err  error
		}{meta: meta, err: err}
	}()

	select {
	case <-st.stageStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for staging to begin")
	}

	stagePath := st.currentStagePath()
	if stagePath == "" {
		t.Fatal("expected stage path to be captured")
	}

	finalExists, err := st.Exists(ctx, targetPath)
	if err != nil {
		t.Fatalf("check final exists mid-stream: %v", err)
	}
	if finalExists {
		t.Fatalf("final object %q should not exist during staging", targetPath)
	}

	stageExists, err := st.Exists(ctx, stagePath)
	if err != nil {
		t.Fatalf("check stage exists mid-stream: %v", err)
	}
	if !stageExists {
		t.Fatalf("stage file %q should exist during staging", stagePath)
	}

	close(st.allowComplete)

	var outcome struct {
		meta services.UploadMetadata
		err  error
	}
	select {
	case outcome = <-resultCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for upload completion")
	}

	if outcome.err != nil {
		t.Fatalf("upload failed: %v", outcome.err)
	}

	if outcome.meta.Path != targetPath {
		t.Fatalf("expected final path %q, got %q", targetPath, outcome.meta.Path)
	}
	if outcome.meta.Checksum != expectedHash {
		t.Fatalf("expected checksum %q, got %q", expectedHash, outcome.meta.Checksum)
	}
	if outcome.meta.Size != int64(len(payload)) {
		t.Fatalf("expected size %d, got %d", len(payload), outcome.meta.Size)
	}

	finalExists, err = st.Exists(ctx, targetPath)
	if err != nil {
		t.Fatalf("check final exists after promote: %v", err)
	}
	if !finalExists {
		t.Fatalf("final object %q should exist after promote", targetPath)
	}

	stageExists, err = st.Exists(ctx, stagePath)
	if err != nil {
		t.Fatalf("check stage removed after promote: %v", err)
	}
	if stageExists {
		t.Fatalf("stage file %q should be removed after promote", stagePath)
	}

	items, err := objectService.List(ctx, "/tenants/acme/docs")
	if err != nil {
		t.Fatalf("list final directory: %v", err)
	}
	found := false
	for _, item := range items {
		if item.Path == targetPath {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected list to include %q, got %+v", targetPath, items)
	}

	rc, err := objectService.Open(ctx, targetPath)
	if err != nil {
		t.Fatalf("open final object: %v", err)
	}
	defer func() { _ = rc.Close() }()

	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read final object: %v", err)
	}
	if !bytes.Equal(content, payload) {
		t.Fatalf("expected final payload %q, got %q", string(payload), string(content))
	}
}
