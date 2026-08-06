# Deployment & Release

> Doc type: operations
>
> Status: Active (deployment shape decided + runbooks landed)
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-06

## Deployment shape

- **Single binary**: `make build` produces `bin/yourtj-hub` (frontend static/dist + GoHTML templates
  go:embed). The binary runs inside a minimal `alpine` container (`deploy/Dockerfile`).
- Runtime deps: SQLite (default, zero external deps), MySQL, or PostgreSQL (main db only; the file
  database `[db.file]` stays SQLite); optional Meilisearch; Casdoor planned.
- **Reverse proxy + SSL by 1Panel** (openresty): `forum.yourtj.de` → `127.0.0.1:5234` (main),
  `dev.yourtj.de` → `127.0.0.1:5235` (dev). Both behind Cloudflare (proxied, origin IP hidden).
- Backend containers bind `127.0.0.1` only; nothing else is exposed publicly.
- **Two instances on one VM** (Ubuntu 24.04, ssh alias `yourtj`), managed as one compose project:
  - `main` — production, `/opt/yourtj/main`
  - `dev` — test line, `/opt/yourtj/dev`
  - DB sync is one-way: dev gets a consistent snapshot of main on each deploy (below).

## Server layout (1Panel container orchestration)

```
/opt/yourtj/
  .env                    # MAIN_PORT/DEV_PORT/MAIN_TAG/DEV_TAG + POSTGRES_USER/POSTGRES_PASSWORD/MEILI_MASTER_KEY (created by init-server.sh)
  docker-compose.yaml     # main + dev services (created by init-server.sh)
  config.toml.example     # template with REPLACE_* placeholders
  build/
    Dockerfile            # alpine + binary
  scripts/                # snapshot-db.sh, sync-db-from-main.sh, backup-db.sh, deploy.sh
  main/
    config.toml           # production config (signingKey, db path) — never in git
    storage/              # sqlite.db + file.db + logs (uid 1000) — PG 部署时 sqlite.db 不产生
  dev/
    config.toml           # dev config
    storage/
  snapshots/
    main/sqlite-*.db      # pre-deploy backups (keep 7) — SQLite 部署
    main/pg-*.sql         # pre-deploy pg_dump backups (keep 7) — PostgreSQL 部署
```

## Branch model & CI/CD

- `dev` is the default branch and the main development line; merges to `dev` trigger
  `.github/workflows/deploy-dev.yml`:
  1. Build single binary (frontend + go build) on GitHub Actions.
  2. Upload binary via scp; SSH: `sync-db-from-main.sh` (auto-detects mode: SQLite `.backup` snapshot
     or PG `pg_dump|psql` rebuild of dev db).
  3. SSH: `deploy.sh dev <binary> dev-<sha> 5235` → build image, compose up, health check, rollback.
- `main` is the production site; merges to `main` trigger `.github/workflows/deploy-main.yml`:
  1. Build single binary on GitHub Actions.
  2. SSH: `backup-db.sh main` (pre-deploy consistent snapshot, keep 7).
  3. SSH: `deploy.sh main <binary> main-<sha> 5234` → build image, compose up, health check,
     auto-rollback to previous image tag on failure.
- **Release gate**: `.github/workflows/release-to-main.yml` (manual `workflow_dispatch`) merges `dev` →
  `main`, bumps the version (`patch` / `minor` / `major`, computed from the latest `vX.Y.Z` tag, first
  release: patch → `v0.0.1`, minor → `v0.1.0`, major → `v1.0.0`), tags it, and pushes via a PAT
  (secret `RELEASE_TOKEN`) so `deploy-main` triggers. Run it from Actions → Release to main → Run
  workflow → choose bump type.
- Why dev syncs main's db: migrations (`app/migration` AutoMigrate + versioned data migrations) run at
  startup, so each dev deploy rehearses the exact migration the next main deploy will run.
- Config is pre-provisioned on the server (`init-server.sh`) and never passes through CI.

## GitHub Actions secrets

| Secret | Value |
|---|---|
| `VM_HOST` | server public IP or hostname (`20.205.27.178`) |
| `VM_USER` | SSH user (e.g. `yourtj`) |
| `VM_SSH_KEY` | private key for that user (full PEM, including `-----BEGIN ...` lines) |

Deploy workflows use `appleboy/scp-action` + `appleboy/ssh-action` with these secrets.

## First-time server setup

```bash
# on the server, as root (or sudo):
sudo bash /opt/yourtj/scripts/init-server.sh \
  https://forum.yourtj.de https://dev.yourtj.de
```

This creates `/opt/yourtj/{.env,build,docker-compose.yaml,main,dev}` with randomized signing keys.
The script itself is deployed to the server by the first CI run (or copy `deploy/` manually).

## Build (local)

```bash
make build     # cd apps/gooseforum/resource && pnpm build → cd apps/gooseforum && go build -o ../../bin/yourtj-hub .
```

## Config & run

- Config: `apps/gooseforum/config.toml` locally; on the server `main/config.toml` / `dev/config.toml`.
- Container-internal port is always `5234`; host mapping via `MAIN_PORT` (5234) / `DEV_PORT` (5235).
- Health probe: `GET /health` returns 200 when service + main db ping succeed, else 503.

## DB migration execution and rollback

