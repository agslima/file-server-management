# ADR 0001: Use ADRs to Lock Architecture Decisions

- Status: Accepted
- Date: 2026-01-28

## Context
Early ambiguity in protocols, trust boundaries, and operational choices increases risk and rework.

## Decision

Document significant architectural decisions as ADRs in `docs/adr/`. Any change that affects the context below requires an ADR:
- trust boundaries,
- data flow contracts,
- security posture,
- operational scalability

## Consequences

- Clariry and onboarding
- Creates an auditable record of design rationale
- Adds lightweight process overhead

## Alternatives considered

- Rely on tribal knowledge (rejected)
- Use a single evolving architecture doc only (rejected)
