# Agents.md — File Engine Working Guide

Scope: this file applies to everything under `file-engine/`.

## Purpose

`file-engine` is the Go data-plane for async, policy-enforced filesystem operations.
It is expected to provide:

- API boundary (gRPC-first with HTTP/JSON gateway support)
- task enqueueing + worker execution
- storage backend abstraction (local/S3/GCS)
- strong auth/authz enforcement at execution boundary
- observable task lifecycle and correlation

## Ground Truth & Current Baseline

Before making changes, treat these as source-of-truth checks:

1. Canonical proto mirror must stay synchronized:
   - `api/proto/fileengine.proto`
   - `proto/fileengine.proto`
2. Baseline checks must pass:
   - `go test ./internal/config ./internal/logger ./internal/worker -v`
3. Integration baseline must pass:
   - `go test ./tests/integration -run TestAsyncCreateFolderFlow -v`
4. Preferred local guardrail script:
   - `./scripts/dev.sh`
5. Upload contract/security changes must keep lifecycle and compatibility checks green:
   - `go test ./internal/server -run "TestUploadLifecycleEndpointsCleanAndDirty|TestUploadChunkAndCompleteRequireAuthorization|TestCompatibilityUploadLifecycleGolden" -v`

Current maturity truth is the capability ledger (`docs/capability-ledger.md`), which currently promotes `CL-001` through `CL-073` with no open target-state exclusions (last reviewed 2026-02-19).

## Required Workflow for Changes

From repo root:

```bash
cd file-engine
```

After edits, always run:

```bash
go test ./internal/config ./internal/logger ./internal/worker -v
go test ./tests/integration -run TestAsyncCreateFolderFlow -v
```

Then run claim-focused checks for the touched area from `docs/capability-ledger.md` (for example, upload lifecycle/compatibility tests when changing upload handlers or API envelopes).

If proto or API contract changes:

```bash
cmp api/proto/fileengine.proto proto/fileengine.proto
# If changed, sync mirror and regenerate artifacts according to project scripts.
```

If storage/auth/authz behavior changes, add/adjust tests in:

- `internal/auth/*_test.go`
- `tests/integration/*`
- relevant package tests (e.g., filesystem/storage adapters)

## Coding Conventions

### Go standards

- Run `gofmt` on touched files.
- Keep imports minimal; remove unused imports.
- Return and wrap errors with useful context.
- Prefer small, explicit interfaces where behavior is shared.
- Avoid duplicating near-identical types across packages.

### Package boundaries

- Keep domain logic in `internal/app` and `internal/services` areas.
- Keep adapter details in `internal/adapters/*`.
- Keep transport concerns in `internal/delivery/*` and server wrappers.
- Worker-specific behavior belongs under `internal/worker` or app task modules.

### API contract discipline

- Do not silently drift endpoint names/fields away from proto and documented API references.
- If request/response payloads change, update docs and generated artifacts in the same PR where possible.

## Security & Governance Expectations

- Treat File Engine as final authorization boundary.
- Enforce deny-by-default behavior when authz data is missing/ambiguous.
- Preserve safe-path normalization and tenant namespace constraints.
- Keep correlation IDs propagated across request -> task -> worker logs.
- Avoid introducing bypasses that allow direct client-controlled tenant scoping.

## Storage & Worker Notes

- Local storage path behavior depends on `FILE_BASE_ROOT`.
- Redis queue behavior depends on `REDIS_ADDR`.
- Backends are selected through `STORAGE_BACKEND` (`local`, `s3`, `gcs`) plus backend-specific env vars.
- New task types should include:
  - deterministic payload schema,
  - explicit status transitions,
  - failure semantics,
  - audit/event hooks where applicable.

## PR/Commit Expectations (for `file-engine` changes)

When touching `file-engine`:

1. Keep change scope narrow and explain architectural impact.
2. Include commands run and outcomes in PR verification section.
3. Call out whether the change is:
   - baseline implementation,
   - hardening/refactor,
   - target-state preparation.
4. If docs are affected, update them in the same PR.

## Known Pitfalls to Avoid

- Editing only one proto mirror file.
- Updating docs without runnable verification.
- Introducing stale compose/setup commands that do not match current paths.
- Marking aspirational controls as implemented without tests.

## Quick Command Reference

From repository root:

```bash
cmp file-engine/api/proto/fileengine.proto file-engine/proto/fileengine.proto
cd file-engine && go test ./internal/config ./internal/logger ./internal/worker -v
cd file-engine && go test ./tests/integration -run TestAsyncCreateFolderFlow -v
./file-engine/scripts/dev.sh
```
