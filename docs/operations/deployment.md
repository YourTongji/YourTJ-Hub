# Deployment & Release

> Doc type: operations
>
> Status: Active (deployment shape decided + runbooks landed)
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-14

## Deployment shape

- **Single binary**: `make build` produces `bin/yourtj-hub` (frontend static/dist + GoHTML templates
  go:embed). The binary runs inside a minimal `alpine` container (`deploy/Dockerfile`).
- Runtime deps: **PostgreSQL 16+ is the default deployment database** (`deploy/config.toml.example`
  `[db.default] connection = "postgres"`); SQLite is the local development/test default; the file
  database `[db.file]` stays SQLite; MySQL is **not supported**; optional Meilisearch; optional
  built-in OIDC Provider ([oidc] in config.toml).
- **Image pipeline**: CI builds the single binary and packages it into an OCI image pushed to GHCR
  (`ghcr.io/yourtongji/yourtj-hub:<instance>-<sha>`; repo is public, so servers pull anonymously —
  no registry credentials on the server).
- **Reverse proxy**: public TLS is terminated by the **1Panel reverse proxy** on the host
  (port 80/443), which proxies `forum.yourtj.de` → `127.0.0.1:5234` (main) and
  `dev.yourtj.de` → `127.0.0.1:5235` (dev). There is no nginx container in the compose file;
  backend containers bind `127.0.0.1` only, and only 1Panel/SSH is exposed publicly.
- **Trusted proxies**: the binary only trusts `127.0.0.1`/`::1` reverse proxies by default
  (`engine.SetTrustedProxies`). 1Panel reverse-proxies from localhost, so
  `server.trusted_proxies` defaults to `["127.0.0.1", "::1"]` (`deploy/config.toml.example`);
  this lets rate-limit IP attribution use the real client IP via `X-Forwarded-For`
  without trusting arbitrary networks.
- **Two instances on one VM** (Debian 12, ssh `root`), managed as one compose project:
  - `main` — production, `/opt/yourtj/main`
  - `dev` — test line, `/opt/yourtj/dev`
  - DB sync is one-way: dev gets a consistent snapshot of main on each deploy (below).
- **Wiki 分站（论坛内嵌）**: wiki 由单二进制直接服务（`/wiki` SSR 视图），无独立部署、无独立
  nginx 容器；旧 VitePress 静态站部署（deploy-wiki.sh / wiki-dist / Waline）已废弃。按 issue
  #219 的用户决策，旧 VitePress 内容不迁移，新原生 Wiki 从空站启动。**存量服务器需退役旧
  wiki 容器/镜像**（旧 bootstrap 流程初始化过的服务器的 compose 仍可能保留
  `wiki-main`/`wiki-dev` 服务定义与运行中的 `yourtj-wiki-main`(127.0.0.1:5284)/
  `yourtj-wiki-dev`(127.0.0.1:5285) 容器）:
  - main CI 部署会先备份并覆盖 `/opt/yourtj/docker-compose.yaml`，随后
    `deploy.sh up -d --remove-orphans` 会停删不再出现在当前 compose 中的旧 wiki 容器。
  - 若要在下次 CI 部署前立即退役：`docker rm -f yourtj-wiki-main yourtj-wiki-dev`
    （`restart: unless-stopped` 不会复活已 `rm` 的容器）；残留镜像手动清理
    `docker images | awk '/yourtj-wiki/{print $3}' | xargs -r docker rmi -f`
    （`deploy.sh` 的镜像清理只保留 `yourtj-hub` 前缀 tag，不含 `yourtj-wiki:*`）。
  - 若反向代理仍把旧 wiki 域名指到 127.0.0.1:5284/5285，退役后需同步摘除路由。

### 旧 VitePress wiki 内容迁移（GitHub 唯一真实源）

论坛 wiki 内容由公开 GitHub 仓库 `YourTongji/YourTJ-Wiki` 维护（PR 协作编辑），
论坛侧只读投影。旧 VitePress 静态站（`wiki-dist`/`deploy-wiki.sh`）内容如需
保留，迁移到 GitHub 仓库后由同步器自动投影：

