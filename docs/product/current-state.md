# Current State & Gaps

> Doc type: implementation inventory
>
> Status: Active
>
> Owner: Product owner, Platform maintainers
>
> Last verified: 2026-08-06

This inventory is based on the current source (apps/gooseforum fork) and states what exists, what is only
a skeleton, and where UI promises diverge from actual behavior. When a later PR changes these
conclusions, update this file in the same PR.

## Solid foundations

- **The forum itself runs**: GooseForum fork (`apps/gooseforum`, Go 1.26 + Gin + Vue 3 + Tailwind),
  single binary (frontend go:embed, `make build` verified locally 2026-08-06), cobra CLI (`serve` /
  mock / rebuild-search-index subcommands).
- **Three-mode rendering**: GoHTML server-side rendering (SEO/no-JS fallback) + JSON payload (SPA
  no-refresh navigation) + pure API; frontend has site/admin dual entry.
- **Feature coverage**: Markdown topics/replies, categories, notifications, direct messages, drafts,
  RBAC moderation, admin panel, theme workbench, i18n (en/zh/ja/it), GitHub OAuth + Casdoor OIDC
  (PKCE), TOTP 2FA + recovery codes, session management (jti + user_sessions, per-session revoke),
  scheduled SQLite backup, slow-SQL logging.
- **Unified-auth verification**: Casdoor numeric-ID path verified during research (sub = numeric ID,
  Incremental rule + explicit numeric ids); OIDC login/binding now wired into the forum with
  server-side numeric-sub enforcement.
- Monorepo structure (apps/packages/services/deploy/docs) + CI (server/web/contract workflows).

## Current key gaps

| Domain | Status | Note |
|---|---|---|
| Forum itself | `Current` | Upstream features complete and runnable; `make build` single binary verified locally (2026-08-06: go vet/test, pnpm typecheck/build, smoke all green) |
| Database | `Decision needed` | SQLite default + MySQL optional today; PostgreSQL migration undecided (upstream migration framework in app/migration) |
| Search | `Partial` | Meilisearch optionally enabled (config [meilisearch]); index sync and search UX incomplete, needs work |
  | Auth | `Partial` | GitHub OAuth + Casdoor OIDC (PKCE/nonce/numeric-sub enforced) integrated; TOTP 2FA for password login; session listing/revoke; Casdoor-side MFA/Passkey pending deployment config |
| Contract | `Partial` | No swagger annotations, no openapi.yaml upstream; packages/api-contract is a placeholder; pipeline not built |
| Mobile | `Planned` | `apps/mobile` is a placeholder dir; Flutter/melos/Riverpod not set up |
| Points | `Planned` | services/credit is a README placeholder; explicitly phase 2, not implemented now |
| Branding | `Partial` | GooseForum branding not yet replaced with yourtj (CLI name, UI copy, config keys) |
| Structural governance | `Partial` | Upstream giant controllers (payload.go 72KB etc.) not split; architecture decisions in note |

## Correctness first

Before expanding features, close these baselines (avoid building on a wrong foundation):

1. Database decision (recommend PostgreSQL) and verify compatibility with the upstream migration framework.
2. Auth chain closed (Casdoor → exchange → JWT), numeric-ID constraint enforced server-side.
3. Contract pipeline (swag or manual openapi → TS/Dart generation) before broad API rework, to prevent
   contract drift.
