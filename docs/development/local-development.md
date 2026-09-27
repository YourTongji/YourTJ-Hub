# Local Environment

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-27

## Dependencies

此处统一维护开发工具版本；各 README 引用本节，避免重复维护。

| 工具 | 当前要求与事实源 |
|---|---|
| Go | `go.mod` 要求 1.26.6，指定工具链 1.26.8；默认 `GOTOOLCHAIN=auto` 可自动获取工具链。见 [go.mod](../../apps/gooseforum/go.mod)。 |
| Node.js / pnpm | Node.js 24、pnpm 11，与 [Web CI](../../.github/workflows/ci-frontend.yml) 一致。前端命令在 `apps/gooseforum/resource/` 的独立 workspace 内执行。 |
| Flutter / Dart | 使用 [Mobile CI](../../.github/workflows/ci-mobile.yml) 验证的 Flutter 3.44.9；[移动工作区](../../apps/mobile/pubspec.yaml)要求 Dart ≥3.12.2、<4.0.0，并声明 Melos 依赖。 |
| Docker Compose | 仅在运行本地 PostgreSQL／Meilisearch 等依赖时需要。 |

## Startup

Run the following from the repository root. Install the frontend dependencies once:

```bash
pnpm --dir apps/gooseforum/resource install --frozen-lockfile

# Generate config.toml if missing, without starting the server; preserves an existing file
(cd apps/gooseforum && go run . --help)
```

Before starting the backend, edit the existing `[app]` section in
`apps/gooseforum/config.toml` to set `env = "local"`. The embedded template defaults to
`"production"`, which serves embedded assets and requires secure session cookies. Local mode
enables Vite resources and HTTP sessions on localhost. Keep the generated signing key and the
rest of the configuration; do not replace the file with this snippet:

```toml
[app]
env = "local"
```

SQLite works without Docker. If you need PostgreSQL or Meilisearch, start the optional dependencies
with `make dev` and configure them as described below. Then use separate terminals:

```bash
# Terminal 1: forum backend (default port 5234)
make server        # = cd apps/gooseforum && go run . serve

# Terminal 2: Vite resources (:3010; browser entry remains http://localhost:5234)
make web           # = cd apps/gooseforum/resource && pnpm dev
```

For a production build (resource → static/dist → Go single binary):

```bash
make build
```

For mobile development (requires Flutter SDK):

```bash
cd apps/mobile && dart pub get
dart run melos bootstrap     # 首次或依赖变更后
dart run melos run analyze   # 全包静态检查
dart run melos run test      # 全包测试
```

## Mobile workspace

- `apps/mobile` is a melos workspace with four packages: `core` (contracts/API client/markdown
  conversion), `auth` (login/TOTP/OIDC/token storage), `ui_kit` (design tokens + Gf* components),
  `forum_app` (routes/pages/state). Scripts (`analyze`/`test`/`gen`) are declared in
  `apps/mobile/pubspec.yaml` under the `melos:` key.
- Design tokens: `ui_kit/lib/src/theme/tokens.json` is the single derived source of the web design
  language (source of truth: `apps/gooseforum/resource/src/styles/tokens.css`). **A PR that changes
  `tokens.css` must update `tokens.json` in the same commit** (contract-style discipline).
- Mobile contract mirrors live in `core/lib/src/gen/*.dart` (see
  [contracts-and-data](../architecture/contracts-and-data.md)).

## Service addresses

| Service | Address | Note |
|---|---|---|
| Forum backend | http://localhost:5234 | config.toml `[server] port` |
| Vite resources | http://localhost:3010 | 设置 `[app] env = "local"` 后，后端将 `/assets` 代理到此处；浏览器访问 5234 |
| meilisearch | http://localhost:7700 | master key: `yourtj-dev-master-key` |
| postgres | localhost:5432 | yourtj/yourtj, db yourtj |

## Mobile → backend

- iOS simulator: `http://localhost:5234` directly
- Android emulator: ordinary API development may use `http://10.0.2.2:5234`; OIDC must instead use
  `http://localhost:5234` through `adb reverse` so the issuer remains an allowed loopback URL
- Physical device: inject `YOURTJ_API_BASE_URL` and `YOURTJ_OIDC_ISSUER` for a reachable HTTPS
  instance; non-loopback HTTP is not a valid OIDC issuer. See the
  [mobile run commands](../../apps/mobile/packages/forum_app/README.md).

## Configuration (config.toml)

GooseForum is configured by `apps/gooseforum/config.toml` (not environment variables):

| Section | Note |
|---|---|
| `[app]` | env (local binds 127.0.0.1; any non-`local` value forces session-cookie `Secure` even when `server.url` is `http://…` — issue #113), debug, maintenance, signingKey, cdn_url |
| `[server]` | url, port (default 5234), accessLog, gzip |
| `[db]` / `[db.default]` / `[db.file]` | SQLite default for local dev/tests; deployments default to PostgreSQL (`[db.default] connection = "postgres"`, issue #11); MySQL is not supported; file db stays SQLite; migration, backup, pool |
| `[meilisearch]` | url, masterkey (optional search) |
| `[log]` | log type/rolling/slow SQL; `level` (debug/info/warn/error), `format` (json/console), `errorPath` (WARN/ERROR separate file), `logIp` (access-log IP, default off) — all require restart |
| `[github]` | GitHub OAuth client |
| `[google]` | Google OAuth client；需要站点设置 `siteUrl` 为与 Google Cloud 完全匹配的绝对回调基址 |

The built-in OIDC Provider is configured from the `[oidc]` section in `config.toml`
(`enabled`, `issuer`, `signing_key_file`, `[[oidc.clients]]`, see `deploy/config.toml.example`); the
values are read at startup via preferences and there is no admin-panel UI to change them (set them in
the file and restart). The endpoints are mounted under `/api/oauth` only when `oidc.enabled = true`.
OIDC clients require the issuer to exactly equal the advertised discovery value, and the provider
only accepts loopback `http` issuers. When `oidc.issuer` is omitted, a loopback `server.url` without
an explicit port is combined with `server.port`, so the default local issuer is
`http://localhost:5234/api/oauth`. The Android emulator reaches that exact address through
`adb reverse tcp:5234 tcp:5234` (see `apps/mobile/scripts/oidc_e2e.sh`); `10.0.2.2` is not a valid
local issuer. Existing local `config.toml` files keep working without regeneration; set
`oidc.issuer` explicitly only when the advertised issuer must differ from the derived site URL.

To run the forum against the local PostgreSQL instead of SQLite, set in `config.toml`:

```toml
[db.default]
connection = "postgres"
url = "host=127.0.0.1 user=yourtj password=yourtj dbname=yourtj port=5432 sslmode=disable"
```

The binary AutoMigrates all main-db models and runs the versioned data migrations on first boot.
`TEST_PG_DSN` can be set to run the gated PostgreSQL integration tests
(`go test ./app/bundles/connect/sqlconnect/...`); `YOURTJ_TEST_PG_URL` gates the migration schema
tests (`go test ./app/migration/ -run 'TestSchema' -v`, see [testing.md](testing.md)).

> config.toml contains signingKey — sensitive; it is gitignored, never commit it.

## Known issues

- Go module fetch: the official proxy can time out; use `GOPROXY=https://goproxy.cn,direct`.
- pnpm `ERR_PNPM_IGNORED_BUILDS`: esbuild must be allowed in
  `apps/gooseforum/resource/pnpm-workspace.yaml`; update `allowBuilds` when adding native deps
  (upstream already handles esbuild, so usually no change needed).
