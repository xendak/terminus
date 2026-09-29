#!/usr/bin/env bash
# db-init.sh — one-time bootstrap of the repo-local PostgreSQL cluster.
#
# initdb into .pg/data (gitignored), bound to the script-owned unix socket
# (.pg/sock) and port 5543, listening on loopback only, trust auth.
# Refuses to run over an existing cluster.
#
# Usage: scripts/db-init.sh        (inside `nix develop`)
set -euo pipefail
. "$(dirname "$0")/lib.sh"

if [ -d "$PGDATA" ]; then
  echo "error: $PGDATA already exists — cluster is initialized." >&2
  echo "       Wipe and redo: rm -rf .pg && scripts/db-init.sh && scripts/db-up.sh" >&2
  exit 1
fi

mkdir -p "$REPO_DIR/.pg" "$PGSOCK"
initdb -D "$PGDATA" -U "$(id -un)" --auth=trust

cat >>"$PGDATA/postgresql.conf" <<EOF

# --- Terminus dev cluster (scripts/db-init.sh) ---
port = $PGPORT
listen_addresses = '127.0.0.1'
unix_socket_directories = '$PGSOCK'
EOF

echo "cluster initialized: $PGDATA" >&2
echo "socket $PGSOCK, port $PGPORT — start with: scripts/db-up.sh" >&2
