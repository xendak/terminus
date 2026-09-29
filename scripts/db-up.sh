#!/usr/bin/env bash
# db-up.sh — start the repo-local cluster, ensure stoptime + stoptime_test exist.
#
# stdout is eval-able; progress goes to stderr:
#   eval "$(scripts/db-up.sh)"   # exports DATABASE_URL / TEST_DATABASE_URL
set -euo pipefail
. "$(dirname "$0")/lib.sh"

[ -d "$PGDATA" ] || { echo "error: no cluster at $PGDATA — run scripts/db-init.sh first" >&2; exit 1; }

# Eval contract: everything the tools print goes to stderr; fd 3 keeps the
# original stdout for the export lines alone.
exec 3>&1 1>&2

if ! pg_ctl status -D "$PGDATA" >/dev/null 2>&1; then
  pg_ctl start -D "$PGDATA" -l "$REPO_DIR/.pg/server.log" -w 3>&-
  echo "cluster started (log: .pg/server.log)"
else
  echo "cluster already running"
fi

for db in stoptime stoptime_test; do
  if [ "$(psql -d postgres -tAc "select 1 from pg_database where datname = '$db'")" != "1" ]; then
    createdb "$db"
    echo "created database $db"
  fi
done

echo "export DATABASE_URL='$(db_url stoptime)'" >&3
echo "export TEST_DATABASE_URL='$(db_url stoptime_test)'" >&3
