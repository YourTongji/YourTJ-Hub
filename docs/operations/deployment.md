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
- Runtime deps: **PostgreSQL 16+ is the default deployment database** (`deploy/config.toml.example`
  `[db.default] connection = "postgres"`); SQLite is the local development/test default; the file
  database `[db.file]` stays SQLite; MySQL is **not supported**; optional Meilisearch; optional
  built-in OIDC Provider ([oidc] in config.toml).
- **Reverse proxy + SSL by 1Panel** (openresty): `forum.yourtj.de` → `127.0.0.1:5234` (main),
  `dev.yourtj.de` → `127.0.0.1:5235` (dev). Both behind Cloudflare (proxied, origin IP hidden).
- Backend containers bind `127.0.0.1` only; nothing else is exposed publicly.
- **Trusted proxies**: the binary only trusts `127.0.0.1`/`::1` reverse proxies by default
  (`engine.SetTrustedProxies`). If an additional proxy sits in front of the binary, add it to
  `server.trusted_proxies` in `config.toml` so rate-limit IP attribution cannot be bypassed via
  a forged `X-Forwarded-For` header.
- **Two instances on one VM** (Ubuntu 24.04, ssh alias `yourtj`), managed as one compose project:
  - `main` — production, `/opt/yourtj/main`
  - `dev` — test line, `/opt/yourtj/dev`
  - DB sync is one-way: dev gets a consistent snapshot of main on each deploy (below).
- **Wiki static site** (VitePress + Pagefind, separate from the binary): two nginx containers
  `wiki-main` (127.0.0.1:5284) / `wiki-dev` (127.0.0.1:5285) in the same compose project, deployed by
  `deploy-wiki.sh` from the `wiki-dist` artifact built in CI. Public reverse-proxy DNS for the wiki is
  a post-merge ops task.

## Server layout (1Panel container orchestration)

