#!/usr/bin/env sh
# Fail-closed wrapper for PostgreSQL archive_command.
# Never returns 0 when pgBackRest is unconfigured so PostgreSQL keeps WAL and
# pg_stat_archiver.failed_count alerts operators instead of silently dropping WAL.
set -eu

WAL_PATH="${1:?usage: pgbackrest-archive-push.sh <wal-path>}"
CONF="${PGBACKREST_CONFIG:-/etc/pgbackrest/pgbackrest.conf}"

if [ ! -s "$CONF" ]; then
  echo "pgbackrest-archive-push: config missing or empty: $CONF" >&2
  exit 1
fi

# Prefer explicit PGBACKREST_STANZA env when non-empty; otherwise read the configured
# non-[global] stanza section directly from pgbackrest.conf (fallback: main).
STANZA="${PGBACKREST_STANZA:-}"
if [ -z "$STANZA" ]; then
  STANZA="$(awk '
    /^[[:space:]]*\[[^]]+\][[:space:]]*$/ {
      sec = $0
      sub(/^[[:space:]]*\[/, "", sec)
      sub(/\][[:space:]]*$/, "", sec)
      if (sec != "global" && sec !~ /:/) {
        print sec
        exit
      }
    }
  ' "$CONF")"
fi
STANZA="${STANZA:-main}"

exec pgbackrest --config="$CONF" --stanza="$STANZA" archive-push "$WAL_PATH"
