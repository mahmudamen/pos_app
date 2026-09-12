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

# Retention: purge backups older than BACKUP_KEEP_DAYS.
find "$BACKUP_DIR" -maxdepth 1 -type f -name 'pos-*.sql.gz' -mtime "+${KEEP_DAYS}" -delete
echo "Retention: keeping ${KEEP_DAYS} days of backups in $BACKUP_DIR"