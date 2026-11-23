# Database Backup, WAL Archiving & Disaster Recovery (Cloudflare R2)

> Doc type: operations
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-11-23

`Current`: this repository ships the complete Cloudflare R2 offsite backup and PostgreSQL
point-in-time recovery (PITR) stack — a `postgres:16-alpine` + `pgBackRest` image
([services/postgres-backup](../../services/postgres-backup/README.md)), daily encrypted backup and
health-check scripts, systemd timers, and an isolated PITR restore verifier. Enabling it on a live
host requires provisioning a dedicated Cloudflare R2 bucket and running the bootstrap steps below.

## Architecture & recovery targets

The forum binary's built-in `[db] backupSqlite` cron only backs up SQLite databases (`file.db`).
When PostgreSQL is the primary database (`[db.default] connection = "postgres"`), PostgreSQL
durability and offsite recovery are owned by two complementary layers stored in a **dedicated
Cloudflare R2 backup bucket** (separate from any media/image upload bucket):

| Layer | Tool | What it backs up | R2 prefix | Encryption | Target RPO |
| --- | --- | --- | --- | --- | --- |
| Physical PG + continuous WAL | `pgBackRest` inside `yourtj-postgres` | Full PG cluster (`yourtj_main` + WAL segments) | `/pgbackrest` | Client-side AES-256-CBC | ≤ 5 min (`archive_timeout=300`) |
| Daily files, keys & logical PG | `restic` on host (`systemd` timer) | Verified `pg_dump -Fc` + `globals.sql`, consistent `sqlite3 .backup` of `main/storage/database/file.db`, `oidc/signing_key.pem`, `push/*.p8`, `agent-state`, `main/config.toml`, `.env`, `docker-compose.yaml`, 1Panel OpenResty configs/certs | `/restic` | Client-side AES-256 + chunk deduplication | ≤ 24 h |
| Pre-deploy local safety net | `deploy/scripts/backup-db.sh` | Compressed `pg_dump` (`pg-*.sql.gz`, mode `0600`, keep 7) | Local `/opt/yourtj/snapshots/main` | Host root-only (`0700`/`0600`) | Pre-deploy |

Local disk guardrails on small hosts (1.6 GiB RAM / 40 GiB disk):

- `deploy/instances/main.json` keeps `[db] backupSqlite = true, keep = 3` (local `file.db` fallback);
  `deploy/instances/dev.json` sets `backupSqlite = false, keep = 1` because `dev` is synced from
  `main` on every deploy.
- `sync-db-from-main.sh` syncs `main` into `dev` directly without writing `snapshots/dev-prev`.
- `deploy/docker-compose.yaml` caps container logs at `20m × 3` (`json-file`) and tunes
  PostgreSQL for low memory (`wal_compression=lz4`, `wal_buffers=16MB`, `max_wal_size=512MB`,
  `effective_cache_size=768MB`, `process-max=1`, `archive-async=n`).

## 1. Cloudflare R2 bucket & token setup

1. In Cloudflare Dashboard → **R2 Object Storage**, create a dedicated private bucket (for example
   `yourtj-prod-backups`). Do **not** reuse the media/image upload bucket.
2. Create an R2 API token scoped **only** to this backup bucket with **Object Read & Write**
   permissions.
3. Generate two independent client-side encryption passphrases locally and store them offline in a
   password manager (losing them makes backups in R2 permanently unreadable):
   ```bash
   openssl rand -base64 32   # PGBACKREST_CIPHER_PASS
   openssl rand -base64 32   # RESTIC_PASSWORD
   ```
4. Optional ransomware hardening: configure an R2 Bucket Lifecycle rule (e.g. expire objects after
   45–60 days) and set `RESTIC_SKIP_PRUNE=true` in `/opt/yourtj/backup/r2.env` if you restrict the
   host token from deleting objects (and keep a separate admin token offline for periodic pruning).

## 2. One-time server bootstrap

On the production host (as `root`):