1. 旧 VitePress 站点仓库的 `docs/`（Markdown 源文件）按新仓库结构整理：
   顶层目录 = namespace，文件 = 页面，front-matter 的 `title` 作为页面标题
   （旧站路径映射为 `<namespace>/<path>.md`）。
2. 旧站的静态资源（图片/附件）随文件一并提交到仓库。同步器只将 `.md` 作为页面投影，
   但会把页面中的仓库相对资源引用重写为论坛二进制提供的受控
   `/wiki/_assets/<repository-path>` 路由；不需要 GitHub raw URL。相对页面链接保留 `.md`
   源文件后缀，投影后会变为无后缀的 `/wiki/...` 路由。作者语义与路径限制见
   [Wiki authoring](../product/wiki-authoring.md)。
3. 旧站评论区（Waline）数据不迁移；如确有保留价值，导出 Waline 评论 JSON
   后以人工方式并入对应页面的论坛回复流（wiki 无评论表，评论走回复流）。
4. 内容提交、PR 合并后，在管理端 `/admin/wiki` → GitHub 同步面板触发一次
   同步，`/wiki` 导航树核对无误后再退役旧容器与路由（见上文 VitePress 退役）。

### Wiki GitHub 唯一真实源同步

wiki 内容由公开 GitHub 仓库 `YourTongji/YourTJ-Wiki` 维护（PR 协作编辑/审核/历史/贡献者），
论坛只保留只读投影。配置 `[wiki.git]` 后启用（见 `deploy/config.toml.example`）：

```toml
[wiki.git]
repo = "https://github.com/YourTongji/YourTJ-Wiki.git"
branch = "main"
clone_dir = "./storage/wiki-repo"
schedule = "0 3 * * *"      # 每日定时同步（默认 03:00）
webhook_secret = ""         # 兼容旧配置的明文密钥；推荐改用管理端「webhook 验签密钥」设置（securestore 加密落库）
```

- **同步触发**：进程启动时异步同步一次 + 每日定时（`[wiki.git].schedule`，默认
  `0 3 * * *`）+ 管理端 `/admin/wiki` → GitHub 同步面板「立即同步」+ GitHub webhook
  （PR merge = push 事件）。
- **命名空间/页面来源**：命名空间 = 仓库顶层目录名（支持中文等 Unicode 字符，目录
  消失自动删除命名空间）；页面 = 目录内 `.md` 文件（路径去 `.md` 后缀；frontmatter
  `title`/`order`/`description` 驱动页面标题/排序与命名空间描述，`index.md` 的
  description/order 写入命名空间元数据）。任意子目录都会在导航树中保留为可折叠目录节点，
  不要求 `index.md`；`index.md` 存在时仍是该目录下可直接访问的页面。同步完成后会重算页面
  到最近祖先索引页的 `parent_id`，所以目录新增、移动、删除或恢复不会保留已删除的父引用。
- **GitHub webhook 配置**（仓库 Settings → Webhooks → Add webhook）：
  - Payload URL：`https://forum.yourtj.de/api/wiki/webhook`（dev 实例用 `https://dev.yourtj.de/api/wiki/webhook`）
  - Content type：`application/json`；Secret：与 webhook 验签密钥一致
    （优先管理端 `/admin/wiki` → 同步面板「Webhook 验签密钥」保存，securestore 加密
    落库；也可用旧 `[wiki.git].webhook_secret` 明文配置）
  - 管理端「Webhook 验签密钥」清除后，即使 config.toml 存在旧明文 `webhook_secret` 也会保持禁用（fail-closed，需删除明文配置才可重新启用）。
  - Events：仅 `push`（PR merge 触发）
  - 验签：`X-Hub-Signature-256` = HMAC-SHA256(secret, body)，验签失败/未配置返回 403/401。
