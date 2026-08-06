# Local Environment

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-06

## Dependencies

- Go 1.26+
- Node 24 + pnpm 11 (the forum frontend workspace lives in `apps/gooseforum/resource/` with its own
  pnpm-workspace.yaml; **note**: the home-directory `/Users/yzxoi/pnpm-workspace.yaml` can interfere
  with pnpm's upward lookup — run pnpm from inside `resource/`)
- Docker + Compose (local dependency services)
- Flutter SDK (mobile planned, not installed yet)

## Startup

```bash
# 1. Start local dependencies (postgres + meilisearch + mariadb + casdoor)
make dev

# 2. Forum backend (default port 5234)
#    First-time setup: place a config.toml in apps/gooseforum (gitignored), based on upstream config
make server        # = cd apps/gooseforum && go run . serve

# 3. Frontend dev server (:3010, vite; run pnpm install first)
make web           # = cd apps/gooseforum/resource && pnpm dev

# 4. Production build: resource → static/dist → go build single binary
make build
```

## Service addresses

| Service | Address | Note |
|---|---|---|
| Forum backend | http://localhost:5234 | config.toml `[server] port` |
| Frontend dev | http://localhost:3010 | vite, hits backend directly |
| casdoor | http://localhost:8001 | unified auth (admin/123, dev) |
| meilisearch | http://localhost:7700 | master key: `yourtj-dev-master-key` |
| postgres | localhost:5432 | yourtj/yourtj, db yourtj (reserved) |
| mariadb | localhost:13306 | casdoor-only |

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
| `[jwtopt]` | validTime (seconds, default 604800 = 7 days) |
| `[db]` / `[db.default]` / `[db.file]` | SQLite by default, MySQL optional; migration, backup, pool |
| `[meilisearch]` | url, masterkey (optional search) |
| `[log]` | log type/rolling/slow SQL |
| `[github]` | GitHub OAuth client (currently the only third-party login) |

> config.toml contains signingKey — sensitive; it is gitignored, never commit it.

## Known issues

- Go module fetch: the official proxy can time out; use `GOPROXY=https://goproxy.cn,direct`.
- pnpm `ERR_PNPM_IGNORED_BUILDS`: esbuild must be allowed in
  `apps/gooseforum/resource/pnpm-workspace.yaml`; update `allowBuilds` when adding native deps
  (upstream already handles esbuild, so usually no change needed).
