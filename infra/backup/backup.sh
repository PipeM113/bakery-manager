#!/usr/bin/env bash
# Backup of the `manager` schema of the Supabase project (docs/despliegue.md, section 7).
#
#   MIGRATE_URL='postgres://manager_app.<ref>:<password>@<session-pooler-host>:5432/postgres' ./backup.sh
#
# MIGRATE_URL is the session pooler (port 5432), the same one the migrations use. Optional:
#   BACKUP_DIR  where the files go (default ~/bakery-backups; keep it OUTSIDE the repo)
#   KEEP        how many backups to keep (default 8)
# Needs pg_dump, pg_restore and psql, and a pg_dump whose major version is the server's or newer
# (Supabase runs PostgreSQL 17: a pg_dump 16 refuses to dump it).
set -euo pipefail

: "${MIGRATE_URL:?MIGRATE_URL is required (session pooler URL of the manager_app role)}"
BACKUP_DIR="${BACKUP_DIR:-$HOME/bakery-backups}"
KEEP="${KEEP:-8}"

server_major=$(psql "$MIGRATE_URL" -Atqc "show server_version_num" | cut -c1-2)
client_major=$(pg_dump --version | sed -E 's/[^0-9]*([0-9]+).*/\1/')
if [ "$client_major" -lt "$server_major" ]; then
  echo "pg_dump $client_major is older than the server ($server_major): install postgresql-client-$server_major and try again." >&2
  exit 2
fi

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"
stamp=$(date +%Y-%m-%d-%H%M)
dump="$BACKUP_DIR/manager-$stamp.dump"
counts="$dump.counts"

# Rows per table, taken just before the dump, so the restore can be checked against them.
psql "$MIGRATE_URL" -Atqc "select tablename from pg_tables where schemaname = 'manager' order by 1" |
  while read -r table; do
    printf '%s %s\n' "$table" "$(psql "$MIGRATE_URL" -Atqc "select count(*) from manager.\"$table\"")"
  done > "$counts"

pg_dump "$MIGRATE_URL" --schema=manager --no-owner --no-privileges -Fc -f "$dump"
chmod 600 "$dump" "$counts"

# The archive must be readable and must list the tables it should hold.
entries=$(pg_restore --list "$dump" | grep -c ' TABLE DATA ' || true)
expected=$(wc -l < "$counts")
if [ "$entries" -lt "$expected" ]; then
  echo "the dump lists $entries tables with data, expected $expected: it is NOT a good backup." >&2
  exit 3
fi

# Keep the newest $KEEP.
ls -1t "$BACKUP_DIR"/manager-*.dump 2>/dev/null | tail -n +$((KEEP + 1)) | while read -r old; do
  rm -f "$old" "$old.counts"
done

echo "ok: $dump ($(du -h "$dump" | cut -f1), $expected tables). Check it with ./restore-check.sh $dump"
