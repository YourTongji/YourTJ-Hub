# YourTJ-Hub 当前状态

> Doc type: status matrix
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-09

## What works

- **Feature coverage**: Markdown topics/replies, categories, notifications, direct messages, drafts,
  RBAC moderation, admin panel, theme workbench, i18n (en/zh/ja/it), GitHub OAuth + Casdoor OIDC
  (PKCE), TOTP 2FA + recovery codes, session management (jti + user_sessions, per-session revoke),
  scheduled SQLite backup, slow-SQL logging, aggregate search (topics/users/categories with scope
  tabs, pinyin/initials), sensitive-word moderation with review queue, terms-of-service page, data
  import/export, pluggable file storage (SQLite BLOB / S3-compatible), admin-managed bot personas
  (Agents) with unique bearer tokens and webhook endpoints.
- **Unified-auth verification**: Casdoor numeric-ID path verified during research (sub = numeric ID,
  Incremental rule + explicit numeric ids); OIDC login/binding now wired into the forum with
  server-side numeric-sub enforcement.
- Monorepo structure (apps/packages/services/deploy/docs) + CI (server/web/contract workflows).

## Current key gaps

| Domain | Status | Note |
|---|---|---|
| Forum itself | `Current` | Upstream features complete and runnable; `make build` single binary verified locally (2026-08-06: go vet/test, pnpm typecheck/build, smoke all green) |
| Database | `Current` | SQLite default, MySQL optional, PostgreSQL main-db support landed (issue #11); file db stays SQLite; data migration from SQLite→PG is manual |
| Search | `Partial` | Aggregate search landed (issue #22): one search box covers topics, users and categories with grouped sections and scope tabs; pinyin/initials matching for users and categories; index sync via topic/user/category events + migration v13 rebuild; per-scope partial degradation; unavailable-state UI fallback |
| Auth | `Partial` | GitHub OAuth + Casdoor OIDC (PKCE/nonce/numeric-sub enforced) integrated; TOTP 2FA for password login; session listing/revoke; Casdoor-side MFA/Passkey pending deployment config |
| Agents (bot personas) | `Partial` | Admin-managed lifecycle and six-operation Agent forum API are `Current`: users row with explicit actor type (human/bot), one token per agent stored only as hash + non-secret prefix (`agt_`), enable/disable/rotate, dedicated bearer authentication, published topic listing, topic/post writes, post windows and aggregate search; bot rows are rejected by all human-auth paths; mention parsing and webhook wakeups remain `Planned` |
| Contract | `Partial` | OpenAPI 3.1 covers password login, mobile OIDC exchange, topic writing and the six-operation Agent forum API; Redocly lint/bundle, generated TypeScript no-diff checks, committed fixtures, real Gin route tests, and manually maintained Web/Dart mirrors are in CI; broader route coverage and automated Dart generation remain `Planned` |
| Mobile | `Partial` | Flutter client (`apps/mobile`, melos: core/auth/ui_kit/forum_app) implemented: YourTJ token theme bridged into pinned TDesign v1 alpha components, iOS-safe branded navigation, Web-aligned persistent list/card topic feeds, dot-grid auth cards, unified Gf form/dialog/status surfaces, browsing/creation/user/search/notification/IM surfaces, OIDC exchange login; CI `ci-mobile`; not yet deployed to stores (push notifications/custom theme sync/ja-it planned later) |
| Points | `Planned` | services/credit is a README placeholder; explicitly phase 2, not implemented now |
| Branding | `Partial` | Default UI copy, activation template, locales and admin brand settings rebranded to YourTJHub (2026-08-07); CLI name (`gooseforum`) and Go module name intentionally kept for upstream merge |
| Structural governance | `Partial` | Upstream giant controllers (payload.go 72KB etc.) not split; architecture decisions in note |
| Storage (files) | `Current` | Pluggable storage: local SQLite BLOB default + S3-compatible object storage (MinIO/COS/OSS/R2), admin panel config + connection test, cursor-driven BLOB→object migration task + `migrate-files` CLI (2026-08-06) |
| Moderation policy | `Current` | Reserved/banned usernames, sensitive-word block or review (ProcessStatus=2 pending queue with admin approve/reject), banned username auto-freezes existing accounts, moderation audit logs (2026-08-06) |
| Terms of service | `Current` | Editable ToS (markdown) in admin, rendered at `/terms`, registration page links and agreement checkbox (2026-08-06) |
| Data import/export | `Current` | Admin panel JSON/CSV export (users/topics/posts, background task + download) and JSON import with per-row validation report and idempotent skip; export files retained 7 days (2026-08-06) |
| Abuse protection | `Current` | Per-action rate limiting (memory fixed-window, IP+user) on register/login/forgot-password/topic.write/post.create/message.send/upload/interact; 429 + Retry-After; captcha switch + new-user post threshold + honeypot + submit-timing detection; all limits hot-tunable in admin settings |

## Correctness first

Before expanding features, close these baselines (avoid building on a wrong foundation):

1. Complete the remaining Casdoor-side MFA/Passkey deployment configuration without weakening the
   numeric-ID and revocable-session invariants already enforced by the forum.
2. Expand OpenAPI and generated-client coverage before broad API rework, so uncovered routes do not
   become a new source of contract drift.
