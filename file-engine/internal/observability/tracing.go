package observability

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	defaultOTELServiceName = "file-engine"
	envOTLPEndpoint        = "OTEL_EXPORTER_OTLP_ENDPOINT"
)

// TracingConfig is a deterministic representation of OTEL export wiring inputs.
type TracingConfig struct {
	ServiceName  string
	OTLPEndpoint string
	ExporterOn   bool
}

// ResolveTracingConfig reads tracing configuration from state-free inputs.
func ResolveTracingConfig(serviceName, otlpEndpoint string) TracingConfig {
	name := strings.TrimSpace(serviceName)
	if name == "" {
		name = defaultOTELServiceName
	}
	endpoint := strings.TrimSpace(otlpEndpoint)
	return TracingConfig{
		ServiceName:  name,
		OTLPEndpoint: endpoint,
		ExporterOn:   endpoint != "",
	}
}

// ResolveTracingConfigFromEnv maps environment variables to deterministic tracing config.
func ResolveTracingConfigFromEnv(serviceName string) TracingConfig {
	return ResolveTracingConfig(serviceName, os.Getenv(envOTLPEndpoint))
}

// InitTracing wires a global TracerProvider for API/worker entrypoints.
// If OTEL_EXPORTER_OTLP_ENDPOINT is unset, tracing stays local/no-op exporter.
func InitTracing(ctx context.Context, cfg TracingConfig) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(cfg.ServiceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("resource init: %w", err)
	}

	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if cfg.ExporterOn {
		exporter, exportErr := newOTLPTraceExporter(ctx, cfg.OTLPEndpoint)
		if exportErr != nil {
			return nil, exportErr
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp.Shutdown, nil
}

func newOTLPTraceExporter(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
	clean := strings.TrimSpace(endpoint)
	if clean == "" {
		return nil, errors.New("otlp endpoint is required")
	}
	parsed, err := url.Parse(clean)
	if err != nil {
		return nil, fmt.Errorf("parse otlp endpoint: %w", err)
	}

	var clientOpts []otlptracegrpc.Option
	switch parsed.Scheme {
	case "http", "https":
		clientOpts = append(clientOpts, otlptracegrpc.WithEndpoint(parsed.Host))
		if parsed.Scheme == "http" {
			clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
		}
	case "":
		clientOpts = append(clientOpts, otlptracegrpc.WithEndpoint(clean), otlptracegrpc.WithInsecure())
	default:
		return nil, fmt.Errorf("unsupported otlp endpoint scheme %q", parsed.Scheme)
	}

	exporter, expErr := otlptracegrpc.New(ctx, clientOpts...)
	if expErr != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", expErr)
	}
	return exporter, nil
}
