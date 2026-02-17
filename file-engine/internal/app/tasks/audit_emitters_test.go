package tasks

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/example/file-engine/internal/logger"
)

func TestNewDualLayerAuditEmitterWithNoOptionalSinksFallsBackToLog(t *testing.T) {
	a := NewDualLayerAuditEmitter(logger.New("debug"), nil, "")
	if _, ok := a.(*logAuditEmitter); !ok {
		t.Fatalf("expected log audit emitter fallback, got %T", a)
	}
}

func TestNewDualLayerAuditEmitterWritesToImmutableSink(t *testing.T) {
	t.Parallel()
	p := t.TempDir() + "/audit.ndjson"

	a := NewDualLayerAuditEmitter(logger.New("debug"), nil, p)
	a.EmitTaskEvent(context.Background(), "task.succeeded", "task-1", "corr-1", "ok")

	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read immutable sink: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, `"event":"task.succeeded"`) {
		t.Fatalf("expected event in sink, got %q", got)
	}
	if !strings.Contains(got, `"task_id":"task-1"`) {
		t.Fatalf("expected task id in sink, got %q", got)
	}
}
