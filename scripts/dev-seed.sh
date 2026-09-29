#!/usr/bin/env bash
# dev-seed.sh — reset the DEV database (stoptime) to the demo dataset:
# migrate, empty every app table, then apply db/seed/golden.sql and
# db/seed/demo/demo.sql.
#
# Destructive for `stoptime` only; never touches stoptime_test (tests seed
# golden alone through testdb.sh). The demo data is relative to
# CURRENT_DATE — rerun it on demo day to move "today" along.
#
# Demo logins (password "stoptime-dev"): admin@stoptime.dev,
# manager@stoptime.dev, driver-a@stoptime.dev, driver-b@stoptime.dev,
# driver-c@stoptime.dev.
#
# Usage: scripts/dev-seed.sh
set -euo pipefail
. "$(dirname "$0")/lib.sh"

[ $# -eq 0 ] || { echo "usage: scripts/dev-seed.sh" >&2; exit 1; }

DB=stoptime
URL="$(db_url "$DB")"

"$(dirname "$0")/migrate.sh" "$DB"

{
  echo "TRUNCATE audit_log, route_stop, route, location, parameter, driver_profile, app_user;"
  echo "\\ir $REPO_DIR/db/seed/golden.sql"
  echo "\\ir $REPO_DIR/db/seed/demo/demo.sql"
} | psql "$URL" -v ON_ERROR_STOP=1 -q --single-transaction -f -

psql "$URL" -tA -c "
  select 'dev-seed: ' || count(*) filter (where status = 'closed') || ' closed, '
         || count(*) filter (where status = 'active') || ' active, '
         || count(*) filter (where status = 'draft') || ' draft routes'
    from route" >&2
echo "dev-seed: $DB ready at $URL (logins: *@stoptime.dev / stoptime-dev)" >&2
