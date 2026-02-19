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
	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/observability"
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
	log     *logger.Logger

	mu          sync.RWMutex
	metadata    map[string]UploadMetadata
	idempotency map[string]idempotencyRecord
}

type idempotencyRecord struct {
	Path       string
	Meta       UploadMetadata
	ErrMessage string
}

func NewUploadService(st storage.Storage, scanner ports.MalwareScanner, policy UploadPolicy) *UploadService {
	return NewUploadServiceWithLogger(st, scanner, policy, nil)
}

func NewUploadServiceWithLogger(st storage.Storage, scanner ports.MalwareScanner, policy UploadPolicy, logg *logger.Logger) *UploadService {
	if policy.RequestTimeout <= 0 {
		policy.RequestTimeout = 10 * time.Second
	}
	return &UploadService{st: st, scanner: scanner, policy: policy, log: logg, metadata: map[string]UploadMetadata{}, idempotency: map[string]idempotencyRecord{}}
}

func (s *UploadService) Upload(ctx context.Context, targetPath string, content []byte) (UploadMetadata, error) {
	return s.UploadStream(ctx, targetPath, bytes.NewReader(content), "")
}

func (s *UploadService) UploadStream(ctx context.Context, targetPath string, content io.Reader, idempotencyKey string) (UploadMetadata, error) {
	normalized, err := security.NormalizeTenantPath(targetPath)
	if err != nil {
		return UploadMetadata{}, err
	}

	if idempotencyKey != "" {
		s.mu.RLock()
		if r, ok := s.idempotency[idempotencyKey]; ok {
			s.mu.RUnlock()
			if r.Path != normalized {
				return UploadMetadata{}, errors.New("idempotency key already used for a different target path")
			}
			if r.ErrMessage != "" {
				return r.Meta, errors.New(r.ErrMessage)
			}
			return r.Meta, nil
		}
		s.mu.RUnlock()
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
		scanStart := time.Now()
		result, err := s.scanner.Scan(tctx, stagePath)
		scanDurationMs := time.Since(scanStart).Milliseconds()
		if err != nil {
			scanStatus = ports.MalwareStatusUnknown
			observability.DefaultMetrics.ObserveMalwareScanDurationMs(scanDurationMs)
			observability.DefaultMetrics.IncMalwareScanVerdict(string(scanStatus))
			if s.log != nil {
				s.log.Event("warn", "upload.scan.completed", map[string]any{"path": normalized, "stage_path": stagePath, "scan_duration_ms": scanDurationMs, "scan_verdict": scanStatus, "scan_engine": "", "scan_error": err.Error()})
			}
		} else {
			scanStatus = result.Status
			scannedBy = result.Engine
			observability.DefaultMetrics.ObserveMalwareScanDurationMs(scanDurationMs)
			observability.DefaultMetrics.IncMalwareScanVerdict(string(scanStatus))
			if s.log != nil {
				s.log.Event("info", "upload.scan.completed", map[string]any{"path": normalized, "stage_path": stagePath, "scan_duration_ms": scanDurationMs, "scan_verdict": scanStatus, "scan_engine": scannedBy})
			}
		}
	}
	if s.policy.RequireCleanScan && scanStatus != ports.MalwareStatusClean {
		meta := UploadMetadata{TenantID: tenantID, Path: normalized, StagePath: stagePath, Size: cr.n, Checksum: checksum, ScanStatus: ports.MalwareStatusQuarantined, ScannedBy: scannedBy, CreatedAt: time.Now().UTC()}
		s.storeMeta(meta, idempotencyKey, "malware scan gate blocked commit")
		return meta, errors.New("malware scan gate blocked commit")
	}

	if err := s.st.Move(tctx, stagePath, normalized); err != nil {
		return UploadMetadata{}, err
	}
	meta := UploadMetadata{TenantID: tenantID, Path: normalized, StagePath: stagePath, Size: cr.n, Checksum: checksum, ScanStatus: scanStatus, ScannedBy: scannedBy, CreatedAt: time.Now().UTC()}
	s.storeMeta(meta, idempotencyKey, "")
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

func (s *UploadService) storeMeta(m UploadMetadata, idempotencyKey, errMessage string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metadata[m.Path] = m
	if idempotencyKey != "" {
		s.idempotency[idempotencyKey] = idempotencyRecord{Path: m.Path, Meta: m, ErrMessage: errMessage}
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
