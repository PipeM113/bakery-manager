#!/usr/bin/env bash
# Proves that a backup restores: loads it into a scratch database of a LOCAL PostgreSQL and compares
# the rows of every table with the counts that backup.sh saved next to it. Never touches Supabase.
#
#   SCRATCH_URL='postgres://postgres:postgres@127.0.0.1:5432/postgres' ./restore-check.sh ~/bakery-backups/manager-<date>.dump
#
# SCRATCH_URL must point to a server where you may create and drop a database (a local one, not the project).
set -euo pipefail

dump="${1:?usage: restore-check.sh <file.dump>}"
: "${SCRATCH_URL:?SCRATCH_URL is required (a LOCAL PostgreSQL where a database can be created)}"
counts="$dump.counts"
[ -f "$counts" ] || { echo "missing $counts (backup.sh writes it next to the dump)" >&2; exit 2; }
case "$SCRATCH_URL" in
  *supabase*|*pooler*) echo "SCRATCH_URL looks like the real project: use a local PostgreSQL." >&2; exit 2 ;;
esac

name="restore_check_$$"
base="${SCRATCH_URL%/*}"
psql "$SCRATCH_URL" -qc "create database $name"
trap 'psql "$SCRATCH_URL" -qc "drop database if exists $name" >/dev/null' EXIT

pg_restore --no-owner --no-privileges -d "$base/$name" "$dump"

bad=0
while read -r table expected; do
  got=$(psql "$base/$name" -Atqc "select count(*) from manager.\"$table\"")
  if [ "$got" != "$expected" ]; then
    echo "DIFFERENT  $table: backup had $expected, restored $got" >&2
    bad=1
  fi
done < "$counts"

if [ "$bad" -ne 0 ]; then echo "the restore does NOT match the backup." >&2; exit 3; fi
echo "ok: $(wc -l < "$counts") tables restored with the same number of rows."