- **运行要求**：服务器需可出站访问 `github.com`（:443）；容器镜像需含 `git` 二进制
  （镜像升级后首次同步会保留仓库原始大小写/Unicode——此前实现做小写归一。对混合
  大小写仓库，首次同步会软删旧的小写路径页面并以仓库实际大小写重建（新 topic，
  原评论/互动不迁移）；当前 `YourTJ-Wiki` 仓库全小写目录，零影响）。
  URL 首段 = 仓库顶层目录名，重命名目录即改变 URL（旧链接不回退解析）。
  （同步用全量 `clone --single-branch` + `fetch` + `reset --hard`，**不使用 pull**；
  全量历史用于页面贡献者统计。存量浅克隆（旧版 `--depth=1`）在下次同步自动
  `fetch --unshallow` 补全历史并重建全部页面贡献者缓存，升级首轮耗时取决于仓库大小）。
- **本地 clone**：默认 `./storage/wiki-repo`（`main`/`dev` 实例各自独立），可被 `[wiki.git].clone_dir` 覆盖。
- **同步记录**：每次同步写入 `wiki_sync_runs`（trigger/status/变更计数/错误），管理端可查最近 20 条；
  同步幂等（正文 sha256 比对），重复同步零变更；软删页面在仓库重新出现时自动恢复（含 topic 生命周期）；
  仓库移除页面 → 页面软删（评论/互动保留），仓库移除顶层目录 → 命名空间自动删除（含贡献者记录）。
- **崩溃恢复**（issue #290）：进程被杀/重启遗留的 `running` 运行行在下次启动、状态读取（管理端
  刷新 `/admin/wiki`）或下次同步开始时统一回收为 `failed`，不会永久禁用手动同步；管理端手动
  同步 accepted 后轮询 `sync/status` + `sync/runs` 直到新 run 行进入终态并刷新页面树（约 5 分钟
  上限，超时提示手动刷新）。
- **重命名/移动（issue #288）**：Git 重命名/移动文件（内容不变）后，同步器按正文 `content_hash`
  唯一匹配收养原页面行——迁移 `path`/`namespace`/`parent_id`、恢复软删并复用原 topic，回复/点赞/
  收藏/订阅与 watcher 通知全部跟随新路径，旧 URL 不再解析（无重定向）。同 hash 多候选（复制）
  或内容同时变化时不做猜测：保持「新建 + 软删旧页」的旧行为（互动保留在旧 topic 上）。

## Server layout (Docker Compose)

```
/opt/yourtj/
  .env                    # MAIN_PORT/DEV_PORT/MAIN_TAG/DEV_TAG/IMAGE_REPO + POSTGRES_*/MEILI_MASTER_KEY (created by init-server.sh)
  docker-compose.yaml     # main + dev + postgres + meilisearch services (created by init-server.sh)
  config.toml.example     # template with REPLACE_* placeholders
  scripts/                # snapshot-db.sh, sync-db-from-main.sh, backup-db.sh, deploy.sh, pgdsn.sh
  main/
    config.toml           # production config (signingKey, db path) — never in git
    storage/              # file.db + logs (uid 1000); PG 部署时 sqlite.db 不产生
  dev/
    config.toml           # dev config
    storage/
  snapshots/
    main/pg-*.sql         # pre-deploy pg_dump backups (keep 7) — PostgreSQL 部署
```

## Branch model & CI/CD

- `dev` is the default branch and the main development line; merges to `dev` trigger
  `.github/workflows/deploy-dev.yml`:
  1. Build single binary (frontend + go build) and push GHCR image `dev-<sha>` on GitHub Actions.
  2. SSH: `sync-db-from-main.sh` (auto-detects mode: SQLite `.backup` snapshot
     or PG `pg_dump|psql` rebuild of dev db).
  3. SSH: `deploy.sh dev dev-<sha> 5235` → pull image, compose up, health check, rollback;
     after a successful deploy the script prunes old images (keeps the newest
     `IMAGE_KEEP_N` tags of the instance prefix including the current one, plus the `prev`
     rollback tag).
     The dev workflow sets `IMAGE_KEEP_N=3` because dev deploys frequently.
- `main` is the production site; merges to `main` trigger `.github/workflows/deploy-main.yml`:
  1. Build single binary and push GHCR image `main-<sha>` on GitHub Actions.
  2. SSH: `backup-db.sh main` (pre-deploy consistent snapshot, keep 7).
  3. SSH: `deploy.sh main main-<sha> 5234` → pull image, compose up, health check,
     auto-rollback to previous image tag on failure; same post-deploy image pruning as dev
     (`IMAGE_KEEP_N=5`, keeps more rollback candidates for production).
