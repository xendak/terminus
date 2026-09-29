#!/usr/bin/env bash
# migrate.sh — apply db/migrations/*.sql in filename order.
#
# One transaction per file (ON_ERROR_STOP + --single-transaction); each applied
# file is recorded in schema_migrations (created by this script — it is
# infrastructure, not schema) and skipped when already present. Applied
# migrations are immutable; fixes are new files (docs/spec/architecture.md).
#
# Usage: scripts/migrate.sh [dbname]    (default: stoptime)
set -euo pipefail
. "$(dirname "$0")/lib.sh"

DB="${1:-stoptime}"
URL="$(db_url "$DB")"
MIGRATIONS_DIR="$REPO_DIR/db/migrations"

psql "$URL" -v ON_ERROR_STOP=1 -q -c \
  "create table if not exists schema_migrations (
     name text primary key,
     applied_at timestamptz not null default now()
   )"

shopt -s nullglob
files=( "$MIGRATIONS_DIR"/*.sql )
shopt -u nullglob

if [ "${#files[@]}" -eq 0 ]; then
  echo "migrate [$DB]: db/migrations/ has no .sql files — nothing to do" >&2
fi

for file in "${files[@]}"; do
  name="$(basename "$file")"
  if [ "$(psql "$URL" -tAc "select 1 from schema_migrations where name = '$name'")" = "1" ]; then
    echo "migrate [$DB]: skip $name (already applied)" >&2
    continue
  fi
  psql "$URL" -v ON_ERROR_STOP=1 --single-transaction -f "$file"
  psql "$URL" -v ON_ERROR_STOP=1 -q -c "insert into schema_migrations (name) values ('$name')"
  echo "migrate [$DB]: applied $name" >&2
done
