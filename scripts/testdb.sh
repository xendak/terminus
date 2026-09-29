#!/usr/bin/env bash
# testdb.sh — rebuild stoptime_test from scratch: drop, create, migrate, and
# with --seed apply db/seed/*.sql in filename order.
#
# Integration tests connect to stoptime_test, never to the dev database
# (docs/spec/architecture.md). Deterministic: safe to rerun before any go test.
#
# Usage: scripts/testdb.sh [--seed]
set -euo pipefail
. "$(dirname "$0")/lib.sh"

seed=0
if [ "${1:-}" = "--seed" ]; then seed=1; shift; fi
[ $# -eq 0 ] || { echo "usage: scripts/testdb.sh [--seed]" >&2; exit 1; }

URL="$(db_url stoptime_test)"

dropdb --if-exists --force stoptime_test
createdb stoptime_test
"$(dirname "$0")/migrate.sh" stoptime_test

if [ "$seed" -eq 1 ]; then
  # Data seeds in glob order. *_check.sql files are assertion scripts, not
  # data — they are skipped here and run explicitly after seeding.
  shopt -s nullglob
  seeds=()
  for file in "$REPO_DIR/db/seed"/*.sql; do
    case "$file" in
      *_check.sql) ;;
      *)           seeds+=("$file") ;;
    esac
  done
  shopt -u nullglob
  if [ "${#seeds[@]}" -eq 0 ]; then
    echo "testdb: --seed given but db/seed/ has no seed .sql files" >&2
    exit 1
  fi
  for file in "${seeds[@]}"; do
    psql "$URL" -v ON_ERROR_STOP=1 --single-transaction -f "$file"
    echo "testdb: seeded $(basename "$file")" >&2
  done
fi

echo "testdb: stoptime_test ready at $URL" >&2