```bash
# 1. Build the PostgreSQL 16 + pgBackRest image (same Alpine musl base as postgres:16-alpine)
docker build -t yourtj-postgres-backup:16-alpine services/postgres-backup

# 2. Install restic, scripts, systemd timers, and seed /opt/yourtj/backup/r2.env (mode 0600)
bash deploy/scripts/install-r2-backup.sh

# 3. Populate /opt/yourtj/backup/r2.env with R2 credentials and encryption passphrases
chmod 0600 /opt/yourtj/backup/r2.env
$EDITOR /opt/yourtj/backup/r2.env

# 4. Render /opt/yourtj/backup/pgbackrest/pgbackrest.conf (mode 0600, owned by PG container uid/gid)
#    and enable the daily backup + 30-minute health-check timers
bash deploy/scripts/install-r2-backup.sh --enable-timers
```

Enable WAL archiving in `/opt/yourtj/.env`:

```dotenv
POSTGRES_IMAGE=yourtj-postgres-backup:16-alpine
PG_ARCHIVE_MODE=on
PG_ARCHIVE_COMMAND=/usr/local/bin/pgbackrest-archive-push.sh %p
PG_ARCHIVE_TIMEOUT=300
```

Recreate only the `postgres` container (existing `pgdata` volume is preserved), create the
pgBackRest stanza, initialize `restic`, and take the first full backup:

```bash
cd /opt/yourtj
docker compose --env-file .env -f docker-compose.yaml up -d postgres

# Initialize pgBackRest stanza and verify WAL archive round-trip to R2
docker exec -u postgres yourtj-postgres pgbackrest --stanza=main stanza-create
docker exec -u postgres yourtj-postgres pgbackrest --stanza=main check

# Initialize the encrypted restic repository in R2 once
bash -c 'set -a; source /opt/yourtj/backup/r2.env; set +a;
  AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID" \
  AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY" \
  RESTIC_REPOSITORY="s3:https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com/${R2_BUCKET}/${R2_RESTIC_PREFIX:-restic}" \
  restic init'

# Run the first full PG + restic backup and confirm health
/opt/yourtj/scripts/backup-r2.sh --pg-type full
/opt/yourtj/scripts/check-backup-health.sh
```

## 3. Ongoing schedule & alerting

`install-r2-backup.sh --enable-timers` installs two systemd timers:

- `yourtj-backup-r2.timer`: runs `/opt/yourtj/scripts/backup-r2.sh` daily at `03:30` (+ up to 10 min
  jitter). Takes a `full` pgBackRest backup on Sunday (`PG_FULL_BACKUP_DOW=7`) and `diff` backups
  Monday–Saturday, plus a verified `pg_dump -Fc` and consistent `file.db`/secrets snapshot via
  `restic`.
- `yourtj-backup-health.timer`: runs `/opt/yourtj/scripts/check-backup-health.sh` every 30 minutes
  and checks host disk usage, `pg_wal` directory size, `pg_stat_archiver` errors/freshness,
  `pgBackRest` latest backup age, and `restic` latest snapshot age.

Set `BACKUP_ALERT_WEBHOOK_URL` in `/opt/yourtj/backup/r2.env` to receive JSON failure alerts.
Inspect logs anytime with:

```bash
journalctl -u yourtj-backup-r2.service -n 50 --no-pager
journalctl -u yourtj-backup-health.service -n 50 --no-pager
docker exec -u postgres yourtj-postgres pgbackrest --stanza=main info
```

## 4. Isolated PITR verification drill (monthly)

Verify recoverability without touching the running database or live `pgdata` volume:

```bash
# Restore latest WAL tip into a temporary container, run table & anonymous-identity checks,
# verify 5% of restic pack data, then destroy the temporary container automatically
/opt/yourtj/scripts/verify-pitr-restore.sh

# Or verify recovery to an exact timestamp (PITR)
/opt/yourtj/scripts/verify-pitr-restore.sh --target-time "2026-11-23 12:00:00+08"
```

