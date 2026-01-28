# ADR 0008: Observability Baseline (Logs, Metrics, Tracing)

- Status: Proposed
- Date: 2026-01-28

## Context
The system spans multiple services and async jobs. Without correlation and tracing, debugging and audits become unreliable.

## Decision
Adopt baseline observability:
- correlationId propagated end-to-end
- structured JSON logs in all services
- metrics for queue depth, job duration, scan duration, error rates
- distributed tracing (OpenTelemetry)

## Consequences
- Faster incident response and clearer audits
- Requires instrumentation work across services

## Alternatives considered
- Logs only (rejected: insufficient for distributed async flows)
