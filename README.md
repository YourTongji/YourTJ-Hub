# YourTJ-Hub

Tongji university campus forum platform monorepo (brand: yourtj). Built directly on [GooseForum](https://github.com/leancodebox/GooseForum) (MIT) with a single-binary deployment; unified auth (Casdoor), search (Meilisearch), and points (credit, phase 2) live as shared infrastructure subdomains.

## Documentation

**[Docs center → docs/README.md](docs/README.md)** (fact-source table + status words)

- Product: [Vision & principles](docs/product/vision-and-principles.md) · [Current state & gaps](docs/product/current-state.md) · [Identity & access](docs/product/identity-and-access.md) · [Points & settlement](docs/product/credit-and-escrow.md)
- Architecture: [System overview & domain boundaries](docs/architecture/system-overview.md) · [Contracts & data](docs/architecture/contracts-and-data.md)
- Development: [Entry point](docs/development/README.md) · [Local environment](docs/development/local-development.md) · [Testing](docs/development/testing.md) · [Pull requests](docs/development/pull-requests.md) · [Documentation governance](docs/development/documentation.md)
- Operations: [Deployment & release](docs/operations/deployment.md)

Read [AGENTS.md](AGENTS.md) and the documents relevant to your change before developing. The repository-level `$yourtj-development` skill lives in `.agents/skills/yourtj-development`.

## Layout

```
yourtj-hub/
├── apps/                  # Deployable applications
│   ├── gooseforum/        # Forum (Go + Vue in one binary; fork of upstream, keeps upstream module name)
│   │   ├── main.go        # Entry point (cobra CLI: serve / mock / rebuild-index ...)
│   │   ├── app/           # Go backend (bundles/console/http/models/service/migration)
│   │   └── resource/      # Vue 3 frontend + gohtml templates (vite output go:embed)
│   └── mobile/            # Flutter (melos workspace: core/auth/ui_kit/forum_app, planned)
├── packages/
│   └── api-contract/      # openapi.yaml contract center (planned)
├── services/              # Base service deployment configs (casdoor / search / credit)
├── deploy/                # Per-environment deployment configs
└── docs/                  # Docs center (product/architecture/development/operations)
```

## Quick start

```bash
# 1. Start local dependencies (postgres + meilisearch + mariadb + casdoor)
make dev

# 2. Forum backend (default port 5234; place a config.toml in apps/gooseforum first)
make server

# 3. Frontend dev server (:3010, vite; run pnpm install in apps/gooseforum/resource first)
make web

# 4. Production build: resource → static/dist → go build single binary
make build
```

See [docs/development/local-development.md](docs/development/local-development.md).

## Reference

We build on [GooseForum](https://github.com/leancodebox/GooseForum) (MIT) — thanks to its author and contributors.

## Decision summary

| Topic | Decision | Record |
|---|---|---|
| Code organization | apps/gooseforum single binary (Go+Vue in one, upstream fork), no frontend/backend split | Decision records live in note |
| Auth | Casdoor unified auth integrated (OIDC + PKCE, numeric user ID enforced server-side, issue #8) | docs/product/identity-and-access.md |
| Mobile state management | Riverpod | docs/architecture/system-overview.md |
| Database / Search | PostgreSQL main-db support landed (issue #11, SQLite default retained); Meilisearch aggregate search with event-driven sync (issue #22) | docs/product/current-state.md |

## Current state

Implementation status and gaps: [docs/product/current-state.md](docs/product/current-state.md) (marked with `Current`/`Partial`/`Planned`/`Decision needed`, no timeline).
