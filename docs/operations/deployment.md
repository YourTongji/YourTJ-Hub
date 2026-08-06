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
- Runtime deps: SQLite (default, zero external deps) or MySQL; optional Meilisearch; Casdoor planned.
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
  .env                    # MAIN_PORT/DEV_PORT/MAIN_TAG/DEV_TAG (created by init-server.sh)
  docker-compose.yaml     # main + dev services (created by init-server.sh)
  config.toml.example     # template with REPLACE_* placeholders
  build/
    Dockerfile            # alpine + binary
  scripts/                # snapshot-db.sh, sync-db-from-main.sh, backup-db.sh, deploy.sh
  main/
    config.toml           # production config (signingKey, db path) — never in git
    storage/              # sqlite.db + file.db + logs (uid 1000)
  dev/
    config.toml           # dev config
    storage/
  snapshots/
    main/sqlite-*.db      # pre-deploy backups (keep 7)
    dev-prev/             # previous dev db before sync (troubleshooting)
```

## Branch model & CI/CD

- `dev` is the main development line; merges to `dev` trigger `.github/workflows/deploy-dev.yml`:
  1. Build single binary (frontend + go build) on GitHub Actions.
  2. Upload binary via scp; SSH: `sync-db-from-main.sh` (SQLite `.backup` snapshot of main db → dev db).
  3. SSH: `deploy.sh dev <binary> dev-<sha> 5235` → build image, compose up, health check, rollback.
- `main` is the production site; merges to `main` trigger `.github/workflows/deploy-main.yml`:
  1. Build single binary on GitHub Actions.
  2. SSH: `backup-db.sh main` (pre-deploy consistent snapshot, keep 7).
  3. SSH: `deploy.sh main <binary> main-<sha> 5234` → build image, compose up, health check,
     auto-rollback to previous image tag on failure.
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
- Pre-deploy snapshot in `snapshots/main/` is the data-level restore point.

## Runbooks to write

- Casdoor production config (domain, certs, client registration)
- Meilisearch index rebuild, backup
- Logging & monitoring (config [log] slow SQL, rolling logs; health probes)