```
/opt/yourtj/
  .env                    # MAIN_PORT/DEV_PORT/MAIN_TAG/DEV_TAG/WIKI_*_TAG + POSTGRES_USER/POSTGRES_PASSWORD/MEILI_MASTER_KEY (created by init-server.sh)
  docker-compose.yaml     # main + dev + meili + wiki-main + wiki-dev services (created by init-server.sh)
  config.toml.example     # template with REPLACE_* placeholders
  build/
    Dockerfile            # alpine + binary
    wiki.Dockerfile       # nginx + wiki static dist
    wiki-dist/            # unpacked wiki dist (deploy-wiki.sh)
  scripts/                # snapshot-db.sh, sync-db-from-main.sh, backup-db.sh, deploy.sh, deploy-wiki.sh, bootstrap-wiki-assets.sh, pgdsn.sh
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
  3. SSH: `deploy.sh dev <binary> dev-<sha> 5235` → build image, compose up, health check, rollback;
     after a successful deploy the script prunes old images (keeps the newest
     `IMAGE_KEEP_N` tags of the instance prefix including the current one, plus the `prev`
     rollback tag) and build cache older than 72h.
     The dev workflow sets `IMAGE_KEEP_N=3` because dev deploys frequently.
  4. SSH: `deploy-wiki.sh dev /tmp/wiki-dist.tar.gz wiki-dev-<sha> 5285` → build nginx image from the
     unpacked static dist, compose up, health check, rollback (the `wiki-build` job builds the site
     and uploads the `wiki-dist` artifact).
- `main` is the production site; merges to `main` trigger `.github/workflows/deploy-main.yml`:
  1. Build single binary on GitHub Actions.
  2. SSH: `backup-db.sh main` (pre-deploy consistent snapshot, keep 7).
  3. SSH: `deploy.sh main <binary> main-<sha> 5234` → build image, compose up, health check,
     auto-rollback to previous image tag on failure; same post-deploy image/cache pruning as dev
     (`IMAGE_KEEP_N=5`, keeps more rollback candidates for production).
  4. SSH: `deploy-wiki.sh main /tmp/wiki-dist.tar.gz wiki-main-<sha> 5284` → same as dev, on 5284.
- **Release gate**: `.github/workflows/release-to-main.yml` (manual `workflow_dispatch`) merges `dev` →
  `main`, bumps the version (`patch` / `minor` / `major`, computed from the latest `vX.Y.Z` tag, first
  release: patch → `v0.0.1`, minor → `v0.1.0`, major → `v1.0.0`), tags it, and pushes via a PAT
  (secret `RELEASE_TOKEN`) so `deploy-main` triggers. Run it from Actions → Release to main → Run
  workflow → choose bump type.
- Why dev syncs main's db: migrations (`app/migration` AutoMigrate + versioned data migrations) run at
  startup, so each dev deploy rehearses the exact migration the next main deploy will run.
- Config is pre-provisioned on the server (`init-server.sh`) and never passes through CI.
- `sync-db-from-main.sh` hard-fails when `main`/`dev` primary DB modes differ (e.g. main already
  migrated to PG while dev is still SQLite): the sync cannot proceed, and the script refuses to
  stop the dev container in that state (parse-before-stop guarantee, issue #134). During the PG
  migration window both instances must be on the same mode before deploying dev.
- Deploy workflows checkout the repo and upload the deploy scripts (`deploy.sh` plus the ones they
  depend on: `backup-db.sh` / `sync-db-from-main.sh` and their shared DSN parser `pgdsn.sh`) to
  `/opt/yourtj/scripts/` before running them, so script fixes reach the server without a
  manual `init-server.sh` re-run. `pgdsn.sh` is a runtime dependency (`source`d by
  `backup-db.sh` / `sync-db-from-main.sh`); keep it in the scp/install list whenever deploying
  script updates.
- Wiki deploy assets are also CI-provisioned: each deploy uploads
  `deploy/build/wiki.Dockerfile` / `wiki.nginx.conf` / `docker-compose.yaml` and runs
  `bootstrap-wiki-assets.sh`, which idempotently installs them under `/opt/yourtj/build` and appends
  missing `WIKI_*` vars to `/opt/yourtj/.env`. Existing servers therefore need no manual
  `init-server.sh` re-run before the first wiki deploy; the compose file is only replaced when the
  `wiki-*` services are missing (preserving any server-side local edits).

## GitHub Actions secrets

| Secret | Value |
|---|---|
| `VM_HOST` | server public IP or hostname (`20.205.27.178`) |
| `VM_USER` | SSH user (e.g. `yourtj`) |
| `VM_SSH_KEY` | private key for that user (full PEM, including `-----BEGIN ...` lines) |
| `WIKI_WALINE_SERVER_URL` | optional Waline comment server URL, injected at wiki build time (`VITE_WALINE_SERVER_URL`); empty = comments disabled |

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
- **Migration failures now abort startup** (since the issue #8 PG fix): if `AutoMigrate` errors,
  `serve` exits non-zero instead of continuing with a partial schema. This makes deploy.sh's
  health-check rollback and container restart policies catch schema problems immediately instead of
  surfacing as runtime API failures (the original issue #8 login/register outage). It also means a
  deploy with an incompatible schema change will roll back — rehearse on dev (which syncs main's db)
  before main.
- Rollback: `deploy.sh` tags the previous image `yourtj-hub:prev` and re-points the instance on
  health-check failure; forward-compatible migrations mean an older binary can still start.
- Pre-deploy snapshot in `snapshots/main/` is the data-level restore point (SQLite).

### Upgrade note: `app.signingKey` is now mandatory and fail-closed

Since the issue #8 build, `serve` refuses to start with the built-in default
signing key (it exits with code 1 instead of silently continuing). The
issue #106 build tightens the guard to **fail-closed**: `serve` now also
refuses any **empty / whitespace-only** value and the deploy template
placeholder `REPLACE_SIGNING_KEY`, because each of them lets an attacker
forge password-reset tokens and take over arbitrary accounts (including
admin). A missing or weak `app.signingKey` is rejected every boot — there is
no fallback key. Existing `config.toml` files created before this change may
omit `app.signingKey`; **before upgrading an existing instance, add a random
signing key** to `main/config.toml` and `dev/config.toml`:

```toml
[app]
signingKey = "<random 32+ byte base64 string>"
```

Generate one with `openssl rand -base64 32`. Without it the new binary exits
immediately on startup; `init-server.sh` already generates a random key for
new installs.

**Rotation after a known/empty-key exposure:** if an instance ever ran with
the built-in default, the `REPLACE_SIGNING_KEY` placeholder, or an empty
`app.signingKey`, treat the signing key as compromised and rotate it. The
symmetric key is shared across three surfaces, so rotation invalidates all of
them at once: forum JWT sessions (users must sign in again), TOTP secret
encryption (AES-GCM key is derived from `app.signingKey`; re-encrypt or
re-enroll TOTP so existing secrets remain decryptable), and any outstanding
password-reset / activation links. Rotating the key is the only way to retire
tokens that were minted under the old key.

**Rotation requires a process restart — hot reload is not supported.** The
signing key feeds more surfaces than the three listed above — it also derives
the session-cookie signing key (sessionstore) and the OIDC opaque-token key
(oidcservice). The surfaces capture it at different points: JWT signing and
session cookies at process start / first use, TOTP encryption on first use, and
reset/activation and OIDC tokens on every call (fail-closed so a weak key is
never accepted). With viper's config watcher enabled, editing `app.signingKey`
at runtime does not rotate them together: the real-time surfaces switch to the
new key immediately, the captured surfaces keep the old value, and TOTP secrets
encrypted under the old key can become undecryptable if the key is swapped
before the first TOTP use. Rotate the key and **restart the process** so all
surfaces rotate consistently.

### Upgrade note: session Cookie `Secure` is now fail-closed by environment

Before issue #113, the `access_token` and goth session cookies decided the
`Secure` flag from the `server.url` scheme: anything `http://` dropped `Secure`
even under `app.env = "production"`. The template default
`server.url = "http://localhost"` thus produced a session cookie without
`Secure` on a production build (CWE-614). Since the issue #113 build, the flag
is fail-closed by environment via `setting.CookieSecure()`: **any `app.env`
other than `"local"` forces `Secure` regardless of `server.url`**, and the
binary logs a startup warning when a non-local `server.url` is non-https and
non-loopback. No `config.toml` change is required for existing instances, but
operators who relied on plain-http production access (e.g. 0.0.0.0 without a
TLS-terminating proxy) should switch `server.url` to the https reverse-proxy
address so browsers actually return the now-`Secure` cookies.

