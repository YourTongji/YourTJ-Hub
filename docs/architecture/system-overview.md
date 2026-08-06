# System Overview & Domain Boundaries

> Doc type: architecture
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-06

## System shape

```
                        ┌─────────────────────┐
                        │     Casdoor (OIDC)   │  Unified auth (numeric user ID, planned)
                        └──────────┬──────────┘
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
           │  Go backend     │   │ search       │  (optional, to be improved)
           └──────┬──────┘   └──────────────┘
                  │
           ┌──────▼──────┐
           │ SQLite/MySQL │ (PostgreSQL migration pending)
           └─────────────┘
```

## Deployment shape

- **Single binary**: forum frontend (Vue 3 output static/dist + GoHTML templates) is fully go:embed'd
  into the Go binary; vite :3010 hits the backend in dev, one file in production. No nginx/CDN split.
- Dependency services (Casdoor/Meilisearch/PostgreSQL/Redis) are orchestrated with docker-compose;
  `services/` holds deployment configs only, not third-party source.

## Domain boundaries (apps/gooseforum upstream layers)

| Layer | Responsibility |
|---|---|
| `app/console` | cobra CLI (serve / mock / rebuild-search-index ...) |
| `app/bundles` | Utilities (connect/eventbus/jwtopt/i18n/captcha/logging/cache ...) |
| `app/models` | GORM models + migrations (app/migration) |
| `app/service` | Business logic (users/topics/mail/oauth/theme ...) |
| `app/http/controllers/api` | JSON API (auth/topic/user/admin/chat/notification/file ...) |
| `app/http/controllers/forum` | Page rendering (GoHTML three-mode: payload + render + SEO) |
| `app/http/middleware` | JWT auth, access log, maintenance mode ... |
| `resource/` | Vue 3 frontend (site/admin dual entry) + templates (gohtml) + static (badges/pic) |

**Boundary rules**
- Business logic in `service`; data access in `models`/repository layer; HTTP in `http/controllers`.
- Cross-domain access (e.g. forum→notifications) goes through the owner's public service API; no
  foreign SQL.
- Frontend output only via `resource/static/dist` (go:embed); do not hand-write DTOs duplicating the
  backend (once the contract pipeline exists).
- Upstream sync: `git merge` upstream main; resolve conflicts with "our changes win" and record it.

## Key flows

### Auth (planned)

- Today: GitHub OAuth (goth, config [github]).
- Planned: Casdoor OIDC unified login (Web standard authorization code; Mobile appauth+PKCE →
  id_token → `POST /api/auth/oidc/exchange` → forum JWT); numeric-ID constraint enforced server-side
  (see identity-and-access.md).

### Search (Partial)

- Meilisearch optionally enabled (config [meilisearch]); index sync and search UX incomplete, to be
  improved.

### Points (phase 2)

- credit is an OIDC client + standalone ledger; the forum acts as a merchant calling the distribution
  API (see credit-and-escrow.md).

## Consistency principles

- The chosen DB is the business fact source; search, cache, counters, hot lists, and feeds are
  rebuildable projections.
- Critical side effects (notifications, index sync, points distribution) are idempotent, retryable,
  observable.
- Contract changes ship in the same PR: Go struct → openapi.yaml → generated output → fixture tests
  (once the pipeline exists).
