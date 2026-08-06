# Deployment & Release

> Doc type: operations
>
> Status: Active (deployment shape decided; concrete runbooks `Planned`)
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

## Release process (Planned)

- CI builds artifacts + version tags (semver);
- Deploy to a test server (follow YourTJ-Platform's PR preview + staging pattern; write the runbook
  when it lands);
- DB migrations: upstream `app/migration` runs at startup (on/off via config [db] migration);
- Rollback: keep the previous binary + forward-compatible migrations (append-only upstream migrations).

## Runbooks to write

- DB migration execution and rollback (after the PostgreSQL decision)
- Casdoor production config (domain, certs, client registration)
- Meilisearch index rebuild, backup
- Logging & monitoring (config [log] slow SQL, rolling logs; health probes)
