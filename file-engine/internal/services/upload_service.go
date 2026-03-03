package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"reflect"
	"sort"
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
	TenantID     string
	Path         string
	StagePath    string
	Size         int64
	Checksum     string
	ScanStatus   ports.MalwareScanStatus
	ScannedBy    string
	StorageClass string
	CreatedAt    time.Time
	RetainUntil  time.Time
	LegalHold    bool
	ArchivedAt   time.Time
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

type IntegrityFailure struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type IntegrityReport struct {
	Checked  int                `json:"checked"`
	Failed   int                `json:"failed"`
	Failures []IntegrityFailure `json:"failures"`
}

type UploadService struct {
	st         storage.Storage
	scanner    ports.MalwareScanner
	policy     UploadPolicy
	governance GovernancePolicy
	log        *logger.Logger

	mu               sync.RWMutex
	metadata         map[string]UploadMetadata
	idempotency      map[string]idempotencyRecord
	initIdempotency  map[string]initIdempotencyRecord
	dlq              map[string]ScanDLQEntry
	dlqSeq           int64
	resumable        map[string]*resumableUpload
	rateByTenant     map[string]tenantRateWindow
	governanceEvents []GovernanceEvent
	sourcePolicy     GovernancePolicy
	sourceVersion    string
	driftDetected    bool
	lastDriftCheck   time.Time
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

type initIdempotencyRecord struct {
	Path      string
	SessionID string
}

type tenantRateWindow struct {
	WindowStart time.Time
	Count       int64
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
	return &UploadService{
		st: st, scanner: scanner, policy: policy, log: logg,
		metadata: map[string]UploadMetadata{}, idempotency: map[string]idempotencyRecord{}, initIdempotency: map[string]initIdempotencyRecord{}, dlq: map[string]ScanDLQEntry{}, resumable: map[string]*resumableUpload{},
		rateByTenant: map[string]tenantRateWindow{},
		governance: GovernancePolicy{
			Default: TenantGovernancePolicy{QuotaBytes: policy.TenantQuotaBytes, ObjectLimit: policy.TenantObjectLimit},
			Tenants: map[string]TenantGovernancePolicy{},
		},
	}
}

func (s *UploadService) StartResumableUpload(targetPath, idempotencyKey string) (string, error) {
	normalized, err := security.NormalizeTenantPath(targetPath)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if idempotencyKey != "" {
		if existing, ok := s.initIdempotency[idempotencyKey]; ok {
			if existing.Path != normalized {
				return "", errors.New("idempotency key already used for a different target path")
			}
			return existing.SessionID, nil
		}
	}
	id, err := newResumableSessionID()
	if err != nil {
		return "", err
	}
	for {
		if _, exists := s.resumable[id]; !exists {
			break
		}
		id, err = newResumableSessionID()
		if err != nil {
			return "", err
		}
	}
	s.resumable[id] = &resumableUpload{TargetPath: normalized}
	if idempotencyKey != "" {
		s.initIdempotency[idempotencyKey] = initIdempotencyRecord{Path: normalized, SessionID: id}
	}
	return id, nil
}

func newResumableSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate resumable session id: %w", err)
	}
	return "upl_" + hex.EncodeToString(b), nil
}

func (s *UploadService) ResolveResumablePath(sessionID, idempotencyKey string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.resumable[sessionID]; ok {
		return r.TargetPath, nil
	}
	if idempotencyKey != "" {
		if existing, found := s.idempotency[idempotencyKey]; found {
			return existing.Path, nil
		}
	}
	return "", errors.New("resumable session not found")
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
		if idempotencyKey != "" {
			if existing, found := s.idempotency[idempotencyKey]; found {
				s.mu.Unlock()
				if existing.ErrMessage != "" {
					return existing.Meta, errors.New(existing.ErrMessage)
				}
				return existing.Meta, nil
			}
		}
		s.mu.Unlock()
		return UploadMetadata{}, errors.New("resumable session not found")
	}
	payload := append([]byte(nil), r.Buffer.Bytes()...)
	target := r.TargetPath
	delete(s.resumable, sessionID)
	s.mu.Unlock()
	return s.UploadStream(ctx, target, bytes.NewReader(payload), idempotencyKey)
}

