package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/observability"
)

type ImmutableSink interface {
	WriteLine(ctx context.Context, line []byte) error
	Type() string
}

type DeadLetterWriter interface {
	Write(ctx context.Context, envelope deadLetterEnvelope) error
}

type deadLetterEnvelope struct {
	SinkType    string    `json:"sink_type"`
	Error       string    `json:"error"`
	PayloadLine string    `json:"payload_line"`
	FailedAt    time.Time `json:"failed_at"`
}

type fileImmutableSink struct {
	path string
	mu   sync.Mutex
}

func (s *fileImmutableSink) Type() string { return "file" }

func (s *fileImmutableSink) WriteLine(_ context.Context, line []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

type localBucketSink struct {
	rootDir string
	bucket  string
	prefix  string
	mu      sync.Mutex
}

func (s *localBucketSink) Type() string { return "bucket" }

func (s *localBucketSink) WriteLine(_ context.Context, line []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.rootDir, s.bucket, s.prefix)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	name := fmt.Sprintf("audit-%d.jsonl", time.Now().UTC().UnixNano())
	path := filepath.Join(dir, name)
	return os.WriteFile(path, append(line, '\n'), 0o600)
}

type lokiSink struct {
	endpoint string
	client   *http.Client
}

func (s *lokiSink) Type() string { return "loki" }

func (s *lokiSink) WriteLine(ctx context.Context, line []byte) error {
	payload := map[string]any{
		"streams": []map[string]any{{
			"stream": map[string]string{"service": "file-engine", "source": "audit"},
			"values": [][]string{{strconv.FormatInt(time.Now().UTC().UnixNano(), 10), string(line)}},
		}},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req) // #nosec G704 -- endpoint is validated at sink construction and is operator-configured
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("loki returned status %d", resp.StatusCode)
	}
	return nil
}

type siemSink struct {
	endpoint string
	apiKey   string
	client   *http.Client
}

func (s *siemSink) Type() string { return "siem" }

func (s *siemSink) WriteLine(ctx context.Context, line []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(append(line, '\n')))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	resp, err := s.client.Do(req) // #nosec G704 -- endpoint is validated at sink construction and is operator-configured
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("siem returned status %d", resp.StatusCode)
	}
	return nil
}

type fileDeadLetterWriter struct {
	path string
	mu   sync.Mutex
}

func (w *fileDeadLetterWriter) Write(_ context.Context, envelope deadLetterEnvelope) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(w.path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	b, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

type retryingSink struct {
	sink            ImmutableSink
	deadLetter      DeadLetterWriter
	log             *logger.Logger
	attempts        int
	retryDelay      time.Duration
	retryMultiplier float64
}

func (s *retryingSink) Type() string { return s.sink.Type() }

func (s *retryingSink) WriteLine(ctx context.Context, line []byte) error {
	attempts := max(s.attempts, 1)
	delay := s.retryDelay
	if delay <= 0 {
		delay = 25 * time.Millisecond
	}
	mult := s.retryMultiplier
	if mult < 1 {
		mult = 2
	}
	var err error
	for i := 1; i <= attempts; i++ {
		err = s.sink.WriteLine(ctx, line)
		if err == nil {
			var rec auditEventRecord
			if jsonErr := json.Unmarshal(line, &rec); jsonErr == nil && !rec.CreatedAt.IsZero() {
				observability.DefaultMetrics.SetAuditSinkLagMs(time.Since(rec.CreatedAt).Milliseconds())
			}
			return nil
		}
		if i < attempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay = time.Duration(float64(delay) * mult)
		}
	}
	if s.deadLetter != nil {
		if dlErr := s.deadLetter.Write(ctx, deadLetterEnvelope{SinkType: s.sink.Type(), Error: err.Error(), PayloadLine: string(line), FailedAt: time.Now().UTC()}); dlErr == nil {
			observability.DefaultMetrics.IncAuditDeadLetter()
		}
	}
	if s.log != nil {
		s.log.Event("warn", "immutable sink failed after retries", map[string]any{"sink": s.sink.Type(), "error": err.Error()})
	}
	return err
}

func BuildImmutableSinkFromEnv(logg *logger.Logger, legacyPath string) ImmutableSink {
	sinkType := strings.TrimSpace(os.Getenv("AUDIT_IMMUTABLE_SINK_TYPE"))
	if sinkType == "" && strings.TrimSpace(legacyPath) != "" {
		sinkType = "file"
	}

	var base ImmutableSink
	switch strings.ToLower(sinkType) {
	case "file":
		p := strings.TrimSpace(legacyPath)
		if p == "" {
			p = strings.TrimSpace(os.Getenv("AUDIT_IMMUTABLE_SINK_PATH"))
		}
		if p == "" {
			return nil
		}
		base = &fileImmutableSink{path: p}
	case "bucket":
		bucket := strings.TrimSpace(os.Getenv("AUDIT_BUCKET_NAME"))
		if bucket == "" {
			return nil
		}
		root := strings.TrimSpace(os.Getenv("AUDIT_BUCKET_LOCAL_DIR"))
		if root == "" {
			root = filepath.Join(os.TempDir(), "audit-bucket")
		}
		base = &localBucketSink{rootDir: root, bucket: bucket, prefix: strings.TrimSpace(os.Getenv("AUDIT_BUCKET_PREFIX"))}
	case "s3", "s3worm", "s3_worm":
		bucket := strings.TrimSpace(os.Getenv("AUDIT_S3_BUCKET"))
		if bucket == "" {
			return nil
		}
		root := strings.TrimSpace(os.Getenv("AUDIT_S3_LOCAL_DIR"))
		if root == "" {
			root = filepath.Join(os.TempDir(), "audit-s3-worm")
		}
		base = &localBucketSink{rootDir: root, bucket: bucket, prefix: strings.TrimSpace(os.Getenv("AUDIT_S3_PREFIX"))}
	case "loki":
		endpoint := strings.TrimSpace(os.Getenv("AUDIT_LOKI_ENDPOINT"))
		if endpoint == "" {
			return nil
		}
		u, err := url.Parse(endpoint)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return nil
		}
		base = &lokiSink{endpoint: endpoint, client: &http.Client{Timeout: 5 * time.Second}}
	case "siem":
		endpoint := strings.TrimSpace(os.Getenv("AUDIT_SIEM_ENDPOINT"))
		if endpoint == "" {
			return nil
		}
		u, err := url.Parse(endpoint)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return nil
		}
		base = &siemSink{
			endpoint: endpoint,
			apiKey:   strings.TrimSpace(os.Getenv("AUDIT_SIEM_API_KEY")),
			client:   &http.Client{Timeout: 5 * time.Second},
		}
	default:
		return nil
	}

	attempts := 3
	if v := strings.TrimSpace(os.Getenv("AUDIT_SINK_RETRY_ATTEMPTS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			attempts = n
		}
	}
	delay := 50 * time.Millisecond
	if v := strings.TrimSpace(os.Getenv("AUDIT_SINK_RETRY_DELAY_MS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			delay = time.Duration(n) * time.Millisecond
		}
	}
	dlPath := strings.TrimSpace(os.Getenv("AUDIT_DEAD_LETTER_PATH"))
	if dlPath == "" {
		dlPath = filepath.Join(os.TempDir(), "file-engine-audit-deadletter.ndjson")
	}

	return &retryingSink{
		sink:            base,
		deadLetter:      &fileDeadLetterWriter{path: dlPath},
		log:             logg,
		attempts:        attempts,
		retryDelay:      delay,
		retryMultiplier: 2,
	}
}
