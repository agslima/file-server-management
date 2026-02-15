# Contributor Guide — Server File Manager Platform

This guide is for developers contributing code: local setup, dev workflows, and standards.

---

## Repo layout (high-level)

- `frontend/` — React/Next.js UI
- `backend/` — Laravel API (auth, RBAC, audit, orchestration)
- `file-engine/` — Go service (filesystem operations + workers)
- `docker/` — container definitions (local + CI)
- `docs/` — architecture, ADRs, role-specific docs

---

## Prerequisites

- Docker + Docker Compose
- Node.js (for frontend)
- PHP + Composer (for backend)
- Go (for file-engine)
- PostgreSQL client (optional, but helpful)

---

## Quick start (local)

Use the canonical onboarding guide in `docs/setup.md`. Baseline-validated path is:

```bash
./file-engine/scripts/dev.sh
```

If you need a full stack compose, note that the root `docker-compose.yml` is **experimental** until validated end-to-end.
Treat it as container-build validation only, not a baseline-validated runtime path.

---

## Development workflow

### Branching
- `main` is stable
- feature branches: `feat/<topic>`
- fix branches: `fix/<topic>`

### Pull requests
Each PR must include:
- summary of change
- how to test locally
- security impact notes (if any)
- related ADRs (if applicable)

---

## Coding standards

### Backend (Laravel)
- Follow Laravel conventions (Controllers thin, Services/Actions for business logic)
- Use Policies for authorization (RBAC + per-path permissions)
- Validate inputs at boundaries (request objects)
- Never accept raw filesystem paths without canonicalization or server-issued references

### File Engine (Go)
- Treat all inputs as untrusted
- Enforce:
  - canonical path resolution
  - root constraints
  - allowlists (extensions/MIME), if applicable
- Use idempotency keys for mutation jobs
- Keep filesystem logic isolated from transport (clean architecture)

### Frontend
- Do not embed authorization logic in the UI
- Render only what the API allows
- Handle async job states gracefully

---

## Testing

### Required test types (target state)
- Backend:
  - unit tests for policies
  - request validation tests
  - integration tests for job creation
- File-engine:
  - unit tests for path canonicalization/root-jail
  - integration tests using a temp filesystem
- End-to-end:
  - create folder flow
  - upload flow with mocked scan verdict

---

## Local security notes
- Do not commit `.env` files or secrets
- Use local-only credentials
- Treat sample data as non-sensitive

---

## Contributing checklist
- [ ] Lint and format
- [ ] Add/adjust tests
- [ ] Update docs if behavior changes
- [ ] Reference ADR if decision-level change
