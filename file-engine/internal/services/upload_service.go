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
	ScanRetryAttempts  int
	ScanRetryBackoff   time.Duration
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

type ScanDLQEntry struct {
	ID         string    `json:"id"`
	Path       string    `json:"path"`
	StagePath  string    `json:"stage_path"`
	Reason     string    `json:"reason"`
	Attempts   int       `json:"attempts"`
	LastError  string    `json:"last_error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	ResolvedAt time.Time `json:"resolved_at,omitempty"`
	Resolved   bool      `json:"resolved"`
}

type CleanupReport struct {
	Deleted int `json:"deleted"`
	Skipped int `json:"skipped"`
}

type UploadService struct {
	st      storage.Storage
	scanner ports.MalwareScanner
	policy  UploadPolicy
	log     *logger.Logger

	mu          sync.RWMutex
	metadata    map[string]UploadMetadata
	idempotency map[string]idempotencyRecord
	dlq         map[string]ScanDLQEntry
	dlqSeq      int64
	resumable   map[string]*resumableUpload
}

type resumableUpload struct {
	TargetPath string
	Buffer     bytes.Buffer
	Offset     int64
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
	if policy.ScanRetryAttempts <= 0 {
		policy.ScanRetryAttempts = 3
	}
	if policy.ScanRetryBackoff <= 0 {
		policy.ScanRetryBackoff = 200 * time.Millisecond
	}
	return &UploadService{st: st, scanner: scanner, policy: policy, log: logg, metadata: map[string]UploadMetadata{}, idempotency: map[string]idempotencyRecord{}, dlq: map[string]ScanDLQEntry{}, resumable: map[string]*resumableUpload{}}
}

func (s *UploadService) StartResumableUpload(targetPath string) (string, error) {
	normalized, err := security.NormalizeTenantPath(targetPath)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dlqSeq++
	id := fmt.Sprintf("resumable-%d", s.dlqSeq)
	s.resumable[id] = &resumableUpload{TargetPath: normalized}
	return id, nil
}

func (s *UploadService) UploadChunk(sessionID string, offset int64, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.resumable[sessionID]
	if !ok {
		return errors.New("resumable session not found")
	}
	if offset != r.Offset {
		return errors.New("invalid chunk offset")
	}
	n, err := r.Buffer.Write(data)
	r.Offset += int64(n)
	return err
}

func (s *UploadService) FinalizeResumableUpload(ctx context.Context, sessionID, idempotencyKey string) (UploadMetadata, error) {
	s.mu.Lock()
	r, ok := s.resumable[sessionID]
	if !ok {
		s.mu.Unlock()
		return UploadMetadata{}, errors.New("resumable session not found")
	}
	payload := append([]byte(nil), r.Buffer.Bytes()...)
	target := r.TargetPath
	delete(s.resumable, sessionID)
	s.mu.Unlock()
	return s.UploadStream(ctx, target, bytes.NewReader(payload), idempotencyKey)
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
	scanAttempts := 0
	scanErrMessage := ""
	if s.scanner != nil {
		scanStart := time.Now()
		result, attempts, err := s.scanWithRetry(tctx, stagePath)
		scanAttempts = attempts
		scanDurationMs := time.Since(scanStart).Milliseconds()
		if err != nil {
			scanStatus = ports.MalwareStatusUnknown
			scanErrMessage = err.Error()
			observability.DefaultMetrics.ObserveMalwareScanDurationMs(scanDurationMs)
			observability.DefaultMetrics.IncMalwareScanVerdict(string(scanStatus))
			if s.log != nil {
				s.log.Event("warn", "upload.scan.completed", map[string]any{"path": normalized, "stage_path": stagePath, "scan_duration_ms": scanDurationMs, "scan_verdict": scanStatus, "scan_engine": "", "scan_error": err.Error(), "scan_attempts": attempts})
			}
		} else {
			scanStatus = result.Status
			scannedBy = result.Engine
			observability.DefaultMetrics.ObserveMalwareScanDurationMs(scanDurationMs)
			observability.DefaultMetrics.IncMalwareScanVerdict(string(scanStatus))
			if s.log != nil {
				s.log.Event("info", "upload.scan.completed", map[string]any{"path": normalized, "stage_path": stagePath, "scan_duration_ms": scanDurationMs, "scan_verdict": scanStatus, "scan_engine": scannedBy, "scan_attempts": attempts})
			}
		}
	}
	if s.policy.RequireCleanScan && scanStatus != ports.MalwareStatusClean {
		meta := UploadMetadata{TenantID: tenantID, Path: normalized, StagePath: stagePath, Size: cr.n, Checksum: checksum, ScanStatus: ports.MalwareStatusQuarantined, ScannedBy: scannedBy, CreatedAt: time.Now().UTC()}
		s.mu.Lock()
		if scanErrMessage != "" {
			s.enqueueScanDLQLocked(meta, "scanner_error", scanAttempts, scanErrMessage)
		}
		s.metadata[meta.Path] = meta
		if idempotencyKey != "" {
			s.idempotency[idempotencyKey] = idempotencyRecord{Path: meta.Path, Meta: meta, ErrMessage: "malware scan gate blocked commit"}
		}
		s.updateOperationalMetricsLocked()
		s.mu.Unlock()
		return meta, errors.New("malware scan gate blocked commit")
	}

	if err := s.st.Move(tctx, stagePath, normalized); err != nil {
		return UploadMetadata{}, err
	}
	meta := UploadMetadata{TenantID: tenantID, Path: normalized, StagePath: stagePath, Size: cr.n, Checksum: checksum, ScanStatus: scanStatus, ScannedBy: scannedBy, CreatedAt: time.Now().UTC()}
	s.storeMeta(meta, idempotencyKey, "")
	return meta, nil
}

func (s *UploadService) scanWithRetry(ctx context.Context, stagePath string) (ports.MalwareScanResult, int, error) {
	attempts := s.policy.ScanRetryAttempts
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		result, err := s.scanner.Scan(ctx, stagePath)
		if err == nil {
			return result, attempt, nil
		}
		lastErr = err
		if attempt < attempts {
			t := time.NewTimer(s.policy.ScanRetryBackoff)
			select {
			case <-ctx.Done():
				t.Stop()
				return ports.MalwareScanResult{}, attempt, ctx.Err()
			case <-t.C:
			}
		}
	}
	return ports.MalwareScanResult{}, attempts, lastErr
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
	s.updateOperationalMetricsLocked()
}

func (s *UploadService) enqueueScanDLQLocked(meta UploadMetadata, reason string, attempts int, errMessage string) {
	s.dlqSeq++
	id := fmt.Sprintf("scan-dlq-%d", s.dlqSeq)
	s.dlq[id] = ScanDLQEntry{ID: id, Path: meta.Path, StagePath: meta.StagePath, Reason: reason, Attempts: attempts, LastError: errMessage, CreatedAt: time.Now().UTC()}
}

func (s *UploadService) updateOperationalMetricsLocked() {
	now := time.Now().UTC()
	var backlog int64
	for _, m := range s.metadata {
		if m.ScanStatus == ports.MalwareStatusQuarantined {
			backlog++
			observability.DefaultMetrics.ObserveQuarantineTimeMs(now.Sub(m.CreatedAt).Milliseconds())
		}
	}
	var dlq int64
	for _, e := range s.dlq {
		if !e.Resolved {
			dlq++
		}
	}
	observability.DefaultMetrics.SetScanBacklog(backlog)
	observability.DefaultMetrics.SetScanDLQSize(dlq)
}

func (s *UploadService) ListScanDLQ(includeResolved bool) []ScanDLQEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ScanDLQEntry, 0, len(s.dlq))
	for _, e := range s.dlq {
		if !includeResolved && e.Resolved {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (s *UploadService) RetryScanDLQ(ctx context.Context, id string) (UploadMetadata, error) {
	s.mu.RLock()
	e, ok := s.dlq[id]
	s.mu.RUnlock()
	if !ok {
		return UploadMetadata{}, errors.New("dlq entry not found")
	}
	if e.Resolved {
		return UploadMetadata{}, errors.New("dlq entry already resolved")
	}

	result, attempts, err := s.scanWithRetry(ctx, e.StagePath)
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.dlq[id]
	entry.Attempts += attempts
	if err != nil {
		entry.LastError = err.Error()
		s.dlq[id] = entry
		s.updateOperationalMetricsLocked()
		return UploadMetadata{}, err
	}
	if result.Status != ports.MalwareStatusClean {
		entry.LastError = "scan verdict not clean"
		s.dlq[id] = entry
		s.updateOperationalMetricsLocked()
		return UploadMetadata{}, errors.New("scan verdict not clean")
	}
	if err := s.st.Move(ctx, e.StagePath, e.Path); err != nil {
		entry.LastError = err.Error()
		s.dlq[id] = entry
		s.updateOperationalMetricsLocked()
		return UploadMetadata{}, err
	}
	meta := s.metadata[e.Path]
	meta.ScanStatus = ports.MalwareStatusClean
	meta.ScannedBy = result.Engine
	s.metadata[e.Path] = meta
	entry.Resolved = true
	entry.ResolvedAt = time.Now().UTC()
	entry.LastError = ""
	s.dlq[id] = entry
	s.updateOperationalMetricsLocked()
	return meta, nil
}

func (s *UploadService) ResolveScanDLQ(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.dlq[id]
	if !ok {
		return false
	}
	e.Resolved = true
	e.ResolvedAt = time.Now().UTC()
	s.dlq[id] = e
	s.updateOperationalMetricsLocked()
	return true
}

func (s *UploadService) CleanupQuarantine(ctx context.Context, ttl time.Duration) (CleanupReport, error) {
	if ttl <= 0 {
		return CleanupReport{}, errors.New("ttl must be > 0")
	}
	objects, err := s.listQuarantineObjects(ctx)
	if err != nil {
		return CleanupReport{}, err
	}
	cutoff := time.Now().UTC().Add(-ttl)
	report := CleanupReport{}
	for _, obj := range objects {
		if obj.IsDir {
			continue
		}
		ageRef := obj.ModifiedAt
		if ageRef.IsZero() {
			ageRef = obj.CreatedAt
		}
		if ageRef.IsZero() || ageRef.After(cutoff) {
			report.Skipped++
			continue
		}
		if !strings.HasPrefix(obj.Path, "/quarantine/") {
			report.Skipped++
			continue
		}
		if err := s.st.Delete(ctx, obj.Path); err != nil {
			report.Skipped++
			continue
		}
		report.Deleted++
	}
	return report, nil
}

func (s *UploadService) listQuarantineObjects(ctx context.Context) ([]storage.ObjectInfo, error) {
	queue := []string{"/quarantine"}
	out := make([]storage.ObjectInfo, 0)
	for len(queue) > 0 {
		prefix := queue[0]
		queue = queue[1:]
		items, err := s.st.List(ctx, prefix)
		if err != nil {
			continue
		}
		for _, it := range items {
			if it.IsDir {
				queue = append(queue, it.Path)
				continue
			}
			out = append(out, it)
		}
	}
	return out, nil
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