- **Release gate**: `.github/workflows/release-to-main.yml` (manual `workflow_dispatch`) merges `dev` →
  `main`, bumps the version (`patch` / `minor` / `major`, computed from the latest `vX.Y.Z` tag, first
  release: patch → `v0.0.1`, minor → `v0.1.0`, major → `v1.0.0`), tags it, and pushes via a PAT
  (secret `RELEASE_TOKEN`) so `deploy-main` triggers. Run it from Actions → `Release / main` → Run
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

## GitHub Actions secrets

| Secret | Value |
|---|---|
| `VM_HOST` | server public IP or hostname (`43.108.84.213`) |
| `VM_USER` | SSH user (`root`) |
| `VM_SSH_KEY` | PEM private key for that user (full PEM, including `-----BEGIN ...` lines; e.g. `YourTJ_Korean.pem`) |
| `VM_SSH_PORT` | SSH port (default 22) |

Deploy workflows use `appleboy/scp-action` + `appleboy/ssh-action` with these secrets.

## First-time server setup

```bash
# on the server, as root (or sudo):
sudo bash /opt/yourtj/scripts/init-server.sh \
  https://forum.yourtj.de https://dev.yourtj.de
```

This creates `/opt/yourtj/{.env,docker-compose.yaml,main,dev}` with randomized signing keys,
PG/Meili passwords, starts `postgres` + `meilisearch`, and creates the `yourtj_main`/`yourtj_dev`
databases. The script itself is deployed to the server by the first CI run (or copy `deploy/` manually).

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
- Rollback: `deploy.sh` tags the previous image `ghcr.io/yourtongji/yourtj-hub:prev` and re-points
  the instance on health-check failure; forward-compatible migrations mean an older binary can still start.

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
  upload JSON (maximum 50 MiB) into a staged background task. The task is checksum-addressed and
  idempotent; identical uploads reuse the same task. The worker applies all rows and topic
  invariants (post_seq, first/last post pointers, counts, posters) in one database transaction,
  so validation failures roll back the batch rather than leaving partial data. Staged files use
  mode `0600`, are deleted after success, and are retained for an explicit replay after failure.
  Administrators can inspect `GET /api/admin/data/import/tasks` and replay a failed task with
  `POST /api/admin/data/import/tasks/:taskId/replay`.
