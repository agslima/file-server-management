# Data Durability & Recovery Runbook

## Scope

This runbook covers backup strategy, restore simulation, and disaster drills for:
- Postgres state (identity/audit/ACL metadata)
- Storage object state (tenant files and staged/quarantine objects)
- Secondary audit sink resilience (DLQ catch-up workflow)

## Backup strategy

### DB backups
- Logical backup cadence: at least daily using `pg_dump`.
- Retention: keep daily backups for 14 days minimum.
- Artifact naming: `db-<UTC_TIMESTAMP>.sql`.

### Storage backups
- Snapshot `FILE_BASE_ROOT` for tenant objects + quarantine/staging paths.
- Retention: keep daily snapshots for 14 days minimum.
- Artifact naming: `storage-<UTC_TIMESTAMP>.tar.gz`.

### Local/dev simulation
Use:

```bash
./scripts/backup_restore_simulation.sh
```

The script performs deterministic backup+restore simulation using local docker-compose services and validates restore signals.

## Restore steps

1. Stop writers (API/worker) to avoid concurrent writes during restore.
2. Restore Postgres metadata backup (`psql < db-*.sql`).
3. Restore storage snapshot (`tar -xzf storage-*.tar.gz` to `FILE_BASE_ROOT`).
4. Start services.
5. Run post-restore verification:
   - readiness check (`/readyz`)
   - integrity verification job (`./scripts/integrity_verify_job.sh`)
   - mutation/read canary (for example `./scripts/e2e/vs001_create_folder.sh`)

## Disaster drills

Run the three deterministic drills:

```bash
./scripts/drills/db_restore_replay.sh
./scripts/drills/storage_corruption_drill.sh
./scripts/drills/audit_sink_catchup_drill.sh
```

Expected outcomes:
- DB restore/replay drill finishes with explicit success marker.
- Storage corruption drill detects corruption via integrity verification endpoint.
- Audit sink drill proves sink outage behavior and documents DLQ catch-up path.

## Integrity verification job

Use the periodic job wrapper:

```bash
./scripts/integrity_verify_job.sh
```

- Calls `POST /admin/v1/integrity:verify` with configurable sample size.
- Returns non-zero when mismatches are detected.
- Emits failure signal consumable by schedulers and alerting wrappers.
