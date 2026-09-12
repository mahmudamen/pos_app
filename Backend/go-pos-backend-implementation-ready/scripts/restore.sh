#!/usr/bin/env bash
# OPS-006: restore a logical backup created by scripts/backup.sh into a
# PostgreSQL database. The target database must already exist.
#
# The embedded .env is NOT applied by this script; export DATABASE_URL for the
# target database yourself (see AGENTS.md: the Go app does not auto-load .env).
#
# Usage:
#   RESTORE_TARGET_URL=postgres://... ./scripts/restore.sh backups/pos-20260101T000000Z.sql.gz
set -euo pipefail

backup="${1:?usage: restore.sh <backup.sql.gz>}"
RESTORE_TARGET_URL="${RESTORE_TARGET_URL:?RESTORE_TARGET_URL must be set to the target database}"

if [[ ! -f "$backup" ]]; then
  echo "backup file not found: $backup" >&2
  exit 1
fi

echo "Restoring into $RESTORE_TARGET_URL"
gunzip -c "$backup" | psql "$RESTORE_TARGET_URL" --set=ON_ERROR_STOP=1
echo "Restore complete."

echo
echo "Post-restore verification (run these manually):"
echo "  1. SELECT count(*) FROM tenants;"
echo "  2. Login a demo user against the restored tenant."
echo "  3. GET /v1/sync/pull on a POS device resyncs from change_seq 0."
echo "See docs/11_OPERATIONS.md (Backups & Restore runbook)."