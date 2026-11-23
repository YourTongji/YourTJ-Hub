# PostgreSQL 16 + pgBackRest Backup Image

> Doc type: operations reference
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-11-23

`Current`: this directory packages a thin Alpine wrapper over `postgres:16-alpine` that adds
`ca-certificates`, `pgbackrest`, and the fail-closed WAL archive wrapper
`pgbackrest-archive-push.sh`. It preserves the exact PostgreSQL 16 Alpine musl binary layout so
existing `pgdata` volumes mount without re-initializing.

## Build on the server

```bash
docker build -t yourtj-postgres-backup:16-alpine services/postgres-backup
```

Verify the installed binary and CA certificates before switching `POSTGRES_IMAGE` in
`/opt/yourtj/.env`:

```bash
docker run --rm yourtj-postgres-backup:16-alpine pgbackrest version
```

## Configuration boundary

- `pgbackrest.conf` is mounted read-only at `/etc/pgbackrest/pgbackrest.conf` (mode `0600`, owned
  by the container's `postgres` UID/GID, `70:70` on `postgres:16-alpine`).
- `pgbackrest-archive-push.sh` fails closed (exits non-zero) if the config is missing or empty, so
  PostgreSQL retains WAL segments and increments `pg_stat_archiver.failed_count` rather than
  silently discarding WAL.
- Cloudflare R2 requires `repo1-s3-region=auto` and `repo1-s3-uri-style=path`. `repo1-bundle=y`,
  `compress-type=lz4`, `process-max=1`, and `archive-async=n` keep R2 Class A calls, RAM usage,
  and local disk spool minimal on small VMs.

See [Database backup & recovery (Cloudflare R2)](../../docs/operations/backup-and-recovery.md)
for full bootstrap, PITR restore, and disaster recovery procedures.
