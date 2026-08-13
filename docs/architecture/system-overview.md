# System Overview & Domain Boundaries

> Doc type: architecture
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-08

## System shape

```
                        ┌──────────────────────┐
                        │  Built-in OIDC       │  forum users (numeric id)
                        │  Provider (/api/oauth)│
                        └──────────┬───────────┘
              OIDC/PKCE            │            OIDC browser flow
       ┌───────────────────────────┼───────────────────────────┐
       │                           │                           │
┌──────▼──────┐           ┌────────▼────────┐          ┌───────▼───────┐
│ apps/mobile │           │ apps/gooseforum │          │ services/     │
│  Flutter    │           │  forum (Go+Vue, │          │ credit (p2)   │
└──────┬──────┘           │  single binary) │          └───────────────┘
       │                  └────────┬────────┘
       │                           │
       └──────────┬────────────────┘
                  │ JSON API (JWT Bearer)
           ┌──────▼──────┐     ┌──────────────┐
           │ apps/gooseforum │──▶│ services/    │  Meilisearch index sync
           │  Go backend     │   │ search       │  (optional, event-driven)
           └──────┬──────┘   └──────────────┘
                  │
           ┌──────▼────────────┐
           │ SQLite/PG        │ (PG default in deployments, issue #11; local dev/tests SQLite; file db stays SQLite)
           └───────────────────┘
```

## Deployment shape

- **Single binary**: forum frontend (Vue 3 output static/dist + GoHTML templates) is fully go:embed'd
  into the Go binary; vite :3010 hits the backend in dev, one file in production. No nginx/CDN split.
- Dependency services (Meilisearch/PostgreSQL/Redis) are orchestrated with docker-compose;
  `services/` holds deployment configs only, not third-party source. Deployments default to
  PostgreSQL for the main database; local development and tests default to SQLite.

## Domain boundaries (apps/gooseforum upstream layers)

| Layer | Responsibility |
|---|---|
| `app/console` | cobra CLI (serve / mock / rebuild-search-index / migrate-files ...) |
| `app/bundles` | Utilities (connect/eventbus/jwtopt/i18n/captcha/logging/cache ...) |
| `app/models` | GORM models + migrations (app/migration) |
| `app/service` | Business logic (users/topics/mail/oauth/theme ...) |
| `app/http/controllers/api` | JSON API (auth/topic/user/admin/chat/notification/file ...) |
| `app/http/controllers/forum` | Page rendering (GoHTML three-mode: payload + render + SEO) |
| `app/http/middleware` | JWT auth, access log, maintenance mode, rate limiting (per-action, IP+user, 429 + Retry-After) ... |
| `resource/` | Vue 3 frontend (site/admin dual entry) + templates (gohtml) + static (badges/pic) |

**Boundary rules**
- Business logic in `service`; data access in `models`/repository layer; HTTP in `http/controllers`.
- Cross-domain access (e.g. forum→notifications) goes through the owner's public service API; no
  foreign SQL.
- Frontend output only via `resource/static/dist` (go:embed). For OpenAPI-covered operations, consume
  generated types rather than duplicating backend DTOs by hand; maintain other endpoint contracts manually
  until they are covered.
- Upstream sync: `git merge` upstream main; resolve conflicts with "our changes win" and record it.

## Key flows

### Auth

- Web: password login (optional forum-side TOTP 2FA), GitHub OAuth (goth, config [github]), and the
  built-in OIDC Provider (authorization code + PKCE S256, numeric `sub` = users.id) for first-party
  clients. Sessions are `jti` + `user_sessions` backed and revocable (see identity-and-access.md).
- Mobile (`Partial`): appauth+PKCE → id_token → `POST /api/auth/oidc/exchange` → forum JWT. The
  Flutter shell and feature pages consume the repository-owned `Gf*` UI API; `ui_kit` maps those
  tokens and components to the pinned TDesign v1 alpha implementation so application pages do not
  depend on pre-release TDesign APIs directly.

### Search (Partial)

- Meilisearch optionally enabled (config [meilisearch]). Aggregate search (one box covering
  topics/users/categories with scope tabs; pinyin/initials matching for users and categories) landed
  in issue #22. Index sync is event-driven (topic/user/category events). When Meilisearch is
  unavailable the search page shows a full unavailable state; per-index failures degrade partially
  via `failedScopes`. The index is a rebuildable projection (`rebuild-search-index` CLI).

### Points (phase 2)

- credit is an OIDC client + standalone ledger; the forum acts as a merchant calling the distribution
  API (see credit-and-escrow.md).

## Consistency principles

- The chosen DB is the business fact source; search, cache, counters, hot lists, and feeds are
  rebuildable projections.
- Critical side effects (notifications, index sync, points distribution) are idempotent, retryable,
  observable.
- For OpenAPI-covered operations, contract changes ship in the same PR: Go behavior/struct →
  `openapi.yaml` → generated TypeScript output → fixture tests. Dart generation remains Planned.
