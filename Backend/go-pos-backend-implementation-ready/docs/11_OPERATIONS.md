# Operations

## Development

PostgreSQL and Redis run through Docker Compose.

## Production topology

```text
Internet
   |
 TLS / Caddy
   |
 Go POS API
   |
 +---------+
 |         |
PostgreSQL Redis
```

PostgreSQL is authoritative.

Redis failure should not corrupt business data.

## Deployment

1. Build immutable application artifact.
2. Apply backward-compatible migrations.
3. Deploy application.
4. Check readiness.
5. Monitor error rate/latency.
6. Roll back application if required.
7. Never roll back destructive schema changes blindly.

## Backups & Restore runbook (OPS-006)

Backups must be automated and periodically restored into an isolated environment.

A backup that has never been restore-tested is not considered verified.

### Backup (logical, pg_dump)

`scripts/backup.sh` streams a gzipped logical dump (`--no-owner --no-privileges`)
to `$BACKUP_DIR` (default `./backups`) and prunes files older than
`$BACKUP_KEEP_DAYS` (default 14).

```bash
set -a; source .env; set +a        # provides DATABASE_URL
./scripts/backup.sh                # -> backups/pos-<UTC stamp>.sql.gz
```

Schedule daily with cron (`crontab -e`):

```cron
30 2 * * * cd /srv/pos && . .env >/dev/null 2>&1 && ./scripts/backup.sh >>/var/log/pos-backup.log 2>&1
```

Notes:

- `--no-owner/--no-privileges` keep roles out of the dump; tenant RLS and the
  `saas` platform tenant are restored as-is.
- On a multi-instance deployment, run exactly one backup writer to avoid
  concurrent dumps.
- Copy the tarball off-host (object storage / another disk). Retention only
  applies inside `$BACKUP_DIR`.

### Restore

```bash
RESTORE_TARGET_URL=postgres://pos_app:...@db-host:5432/pos ./scripts/restore.sh backups/pos-<stamp>.sql.gz
```

The target database must already exist (e.g. `createdb pos_restore_test`).

### Restore verification (mandatory)

A backup is only "verified" when it successfully restores **and** passes a smoke
test in an isolated database:

1. `createdb pos_restore_test`.
2. Restore into it (URL above, changing the database name).
3. `SELECT count(*) FROM tenants;` — expect the same number as the source.
4. Boot the API against the restored DB and run one login + one sale.
5. Confirm a POS device re-pulls from `change_seq 0` (`GET /v1/sync/pull`) and
   converges with the restored snapshot.

### Rollback

- Application-only rollback is safe any time: keep the previous immutable build
  and redeploy it; PostgreSQL authored the data, migrations are forward-only.
- Never roll back a destructive schema change blindly — that requires a restore
  from the last verified backup taken before the migration.
- Back up *before* running `make migrate` on production: `./scripts/backup.sh`.

## Monitoring

Monitor:

- request rate;
- 4xx/5xx rate;
- latency p50/p95/p99;
- database pool exhaustion;
- database errors;
- Redis errors;
- authentication failures;
- sync failures;
- process restarts.

## Security

Production requires:

- TLS;
- restricted database network access;
- least-privilege DB credentials;
- secret management;
- explicit CORS;
- firewall policy;
- OS patching;
- log retention policy.