- Migrations run at startup (`[db] migration = "on"`); append-only upstream style.
- Rollback: `deploy.sh` tags the previous image `yourtj-hub:prev` and re-points the instance on
  health-check failure; forward-compatible migrations mean an older binary can still start.
- Pre-deploy snapshot in `snapshots/main/` is the data-level restore point (SQLite).

### Upgrade note: `app.signingKey` is now mandatory

Since the issue #8 build, `serve` refuses to start with the built-in default
signing key (it exits with code 1 instead of silently continuing). Existing
`config.toml` files created before this change may omit `app.signingKey`;
**before upgrading an existing instance, add a random signing key** to
`main/config.toml` and `dev/config.toml`:

```toml
[app]
signingKey = "<random 32+ byte base64 string>"
```

Generate one with `openssl rand -base64 32`. Without it the new binary exits
immediately on startup; `init-server.sh` already generates a random key for
new installs.

### Unique-index preflight (user_o_auth provider_uid)

Issue #8 added a unique index on `(provider, provider_uid)` in `user_o_auth`. On databases that
already contain rows, `AutoMigrate` fails silently if duplicates exist (the schema logger records the
error and startup continues). Before deploying a build containing this index, run the duplicate check
on the live database and clean up any duplicates:

```sql
-- SQLite / MySQL
SELECT provider, provider_uid, COUNT(*) FROM user_o_auth GROUP BY provider, provider_uid HAVING COUNT(*) > 1;
-- PostgreSQL
SELECT provider, provider_uid, COUNT(*) FROM user_o_auth GROUP BY provider, provider_uid HAVING COUNT(*) > 1;
```

If duplicates exist, keep the row with the earliest `created_at` (or the one owned by the active
account) and delete the rest before the upgrade; the index creation will then succeed.
## PostgreSQL support

Since issue #11 the main database (`[db.default]`) can run on PostgreSQL 16+ in addition to the
default SQLite and optional MySQL. The file database (`[db.file]`, attachment BLOBs) remains SQLite.

### Enable PostgreSQL

1. Uncomment the `postgres` service in `deploy/docker-compose.yaml` and set `POSTGRES_USER` /
   `POSTGRES_PASSWORD` in `/opt/yourtj/.env`.
2. Create **two separate databases** to keep main (production) and dev (test) isolated, matching the
   SQLite deployment model (dev is one-way synced from main, never written directly):
   ```bash
   # 在 postgres 容器内执行
   docker compose exec postgres psql -U yourtj -d postgres -c \
     "CREATE DATABASE yourtj_main; CREATE DATABASE yourtj_dev;"
   ```
   Do **not** point both instances at the same database — dev migrations/writes would land on
   production data.
3. In `main/config.toml` set:
   ```toml
   [db.default]
   connection = "postgres"
   url = "host=postgres user=yourtj password=<secret> dbname=yourtj_main port=5432 sslmode=disable"
   ```
   In `dev/config.toml` use the same DSN but `dbname=yourtj_dev`. `host=postgres` is the Compose
   service name: the forum and postgres containers share the compose network, so `127.0.0.1` inside
   the forum container would point at the forum container itself and fail to connect.
   `url` accepts libpq key=value or URL DSN formats.
4. Start the instance. On first boot the binary runs AutoMigrate (all main-db models) and the
   versioned data migrations v1-v12 from scratch, then serves.

### SQLite → PostgreSQL data migration (manual, no automated tool)

The Blueprint explicitly does not ship an automated SQLite→PG migration tool. To move an existing
instance:

1. `pg_dump` the target schema (or let the binary AutoMigrate an empty PG database first).
2. Export data from SQLite (`sqlite3 main.db .dump`) and load it into PG with type adjustments
   (`bigint unsigned` → `bigint`, `tinyint` → `smallint`, `datetime` → `timestamp`).
3. Set `[db.default] connection="postgres"`, start the binary, verify `/health` 200 and spot-check
   topics/posts/users reads.
4. Keep the SQLite file as the rollback snapshot until the PG instance has been stable.

### Backup and sync scripts under PostgreSQL

- `backup-db.sh` / `sync-db-from-main.sh` auto-detect the main-db mode from each instance's
  `config.toml`:
  - SQLite: use the SQLite `.backup` API (unchanged legacy path).
  - PostgreSQL: `backup-db.sh` runs `pg_dump` into `snapshots/<instance>/pg-<db>-<ts>.sql`;
    `sync-db-from-main.sh` drops/recreates `yourtj_dev` and pipes
    `pg_dump -d yourtj_main | psql -d yourtj_dev` (dev is a clean one-way snapshot of main).
- Manual equivalent inside the postgres container:
  ```bash
  docker compose exec postgres sh -c \
    'pg_dump -U yourtj -d yourtj_main | psql -U yourtj -d yourtj_dev'
  ```
- `snapshot-db.sh` remains a generic SQLite snapshot helper; it is not used by PG deployments.

### Logging configuration (issue #11)

`[log]` supports `level` (debug/info/warn/error, default info), `format` (json/console, default json),
`errorPath` (WARN/ERROR go to a separate rotating file, e.g. `run.error.log`), and `logIp`
(default false — access logs omit the client IP for privacy). Changing these requires a restart.

## Runbooks to write

- Casdoor production config (domain, certs, client registration)
- Meilisearch index rebuild, backup
- Logging & monitoring (config [log] slow SQL, rolling logs; health probes)
