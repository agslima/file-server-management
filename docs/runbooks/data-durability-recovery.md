# Data Durability & Recovery Runbook

## Scope

This runbook covers backup strategy, restore simulation, and disaster drills for:
- Postgres state (identity/audit/ACL metadata)
- Storage object state (tenant files and staged/quarantine objects)
- Secondary audit sink resilience (DLQ catch-up workflow)

## Recovery objectives (dev-grade contract)

- **DB restore RTO:** `PT30M`
- **DB RPO:** `PT24H`
- **Storage restore RTO:** `PT45M`
- **Storage RPO:** `PT24H`

These defaults are encoded in script env vars and can be overridden per environment.

## Backup strategy

### DB backups
- Logical backup cadence: at least daily using `pg_dump`.
- Retention: keep daily backups for 14 days minimum.
- Artifact naming: `db-<UTC_TIMESTAMP>.sql`.

### Storage backups
- Snapshot `FILE_BASE_ROOT` for tenant objects + quarantine/staging paths.
- Retention: keep daily snapshots for 14 days minimum.
- Artifact naming: `storage-<UTC_TIMESTAMP>.tar.gz`.

### Local/dev simulation (script-backed)

```bash
./scripts/backup_restore_simulation.sh
```

The script prints each backup/restore step, includes active RTO/RPO targets in output, and emits `BACKUP_RESTORE_SIMULATION_OK`.

## Restore steps

1. Stop writers (API/worker) to avoid concurrent writes during restore.
2. Restore Postgres metadata backup (`psql < db-*.sql`).
3. Restore storage snapshot (`tar -xzf storage-*.tar.gz` to `FILE_BASE_ROOT`).
4. Start services.
5. Run post-restore verification:
   - readiness check (`/readyz`)
   - integrity verification job (`./scripts/integrity_verify_job.sh`)
   - mutation/read canary (for example `./scripts/e2e/vs001_create_folder.sh`)

## Disaster drills (script-backed)

Run the three drills:

```bash
./scripts/drills/db_restore_replay.sh
./scripts/drills/storage_corruption_drill.sh
./scripts/drills/audit_sink_catchup_drill.sh
```

Expected outcomes:
- DB restore/replay drill prints deterministic step logs and exits with `DB_RESTORE_REPLAY_DRILL_OK`.
- Storage corruption drill prints deterministic operator steps and exits with `STORAGE_CORRUPTION_DRILL_GUIDE_OK`.
- Audit sink drill proves sink outage behavior and documents DLQ catch-up path.

## Integrity verification policy (sample-based)

```bash
./scripts/integrity_verify_job.sh
```

Configurable policy controls:
- `SAMPLE_SIZE` (default `25`)
- `FAILURE_THRESHOLD` (default `0`)
- `IGNORE_PATHS` (comma-separated false-positive suppression list)

Semantics:
- The endpoint returns `409` only when `failed > failure_threshold` after ignore-path filtering.
- The response includes `ignored_failures`, `sample_size`, and `failure_threshold` for auditability.

## Audit-ready evidence bundle

Generate one-command evidence pack:

```bash
./scripts/generate_durability_evidence_pack.sh
```

The pack includes:
- access review command pointers
- durability config snapshot (RTO/RPO + integrity policy defaults)
- drill transcript pointers
- deterministic marker: `DURABILITY_EVIDENCE_PACK_OK`
