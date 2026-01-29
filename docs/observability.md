# Observability Strategy & Structured Logging Spec (Platform Engineering Standard)

This document defines **logging, metrics, and tracing** standards for the File Engine platform. It is written to support production-grade operations and ingestion into systems like **Datadog**, **ELK**, or **Grafana Loki**.

---

## 1) Goals & Non-Goals

### Goals
- Provide consistent, queryable logs across API and worker services.
- Enable request correlation across services (HTTP ↔ gRPC ↔ queue ↔ worker).
- Support distributed tracing (Trace/Span IDs) and context propagation (user, request, tenant).
- Keep log volume manageable while maximizing signal-to-noise.

### Non-Goals (for now)
- Defining vendor-specific dashboards (Datadog/Loki) – covered in runbooks.
- Full audit log immutability guarantees (can be added via sink/WORM storage).

---

## 2) Log Level Standards

### 2.1 Severity Matrix (Audience + Expectations)

| Level | When to use | Audience | Paging-worthy | Examples |
|---|---|---|---|---|
| `DEBUG` | Developer diagnostics, local/staging deep debugging | Developers | No | payload shape, branch decisions |
| `INFO` | Expected lifecycle events, key business milestones | Dev + Ops | No | “task enqueued”, “worker started” |
| `WARN` | Unexpected conditions that may self-heal; degraded experience | Operators | Sometimes | transient storage errors, retries |
| `ERROR` | Failed operation requiring investigation; request/task failed | Operators | Often | task failed, dependency down |
| `FATAL` | Service cannot start/continue; must exit | Operators | Yes | config invalid, cannot bind port |
| `PANIC` | Unhandled exception – should be treated as a bug | Dev + Ops | Yes | unexpected nil pointer etc. |

**Policy**
- Default level in production: `INFO`
- Use `DEBUG` only temporarily and ideally scoped (sampled / per-request).
- `WARN` must include actionable context (what degraded, what auto-recovery did).
- `ERROR` must include correlation IDs + error class + relevant metadata.

---

## 3) Structured Logging Specification (SRE/DevOps Approach)

### 3.1 Required JSON Schema (Log Event Envelope)

All services MUST emit JSON logs with these required fields:

```json
{
  "timestamp": "2026-01-28T12:00:00.123Z",
  "severity": "INFO",
  "service": "file-engine-api",
  "env": "prod",
  "version": "git-sha-or-semver",
  "message": "Task enqueued",
  "request_id": "req_01HR...",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "user_id": "user-42",
  "tenant_id": "tenant-123",
  "operation": "create_folder",
  "path": "/tenants/tenant-123/projects/alpha",
  "status": "queued",
  "duration_ms": 12,
  "error": null
}
```

### 3.2 Field Definitions (Required vs Optional)

**Required**
- `timestamp` (RFC3339Nano, UTC)
- `severity` (`DEBUG|INFO|WARN|ERROR|FATAL|PANIC`)
- `service` (stable service name: `file-engine-api`, `file-engine-worker`)
- `env` (`dev|staging|prod`)
- `message` (human-readable summary)

**Strongly recommended**
- `request_id` (unique per inbound request)
- `trace_id`, `span_id` (OpenTelemetry compatible)
- `operation` (stable operation key, e.g. `create_folder`, `upload_initiate`)
- `duration_ms` (for completed operations)
- `status` (result status)

**Context**
- `user_id` (from JWT `sub`)
- `roles` (array) – **optional**, but don’t log if too sensitive/large
- `tenant_id` (if multi-tenant)
- `path` (normalized)

**Error object** (when `severity >= ERROR`)
```json
{
  "error": {
    "type": "dependency_unavailable",
    "message": "redis connection refused",
    "stack": "optional stack trace",
    "cause": "optional root cause",
    "retryable": true
  }
}
```

### 3.3 Redaction & PII Rules
- Never log secrets: tokens, passwords, keys, DSNs with passwords.
- Avoid logging raw file contents or large payloads.
- If you must log request payloads in DEBUG, **redact** sensitive fields and cap size.

---

## 4) Request Correlation & Distributed Tracing

### 4.1 Correlation IDs (HTTP)
Use these headers:
- `X-Request-Id`: stable request ID (generated if missing)
- `traceparent`: W3C Trace Context (preferred)
- `X-B3-TraceId`, `X-B3-SpanId`: optional compatibility (Zipkin/B3)

