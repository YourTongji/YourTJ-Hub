# AGENTS.md — yourtj-hub

Operating guide for anyone (human or AI agent) changing this repository.

Before changing anything, read this file, [`docs/README.md`](docs/README.md),
[`docs/development/README.md`](docs/development/README.md), and the product/architecture/operations
documents directly affected by the request. Use the repository `$yourtj-development` skill for
implementation, testing, review, CI, or PR work.

---

## 1. What this is

yourtj-hub is the monorepo for the Tongji university campus forum platform (brand: yourtj, distinct from
the archived YourTJ-Platform). The forum is the core product — a **direct modification of the upstream
GooseForum, keeping the single-binary deployment**. Unified auth (built-in OIDC Provider), search (Meilisearch), and
points (credit, phase 2) are shared infrastructure subdomains. Database, search, and structure may all
be changed, but the "Go + Vue in one binary, frontend go:embed into the binary" deployment shape is kept.

- Forum: **Go 1.26 + Gin + Vue 3 + Tailwind**, at `apps/gooseforum` (fork of upstream; module path
  `github.com/YourTongji/YourTJ-Hub/apps/gooseforum`, diverged from upstream's `github.com/leancodebox/GooseForum`).
- Backend layers (upstream structure): `app/bundles` (utilities) → `app/models` (GORM models) →
  `app/service` (business) → `app/http/controllers/{api,forum}` (JSON API + GoHTML three-mode rendering).
- Frontend: `apps/gooseforum/resource` (Vue 3 + Vite, site/admin dual entry), built output
  `resource/static/dist` go:embed; GoHTML templates in `resource/templates` keep server-side rendering (three-mode).
- Database: SQLite default, MySQL optional (`config.toml [db]`); **PostgreSQL supported for the main
  database since issue #11** (file db stays SQLite).
- Search: **Meilisearch** (`config.toml [meilisearch]`, optional); aggregate search (topics/users/
  categories, pinyin/initials) landed (issue #22); event-driven index sync, rebuildable projection.
- Mobile: **Flutter** (`apps/mobile`, melos workspace, Riverpod, **Partial**).
- Auth: GitHub OAuth (goth) + **built-in OIDC Provider** (`/api/oauth`, authorization code + PKCE S256,
  RS256 id_token, opaque access tokens, numeric `sub` = users.id); TOTP 2FA and session management
  (`jti` + `user_sessions`) in place. Casdoor is not enabled.
- Contract: **Partial** — `packages/api-contract/openapi.yaml` is the controlled contract center for
  password login, login public-key retrieval, TOTP login verification and account management, logout,
  mobile OIDC exchange, session management (list/revoke/revoke-all), and topic writing, with lint/bundle,
  generated TypeScript types, fixtures, and route-level HTTP tests;
  paths are split per domain under `paths/`; broader route coverage still needs manual or
  annotation-based work.
- Points: credit (linux-do) phase 2, merchant model, not implemented this phase.

## 2. Repository layout & boundary rules

```
apps/
  gooseforum/  The forum itself (upstream fork; module path github.com/YourTongji/YourTJ-Hub/apps/gooseforum)
    main.go            Entry point (cobra: serve / mock / rebuild-search-index subcommands)
    config.toml       Runtime config (gitignored; bring your own locally)
    app/              Go backend (bundles/console/datastruct/http/migration/models/service)
    resource/         Vue 3 frontend + gohtml templates + @gooseforum/client package
    docs/             Upstream-owned docs (reference only)
  mobile/      Flutter melos workspace (core/auth/ui_kit/forum_app)
packages/
  api-contract/  openapi.yaml + gen scripts + fixtures + contract tests (Partial)
services/
  casdoor/   Archived Casdoor deployment config (not enabled; built-in OIDC Provider replaces it)
  search/    Meilisearch deployment config
  credit/    Points (phase 2 placeholder)
deploy/      Per-environment compose + env.example
docs/        Docs center (product/architecture/development/operations)
```

**Boundary rules**
- `apps/gooseforum` is the only place the forum is implemented; business logic in `service`, data access
  in `models`/repository layer, HTTP in `http/controllers`.
- Cross-domain access goes through the owner's public API; no foreign SQL against other domains' tables.
- Frontend output only via `resource/static/dist` (go:embed); vite :3010 hits the backend in dev,
  single binary in production.
- `services/` holds deployment configs only, not third-party source (Meilisearch/credit are
  off-the-shelf components; Casdoor is archived and not enabled).
- Upstream sync: `git merge` upstream main; resolve conflicts with "our changes win" and record it. After
  merging, rewrite upstream's `github.com/leancodebox/GooseForum` import prefix to
  `github.com/YourTongji/YourTJ-Hub/apps/gooseforum` (upstream files keep the old prefix), then run `go mod tidy`.

## 3. Hard constraints

- Deployment shape is a **single binary** (go:embed webdist/static-dist); no nginx/CDN split.
- User IDs must be **numeric** (uint64) — credit's `GetID()` only accepts numeric sub; the built-in
  OIDC Provider always issues `sub` = users.id (uint64 decimal string).
- The forum `users` table is the identity source; the forum JWT is a session credential, not identity
  truth, and is never issued to external OIDC clients.
- For OpenAPI-covered operations, contract changes ship in the same PR: backend behavior/structs →
  `openapi.yaml` → generated TypeScript output → fixture contract tests. Until the pipeline fully
  covers an operation, contract changes must also update the mobile Dart mirrors in
  `apps/mobile/packages/core/lib/src/gen/` (same PR) and web TS types
  (`resource/packages/client/src/contracts/`) in the same commit. Dart generation remains Planned.
- Design-token changes ship in the same PR: changing `resource/src/styles/tokens.css` requires
  updating `apps/mobile/packages/ui_kit/lib/src/theme/tokens.json` in the same commit.
- Docs use the four implementation status words (`Current`/`Partial`/`Planned`/`Decision needed`),
  see docs/README.md.
- Docs describe only the currently supported model — no timeline or milestones
  (see docs/development/documentation.md).
- Research files goto research/, and should not be included in git.

## 4. Verification

- Backend: `cd apps/gooseforum && go vet ./... && go test ./...` (use `GOPROXY=https://goproxy.cn,direct`
  if module fetch times out). **Any model/migration change must also pass the PostgreSQL migration
  tests**: `YOURTJ_TEST_PG_URL="host=127.0.0.1 port=5432 user=postgres password=postgres dbname=postgres sslmode=disable" go test ./app/migration/ -run 'TestSchema' -v`
  (both `TestSchemaMigratesOnPostgreSQL` and `TestSchemaUpgradeCreatesNewTablesOnPostgreSQL` in
  `app/migration/migration_pg_test.go`; spin up `postgres:16-alpine` locally — CI runs the same
  command in `ci-backend-pg`). MySQL-only type tags (`bigint unsigned` / `datetime` / `tinyint`)
  break PG and are forbidden in models.
- Web: `cd apps/gooseforum/resource && pnpm typecheck && pnpm test && pnpm build` (output into resource/static/dist)
- Full build: `make build` (resource → go build single binary `bin/yourtj-hub`)
- Smoke: run `./bin/yourtj-hub serve` then curl the homepage/API (port from config.toml, default 5234)
- Report the commands actually run and their results; a local subset is not CI passing.

## 5. Git & PR discipline

- `dev` is the main development line: create `feat/<topic>` / `fix/<topic>` / `docs/<topic>` from
  `origin/dev`, open PRs against `dev`; CI builds and auto-deploys `dev` to the test instance.
- `main` is the production site: merges to `main` go through PR + CI and auto-deploy to the prod
  instance. Never develop directly on `main` or `dev`.
- The dev instance syncs a consistent snapshot of the main database on each deploy (see
  `docs/operations/deployment.md`), so DB migrations are rehearsed on dev before reaching main.
- Stage only files this task owns; leave unrelated dirty/untracked files alone.
- Commit/push/open a PR only when the user explicitly asks.
- Never push to protected branches; releases go through PR + CI.
- Commit messages use concise conventional types (`feat:`/`fix:`/`docs:`/`refactor:`/`chore:`).

## 6. Reference

- [Docs center](docs/README.md) (fact-source table + status words)
- [Development entry](docs/development/README.md)
- Architecture decision records live in the project note (yourtj-hub ADR note), not in git
- Upstream: GooseForum (apps/gooseforum, the fork itself); YourTJ-Platform (local, same-brand archived repo)
