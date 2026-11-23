#!/usr/bin/env sh
# Fail-closed wrapper for PostgreSQL archive_command.
# Never returns 0 when pgBackRest is unconfigured so PostgreSQL keeps WAL and
# pg_stat_archiver.failed_count alerts operators instead of silently dropping WAL.
set -eu

WAL_PATH="${1:?usage: pgbackrest-archive-push.sh <wal-path>}"
CONF="${PGBACKREST_CONFIG:-/etc/pgbackrest/pgbackrest.conf}"
STANZA="${PGBACKREST_STANZA:-main}"

if [ ! -s "$CONF" ]; then
  echo "pgbackrest-archive-push: config missing or empty: $CONF" >&2
  exit 1
fi

exec pgbackrest --config="$CONF" --stanza="$STANZA" archive-push "$WAL_PATH"
