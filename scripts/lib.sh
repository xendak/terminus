# lib.sh — shared facts for the StopTime cluster scripts. Sourced, never run.
#
# The cluster is repo-local (gitignored .pg/) and owned by the scripts:
# unix socket .pg/sock + TCP port 5543, loopback only, trust auth (dev).
# Nothing global is touched (docs/spec/architecture.md, "Environment").
# Requires the devshell tools (postgresql_18): run inside `nix develop`.

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PGDATA="$REPO_DIR/.pg/data"
export PGSOCK="$REPO_DIR/.pg/sock"
export PGPORT="5543"
export PGHOST="$PGSOCK"

if ! command -v psql >/dev/null 2>&1; then
  echo "error: psql not found — run inside the devshell: nix develop" >&2
  exit 1
fi

# db_url <dbname> — libpq URI for a database on the repo-local cluster.
db_url() {
  printf 'postgresql:///%s?host=%s&port=%s' "$1" "$PGSOCK" "$PGPORT"
}