### Unique-index preflight (user_o_auth provider_uid)

Issue #8 added a unique index on `(provider, provider_uid)` in `user_o_auth`. On databases that
already contain rows, `AutoMigrate` fails silently if duplicates exist (the schema logger records the
error and startup continues). Before deploying a build containing this index, run the duplicate check
on the live database and clean up any duplicates:

```sql
-- SQLite / PostgreSQL
SELECT provider, provider_uid, COUNT(*) FROM user_o_auth GROUP BY provider, provider_uid HAVING COUNT(*) > 1;
```

If duplicates exist, keep the row with the earliest `created_at` (or the one owned by the active
account) and delete the rest before the upgrade; the index creation will then succeed.

### Unique username migration preflight

The `users.username` unique index is shared by human and bot accounts. Before `AutoMigrate` creates
that index, startup checks an existing `users` table for blank or duplicate usernames. The binary
does not rewrite identity data automatically: if dirty rows exist, startup exits non-zero with the
blank-row count, up to ten duplicate usernames, and an instruction to assign non-empty globally
unique usernames before restarting. Because dev receives a production snapshot, resolve the report
on the authoritative main dataset, resync dev, and rehearse the migration there before releasing to
main.

Operator checks for all supported databases:

```sql
SELECT COUNT(*) FROM users WHERE username = '';
SELECT username, COUNT(*) FROM users
WHERE username <> ''
GROUP BY username
HAVING COUNT(*) > 1;
```

## Database: PostgreSQL default

PostgreSQL 16+ is the default deployment database (`[db.default] connection = "postgres"`,
`deploy/config.toml.example`). SQLite remains the local development/test default
(`apps/gooseforum/config.toml`, in-memory tests) and the file database (`[db.file]`, attachment
BLOBs) is fixed SQLite. MySQL is **not supported** and its connection driver was removed.

### Deploy on PostgreSQL

Fresh installs default to PostgreSQL end to end — `init-server.sh` does the
provisioning automatically:

1. Generates `POSTGRES_USER` / a random `POSTGRES_PASSWORD` / `POSTGRES_DB` into
   `/opt/yourtj/.env` (missing keys are appended on existing servers; empty
   `POSTGRES_PASSWORD` is replaced with a random value).
2. Starts the `postgres` service (defined in `deploy/docker-compose.yaml`) and waits for it,
   then creates **two separate databases** — `yourtj_main` and `yourtj_dev` — so main
   (production) and dev (test) stay isolated (dev is one-way synced from main, never written
   directly). Do **not** point both instances at the same database — dev migrations/writes
   would land on production data.
3. Generates `main/config.toml` / `dev/config.toml` from `deploy/config.toml.example`,
   substituting `REPLACE_POSTGRES_DSN` with a real DSN per instance:
   ```toml
   [db.default]
   connection = "postgres"
   url = "host=postgres user=<user> password=<secret> dbname=yourtj_main port=5432 sslmode=disable"  # dev 用 yourtj_dev
   ```
   `host=postgres` is the Compose service name: the forum and postgres containers share the
   compose network, so `127.0.0.1` inside the forum container would point at the forum
   container itself and fail to connect. `url` accepts libpq key=value or URL DSN formats.
