# Performance budgets and capacity planning

[//]: # (owner: Platform engineer rotation)
[//]: # (review_cadence: Quarterly)
[//]: # (last_reviewed: 2026-02-26)

## Purpose

This document turns load scripts into an explicit budget contract (what "good" means), defines error-budget investigation triggers, and gives rough component sizing guidance for dev/stage/prod-like environments.

## SLO + budget targets

The budget envelope is enforced by `file-engine/tests/load/smoke.js` (CI) and observed over a longer window by `file-engine/tests/load/soak.js` (nightly).

| Flow | Budget target (p95) | Error budget target |
| --- | --- | --- |
| List (gRPC list path; tracked via profiling/tests) | `< 250ms` | `< 1%` failed requests |
| Download (`/v1/objects:download`) | `< 300ms` (smoke), `< 350ms` (soak) | `< 1%` smoke, `< 2%` soak |
| Upload complete (`/v1/uploads/{id}:complete`) | `< 1200ms` (smoke), `< 1500ms` (soak) | shared global budget |
| Mutations (`/v1/folders`, upload initiate/chunk) | `< 800ms` (smoke), `< 900ms` (soak) | shared global budget |
| Task status (`/v1/tasks/{task_id}`) | `< 250ms` | `< 1%` failed requests |

## Error budget policy

Investigation is mandatory when one of the following occurs:

1. Smoke workflow fails a threshold in pull request validation (immediate block).
2. Nightly soak fails thresholds for two consecutive runs.
3. Weekly aggregate error rate exceeds `1%` for core flows.
4. p95 latency regresses by `>20%` week-over-week for any core flow.

Recommended first response:

- capture new profile artifacts with `file-engine/scripts/capture_hotpath_profile.sh`
- compare request-rate and queue-depth telemetry
- run `scripts/drills/dependency_backpressure.sh` to validate reject-path behavior

## CI budget gate

`/.github/workflows/load-tests.yml` now starts a local stack (`redis`, `postgres`, `file-engine`, `file-engine-worker`) and executes k6 smoke as a hard gate for pull requests and pushes to `main`.

Nightly schedule executes the soak profile against the same local topology for trend monitoring.

## Rough capacity guidance

> Values below are conservative starting points for planning and should be refined using production telemetry.

### Per-component sizing baseline

| Component | CPU baseline | Memory baseline | Notes |
| --- | --- | --- | --- |
| `file-engine` API | 500m | 512Mi | Increase to 1 vCPU when sustained upload-complete p95 exceeds 1s |
| `file-engine-worker` | 1000m | 1Gi | Prefer horizontal scale before vertical for queue catch-up |
| Redis queue | 500m | 512Mi | Keep used memory <70%; set maxmemory policy explicitly |
| Postgres | 1000m | 2Gi | Required for tasks/audit durability; ensure IOPS headroom |
| ClamAV (if enabled) | 1000m | 2Gi | Scan throughput bound; isolate from API CPU pool |

### Queue sizing rule-of-thumb

- target steady-state queue depth: `< 2x` average per-minute enqueue volume
- target drain SLO after spike: `< 10 minutes`
- provision worker concurrency so `drain_rate >= 1.5x peak_enqueue_rate`

### Storage throughput assumptions

- Download-heavy profile assumes at least `50 MB/s` aggregate read throughput per API instance.
- Upload-complete path assumes low-latency metadata + promote operations (`< 50ms` storage metadata RTT median).
- If shared filesystem latency rises above `100ms` median, scale workers down and increase backpressure thresholds before user-facing errors spike.

## Reproducible hot-path profiling

Use this script to capture CPU + heap profiles for the two hottest HTTP paths (`download`, `upload complete`):

```bash
./file-engine/scripts/capture_hotpath_profile.sh
```

Artifacts are written to `file-engine/artifacts/pprof/` and can be inspected with:

```bash
go tool pprof -top file-engine/artifacts/pprof/hotpath.cpu.pprof
go tool pprof -top file-engine/artifacts/pprof/hotpath.mem.pprof
```
