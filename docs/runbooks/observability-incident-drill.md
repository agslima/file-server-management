# Observability Incident Drill Runbook

## Goal

Prove operators can detect and diagnose failures quickly with trace + metrics + alert rules.

## Preconditions

- Stack running with `--profile observability`.
- Metrics endpoint available at `http://localhost:8080/metrics`.
- Alert rules checked in under `monitoring/alerts/file-engine-alerts.yml`.

## Drill script

```bash
./scripts/drills/observability_incident_drill.sh
```

Script confirms baseline signals are present and prints expected signatures for three break scenarios.

## Scenarios

1. **Break scanner**
   - Set scanner backend to unreachable ClamAV endpoint.
   - Expected signals:
     - logs: `upload.scan.completed` with `scan_error`
     - metrics: `fileengine_scan_dlq_size > 0`, `fileengine_scan_backlog > 0`
     - alert: `FileEngineScanDLQGrowing`

2. **Break audit sink**
   - Point SIEM/Loki sink endpoint to failing URL.
   - Expected signals:
     - metric increase: `fileengine_audit_sink_failures_total`
     - lag movement: `fileengine_audit_sink_lag_ms`
     - alert: `FileEngineAuditSinkFailures`

3. **Slow DB**
   - Inject latency/failure in Postgres path.
   - Expected signals:
     - queue lag growth (`fileengine_queue_lag_ms_max`)
     - readiness degradation on `/readyz`
     - traces in Jaeger show long DB spans

## Related assets

- Dashboard template: `monitoring/dashboards/file-engine-golden-signals.json`
- Alerts as code: `monitoring/alerts/file-engine-alerts.yml`
- Alert rule checker: `./scripts/validate-alert-rules.sh`