- Export files are written to `data/export/` inside the storage dir with mode 0600
  (owner-only) and cleaned up after 7 days (daily cron). Export contains user emails —
  treat downloads as sensitive. Export creation and download are recorded in the
  operation audit log (`opt_record`, issue #324).

## 管理端设置密钥（issue #324）

- SMTP 密码、对象存储 accessKey/secretKey、HTTP 通知端点 secret 均以 securestore
  AES-256-GCM 密文落库（与一系统 Cookie / wiki webhook secret 同一模式），管理端
  GET 仅回显是否已配置，绝不回显明文或密文；保存时密钥字段留空表示保留已存值。
- 升级到包含 v25 数据迁移的版本后，存量明文密钥会在下次启动时自动加密迁移
  （幂等；迁移失败不推进版本，下次启动重试）。

## 一系统排课同步（course-pk-sync，issue #186）

将同济一系统（1.tongji.edu.cn）排课数据分页同步到 PK 域，并重建 `teacher_timeslots`。

> **管理端入口（推荐，issue #248）**：部署实例的排课器学期下拉为空，通常是因为
> `pk_calendar` 尚无数据且未同步。无需登录服务器，在**管理端 → 设置 → 一系统同步**
> 页面即可：
> 1. 配置一系统 Cookie（加密落库，不存明文）；
> 2. 输入一系统数字学期 ID（如 `121`）或已同步过的学期名（如 `2025-2026-1`）点「立即同步」；
> 3. 同步在后台执行（`POST /api/admin/pk/sync-calendar`），页面「同步状态」列表每 3s 轮询
>    `GET /api/admin/pk/sync-status`（`pk_fetch_log` 游标）直至结束，可看到行数/进度/失败原因。
>
> 未配置任何 Cookie 来源（管理端设置/`ONESYSTEM_COOKIE` 环境变量）时入口会拒绝触发。
> 同一学期同步中的并发仍受 fetchlog 1 小时 running 窗口保护（见下）。

CLI 同步（运维 cron 等自动化场景）：

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

应用内定时任务默认开启。若实例只运行持久化 worker、由外部 cron 触发维护命令，
可在 `config.toml` 设置 `[cron].enabled = false`；该开关只停止应用内 scheduler，
不会停用 task queue worker 或手动 CLI。

- 行为保证：同一学期重复执行先清空再全量重写（幂等，不翻倍）；同步中断后重跑从失败批次
  续跑（`pk_fetch_log` 游标），不回滚已成功批次；Cookie 失效时报 HTTP 状态与提示并标记
  fetchlog `failed`，且不删除存量数据；无效 Cookie 不会破坏已同步数据。
- 并发防护：同一学期存在 1 小时内的 `running` fetchlog 时拒绝新同步（避免两个进程互相删数据）；
  进程崩溃后若需立即重跑，可等待窗口过期或手动清掉该学期 `pk_fetch_log`。
- 注意：`app.signingKey` 轮换会使管理端已存的一系统 Cookie 密文失效（与 TOTP 相同），
  需到管理端重新保存。

### 学期起止日期（可选，config 维护）

一系统 manualArrange 数据不含学期起止日期。排课器「当前周次」自动定位与学期日期条
展示依赖 `config.toml [pk.semester_dates]`（键 = 学期标记 `pk_calendar.calendar_id_i18n`，
如 `2025-2026-1`；值 = start/end 纯日期）。`course-pk-sync` 同步该学期时写入
`pk_calendar.start_date/end_date`，P1 `/api/pk/calendars` 原样返回（未配置为 null）：

```toml
[pk.semester_dates."2025-2026-1"]
start = "2025-09-08"
end = "2026-01-18"
```

- 修改日期后需重跑该学期同步（幂等）才会写入数据库。
- 未配置的学期两列为 NULL，排课器周次仍可手动选择（「当前周次」开关禁用）。

## 服务器迁移 runbook（旧机 → 新机）

旧生产服务器（20.205.27.178, 1Panel）迁移到新机（43.108.84.213, Docker Compose + GHCR）的完整步骤。
**原则：signingKey 必须原样复制，绝不重新生成**（否则全部会话 / TOTP / 重置链接失效，fail-closed）。

### 0. 前置

- 新机已装 Docker Engine + Compose + 2G swap（`deploy/scripts/init-server.sh` 会在 CI 首次部署时下发，
  也可手动拷贝 `deploy/` 后在服务器执行）。
- 仓库侧 GHCR 镜像流 PR 已合并；**合并后先更新 GitHub Secrets（`VM_HOST` → 新机 IP、
  `VM_USER` → `root`、`VM_SSH_KEY` → PEM 内容）再触发 dev 部署**。反代统一由 1Panel 承担
  （自管 nginx 容器已移除, 不再有 :80 与 openresty 冲突的问题）。
- **GHCR 包可见性**：首次 build-image 推送后，GitHub Packages → `yourtj-hub` → Settings →
  Change visibility → **Public**（默认 private，服务器匿名 pull 会 401）。
- 备份密钥：`~/Documents/YourTJ_Korean.pem`（新机 SSH PEM）。

### 1. 新机初始化

```bash
ssh -i ~/Documents/YourTJ_Korean.pem root@43.108.84.213
mkdir -p /opt/yourtj && cd /opt/yourtj
# 从仓库拷贝 deploy/ 目录后:
sudo bash deploy/scripts/init-server.sh https://forum.yourtj.de https://dev.yourtj.de
```

这会生成 `/opt/yourtj/{.env,docker-compose.yaml,main/config.toml,dev/config.toml}`，
启动 postgres + meilisearch，创建 `yourtj_main` / `yourtj_dev` 数据库。

### 2. 数据迁移（main + dev）

在旧机（1Panel 部署）上导出，再导入新机。**旧机 / 新机 PostgreSQL 版本一致（16）**。

> **⚠️ 旧机是 1Panel 管理**：下列命令假设 compose 项目在 `/opt/yourtj`（与仓库约定一致）。
> 实际路径以旧机 1Panel 配置为准（1Panel 项目目录可能是 `/opt/1panel/apps/...` 或自定义），
> 执行前先 `ls` 确认。`docker compose` 命令在 1Panel 的 compose 项目目录下执行；
> 1Panel 的 compose 版本若为 v1（无 `exec -T` 的 `-T` 参数），去掉 `-T`。

```bash
# 旧机: 导出主库到宿主机 /tmp(exec -T 流式输出, 重定向在宿主机执行;
#       若在容器内重定向, 文件落在容器 /tmp, 宿主机 scp 找不到)
docker compose exec -T postgres sh -c 'pg_dump -U yourtj -d yourtj_main' > /tmp/yourtj_main.sql
docker compose exec -T postgres sh -c 'pg_dump -U yourtj -d yourtj_dev' > /tmp/yourtj_dev.sql

# 旧机 → 新机: 直传
scp -i ~/Documents/YourTJ_Korean.pem /tmp/yourtj_main.sql /tmp/yourtj_dev.sql \
  root@43.108.84.213:/tmp/

# 新机: 导入(在 postgres 容器内)
docker compose exec -T postgres sh -c 'psql -U yourtj -d yourtj_main' < /tmp/yourtj_main.sql
docker compose exec -T postgres sh -c 'psql -U yourtj -d yourtj_dev' < /tmp/yourtj_dev.sql
```

文件库 `file.db`（SQLite，附件 BLOB）直接拷贝。**注意实际路径是
`storage/database/file.db`**（见 config.toml `[db.file].path`）：

```bash
# 旧机: 用 sqlite3 .backup 做一致性快照(实例仍在运行, 直接 scp 活库会拿到 torn copy;
#       与 backup-db.sh 相同做法; 旧机已装 sqlite3, init-server.sh 也会装)
sqlite3 /opt/yourtj/main/storage/database/file.db ".backup '/tmp/main-file.db'"
sqlite3 /opt/yourtj/dev/storage/database/file.db ".backup '/tmp/dev-file.db'"
# 或直接复用旧机已有的 backup-db.sh(它已做一致性快照):
#   /opt/yourtj/scripts/backup-db.sh main && scp ... root@<旧机>:/opt/yourtj/snapshots/main/file-*.db /tmp/main-file.db

# 旧机 → 新机
scp -i ~/Documents/YourTJ_Korean.pem /tmp/main-file.db /tmp/dev-file.db \
  root@43.108.84.213:/tmp/

# 新机: 安装 + 属主必须是容器内 app uid(1000), 否则附件写入报权限错误
install -m 0664 /tmp/main-file.db /opt/yourtj/main/storage/database/file.db
install -m 0664 /tmp/dev-file.db /opt/yourtj/dev/storage/database/file.db
chown 1000:1000 /opt/yourtj/main/storage/database/file.db /opt/yourtj/dev/storage/database/file.db
```

### 3. 配置迁移（完整拷贝 + reconcile，关键！）

**不要只复制 signingKey**：`config.toml` 还包含 GitHub OAuth `client_id`/`client_secret`、
OIDC 设置（含 signing_key_file PEM）、一系统同步 Cookie 密文等，全部需要原样迁移：

```bash
# 旧机: 完整拷贝两个 config.toml 与 OIDC 密钥文件
scp -i ~/Documents/YourTJ_Korean.pem root@<旧机>:/opt/yourtj/main/config.toml /tmp/main-config.toml
scp -i ~/Documents/YourTJ_Korean.pem root@<旧机>:/opt/yourtj/dev/config.toml /tmp/dev-config.toml
# 若 [oidc] signing_key_file 启用, 一并拷贝对应 PEM

# 新机: 用旧配置覆盖, 然后 reconcile 以下部署相关键:
#   - [db.default] url: host 改为 postgres(compose 服务名), 不要沿用旧机 127.0.0.1
#   - [server] trusted_proxies: 1Panel 本机回源 → ["127.0.0.1", "::1"]
#   - [meilisearch] masterkey: 与 /opt/yourtj/.env 的 MEILI_MASTER_KEY 一致
#   - [server] url: 保持 https://forum.yourtj.de / https://dev.yourtj.de
install -m 0644 /tmp/main-config.toml /opt/yourtj/main/config.toml
install -m 0644 /tmp/dev-config.toml /opt/yourtj/dev/config.toml
```

### 4. 首次部署 + 健康检查

```bash
# 手动拉取并启动(等价于 CI deploy.sh main main-<sha> 5234)
cd /opt/yourtj
IMAGE_KEEP_N=5 bash scripts/deploy.sh main main-<latest-sha> 5234
IMAGE_KEEP_N=3 bash scripts/deploy.sh dev dev-<latest-sha> 5235
# 验证
curl -fsS http://127.0.0.1:5234/health && echo MAIN_OK
curl -fsS http://127.0.0.1:5235/health && echo DEV_OK
curl -fsS -H "Host: forum.yourtj.de" http://127.0.0.1/ | head -5   # 经 1Panel 反代
```

### 5. Cloudflare SSL 模式 + DNS 切换

**先改 SSL/TLS 模式，再切 DNS**（否则切换瞬间 521/525）：

1. **Cloudflare SSL/TLS 模式**：新旧机均由 1Panel 反代终止 TLS（保持与旧机一致的
   模式，如 Full (strict)：Cloudflare → origin 走 HTTPS:443）。论坛容器只监听
   127.0.0.1 回源端口，不直接暴露 80/443。
2. **DNS**：`forum.yourtj.de` / `dev.yourtj.de` 的 A 记录从旧机 IP（20.205.27.178）
   改为新机 IP（43.108.84.213）。
3. 等 TTL 过后从外网验证 `https://forum.yourtj.de` 与 `https://dev.yourtj.de` 可达、
   登录/发帖/附件/搜索 spot-check。
4. **观察期（≥7 天）内旧机保持运行**；确认稳定后退役旧机（`docker compose down`，保留数据快照）。
5. 更新 GitHub Secrets `VM_HOST` → 新机 IP；旧机从 CI 摘除。

### 风险与回滚

- 迁移窗口内的新写入不会进入 dump；**切换前在旧机再跑一次 `backup-db.sh main`** 生成最新快照，
  接受分钟级数据窗口。
- **回滚不是简单的 DNS 切回**：DNS 切换到新机后，新机接受了新的发帖/注册/附件写入。
  若此时切回旧机，这些新写入会丢失。因此回滚前必须**反向同步**：
  1. 停写：临时把 Cloudflare 页面规则或新机 1Panel 反代置为维护模式（或直接切 DNS 前先接受
     "最近写入可能丢失" 的窗口）；
  2. 从新机 `pg_dump yourtj_main/yourtj_dev` 回灌旧机对应库（与 §2 相同方式反向）；
  3. 把新机 `storage/database/file.db` 拷回旧机对应路径并 `chown 1000:1000`；
  4. 再切 DNS 回旧机。
  - 若回滚发生在切换后很短时间内且写入量可忽略，可接受不回灌，但文档不承诺"数据无损"。
- Meilisearch 索引不迁移，首次启动后由 `rebuild-search-index` 重建（ADR-003：索引是可重建投影）。
- 搜索投影任务采用有界重试；Meilisearch 短时不可用时，`topic-search.*`、
  `user-search.*` 或 `category-search.*` 任务可能进入 `failed`，不会自动无限重试。
  Meilisearch 恢复后检查 `task_queue`，并运行 `rebuild-search-index` 做一次全量对账；
  该命令是运维恢复动作，不依赖旧任务仍处于 pending。

## Runbooks to write

- Built-in OIDC Provider production config ([oidc] in config.toml: enabled, issuer, signing key, clients)
- Meilisearch index rebuild, backup
- Logging & monitoring (config [log] slow SQL, rolling logs; health probes)
