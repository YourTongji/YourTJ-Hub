# Local Environment

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-09

## Dependencies

- Go 1.26+
- Node 24 + pnpm 11 (the forum frontend workspace lives in `apps/gooseforum/resource/` with its own
  pnpm-workspace.yaml; **note**: the home-directory `/Users/yzxoi/pnpm-workspace.yaml` can interfere
  with pnpm's upward lookup — run pnpm from inside `resource/`)
- Docker + Compose (local dependency services)
- Flutter SDK (mobile; workspace-local clone under `.flutter-sdk/` when external paths are blocked,
  otherwise a normal install ≥3.27) + melos (`dart pub global activate melos`)

## Startup

```bash
# 1. Start local dependencies (postgres + meilisearch)
make dev

# 2. Forum backend (default port 5234)
#    First start creates apps/gooseforum/config.toml from the embedded template (gitignored)
make server        # = cd apps/gooseforum && go run . serve

# 3. Frontend dev server (:3010, vite; run pnpm install first)
make web           # = cd apps/gooseforum/resource && pnpm dev

# 4. Production build: resource → static/dist → go build single binary
make build
```
# 5. Mobile app (Flutter, apps/mobile melos workspace; requires Flutter SDK + melos)
cd apps/mobile && melos bootstrap   # 首次或依赖变更后
melos run analyze                    # 全包静态检查
melos run test                       # 全包测试
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
| Frontend dev | http://localhost:3010 | vite, hits backend directly |
| meilisearch | http://localhost:7700 | master key: `yourtj-dev-master-key` |
| postgres | localhost:5432 | yourtj/yourtj, db yourtj (reserved) |

## Mobile → backend

- iOS simulator: `http://localhost:5234` directly
- Android emulator: `http://10.0.2.2:5234`
- Physical device: LAN IP (inject baseUrl via dart-define, when mobile lands)

## Configuration (config.toml)

GooseForum is configured by `apps/gooseforum/config.toml` (not environment variables):

| Section | Note |
|---|---|
| `[app]` | env (local binds 127.0.0.1), debug, maintenance, signingKey, cdn_url |
| `[server]` | url, port (default 5234), accessLog, gzip |
| `[db]` / `[db.default]` / `[db.file]` | SQLite default; main db (`[db.default]`) also supports MySQL and PostgreSQL (issue #11); file db stays SQLite; migration, backup, pool |
| `[meilisearch]` | url, masterkey (optional search) |
| `[log]` | log type/rolling/slow SQL; `level` (debug/info/warn/error), `format` (json/console), `errorPath` (WARN/ERROR separate file), `logIp` (access-log IP, default off) — all require restart |
| `[github]` | GitHub OAuth client |

The built-in OIDC Provider is configured from the `[oidc]` section in `config.toml`
(`enabled`, `issuer`, `signing_key_file`, `[[oidc.clients]]`, see `deploy/config.toml.example`); the
values are read at startup via preferences and there is no admin-panel UI to change them (set them in
the file and restart). The endpoints are mounted under `/api/oauth` only when `oidc.enabled = true`.

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
