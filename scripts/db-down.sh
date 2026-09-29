#!/usr/bin/env bash
# db-down.sh — stop the repo-local cluster (fast mode).
set -euo pipefail
. "$(dirname "$0")/lib.sh"

[ -d "$PGDATA" ] || { echo "error: no cluster at $PGDATA" >&2; exit 1; }

if pg_ctl status -D "$PGDATA" >/dev/null 2>&1; then
  pg_ctl stop -D "$PGDATA" -m fast -w
  echo "cluster stopped" >&2
else
  echo "cluster already stopped" >&2
fi
