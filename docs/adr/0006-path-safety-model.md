# ADR 0006: Path Safety Model (Canonicalization + Root Jail + Server-Issued References)

- Status: Proposed
- Date: 2026-01-28

## Context

User-provided paths enable traversal attacks and inconsistent encoding issues.

## Decision

- Canonicalize all paths in the File Engine
- Enforce root-jail constraints (deny escaping allowed roots)
- Prefer server-issued path references (tokens) instead of raw path strings

## Consequences

- Strong defense against traversal and encoding tricks
- Requires token design and storage/validation strategy

## Alternatives considered

- String sanitization only (rejected: insufficient)
