# ADR 0004: Select Queue Technology (Redis vs Kafka)

- Status: Proposed
- Date: 2026-01-28

## Context

Docs reference Redis and Kafka interchangeably. The system needs reliable job processing, retries, and DLQ.

## Decision

Short-term: Redis-based queue for simplicity (local dev + early stages).
Long-term: Kafka may be adopted if event streaming, replay, and high throughput become primary requirements.

## Consequences

- Redis accelerates delivery and reduces operational burden initially
- Kafka adoption later requires migration plan and schema governance

## Alternatives considered

- Kafka from day one (rejected: higher ops complexity early)
- SQS-only (rejected: cloud lock-in for initial stage)