func (s *UploadService) SetGovernancePolicy(p GovernancePolicy) error {
	if err := p.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.Tenants == nil {
		p.Tenants = map[string]TenantGovernancePolicy{}
	}
	s.governance = p
	return nil
}

func (s *UploadService) UpdateGovernancePolicy(actorID string, p GovernancePolicy) (string, string, error) {
	if err := p.Validate(); err != nil {
		return "", "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.Tenants == nil {
		p.Tenants = map[string]TenantGovernancePolicy{}
	}
	beforeHash := governancePolicyHash(s.governance)
	afterHash := governancePolicyHash(p)
	s.governance = p
	s.appendGovernanceEventLocked(actorID, "control-plane", "policy_update", "/governance/policy", "allow", fmt.Sprintf("before_hash=%s after_hash=%s", beforeHash, afterHash))
	return beforeHash, afterHash, nil
}

func governancePolicyHash(p GovernancePolicy) string {
	b, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *UploadService) GovernanceEvents() []GovernanceEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]GovernanceEvent, len(s.governanceEvents))
	copy(out, s.governanceEvents)
	return out
}

type EffectiveGovernancePolicy struct {
	TenantID      string                 `json:"tenant_id"`
	TenantPolicy  TenantGovernancePolicy `json:"tenant_policy"`
	Lifecycle     LifecyclePolicy        `json:"lifecycle"`
	PathHolds     []string               `json:"path_holds"`
	SourceVersion string                 `json:"source_version,omitempty"`
	DriftDetected bool                   `json:"drift_detected"`
	LastDriftAt   time.Time              `json:"last_drift_check_at,omitempty"`
}

func (s *UploadService) EffectivePolicy(tenantID string) EffectiveGovernancePolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return EffectiveGovernancePolicy{
		TenantID:      tenantID,
		TenantPolicy:  s.tenantGovernancePolicy(tenantID),
		Lifecycle:     s.governance.Lifecycle,
		PathHolds:     append([]string(nil), s.governance.PathHolds...),
		SourceVersion: s.sourceVersion,
		DriftDetected: s.driftDetected,
		LastDriftAt:   s.lastDriftCheck,
	}
}

func (s *UploadService) SetGovernanceSource(p GovernancePolicy, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sourcePolicy = p
	s.sourceVersion = version
}

func (s *UploadService) CheckGovernanceDrift(actorID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastDriftCheck = time.Now().UTC()
	drift := !reflect.DeepEqual(s.governance, s.sourcePolicy)
	reason := "in_sync"
	if drift {
		reason = "runtime_policy_mismatch"
	}
	s.driftDetected = drift
	s.appendGovernanceEventLocked(actorID, "control-plane", "policy_drift_check", "/governance/policy", "allow", reason)
	observability.DefaultMetrics.ObserveGovernanceDrift(drift, reason)
	return drift
}

func (s *UploadService) appendGovernanceEventLocked(actorID, tenantID, action, objectPath, decision, reason string) {
	s.governanceEvents = append(s.governanceEvents, GovernanceEvent{
		Timestamp: time.Now().UTC(),
		ActorID:   actorID,
		TenantID:  tenantID,
		Action:    action,
		Path:      objectPath,
		Decision:  decision,
		Reason:    reason,
	})
}

func (s *UploadService) enforceTenantRateLimitLocked(tenantID string, maxPerMinute int64) error {
	if maxPerMinute <= 0 {
		return nil
	}
	now := time.Now().UTC()
	window := s.rateByTenant[tenantID]
	if window.WindowStart.IsZero() || now.Sub(window.WindowStart) >= time.Minute {
		window = tenantRateWindow{WindowStart: now}
	}
	if window.Count >= maxPerMinute {
		return errors.New("tenant upload rate limit exceeded")
	}
	window.Count++
	s.rateByTenant[tenantID] = window
	return nil
}

