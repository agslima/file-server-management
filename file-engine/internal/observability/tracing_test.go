package observability

import (
	"context"
	"testing"
)

func TestResolveTracingConfigDefaultsAndExporterToggle(t *testing.T) {
	t.Parallel()

	cfg := ResolveTracingConfig("", "")
	if cfg.ServiceName != "file-engine" {
		t.Fatalf("expected default service name, got %q", cfg.ServiceName)
	}
	if cfg.ExporterOn {
		t.Fatalf("expected exporter off when endpoint is empty")
	}

	cfg = ResolveTracingConfig("file-engine-api", "http://collector:4317")
	if cfg.ServiceName != "file-engine-api" {
		t.Fatalf("expected service name override, got %q", cfg.ServiceName)
	}
	if !cfg.ExporterOn {
		t.Fatalf("expected exporter on when endpoint is provided")
	}
}

func TestInitTracingRejectsUnsupportedEndpointScheme(t *testing.T) {
	t.Parallel()

	cfg := ResolveTracingConfig("file-engine-api", "ftp://collector:4317")
	_, err := InitTracing(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected invalid endpoint scheme to fail")
	}
}

func TestInitTracingWithoutExporterIsDeterministic(t *testing.T) {
	t.Parallel()

	cfg := ResolveTracingConfig("file-engine-worker", "")
	shutdown, err := InitTracing(context.Background(), cfg)
	if err != nil {
		t.Fatalf("init tracing without exporter: %v", err)
	}
	if shutdown == nil {
		t.Fatalf("expected shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown tracing: %v", err)
	}
}
