# ADR 0002: Adopt Hybrid Architecture (Laravel Orchestrator + Go File Engine)

- Status: Proposed
- Date: 2026-01-28

## Context

Filesystem operations are I/O heavy, concurrency-sensitive, and must be hardened against path attacks. Business logic requires strong auth/RBAC and rapid iteration.

## Decision

Use:
- Laravel for authentication, RBAC, business rules, audit logs, API orchestration
- Go for filesystem operations and worker execution

## Consequences

- API remains responsive while file operations scale independently
- Requires dual-language build/test pipelines
- Requires strict contract and tracing to avoid debugging complexity

## Alternatives considered

- PHP-only (rejected: poor fit for heavy concurrent filesystem work)
- Go-only (rejected: slower to build auth/RBAC/admin features)
- Monolith with internal Go module (rejected: weaker isolation, harder scaling)
