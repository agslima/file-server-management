package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/example/file-engine/internal/app/ports"
	"github.com/example/file-engine/internal/security"
	"github.com/example/file-engine/internal/storage"
)

type UploadPolicy struct {
	MaxObjectSizeBytes int64
	TenantQuotaBytes   int64
	TenantObjectLimit  int64
	RequestTimeout     time.Duration
	RequireCleanScan   bool
}

type UploadMetadata struct {
	TenantID   string
	Path       string
	StagePath  string
	Size       int64
	Checksum   string
	ScanStatus ports.MalwareScanStatus
	ScannedBy  string
	CreatedAt  time.Time
}

type UploadService struct {
	st      storage.Storage
	scanner ports.MalwareScanner
	policy  UploadPolicy

	mu          sync.RWMutex
	metadata    map[string]UploadMetadata
	idempotency map[string]UploadMetadata
}

func NewUploadService(st storage.Storage, scanner ports.MalwareScanner, policy UploadPolicy) *UploadService {
	if policy.RequestTimeout <= 0 {
		policy.RequestTimeout = 10 * time.Second
	}
	return &UploadService{st: st, scanner: scanner, policy: policy, metadata: map[string]UploadMetadata{}, idempotency: map[string]UploadMetadata{}}
}

func (s *UploadService) Upload(ctx context.Context, targetPath string, content []byte) (UploadMetadata, error) {
	return s.UploadStream(ctx, targetPath, bytes.NewReader(content), "")
}

func (s *UploadService) UploadStream(ctx context.Context, targetPath string, content io.Reader, idempotencyKey string) (UploadMetadata, error) {
	if idempotencyKey != "" {
		s.mu.RLock()
		if m, ok := s.idempotency[idempotencyKey]; ok {
			s.mu.RUnlock()
			return m, nil
		}
		s.mu.RUnlock()
	}

	normalized, err := security.NormalizeTenantPath(targetPath)
	if err != nil {
		return UploadMetadata{}, err
	}
	tenantID := tenantFromPath(normalized)
	if tenantID == "" {
		return UploadMetadata{}, errors.New("invalid tenant path")
	}
	if s.policy.TenantQuotaBytes > 0 {
		used, objects := s.tenantUsage(tenantID)
		if s.policy.TenantObjectLimit > 0 && objects >= s.policy.TenantObjectLimit {
			return UploadMetadata{}, errors.New("tenant object count limit exceeded")
		}
		if used >= s.policy.TenantQuotaBytes {
			return UploadMetadata{}, errors.New("tenant quota exceeded")
		}
	}

	tctx, cancel := context.WithTimeout(ctx, s.policy.RequestTimeout)
	defer cancel()

	stagePath := path.Join("/quarantine", tenantID, fmt.Sprintf("upload-%d.bin", time.Now().UTC().UnixNano()))
	h := sha256.New()
	cr := &countingReader{r: content}
	tr := io.TeeReader(cr, h)

	if err := s.st.AtomicWrite(tctx, stagePath, tr); err != nil {
		return UploadMetadata{}, err
	}
	if s.policy.MaxObjectSizeBytes > 0 && cr.n > s.policy.MaxObjectSizeBytes {
		_ = s.st.Delete(tctx, stagePath)
		return UploadMetadata{}, errors.New("max object size exceeded")
	}
	if s.policy.TenantQuotaBytes > 0 {
		used, _ := s.tenantUsage(tenantID)
		if used+cr.n > s.policy.TenantQuotaBytes {
			_ = s.st.Delete(tctx, stagePath)
			return UploadMetadata{}, errors.New("tenant quota exceeded")
		}
	}

	checksum := hex.EncodeToString(h.Sum(nil))
	scanStatus := ports.MalwareStatusUnknown
	scannedBy := ""
	if s.scanner != nil {
		result, err := s.scanner.Scan(tctx, stagePath)
		if err != nil {
			scanStatus = ports.MalwareStatusUnknown
		} else {
			scanStatus = result.Status
			scannedBy = result.Engine
		}
	}
	if s.policy.RequireCleanScan && scanStatus != ports.MalwareStatusClean {
		meta := UploadMetadata{TenantID: tenantID, Path: normalized, StagePath: stagePath, Size: cr.n, Checksum: checksum, ScanStatus: ports.MalwareStatusQuarantined, ScannedBy: scannedBy, CreatedAt: time.Now().UTC()}
		s.storeMeta(meta, idempotencyKey)
		return meta, errors.New("malware scan gate blocked commit")
	}

	if err := s.st.Move(tctx, stagePath, normalized); err != nil {
		return UploadMetadata{}, err
	}
	meta := UploadMetadata{TenantID: tenantID, Path: normalized, StagePath: stagePath, Size: cr.n, Checksum: checksum, ScanStatus: scanStatus, ScannedBy: scannedBy, CreatedAt: time.Now().UTC()}
	s.storeMeta(meta, idempotencyKey)
	return meta, nil
}

func (s *UploadService) VerifyIntegrity(path string, content []byte) error {
	sum := sha256.Sum256(content)
	return s.VerifyIntegrityHash(path, hex.EncodeToString(sum[:]))
}

func (s *UploadService) VerifyIntegrityHash(path, checksumHex string) error {
	normalized, err := security.NormalizeTenantPath(path)
	if err != nil {
		return err
	}
	s.mu.RLock()
	meta, ok := s.metadata[normalized]
	s.mu.RUnlock()
	if !ok || meta.Checksum == "" {
		return nil
	}
	if checksumHex != meta.Checksum {
		return errors.New("checksum mismatch")
	}
	return nil
}

func (s *UploadService) tenantUsage(tenantID string) (int64, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var total, count int64
	for _, m := range s.metadata {
		if m.TenantID == tenantID && m.ScanStatus != ports.MalwareStatusQuarantined {
			total += m.Size
			count++
		}
	}
	return total, count
}

func (s *UploadService) storeMeta(m UploadMetadata, idempotencyKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metadata[m.Path] = m
	if idempotencyKey != "" {
		s.idempotency[idempotencyKey] = m
	}
}

func tenantFromPath(p string) string {
	parts := strings.Split(strings.TrimPrefix(p, "/"), "/")
	if len(parts) < 2 || parts[0] != "tenants" {
		return ""
	}
	return parts[1]
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