4. Start the instance (`deploy.sh` / `docker compose up -d main dev`; compose starts
   `postgres` first via `depends_on: service_healthy`). On first boot the binary runs
   AutoMigrate (all main-db models) and the versioned data migrations from scratch, then
   serves.


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
  - PostgreSQL mode also snapshots dev's existing `file.db` and copies main's `file.db` to dev, because `[db.file]` remains SQLite even when the main database is PostgreSQL.
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

## Storage (object storage) configuration

- Files default to SQLite BLOB (no external dependency). To move uploads to an S3-compatible
  object store (MinIO / Tencent COS / Alibaba OSS / Cloudflare R2), configure it in the admin panel
  (设置 → 存储设置): provider `s3`, endpoint, bucket, region, bucket lookup, access/secret keys,
  optional public URL prefix (CDN direct reads).
- Credentials are stored in `page_config` (same handling as SMTP password today). Keep the bucket
  private; the forum proxies reads through `/file/img/*` unless a public prefix is configured.
- Addressing: Alibaba OSS and Tencent COS (buckets created after 2024-01-01) only support
  virtual-hosted style — set bucket lookup `dns` and the bucket region explicitly. MinIO/R2 accept
  `auto`/`path`. The endpoint may include `https://`; the forum strips the scheme and derives TLS
  from it (or from the secure toggle).
- Migrating existing BLOB files: admin panel (存储设置 → 迁移文件) creates a background task with
  progress; or run `./bin/yourtj-hub migrate-files --endpoint ... --bucket ... [--clear-after-migrate]`
  on the server. Migration is cursor-driven and resumable; the BLOB column is kept unless
  `--clear-after-migrate` is set, so reads stay correct during/after migration.

## Data export/import

- Admin panel (数据管理): export users/topics/posts (plus derived topic_category_index /
  topic_user_stat when selected) as JSON or CSV via a background task, then download;
  import JSON with a per-row validation report and idempotent skip; topic invariants
  (post_seq, first/last post pointers, counts, posters) are preserved and rebuilt on import.
- Export files are written to `data/export/` inside the storage dir and cleaned up after 7 days
  (daily cron). Export contains user emails — treat downloads as sensitive.

## 一系统排课同步（course-pk-sync，issue #186）

将同济一系统（1.tongji.edu.cn）排课数据分页同步到 PK 域，并重建 `teacher_timeslots`。

```bash
# 首次同步请用数字 calendarId（或 --calendar-id）；学期名（2025-2026-1）需在 pk_calendar
# 已有记录后才可反查（同一学期同步过一次即可）
./bin/yourtj-hub course-pk-sync 121
./bin/yourtj-hub course-pk-sync 2025-2026-1 --calendar-id 121
./bin/yourtj-hub course-pk-sync 2025-2026-1   # 学期名在已同步过的实例上可用

# 连同步前 3 个学期（选课季加频/补历史）
./bin/yourtj-hub course-pk-sync 121 --depth 3

# 同步后物化到课程目录（默认关闭；写 course/course_alias/course_instructor + 课程搜索 outbox）
./bin/yourtj-hub course-pk-sync 121 --materialize
```

凭证优先级：`--onesystem-cookie` 参数 > `ONESYSTEM_COOKIE` 环境变量 > 管理端设置
（设置 → 一系统同步；`save-onesystem-settings` 仅落库 securestore 密文，不存明文）。
- 运维 cron（每日，选课季加频；应用内不自造调度器）：

  ```bash
  # 每日 02:30 同步当前学期
  30 2 * * * cd /srv/yourtj-hub && ONESYSTEM_COOKIE='JWTUser=…; JSESSIONID=…' ./bin/yourtj-hub course-pk-sync 121
  ```

- 行为保证：同一学期重复执行先清空再全量重写（幂等，不翻倍）；同步中断后重跑从失败批次
  续跑（`pk_fetch_log` 游标），不回滚已成功批次；Cookie 失效时报 HTTP 状态与提示并标记
  fetchlog `failed`，且不删除存量数据；无效 Cookie 不会破坏已同步数据。
- 并发防护：同一学期存在 1 小时内的 `running` fetchlog 时拒绝新同步（避免两个进程互相删数据）；
  进程崩溃后若需立即重跑，可等待窗口过期或手动清掉该学期 `pk_fetch_log`。
- 注意：`app.signingKey` 轮换会使管理端已存的一系统 Cookie 密文失效（与 TOTP 相同），
  需到管理端重新保存。

## Runbooks to write

- Built-in OIDC Provider production config ([oidc] in config.toml: enabled, issuer, signing key, clients)
- Meilisearch index rebuild, backup
- Logging & monitoring (config [log] slow SQL, rolling logs; health probes)
