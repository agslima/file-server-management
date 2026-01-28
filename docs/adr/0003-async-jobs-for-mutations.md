# ADR 0003: Use Asynchronous Jobs for Filesystem Mutations

- Status: Proposed
- Date: 2026-01-28

## Context

Folder creation, uploads, and move/rename/delete can be slow and unreliable depending on filesystem latency and scanning time.

## Decision

All filesystem mutations will be executed asynchronously using a job model:
`PENDING → RUNNING → SUCCEEDED | FAILED | QUARANTINED`

## Consequences

- Improves resilience and throughput
- Requires user-visible job tracking and clear UX
- Requires idempotency and retry/DLQ governance

## Alternatives considered
- Synchronous mutations (rejected: timeouts and API saturation)
