# YourTJ-Hub 当前状态

> Doc type: status matrix
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-06

## What works

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
| Database | `Current` | SQLite default, MySQL optional, PostgreSQL main-db support landed (issue #11); file db stays SQLite; data migration from SQLite→PG is manual |
| Search | `Partial` | Meilisearch optionally enabled (config [meilisearch]); index sync wired via topic events (publish/update/delete), unavailable-state UI fallback landed; full-text UX still minimal |
| Auth | `Partial` | GitHub OAuth + Casdoor OIDC (PKCE/nonce/numeric-sub enforced) integrated; TOTP 2FA for password login; session listing/revoke; Casdoor-side MFA/Passkey pending deployment config |
| Contract | `Partial` | No swagger annotations, no openapi.yaml upstream; packages/api-contract is a placeholder; pipeline not built |
| Mobile | `Planned` | `apps/mobile` is a placeholder dir; Flutter/melos/Riverpod not set up |
| Points | `Planned` | services/credit is a README placeholder; explicitly phase 2, not implemented now |
| Branding | `Partial` | GooseForum branding not yet replaced with yourtj (CLI name, UI copy, config keys) |
| Structural governance | `Partial` | Upstream giant controllers (payload.go 72KB etc.) not split; architecture decisions in note |
| Abuse protection | `Current` | Per-action rate limiting (memory fixed-window, IP+user) on register/login/forgot-password/topic.write/post.create/message.send/upload/interact; 429 + Retry-After; captcha switch + new-user post threshold + honeypot + submit-timing detection; all limits hot-tunable in admin settings |

## Correctness first

Before expanding features, close these baselines (avoid building on a wrong foundation):

1. Auth chain closed (Casdoor → exchange → JWT), numeric-ID constraint enforced server-side.
2. Contract pipeline (swag or manual openapi → TS/Dart generation) before broad API rework, to prevent
   contract drift.
