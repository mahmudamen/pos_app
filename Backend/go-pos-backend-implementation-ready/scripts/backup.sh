#!/usr/bin/env bash
# OPS-006: logical backup of the pos-api PostgreSQL database.
#
# Typically scheduled daily via cron or a systemd timer; see
# docs/11_OPERATIONS.md (Backups & Restore runbook).
set -euo pipefail

DATABASE_URL="${DATABASE_URL:?DATABASE_URL must be set, e.g. copied from .env.example}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
KEEP_DAYS="${BACKUP_KEEP_DAYS:-14}"

mkdir -p "$BACKUP_DIR"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
out="$BACKUP_DIR/pos-${stamp}.sql.gz"

# gzip streams pg_dump output; elapsed strong is corrupted backups (0 bytes).
# --no-owner / --no-privileges keep roles out of the dump and tenant RLS intact.
echo "Backing up to $out"
pg_dump "$DATABASE_URL" --no-owner --no-privileges | gzip -9 > "$out"

size="$(du -h "$out" | cut -f1)"
echo "Backup complete: $out ($size)"

# A truncated dump is worse than none: verify the archive before anything else
# trusts it, and drop it rather than let rotation keep a bad file.
gzip -t "$out" || { rm -f "$out"; echo "Backup FAILED integrity check (gzip); removed $out"; exit 1; }
[ -s "$out" ] || { echo "Backup FAILED: empty"; exit 1; }

# Data integrity alarm, deliberately AFTER the dump: the dump is written and
# verified by now, so it is still wanted on disk, but any corrupt row is
# inherited by every future restore of it.
#
# The pattern is the one migration 051 enforces on users.password_hash
# (internal/infrastructure/database/migrations/051_users_password_hash_format.sql).
# A value that does not match cannot have been produced by bcrypt, so
# CompareHashAndPassword rejects it outright: the account answers 401 to a
# correct password and nothing anywhere reports an error. That is how a
# production admin account was found broken only by restoring a dump by hand.
read -r -d '' hash_check_sql <<'SQL' || true
SELECT count(*) FROM users;
SELECT coalesce(string_agg(email, ', ' ORDER BY email), '')
FROM users
WHERE NOT (password_hash ~ '^\$2[aby]\$[0-9]{2}\$[./A-Za-z0-9]{53}$');
SQL
mapfile -t hash_check < <(psql "$DATABASE_URL" -tA -c "$hash_check_sql")

# A check that inspected nothing is not a passing check. `users` is FORCE RLS,
# so a least-privilege backup role would see zero rows here and every corrupt
# hash would pass unnoticed — the failure mode this whole block exists to avoid.
total="${hash_check[0]:-}"
if [ -z "$total" ] || [ "$total" -eq 0 ] 2>/dev/null; then
    echo "ERROR: integrity check saw ${total:-no} user rows (wrong role or FORCE RLS?)" >&2
    echo "       refusing to report this backup as clean" >&2
    exit 1
fi
corrupt="${hash_check[1]:-}"
if [ -n "$corrupt" ]; then
    echo "ERROR: corrupt users.password_hash (not bcrypt) for: $corrupt" >&2
    echo "       the dump above is valid and retained; these accounts will 401 on correct passwords until repaired" >&2
    exit 1
fi

# Retention: purge backups older than BACKUP_KEEP_DAYS.
find "$BACKUP_DIR" -maxdepth 1 -type f -name 'pos-*.sql.gz' -mtime "+${KEEP_DAYS}" -delete
echo "Retention: keeping ${KEEP_DAYS} days of backups in $BACKUP_DIR"