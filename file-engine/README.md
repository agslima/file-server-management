# File Engine (Go) — Data Plane

This directory contains the Go File Engine, which is the **final authorization and execution boundary** for filesystem mutations. It exposes a gRPC-first API and runs async tasks via a worker.

## Canonical docs (read first)

- Project overview and status: `../README.md`
- Capability truth table: `../docs/capability-ledger.md`
- Local setup: `../docs/setup.md`
- File Engine architecture: `../docs/architecture_file-engine.md`
- API reference: `../docs/api-reference.md`
- Storage backends: `../docs/storage_backends.md`

If guidance conflicts, prefer: capability ledger → setup → architecture docs.

---

## Current baseline (validated)

The baseline is intentionally small and is enforced via the capability ledger. The verified vertical slice is the **async create-folder flow** with task status, audit events, and correlation IDs.

Run the baseline checks from repo root:

```bash
./file-engine/scripts/dev.sh
```

Key validations included:

- Proto mirror sync (`api/proto` ↔ `proto`)
- Module-level tests for config/logger/worker
- Handler-level auth/task tests
- Integration async folder flow

---

## Local run (scaffold-level)

For debugging the API/worker locally, follow `../docs/setup.md`. Note that HTTP gateway wiring is still scaffold-level until real generated gateway code is committed. Use this path for debugging, not as a validated REST contract.

---

## Contract discipline

- Canonical proto: `api/proto/fileengine.proto`
- Mirror proto: `proto/fileengine.proto`

These must remain identical. Use `cmp` as in the dev script.

---

## Notes

- Tenant scope is resolved server-side via `TENANT_MEMBERSHIPS` (dev only) or a future authoritative source.
- AuthN requires `JWT_SECRET` or `JWT_PUBLIC_KEY_PEM`.
- File Engine is the **final authz gate** (tenant membership + RBAC/ACL + path safety).
