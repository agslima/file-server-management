package integration

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/example/file-engine/internal/app/tasks"
	"github.com/example/file-engine/internal/logger"
	"github.com/example/file-engine/internal/observability"
)

func TestAuditExternalSinkDeliveryWithDLQAndLagMetrics(t *testing.T) {
	origMetrics := observability.DefaultMetrics
	observability.DefaultMetrics = observability.NewMetrics()
	t.Cleanup(func() { observability.DefaultMetrics = origMetrics })

	t.Run("s3_worm_adapter_writes_object", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("AUDIT_IMMUTABLE_SINK_TYPE", "s3_worm")
		t.Setenv("AUDIT_S3_BUCKET", "audit-archive")
		t.Setenv("AUDIT_S3_PREFIX", "events")
		t.Setenv("AUDIT_S3_LOCAL_DIR", root)
		t.Setenv("AUDIT_DEAD_LETTER_PATH", filepath.Join(root, "dlq.ndjson"))

		sink := tasks.BuildImmutableSinkFromEnv(logger.New("debug"), "")
		if sink == nil {
			t.Fatalf("expected s3_worm sink")
		}
		line := `{"event":"task.succeeded","created_at":"` + time.Now().Add(-2*time.Second).UTC().Format(time.RFC3339Nano) + `"}`
		if err := sink.WriteLine(context.Background(), []byte(line)); err != nil {
			t.Fatalf("write line: %v", err)
		}

		matches, err := filepath.Glob(filepath.Join(root, "audit-archive", "events", "*.jsonl"))
		if err != nil {
			t.Fatalf("glob: %v", err)
		}
		if len(matches) == 0 {
			t.Fatalf("expected jsonl object to be created")
		}

		snapshot := observability.DefaultMetrics.SnapshotPrometheus()
		lagLine := metricLine(snapshot, "fileengine_audit_sink_lag_ms")
		lagMs, err := strconv.ParseInt(strings.TrimSpace(lagLine), 10, 64)
		if err != nil {
			t.Fatalf("parse lag metric %q: %v", lagLine, err)
		}
		if lagMs <= 0 {
			t.Fatalf("expected lag metric > 0, got %d", lagMs)
		}
	})

	t.Run("loki_adapter_posts_stream_payload", func(t *testing.T) {
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

		sink := tasks.BuildImmutableSinkFromEnv(logger.New("debug"), "")
		if sink == nil {
			t.Fatalf("expected loki sink")
		}
		if err := sink.WriteLine(context.Background(), []byte(`{"event":"task.processing"}`)); err != nil {
			t.Fatalf("write line: %v", err)
		}
		if !strings.Contains(received, "streams") {
			t.Fatalf("expected loki payload, got %q", received)
		}
	})

	t.Run("siem_failures_retry_then_dead_letter", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer srv.Close()

		dlq := filepath.Join(t.TempDir(), "deadletter.ndjson")
		t.Setenv("AUDIT_IMMUTABLE_SINK_TYPE", "siem")
		t.Setenv("AUDIT_SIEM_ENDPOINT", srv.URL)
		t.Setenv("AUDIT_SIEM_API_KEY", "audit-token")
		t.Setenv("AUDIT_SINK_RETRY_ATTEMPTS", "2")
		t.Setenv("AUDIT_SINK_RETRY_DELAY_MS", "1")
		t.Setenv("AUDIT_DEAD_LETTER_PATH", dlq)

		sink := tasks.BuildImmutableSinkFromEnv(logger.New("debug"), "")
		if sink == nil {
			t.Fatalf("expected siem sink")
		}
		err := sink.WriteLine(context.Background(), []byte(`{"event":"task.failed"}`))
		if err == nil {
			t.Fatalf("expected siem write to fail")
		}
		dlqBytes, readErr := os.ReadFile(dlq)
		if readErr != nil {
			t.Fatalf("read dead letter: %v", readErr)
		}
		if !strings.Contains(string(dlqBytes), `"sink_type":"siem"`) {
			t.Fatalf("expected siem dead letter envelope, got %q", string(dlqBytes))
		}
		snapshot := observability.DefaultMetrics.SnapshotPrometheus()
		if !strings.Contains(snapshot, "fileengine_audit_dead_letters_total 1") {
			t.Fatalf("expected dead-letter metric increment, snapshot=%s", snapshot)
		}
	})
}

func metricLine(snapshot, key string) string {
	for _, line := range strings.Split(snapshot, "\n") {
		if strings.HasPrefix(line, key+" ") {
			return strings.TrimPrefix(line, key+" ")
		}
	}
	return ""
}