func (s *UploadService) Upload(ctx context.Context, targetPath string, content []byte) (UploadMetadata, error) {
	return s.UploadStream(ctx, targetPath, bytes.NewReader(content), "")
}

func (s *UploadService) UploadStream(ctx context.Context, targetPath string, content io.Reader, idempotencyKey string) (UploadMetadata, error) {
	normalized, err := security.NormalizeTenantPath(targetPath)
	if err != nil {
		return UploadMetadata{}, fmt.Errorf("normalize tenant path %q: %w", targetPath, err)
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
	tenantPolicy := s.tenantGovernancePolicy(tenantID)
	if tenantPolicy.QuotaBytes <= 0 {
		tenantPolicy.QuotaBytes = s.policy.TenantQuotaBytes
	}
	if tenantPolicy.ObjectLimit <= 0 {
		tenantPolicy.ObjectLimit = s.policy.TenantObjectLimit
	}
	s.mu.Lock()
	if err := s.enforceTenantRateLimitLocked(tenantID, tenantPolicy.RequestsPerMinute); err != nil {
		s.appendGovernanceEventLocked("system", tenantID, "upload", normalized, "deny", "rate_limit")
		s.mu.Unlock()
		return UploadMetadata{}, err
	}
	s.mu.Unlock()
	if tenantPolicy.QuotaBytes > 0 {
		used, objects := s.tenantUsage(tenantID)
		if tenantPolicy.ObjectLimit > 0 && objects >= tenantPolicy.ObjectLimit {
			s.mu.Lock()
			s.appendGovernanceEventLocked("system", tenantID, "upload", normalized, "deny", "object_limit")
			s.mu.Unlock()
			return UploadMetadata{}, errors.New("tenant object count limit exceeded")
		}
		if used >= tenantPolicy.QuotaBytes {
			s.mu.Lock()
			s.appendGovernanceEventLocked("system", tenantID, "upload", normalized, "deny", "quota_bytes")
			s.mu.Unlock()
			return UploadMetadata{}, errors.New("tenant quota exceeded")
		}
	}

	tctx, cancel := context.WithTimeout(ctx, s.policy.RequestTimeout)
	defer cancel()

	stageName := strings.TrimSpace(path.Base(normalized))
	if stageName == "" || stageName == "." || stageName == "/" {
		stageName = "upload.bin"
	}
	stagePath := path.Join("/quarantine", tenantID, fmt.Sprintf("%d-%s", time.Now().UTC().UnixNano(), stageName))
	h := sha256.New()
	cr := &countingReader{r: content}
	tr := io.TeeReader(cr, h)

	if err := s.st.AtomicWrite(tctx, stagePath, tr); err != nil {
		return UploadMetadata{}, fmt.Errorf("write staged object %q: %w", stagePath, err)
	}
	if s.policy.MaxObjectSizeBytes > 0 && cr.n > s.policy.MaxObjectSizeBytes {
		_ = s.st.Delete(tctx, stagePath)
		return UploadMetadata{}, errors.New("max object size exceeded")
	}
	if tenantPolicy.QuotaBytes > 0 {
		used, _ := s.tenantUsage(tenantID)
		if used+cr.n > tenantPolicy.QuotaBytes {
			_ = s.st.Delete(tctx, stagePath)
			s.mu.Lock()
			s.appendGovernanceEventLocked("system", tenantID, "upload", normalized, "deny", "quota_bytes")
			s.mu.Unlock()
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
		meta := UploadMetadata{TenantID: tenantID, Path: normalized, StagePath: stagePath, Size: cr.n, Checksum: checksum, ScanStatus: ports.MalwareStatusQuarantined, ScannedBy: scannedBy, StorageClass: "standard", CreatedAt: time.Now().UTC(), RetainUntil: time.Now().UTC().Add(time.Duration(tenantPolicy.RetentionSeconds) * time.Second), LegalHold: tenantPolicy.LegalHold || s.pathUnderLegalHold(normalized)}
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
		return UploadMetadata{}, fmt.Errorf("storage move %q -> %q: %w", stagePath, normalized, err)
	}
	meta := UploadMetadata{TenantID: tenantID, Path: normalized, StagePath: stagePath, Size: cr.n, Checksum: checksum, ScanStatus: scanStatus, ScannedBy: scannedBy, StorageClass: "standard", CreatedAt: time.Now().UTC(), RetainUntil: time.Now().UTC().Add(time.Duration(tenantPolicy.RetentionSeconds) * time.Second), LegalHold: tenantPolicy.LegalHold || s.pathUnderLegalHold(normalized)}
	s.storeMeta(meta, idempotencyKey, "")
	return meta, nil
}

func (s *UploadService) scanWithRetry(ctx context.Context, stagePath string) (ports.MalwareScanResult, int, error) {
	attempts := max(s.policy.ScanRetryAttempts, 1)
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
		return UploadMetadata{}, fmt.Errorf("scan %q: %w", e.StagePath, err)
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
	return s.listPrefixObjects(ctx, "/quarantine")
}

func (s *UploadService) MoveObject(ctx context.Context, actorID, sourcePath, destinationPath string) (UploadMetadata, error) {
	src, err := security.NormalizeTenantPath(sourcePath)
	if err != nil {
		return UploadMetadata{}, fmt.Errorf("normalize tenant path %q: %w", sourcePath, err)
	}
	dst, err := security.NormalizeTenantPath(destinationPath)
	if err != nil {
		return UploadMetadata{}, fmt.Errorf("normalize tenant path %q: %w", destinationPath, err)
	}
	srcTenant := tenantFromPath(src)
	dstTenant := tenantFromPath(dst)
	if srcTenant == "" || srcTenant != dstTenant {
		return UploadMetadata{}, errors.New("cross-tenant move is not allowed")
	}

	s.mu.Lock()
	meta, ok := s.metadata[src]
	if ok {
		meta.Path = dst
	}
	s.mu.Unlock()

	if err := s.st.Move(ctx, src, dst); err != nil {
		return UploadMetadata{}, fmt.Errorf("storage move %q -> %q: %w", src, dst, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if ok {
		delete(s.metadata, src)
		s.metadata[dst] = meta
	}
	s.appendGovernanceEventLocked(actorID, srcTenant, "move", dst, "allow", "")
	s.updateOperationalMetricsLocked()
	return meta, nil
}

func (s *UploadService) RestoreQuarantinedObject(ctx context.Context, actorID, objectPath string, forceReprocess bool) (UploadMetadata, error) {
	normalized, err := security.NormalizeTenantPath(objectPath)
	if err != nil {
		return UploadMetadata{}, fmt.Errorf("normalize tenant path %q: %w", objectPath, err)
	}
	tenantID := tenantFromPath(normalized)

	s.mu.RLock()
	meta, ok := s.metadata[normalized]
	s.mu.RUnlock()
	if !ok {
		return UploadMetadata{}, errors.New("object metadata not found")
	}
	if strings.TrimSpace(meta.StagePath) == "" {
		return UploadMetadata{}, errors.New("object has no quarantine stage path")
	}

	if forceReprocess {
		result, _, err := s.scanWithRetry(ctx, meta.StagePath)
		if err != nil {
			return UploadMetadata{}, fmt.Errorf("scan %q: %w", meta.StagePath, err)
		}
		if result.Status != ports.MalwareStatusClean {
			return UploadMetadata{}, errors.New("scan verdict not clean")
		}
		meta.ScanStatus = result.Status
		meta.ScannedBy = result.Engine
	} else {
		meta.ScanStatus = ports.MalwareStatusClean
	}

	if err := s.st.Move(ctx, meta.StagePath, normalized); err != nil {
		return UploadMetadata{}, fmt.Errorf("storage move %q -> %q: %w", meta.StagePath, normalized, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.metadata[normalized] = meta
	s.appendGovernanceEventLocked(actorID, tenantID, "restore_quarantine", normalized, "allow", map[bool]string{true: "force_reprocess", false: "operator_restore"}[forceReprocess])
	s.updateOperationalMetricsLocked()
	return meta, nil
}

func (s *UploadService) VerifyIntegritySample(ctx context.Context, actorID string, sampleSize int) IntegrityReport {
	if sampleSize <= 0 {
		sampleSize = 10
	}
	s.mu.RLock()
	paths := make([]string, 0, len(s.metadata))
	for p := range s.metadata {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	if len(paths) > sampleSize {
		paths = paths[:sampleSize]
	}
	metas := make([]UploadMetadata, 0, len(paths))
	for _, p := range paths {
		metas = append(metas, s.metadata[p])
	}
	s.mu.RUnlock()

	report := IntegrityReport{Failures: make([]IntegrityFailure, 0)}
	for _, meta := range metas {
		report.Checked++
		rc, err := s.st.Open(ctx, meta.Path)
		if err != nil {
			report.Failed++
			report.Failures = append(report.Failures, IntegrityFailure{Path: meta.Path, Reason: "object_missing"})
			observability.DefaultMetrics.IncIntegrityCheck(false)
			s.mu.Lock()
			s.appendGovernanceEventLocked(actorID, meta.TenantID, "integrity_verify", meta.Path, "deny", "object_missing")
			s.mu.Unlock()
			continue
		}
		h := sha256.New()
		cr := &countingReader{r: rc}
		_, copyErr := io.Copy(h, cr)
		_ = rc.Close()
		if copyErr != nil {
			report.Failed++
			report.Failures = append(report.Failures, IntegrityFailure{Path: meta.Path, Reason: "read_error"})
			observability.DefaultMetrics.IncIntegrityCheck(false)
			s.mu.Lock()
			s.appendGovernanceEventLocked(actorID, meta.TenantID, "integrity_verify", meta.Path, "deny", "read_error")
			s.mu.Unlock()
			continue
		}
		digest := hex.EncodeToString(h.Sum(nil))
		if meta.Checksum != "" && digest != meta.Checksum {
			report.Failed++
			report.Failures = append(report.Failures, IntegrityFailure{Path: meta.Path, Reason: "checksum_mismatch"})
			observability.DefaultMetrics.IncIntegrityCheck(false)
			s.mu.Lock()
			s.appendGovernanceEventLocked(actorID, meta.TenantID, "integrity_verify", meta.Path, "deny", "checksum_mismatch")
			s.mu.Unlock()
			continue
		}
		if meta.Size > 0 && cr.n != meta.Size {
			report.Failed++
			report.Failures = append(report.Failures, IntegrityFailure{Path: meta.Path, Reason: "size_mismatch"})
			observability.DefaultMetrics.IncIntegrityCheck(false)
			s.mu.Lock()
			s.appendGovernanceEventLocked(actorID, meta.TenantID, "integrity_verify", meta.Path, "deny", "size_mismatch")
			s.mu.Unlock()
			continue
		}
		observability.DefaultMetrics.IncIntegrityCheck(true)
		s.mu.Lock()
		s.appendGovernanceEventLocked(actorID, meta.TenantID, "integrity_verify", meta.Path, "allow", "ok")
		s.mu.Unlock()
	}
	return report
}

func (s *UploadService) DeleteObject(ctx context.Context, actorID, objectPath string) error {
	normalized, err := security.NormalizeTenantPath(objectPath)
	if err != nil {
		return err
	}
	tenantID := tenantFromPath(normalized)
	s.mu.Lock()
	meta, ok := s.metadata[normalized]
	if ok {
		if meta.LegalHold || s.pathUnderLegalHold(normalized) {
			s.appendGovernanceEventLocked(actorID, tenantID, "delete", normalized, "deny", "legal_hold")
			s.mu.Unlock()
			return errors.New("deletion blocked by legal hold")
		}
		if !meta.RetainUntil.IsZero() && time.Now().UTC().Before(meta.RetainUntil) {
			s.appendGovernanceEventLocked(actorID, tenantID, "delete", normalized, "deny", "retention")
			s.mu.Unlock()
			return errors.New("deletion blocked by retention policy")
		}
	}
	s.mu.Unlock()
	if err := s.st.Delete(ctx, normalized); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.metadata, normalized)
	s.appendGovernanceEventLocked(actorID, tenantID, "delete", normalized, "allow", "")
	s.updateOperationalMetricsLocked()
	s.mu.Unlock()
	return nil
}

func (s *UploadService) CleanupLifecycle(ctx context.Context) (map[string]CleanupReport, error) {
	reports := map[string]CleanupReport{}
	if rep := s.applyArchiveLifecycle(time.Now().UTC()); rep.Deleted > 0 || rep.Skipped > 0 {
		reports["archive"] = rep
	}
	if ttl := time.Duration(s.governance.Lifecycle.QuarantineTTLSeconds) * time.Second; ttl > 0 {
		rep, err := s.cleanupPrefix(ctx, "/quarantine", ttl)
		if err != nil {
			return nil, err
		}
		reports["quarantine"] = rep
	}
	if ttl := time.Duration(s.governance.Lifecycle.OrphanStagingTTLSeconds) * time.Second; ttl > 0 {
		rep, err := s.cleanupPrefix(ctx, "/staging", ttl)
		if err != nil {
			return nil, err
		}
		reports["staging"] = rep
	}
	return reports, nil
}

func (s *UploadService) applyArchiveLifecycle(now time.Time) CleanupReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	report := CleanupReport{}
	for p, meta := range s.metadata {
		if strings.EqualFold(meta.StorageClass, "archive") {
			report.Skipped++
			continue
		}
		tenantID := tenantFromPath(p)
		policy := s.tenantGovernancePolicy(tenantID)
		if policy.ArchiveAfterDays <= 0 {
			report.Skipped++
			continue
		}
		if meta.CreatedAt.IsZero() || now.Before(meta.CreatedAt.Add(time.Duration(policy.ArchiveAfterDays)*24*time.Hour)) {
			report.Skipped++
			continue
		}
		meta.StorageClass = "archive"
		if strings.TrimSpace(policy.ArchiveClass) != "" {
			meta.StorageClass = strings.TrimSpace(policy.ArchiveClass)
		}
		meta.ArchivedAt = now
		s.metadata[p] = meta
		report.Deleted++
		s.appendGovernanceEventLocked("system", tenantID, "archive_transition", p, "allow", meta.StorageClass)
		observability.DefaultMetrics.IncArchiveTransition()
	}
	return report
}

func (s *UploadService) cleanupPrefix(ctx context.Context, prefix string, ttl time.Duration) (CleanupReport, error) {
	if ttl <= 0 {
		return CleanupReport{}, errors.New("ttl must be > 0")
	}
	objects, err := s.listPrefixObjects(ctx, prefix)
	if err != nil {
		return CleanupReport{}, err
	}
	cutoff := time.Now().UTC().Add(-ttl)
	report := CleanupReport{}
	for _, obj := range objects {
		if obj.IsDir {
			continue
		}
		if !strings.HasPrefix(obj.Path, prefix+"/") {
			report.Skipped++
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
		if err := s.st.Delete(ctx, obj.Path); err != nil {
			report.Skipped++
			continue
		}
		report.Deleted++
	}
	return report, nil
}

func (s *UploadService) listPrefixObjects(ctx context.Context, root string) ([]storage.ObjectInfo, error) {
	queue := []string{root}
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
