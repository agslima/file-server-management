# Observability Incident Drill Runbook

## Goal

Prove operators can detect and diagnose OTEL/alerting failures quickly with deterministic drills and runbook-linked signals.

## Preconditions

- Stack running with `--profile observability`.
- Metrics endpoint available at `http://localhost:8080/metrics`.
- Alert rules checked in under `monitoring/alerts/file-engine-alerts.yml`.

## Drill scripts

```bash
./scripts/check-otel-connectivity.sh
./scripts/drills/production_deployment_hardening.sh
```

`check-otel-connectivity.sh` fails fast when service OTEL exporter endpoints are misconfigured and verifies Jaeger actually contains exported traces from both `file-engine-api` and `file-engine-worker` after a backend-driven request.

`production_deployment_hardening.sh` validates alert rules and runs three deterministic drills:

1. `sink_down.sh`
2. `scanner_down.sh`
3. `otel_exporter_down.sh`

## Scenarios + expected deterministic signals

1. **Sink down (`scripts/drills/sink_down.sh`)**
   - Simulation: integration test drives immutable sink retries into dead-letter path.
   - Expected signals:
     - `fileengine_audit_sink_failures_total` increments
     - dead-letter payload exists for sink write attempts
     - `fileengine_audit_sink_lag_ms` records lag movement

2. **Scanner down (`scripts/drills/scanner_down.sh`)**
   - Simulation: service test forces scanner failure path and enqueues scan DLQ entry.
   - Expected signals:
     - upload scan failure metadata contains scanner error
     - `fileengine_scan_dlq_size` increases
     - operator remediation remains available through scan DLQ admin workflow

3. **OTEL exporter down (`scripts/drills/otel_exporter_down.sh`)**
   - Simulation: observability tests verify unsupported endpoint rejection + no-export fallback behavior.
   - Expected signals:
     - startup validation rejects invalid exporter endpoint scheme
     - no-export fallback is deterministic (`InitTracing` still returns shutdown handler)

## Runbook closure checks

For incident-drill closure, run:

```bash
./scripts/check-otel-connectivity.sh
./scripts/drills/production_deployment_hardening.sh
```

Expected output markers:

- `OTEL_CONNECTIVITY_OK`
- `PRODUCTION_DEPLOYMENT_HARDENING_DRILLS_OK`

## Related assets

- Dashboard template: `monitoring/dashboards/file-engine-golden-signals.json`
- Alerts as code: `monitoring/alerts/file-engine-alerts.yml`
- Alert rule checker: `./scripts/validate-alert-rules.sh`
