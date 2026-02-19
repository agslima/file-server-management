package tasks

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/observability"
)

type flakySink struct {
	fails int
}

func (f *flakySink) Type() string { return "flaky" }

func (f *flakySink) WriteLine(_ context.Context, _ []byte) error {
	if f.fails > 0 {
		f.fails--
		return errors.New("boom")
	}
	return nil
}

func TestRetryingSinkWritesDeadLetterAfterExhaustingRetries(t *testing.T) {
	t.Parallel()
	orig := observability.DefaultMetrics
	observability.DefaultMetrics = observability.NewMetrics()
	t.Cleanup(func() { observability.DefaultMetrics = orig })

	dl := filepath.Join(t.TempDir(), "deadletter.ndjson")
	r := &retryingSink{
		sink:            &flakySink{fails: 3},
		deadLetter:      &fileDeadLetterWriter{path: dl},
		log:             logger.New("debug"),
		attempts:        2,
		retryDelay:      1 * time.Millisecond,
		retryMultiplier: 1,
	}

	err := r.WriteLine(context.Background(), []byte(`{"event":"x"}`))
	if err == nil {
		t.Fatalf("expected write to fail")
	}
	b, readErr := os.ReadFile(dl)
	if readErr != nil {
		t.Fatalf("read dead letter file: %v", readErr)
	}
	if !strings.Contains(string(b), `"sink_type":"flaky"`) {
		t.Fatalf("expected dead letter sink type, got %q", string(b))
	}
	snapshot := observability.DefaultMetrics.SnapshotPrometheus()
	if !strings.Contains(snapshot, "fileengine_audit_dead_letters_total 1") {
		t.Fatalf("expected dead-letter metric increment, snapshot=%s", snapshot)
	}
}

func TestBuildImmutableSinkFromEnvBucketWritesJSONL(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AUDIT_IMMUTABLE_SINK_TYPE", "bucket")
	t.Setenv("AUDIT_BUCKET_NAME", "audit-bucket")
	t.Setenv("AUDIT_BUCKET_PREFIX", "events")
	t.Setenv("AUDIT_BUCKET_LOCAL_DIR", root)
	t.Setenv("AUDIT_DEAD_LETTER_PATH", filepath.Join(root, "dlq.ndjson"))

	s := BuildImmutableSinkFromEnv(logger.New("debug"), "")
	if s == nil {
		t.Fatalf("expected bucket sink")
	}
	if err := s.WriteLine(context.Background(), []byte(`{"event":"bucket"}`)); err != nil {
		t.Fatalf("write line: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(root, "audit-bucket", "events", "*.jsonl"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected bucket jsonl object to be created")
	}
}

func TestBuildImmutableSinkFromEnvLokiPostsLine(t *testing.T) {
	received := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		b, _ := io.ReadAll(r.Body)
		received = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	t.Setenv("AUDIT_IMMUTABLE_SINK_TYPE", "loki")
	t.Setenv("AUDIT_LOKI_ENDPOINT", srv.URL)
	t.Setenv("AUDIT_DEAD_LETTER_PATH", filepath.Join(t.TempDir(), "dlq.ndjson"))

	s := BuildImmutableSinkFromEnv(logger.New("debug"), "")
	if s == nil {
		t.Fatalf("expected loki sink")
	}
	if err := s.WriteLine(context.Background(), []byte(`{"event":"loki"}`)); err != nil {
		t.Fatalf("write line: %v", err)
	}
	if !strings.Contains(received, "streams") {
		t.Fatalf("expected loki payload, got %q", received)
	}
}

func TestBuildImmutableSinkFromEnvSIEMPostsNDJSONWithAuth(t *testing.T) {
	received := ""
	authHeader := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()
		b, _ := io.ReadAll(r.Body)
		received = string(b)
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	t.Setenv("AUDIT_IMMUTABLE_SINK_TYPE", "siem")
	t.Setenv("AUDIT_SIEM_ENDPOINT", srv.URL)
	t.Setenv("AUDIT_SIEM_API_KEY", "top-secret")
	t.Setenv("AUDIT_DEAD_LETTER_PATH", filepath.Join(t.TempDir(), "dlq.ndjson"))

	s := BuildImmutableSinkFromEnv(logger.New("debug"), "")
	if s == nil {
		t.Fatalf("expected siem sink")
	}
	if err := s.WriteLine(context.Background(), []byte(`{"event":"siem"}`)); err != nil {
		t.Fatalf("write line: %v", err)
	}
	if !strings.Contains(received, `{"event":"siem"}`) {
		t.Fatalf("expected siem payload, got %q", received)
	}
	if authHeader != "Bearer top-secret" {
		t.Fatalf("expected authorization header, got %q", authHeader)
	}
}

func TestBuildImmutableSinkFromEnvS3WormWritesJSONL(t *testing.T) {
	root := t.TempDir()
	t.Setenv("AUDIT_IMMUTABLE_SINK_TYPE", "s3_worm")
	t.Setenv("AUDIT_S3_BUCKET", "audit-archive")
	t.Setenv("AUDIT_S3_PREFIX", "worm")
	t.Setenv("AUDIT_S3_LOCAL_DIR", root)
	t.Setenv("AUDIT_DEAD_LETTER_PATH", filepath.Join(root, "dlq.ndjson"))

	s := BuildImmutableSinkFromEnv(logger.New("debug"), "")
	if s == nil {
		t.Fatalf("expected s3_worm sink")
	}
	if err := s.WriteLine(context.Background(), []byte(`{"event":"s3_worm"}`)); err != nil {
		t.Fatalf("write line: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(root, "audit-archive", "worm", "*.jsonl"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected s3_worm jsonl object to be created")
	}
}
