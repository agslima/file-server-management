# ADR 0005: Service-to-Service Authentication (mTLS + Scoped Tokens)

- Status: Proposed
- Date: 2026-01-28

## Context

The File Engine performs privileged filesystem operations. Calls must be authenticated, authorized, and protected against replay.

## Decision

Use:
- mTLS between Laravel and Go (in-cluster)
- short-lived scoped tokens for operation claims (jobId, action, expiry)
- idempotency key enforcement for mutation jobs

## Consequences

- Strong protection against spoofing and misuse
- Requires certificate management and token issuance logic

## Alternatives considered

- Network-only trust (rejected: insufficient)
- Shared static API key (rejected: weak rotation and scope)
