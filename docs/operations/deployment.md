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
  go:embed).
- Runtime deps: SQLite (default, zero external deps) or MySQL; optional Meilisearch; Casdoor planned.
- No nginx/CDN split in production; if a reverse proxy is needed, campus infrastructure provides it
  (the single binary stays).
- **Two instances on one VM** (Ubuntu 24.04, ssh alias `yourtj`):
  - `main` — production, `/srv/yourtj/main`, systemd `yourtj-main.service`
  - `dev` — test line, `/srv/yourtj/dev`, systemd `yourtj-dev.service`
  - DB sync is one-way: dev gets a consistent snapshot of main on each deploy (below).

## Branch model & CI/CD

- `dev` is the main development line; merges to `dev` trigger `.github/workflows/deploy-dev.yml`:
  1. Build single binary (frontend + go build) on GitHub Actions.
  2. SSH to VM: `sync-db-from-main.sh` (SQLite `.backup` API snapshot of main db → dev db).
  3. Install binary, restart `yourtj-dev`, health check on port 5235.
- `main` is the production site; merges to `main` trigger `.github/workflows/deploy-main.yml`:
  1. Build single binary on GitHub Actions.
  2. SSH to VM: `backup-db.sh main` (pre-deploy consistent snapshot, keep 7).
  3. Install binary, restart `yourtj-main`, health check on configured port (default 5234);
     auto-rollback to previous binary if health check fails.
- Why dev syncs main's db: migrations (`app/migration` AutoMigrate + versioned data migrations) run at
  startup, so each dev deploy rehearses the exact migration the next main deploy will run.

## Server layout

```
/srv/yourtj/
  scripts/            # snapshot-db.sh, sync-db-from-main.sh, backup-db.sh, deploy.sh
  main/
    bin/yourtj-hub
    config.toml       # production config (signingKey, port, db path) — never in git
    storage/          # sqlite.db + file.db + logs
  dev/
    bin/yourtj-hub
    config.toml       # dev config (port 5235, separate meilisearch/casdoor if needed)
    storage/
  snapshots/
    main/sqlite-*.db  # pre-deploy backups (keep 7)
    dev-prev/         # previous dev db before sync (troubleshooting)
```

Systemd units: `deploy/systemd/yourtj-main.service`, `deploy/systemd/yourtj-dev.service`
(install to `/etc/systemd/system/`, `systemctl daemon-reload && enable --now`).

Config is pre-provisioned on the server and never passes through CI (no secrets in workflow).

## Build

```bash
make build     # cd apps/gooseforum/resource && pnpm build → cd apps/gooseforum && go build -o ../../bin/yourtj-hub .
```

## Config & run

```bash
# Config: apps/gooseforum/config.toml (see docs/development/local-development.md)
cd apps/gooseforum && cp <reference config> config.toml

# Run
./bin/yourtj-hub serve        # listens on config.toml [server] port (default 5234)
```

- `env = "production"` listens on all interfaces; `local` binds 127.0.0.1 only.
- Other CLI: `./bin/yourtj-hub --help` (mock data, rebuild-search-index, etc.).
- Health probe: `GET /health` returns 200 when service + main db ping succeed, else 503.

## DB migration execution and rollback

- Migrations run at startup (`[db] migration = "on"`); append-only upstream style.
- Rollback: keep the previous binary (`deploy.sh` saves `yourtj-hub.prev`); forward-compatible
  migrations mean an older binary can still start against the migrated db.
- Pre-deploy snapshot in `snapshots/main/` is the data-level restore point.

## Runbooks to write

- Casdoor production config (domain, certs, client registration)
- Meilisearch index rebuild, backup
- Logging & monitoring (config [log] slow SQL, rolling logs; health probes)
