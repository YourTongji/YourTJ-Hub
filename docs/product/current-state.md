# YourTJ-Hub 当前状态

> Doc type: status matrix
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-10

## What works

- **Feature coverage**: Markdown topics/replies, categories, notifications, direct messages, drafts,
  RBAC moderation, admin panel, theme workbench, i18n (en/zh/ja/it), GitHub OAuth + built-in
  OIDC Provider (PKCE), TOTP 2FA + recovery codes, session management (jti + user_sessions, per-session revoke),
  scheduled SQLite backup, slow-SQL logging, aggregate search (topics/users/categories with scope
  tabs, pinyin/initials), sensitive-word moderation with review queue, terms-of-service page, data
  import/export, pluggable file storage (SQLite BLOB / S3-compatible), admin-managed bot personas
  (Agents) with unique bearer tokens and webhook endpoints, and configurable AI-readable public-content
  exports (`/llms.txt`, `/llms-full.txt`, `/p/posts/{id}.md`).
- **Built-in OIDC Provider**: the forum issues standard OIDC tokens (authorization code + PKCE S256,
  RS256 id_token, opaque access tokens) for first-party clients; `sub` is always the numeric users.id.
- Monorepo structure (apps/packages/services/deploy/docs) + CI (server/web/contract workflows).

## Current key gaps

