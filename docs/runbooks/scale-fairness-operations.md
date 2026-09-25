# Scale & Fairness Operations Runbook

[//]: # (owner: Platform Engineers)
[//]: # (review_cadence: Per release)
[//]: # (last_reviewed: 2026-02-24)

## Core SLOs (published)

| Flow | SLI | SLO target |
| :-- | :-- | :-- |
| List/download | `http_request_duration_ms` for `GET /v1/objects:download` | p95 < 1200ms during smoke profile |
| Upload completion | latency for `POST /v1/uploads/{uploadId}:complete` | p95 < 2000ms during smoke profile |
| Task completion | create-folder task `queued -> success` completion time | p95 < 5s in baseline integration flow |

## Load profiles

- **Smoke (CI):** `file-engine/tests/load/smoke.js` (deterministic, short run).
- **Soak (nightly):** `file-engine/tests/load/soak.js` (scheduled via workflow cron).

## Fairness + noisy-neighbor controls

- Upload ingress applies **per-tenant** and **per-actor** request/concurrency limits.
- Create-folder enqueue applies **per-tenant** and **per-actor** mutation rate limits.
- Throttled responses use a stable envelope:
  - `error.code = THROTTLED`
  - `error.reason = rate_limited | concurrency_limited`
  - `error.retryable = true`
- Audit evidence is emitted as `upload.throttled` / `task.enqueue.throttled`.

## Backpressure behavior drills

Run from repo root:

```bash
./scripts/drills/dependency_backpressure.sh
```

Expected deterministic signal:

- readiness checks emit failing dependency reason in `/readyz` payload
- queue/unavailable mutation path returns deterministic unavailable rejection
- upload governance controls remain enforceable under constrained conditions

## Operator checklist

1. Confirm active SLO dashboard and burn-rate alerts.
2. Run smoke profile before release promotion.
3. Verify nightly soak artifact for latency + error-rate drift.
4. Execute dependency drill and attach transcript to incident/run review.
