# AGENTS.md

Scope: `backend/` (Laravel control-plane scaffold)

## Read this first

For any work in `backend/`, use these sources in order:

1. `docs/capability-ledger.md` (canonical implemented-vs-target status)
2. `README.md` (project architecture and current maturity framing)
3. `docs/project-alignment-review.md` (drift risks and improvement priorities)
4. `.github/AGENTS.md` (repo-wide operating conventions)

If guidance conflicts, prefer the same order above.

## Backend reality check (important)

- `backend/` is currently scaffold-level, not a feature-complete control plane.
- Do **not** present backend endpoints as production-ready unless validated by runnable checks.
- Keep claims aligned with the ledger (`CL-008` + `CL-018` currently define the backend scaffold contract).

## Change expectations for backend edits

- Follow Laravel/PHP conventions and PSR-4 compatibility.
- Prefer small, explicit changes over broad scaffold rewrites.
- Preserve clear separation of concerns:
  - Controllers: request/response orchestration
  - Services: integration/business logic
  - Routes/config: minimal, declarative wiring
- Avoid introducing target-state claims in code comments, docs, or PR text without validation evidence.

## Validation baseline for backend changes

Run from repository root unless noted:

- `cd backend && composer validate --strict`
- `cd backend && php -l app/Http/Controllers/FolderController.php && php -l app/Http/Controllers/TaskController.php && php -l app/Services/FileEngineService.php`

If you add executable backend behavior, also add/update focused tests and include exact commands used to validate.

## Documentation hygiene

When backend behavior or boundaries change, update the minimum necessary docs so they stay consistent:

- `README.md` (only if user-visible architecture/status language changes)
- `docs/capability-ledger.md` (when a backend claim or validation command changes)
- `docs/project-alignment-review.md` (only when alignment findings/actions are materially affected)

Prefer linking to canonical docs rather than duplicating instructions.