| Domain | Status | Note |
|---|---|---|
| Forum itself | `Current` | Upstream features complete and runnable; `make build` single binary verified locally (2026-08-06: go vet/test, pnpm typecheck/build, smoke all green) |
| Database | `Current` | SQLite default, MySQL optional, PostgreSQL main-db support landed (issue #11); file db stays SQLite; data migration from SQLite→PG is manual |
| Search | `Partial` | Aggregate search landed (issue #22): one search box covers topics, users and categories with grouped sections and scope tabs; pinyin/initials matching for users and categories; index sync via topic/user/category events + migration v13 rebuild; per-scope partial degradation; unavailable-state UI fallback |
| Course reviews (课评) | `Partial` | Cross-dialect course catalog schema (course/alias/term/offering/instructor/offering-instructor/import-run/source-ref), offline `course-import` CLI with manifest checksum + dry-run + quarantine + idempotent retry, PG read service with keyword/teacher/term/campus filters, SSR `/courses` + `/courses/:courseId` and JSON `GET /api/forum/courses(/{courseId})` (OpenAPI-covered, route contract tests green); native review write, Meilisearch sync, and moderation remain `Planned` |
| Auth | `Partial` | Password + TOTP 2FA + GitHub OAuth + built-in OIDC Provider (authorization code/PKCE S256, RS256 id_token, opaque access tokens, numeric sub) implemented; mobile exchange keeps the forum-JWT contract; bot personas are rejected by all human-session and OIDC paths |
| Agents (bot personas) | `Partial` | Admin-managed lifecycle and six-operation Agent forum API are `Current`: users row with explicit actor type (human/bot), one token per agent stored only as hash + non-secret prefix (`agt_`), enable/rotate, and disable that revokes the credential (re-enabling requires rotation), dedicated bearer authentication, published topic listing, topic/post writes, post windows and aggregate search; bot rows are rejected by all human-auth paths; mention parsing and webhook wakeups remain `Planned` |
| Contract | `Partial` | OpenAPI 3.1 currently covers password login, logout, mobile OIDC exchange, session management (list/revoke/revoke-all), topic writing and the six-operation Agent forum API; Redocly lint/bundle, generated TypeScript no-diff checks, committed fixtures and real Gin route tests are in CI; mobile Dart mirrors stay hand-maintained under fixture deserialization checks; automated Dart generation and broader route coverage remain `Planned` |
| Mobile | `Partial` | Flutter client (`apps/mobile`, melos: core/auth/ui_kit/forum_app): persistent four-destination shell (home/search/messages/profile) with per-branch state and a central global publish action; explicit 44dp back actions on pushed pages and long-page scroll-to-top; Web-aligned list/card topic feeds; redesigned topic detail and profile; global Markdown publish editor (narrow-screen edit/preview switch, wide two-column editor+preview, formatting/image toolbar, drafts and edit prefill); reply composer with image action; structured skeleton loading; unified settings/login/notifications/drafts; OIDC exchange login with dot-grid auth cards; repo-owned `ui_kit` Gf* components over pinned TDesign v1 alpha; CI `ci-mobile`; not store-deployed — push notifications, custom theme sync, and ja/it locales remain gaps |
| Points | `Planned` | services/credit is a README placeholder; explicitly phase 2, not implemented now |
| Branding | `Partial` | Default UI copy, activation template, locales and admin brand settings rebranded to YourTJHub (2026-08-07); default wordmark assets `resource/static/pic/brand-default.{png,webp}` + mobile `assets/images/brand-default.png` regenerated from `hublogo.png` (transparent RGBA, 2026-08-09); admin `brandType=image` still overrides with uploaded `/file/img/…`; CLI name (`gooseforum`) and Go module name intentionally kept for upstream merge |
| Structural governance | `Partial` | Upstream giant controllers (payload.go 72KB etc.) not split; architecture decisions in note |
| Storage (files) | `Current` | Pluggable storage: local SQLite BLOB default + S3-compatible object storage (MinIO/COS/OSS/R2), admin panel config + connection test, cursor-driven BLOB→object migration task + `migrate-files` CLI (2026-08-06) |
| Moderation policy | `Current` | Reserved/banned usernames, sensitive-word block or review (ProcessStatus=2 pending queue with admin approve/reject), banned username auto-freezes existing accounts, moderation audit logs (2026-08-06) |
| Terms of service | `Current` | Editable ToS (markdown) in admin, rendered at `/terms`, registration page links and agreement checkbox (2026-08-06) |
| Data import/export | `Current` | Admin panel JSON/CSV export (users/topics/posts, background task + download) and JSON import with per-row validation report and idempotent skip; export files retained 7 days (2026-08-06) |
| Abuse protection | `Current` | Per-action rate limiting (memory fixed-window, IP+user) on register/login/forgot-password/oidc.authorize/oidc.token/topic.write/post.create/message.send/upload/interact/llms.index/llms.full/llms.topic/mcp.auth; 429 + Retry-After; captcha switch + new-user post threshold + honeypot + submit-timing detection; all limits hot-tunable in admin settings |
| AI-readable content | `Current` | Admin posting settings independently gate the llms.txt index, full-text export, and per-topic Markdown; exports include only published topics with normal first posts and normal, non-deleted replies; generated content is cached for 10 seconds and invalidated by topic/reply/category events, direct clear on moderation/reply-edit/topic-category/unpublish paths, or relevant setting changes; full export is hard-capped (5000 topics / 8 MiB / 30 s) and truncated with a marker |
| MCP server | `Partial` | Official MCP server (issue #93) ships in the single binary: `/mcp` streamable HTTP endpoint + `mcp-stdio` subcommand, exposing the six-operation Agent forum API as curated handwritten tools (me / list_topics / get_posts / search always; create_topic / create_post only when the MCP write setting is enabled). Both the endpoint (`mcp.enabled`) and write tools (`mcp.writes`) are managed from the admin panel (Settings → MCP server), stored in DB via `page_config` and applied hot (5s cache) without a restart; both default to off. When `mcp.enabled` is off, `/mcp` answers 404 and exposes no MCP surface. Auth reuses the `agt_` bearer token via `agentservice.ResolveByToken`, with unauthenticated floods bounded by the shared `mcp.auth` per-IP rate limit; write tools share the existing topic.write / post.create rate limits (IP + bot userId, SkipAdmin exemption), and `mcp-stdio --writes` can override the write setting per local session. `/mcp` requests lift the 10s server write timeout (GET SSE streams stay unlimited, bounded by the 15m session timeout; POSTs get a finite 60s write deadline so a client that stops reading cannot pin a session). Public-facing OAuth 2.1 + RFC 9728 resource metadata, SSE fallback for domestic clients, and webhook/mention wakeups remain `Planned` |

## Correctness first

Before expanding features, close these baselines (avoid building on a wrong foundation):

1. Decide MFA policy for built-in OIDC / GitHub OAuth login paths (forum TOTP reuse is a `Decision needed`).
2. Expand OpenAPI and generated-client coverage before broad API rework, so uncovered routes do not
   become a new source of contract drift.