## 5. Recovery runbooks

### A. Point-in-time recovery (accidental DELETE / DROP / bad migration)

1. Stop forum writes to prevent divergent transactions:
   ```bash
   cd /opt/yourtj
   docker compose --env-file .env -f docker-compose.yaml stop main dev
   ```
2. First rehearse the chosen timestamp in isolation and confirm the target rows exist:
   ```bash
   /opt/yourtj/scripts/verify-pitr-restore.sh --target-time "YYYY-MM-DD HH:MM:SS+08"
   ```
3. Take a forensic copy of the current `pgdata` volume before overwriting anything, stop `postgres`,
   then run `pgbackrest restore` targeting the verified timestamp:
   ```bash
   docker compose --env-file .env -f docker-compose.yaml stop postgres
   docker run --rm --user 70:70 \
     -v yourtj_pgdata:/var/lib/postgresql/data \
     -v /opt/yourtj/backup/pgbackrest:/etc/pgbackrest:ro \
     --entrypoint pgbackrest \
     yourtj-postgres-backup:16-alpine \
     --stanza=main --delta --type=time --target="YYYY-MM-DD HH:MM:SS+08" \
     --target-action=promote restore
   ```
4. Start `postgres` and wait until `SELECT NOT pg_is_in_recovery();` returns `t`.
5. Rotate the Agent epoch before restarting `main` (see
   [Agent isolation during database replacement](deployment.md#agent-isolation-during-database-replacement)):
   ```bash
   python3 /opt/yourtj/scripts/rotate-agent-epoch.py /opt/yourtj/main/storage/agent-state
   ```
6. Start `main` (`docker compose up -d main`), verify `/health`, anonymous identity invariants
   ([anonymous-identity.md](anonymous-identity.md#migration-backup-and-recovery)), and trigger
   Meilisearch reconciliation (`/admin/search`) so search projections match the restored database.

### B. Full disaster recovery (new VM from scratch)

1. Provision a clean host with Docker Engine + Compose + `restic` + `sqlite3`, build
   `yourtj-postgres-backup:16-alpine`, and recreate `/opt/yourtj/backup/r2.env` (`0600`) and
   `/opt/yourtj/backup/pgbackrest/pgbackrest.conf` (`0600`, uid/gid `70:70`) using your offline
   R2 credentials and encryption passphrases.
2. Restore files, keys, and `file.db` from `restic`:
   ```bash
   set -a; source /opt/yourtj/backup/r2.env; set +a
   export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
   export AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
   export RESTIC_REPOSITORY="s3:https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com/${R2_BUCKET}/${R2_RESTIC_PREFIX:-restic}"
   restic restore latest --tag yourtj-main --target /tmp/yourtj-restore
   ```
   Copy back `/opt/yourtj/.env` (`0600`), `docker-compose.yaml`, `main/config.toml` (`0600` or
   `0640`), `main/storage/database/file.db`, `main/storage/oidc/signing_key.pem` (`0600`),
   `main/storage/push/*` (`0600`), and `main/storage/agent-state/` (`chown -R 1000:1000` on
   `main/storage`). **Preserve `app.signingKey` and `oidc/signing_key.pem` verbatim** so sessions,
   TOTP secrets, and encrypted settings stay decryptable.
3. Restore PostgreSQL from R2 via `pgbackrest --stanza=main --target-action=promote restore` into
   the `yourtj_pgdata` volume (or fall back to `pg_restore` using `postgres/yourtj_main.dump` from
   the restored `restic` snapshot).
4. Start `postgres`, `meilisearch`, and `main`, verify `/health`, restore 1Panel OpenResty configs
   and TLS certs from `/tmp/yourtj-restore/openresty`, rebuild Meilisearch indexes, and delete
   `/tmp/yourtj-restore`.

See [Deployment & release](deployment.md) and
[Anonymous identity operations](anonymous-identity.md) for deployment topology and restricted-table
recovery invariants.