**Rules**
- API must accept inbound `traceparent` and continue the trace.
- If no trace headers are present, API generates a new trace.

### 4.2 gRPC
Propagate tracing via gRPC metadata:
- `traceparent`
- `x-request-id` (if you use it as a stable app correlation ID)

### 4.3 Queue (Redis tasks)
When enqueueing tasks, include correlation context in the task payload:

```json
{
  "type": "create_folder",
  "params": { "parent": "/tenants/1", "name": "reports" },
  "meta": {
    "request_id": "req_01HR...",
    "trace_id": "4bf92f...",
    "span_id": "00f067...",
    "user_id": "user-42",
    "tenant_id": "tenant-123"
  }
}
```

Worker MUST:
- Extract `meta` from task payload
- Create/continue a trace for the worker execution span
- Log all task lifecycle events with the same IDs

---

## 5) Context Propagation (User IDs, Request IDs, Metadata)

### 5.1 In-process propagation (Go)
Use `context.Context` as the canonical carrier:
- `auth.AuthContext` (user, roles)
- request ID
- trace context (OpenTelemetry span in context)

**Design rules**
- No global variables for request state.
- Each service boundary must rehydrate context from headers/metadata/payload.
- Add helper functions: `RequestIDFromContext`, `WithRequestID`, etc.

### 5.2 What to propagate
Minimum set:
- `request_id`
- `trace_id`/`span_id`
- `user_id`
- `tenant_id` (if applicable)
- `operation`

---

## 6) Configuration & Tuning

### 6.1 Environment variables
Recommended configuration flags:

```dotenv
# logging
LOG_FORMAT=json          # json|text
LOG_LEVEL=info           # debug|info|warn|error
LOG_SAMPLING_RATE=1.0    # 0.0 - 1.0 (optional)

# tracing
ENABLE_TRACING=true
OTEL_SERVICE_NAME=file-engine-api
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1

# metrics
ENABLE_METRICS=true
METRICS_ADDR=:9090
```

### 6.2 Log format
- Production: **JSON** (required for Loki/Datadog/ELK parsing)
- Local dev: JSON or text (text is OK if you prefer readability)

### 6.3 Dynamic level adjustment
Options (pick one):
1) **Hot-reload via SIGHUP** (Unix) to reread env/config
2) **Admin endpoint** (protected) to set level temporarily
3) **Feature flag** (central config)

**Policy**
- Must be time-bound in production (`DEBUG` expires after N minutes).
- Audit changes to log level (who, when, why).

### 6.4 Managing volume
- Use `DEBUG` sparingly and avoid payload logging.
- Use sampling for noisy logs (e.g., per-request INFO access logs).
- Emit high-cardinality fields cautiously (e.g., full paths can be high cardinality; consider hashing or truncation for metrics but keep in logs for debugging).

---

## 7) Implementation Guidelines (Recommended Baseline)

### 7.1 OpenTelemetry
- Use OTel SDK for Go.
- Add HTTP middleware and gRPC interceptors for tracing:
  - start span per request
  - add attributes: `operation`, `path`, `user_id` (careful with PII)
- Export traces via OTLP to a collector.

### 7.2 Logging library expectations
Any structured logger is fine if it produces consistent JSON, e.g.:
- zap
- zerolog
- slog (Go stdlib) with JSON handler

### 7.3 Service naming
- `file-engine-api`
- `file-engine-worker`

---

## 8) Golden Queries (Operators)

Useful queries in Loki/Datadog/ELK:
- Error rate by service:
  - `service="file-engine-api" severity="ERROR"`
- Trace a single request:
  - `request_id="req_01HR..."`
- Trace a user flow:
  - `user_id="user-42" operation="create_folder"`
- Find retries:
  - `message~"retry" OR error.retryable=true`

---

## 9) Appendix: Suggested Standard Fields

**Always include**
- `service`, `env`, `severity`, `message`, `timestamp`

**When request-scoped**
- `request_id`, `trace_id`, `span_id`

**When auth-scoped**
- `user_id`, `tenant_id`

**When action-scoped**
- `operation`, `path`, `task_id`, `status`, `duration_ms`
