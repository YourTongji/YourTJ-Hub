# Contracts, Data & Derived Projections

> Doc type: architecture
>
> Status: Active (contract pipeline `Partial`)
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-28

## Contract status

The contract capability is **Partial** (all JSON routes are described; generated Dart and a few
protocol surfaces remain open). The controlled OpenAPI 3.1 entry point is
`packages/api-contract/openapi.yaml`. Every `/api` JSON route is covered by an OpenAPI operation —
the generated `packages/api-contract/coverage-matrix.md` is the authoritative route-by-route list,
and CI rejects any route that is neither contracted nor listed. By domain:

- auth and account: password login, login public key, TOTP (verify + management), logout,
  registration/password recovery, mobile OIDC exchange, session management, captcha, user-card,
  profile/email/username/avatar/badge settings, upload-avatar, change-password, OAuth
  bindings/unbind, and the user content lifecycle (my-content, deleted-content, restore,
  batch-delete, purge, privacy-erase, content-event, account-close);
- forum: topic write, post CRUD/window/revisions, topic status/delete, like/bookmark/watch on
  topics and posts, follow-user, report, aggregate search, site statistics, notifications/unread,
  chat, and the moderator workbench (`/api/forum/moderation/*`);
- admin console (`/api/admin/*`): user/role/category/moderator management, topic/post moderation,
  agent administration, operation records, traffic overview, page settings, site settings, and
  data import/export;
- Agent public API (`/api/v1/agent/*`), course catalog + reviews + moderation, and the PK
  scheduler;
- Wiki 域（`paths/wiki.yaml` + `paths/wiki-sync.yaml`，GitHub 唯一真实源模型）：公开读
  `GET /api/wiki/{tree,namespaces,home}`；管理端 `/api/admin/wiki/*`（PageManager：只读树 +
  `sync/status` / `sync` / `sync/runs` / `sync/webhook-secret` 读写 + asset CDN 设置）与公开
  `POST /api/wiki/webhook`（GitHub push 事件，HMAC-SHA256 验签）。写即发布/CAS/版本历史/回滚/
  diff/编辑者/命名空间 CRUD 等站内写端点均已**退役**（编辑/审核/历史/贡献者走 GitHub PR，
  命名空间由仓库顶层目录同步驱动），不得重新加入契约——覆盖门禁会拦下任何未申报的路由变化。
  公开读与 `/api/admin/wiki/{tree,sync/status,sync/runs}` 在数据库查询失败时返回
  **HTTP 500 + `wiki.readFailed`**（契约已声明 500 响应），与真实空 wiki（200 + 空结果）
  严格区分（issue #287）；GitHub SSOT 下站内无「编辑者」概念，`wiki.home` 的 `recent[]` 与
  详情负载的 `editorId`/`editorName` 字段已移除；Git 作者信息由详情页 `contributors[]` 提供
  （无论坛数字用户 ID）：同步器从仓库 `git log` 按 email 聚合贡献者与提交数，GitHub noreply
  隐私邮箱解析出 `username` → 前端拼 `avatarUrl`（`github.com/{user}.png`）与 `githubUrl`
  外链；自定义邮箱贡献者两者为空（前端降级首字母占位）（issues #291/#310）。

- Wiki 局内搜索 `GET /api/wiki/search` 使用 Meilisearch 的 `wiki_pages` 段落索引：每个段落
  一个文档，公开 API 再按页面聚合结果。`q` 最长 100 个字符，`limit` 默认 12、最大 20；
  `total` 是去重后的公开页面数，不是段落命中数；Meilisearch 不可用时返回 HTTP 200、空
  `items` 与 `searchUnavailable: true`。结果中的 `anchors` 来源于 `wiki_pages.para_anchors`
  投影，前端用 `#s-<n>` 精确跳转。页面更新、无段落页面和 Wiki 软删都会先清理该页面旧
  段落文档，搜索索引只作为可重建投影。

The remaining **Partial** gaps are not missing routes but: the OIDC Provider standard endpoint
suite (separate OAuth/OIDC contract track), AI-readable text surfaces (`/llms.txt` etc.), and
hand-maintained Dart mirrors (generation remains Planned).

Paths are split per domain under `packages/api-contract/paths/` (for example `auth.yaml`,
`auth-sessions.yaml`, `forum-topics.yaml`); new coverage adds a new per-domain file instead of
extending an existing one, so parallel contract PRs only meet in the `openapi.yaml` entry point and
the generated TypeScript output.

The first coverage intentionally describes the current legacy wire behavior. A business failure commonly
uses HTTP `200` with `{ "code": 1, "result": null, "messageCode": ... }`; consumers must inspect the
JSON envelope rather than treating every 2xx status as application success. Middleware-owned failures
such as unauthenticated access, frozen or unresolvable authenticated accounts, and rate limiting remain
HTTP `401`, `403`, and `429` with the same failure envelope. Topic write's current permissive `UpButterReq`
wrapper reports malformed or incomplete JSON as
an HTTP `200` validation failure, not a guaranteed `400`.

## HTTP method contract: HEAD vs GET (issue #411)

The single-binary app is served directly by Gin behind a CDN / reverse proxy
(1Panel + Cloudflare in production; no in-compose nginx). Probes, cache
warmers, and CDN origin checks frequently use `HEAD`; this section is the
contract for what `HEAD` means on the forum's own listener
(`apps/gooseforum/app/http/routes/bridge.go` → `RegisterByGin`).

### Mechanism: Gin v1.12 does not synthesize HEAD for GET

Gin registers and routes methods **exactly** as declared: a `HEAD` request
only matches a route explicitly registered for `HEAD`. In this codebase the
complete explicit HEAD surface on the **production** listener is produced by
two calls and one helper:

- `StaticFS("assets", ...)` and `StaticFS("static", ...)` register
  `GET` **and** `HEAD` for `/assets/*filepath` and `/static/*filepath`
  (`route4api.go` `assertRouter`);
- `Any("/mcp")` registers `HEAD` alongside every other method
  (`mcpRoute.go`).

In dev mode (`app.env != production`) `assertRouter` additionally
registers a Vite reverse proxy via `Any("assets/*path")` (`route4api.go`),
which answers HEAD for `/assets/*` as well. The contract tests and the
tables below exercise the production listener.

Everything else — SSR pages (`/`, `/p/post/:id`, ...), `/health`,
`/robots.txt` / `/sitemap.xml` / `/rss.xml`, JSON APIs (`/api/...`), and the
file download route `/file/img/*filename` — is GET-only. `HEAD` on those
paths is not a route error but the plain NoRoute handler, i.e. the same
`404` JSON envelope an unregistered method gets. The Go `net/http` server
then strips the response body on the wire (RFC 9110: HEAD responses never
carry a body), so a CDN or proxy never sees NoRoute's JSON payload on a HEAD
request.

### Supported HEAD surface (static assets)

The two static mounts answer HEAD, and each carries its own production cache
header via a mount-level subgroup in `assertRouter` (`AssetsCache` on
`/assets`, `BrowserCache` on `/static`; Gin captures group middleware when
each route is registered, so neither header leaks to the other mount). The
contract therefore promises:

- `/static/*filepath`: HEAD == GET status and headers, **including** the
  long-public `Cache-Control` (`public, max-age=18144000`);
- `/assets/*filepath`: HEAD == GET status and headers, **including** the
  immutable `Cache-Control` (`public, max-age=31536000, immutable` — Vite
  filenames carry a content hash, so a byte change always changes the URL);

both with an empty body on the wire.

| Request | Status | Body | Content-Length | Content-Type / Cache-Control |
|---|---|---|---|---|
| `GET /static/...` | 200 | full file | set | set |
| `HEAD /static/...` | 200 (same as GET) | empty (never sent) | set, same as GET | same as GET |
| `HEAD /static/...` with `Accept-Encoding: gzip` | 200 | empty | unset (no body was written, so the gzip middleware cannot size it) | Content-Type present; no Content-Encoding |
| `GET` / `HEAD /assets/...` (built entry) | 200 | full file / empty | set, same across methods | Content-Type set; immutable `Cache-Control` |

`HEAD` for an existing asset is therefore safe for cache existence checks
(both mounts guarantee their production `Cache-Control`).
`HEAD`/`GET` for a *missing* asset are both the same NoRoute 404: gin's
StaticFS handler hands failed file opens to the engine's NoRoute chain
(`controllers.NotFound`), so both methods answer the identical 404 JSON
envelope — HEAD simply never sends the body on the wire. Failure responses
carry `Cache-Control: no-store`, never a mount max-age: both cache
middlewares defer the header decision to the final response status
(`httputil.DeferCacheHeader`), so a missing content-hashed chunk during a
deploy rollback cannot be pinned into browser or shared caches. The `/assets`
contract tests probe a file the Vite build actually emits
(`static/dist/.vite/manifest.json`, site entry) and skip when the build
output is absent — `dist/` is never committed and CI never builds the
frontend.

### Unsupported HEAD surface (dynamic GET routes)

| Request | GET status | HEAD status |
|---|---|---|
| `/health` | 200 (db ping ok) / 503 | **404** |
| `/robots.txt`, `/sitemap.xml` | 200 | **404** |
| `/` (SSR home) | 200 | **404** |
| `/api/login-public-key`, `/api/forum/get-site-statistics`, ... | 200 | **404** |
| `/file/img/:filename` (existing upload) | 200 | **404** (no HEAD route registered) |
| write-only endpoints (`POST /api/forum/topics/write`, `POST /file/img-upload`, ...) | 404 (method not routed for anonymous GET) | **404** — the controller never executes |

Consequences for operators and integrators:

- **Probes must use `GET`.** `deploy.sh`, compose health checks, and 1Panel /
  Cloudflare health endpoints already use `GET /health` (curl `-fsS`). Do not
  switch monitor probes to `HEAD /health`: they would receive the stable
  404 NoRoute response even when the service is healthy.
- **HEAD has no *business* side effects — but it is not free on `/mcp`.**
  Write controllers are reachable only via their registered method
  (`POST`/`PATCH`/`DELETE`/`PUT`). No write route has a HEAD registration, so
  a HEAD request never triggers a write; unregistered methods on write paths
  answer the stable 404 envelope. `Any("/mcp")` does register HEAD, so
  `HEAD /mcp` runs the `mcp.auth` per-IP rate limiter and consumes the same
  quota as any other `/mcp` request — repeated HEAD probes can exhaust the
  quota and make legitimate MCP calls answer 429. Health probes must use
  `GET /health`, which is outside the MCP rate limits. If a future change
  needs HEAD on a dynamic route, register it explicitly and run the
  routes-snapshot gate (`TestRoutesSnapshot`) — the snapshot carries HEAD
  only for the two static mounts (`/assets/*filepath`, `/static/*filepath`)
  and `/mcp` today, and `TestHeadRouteRegistrationContract` pins that exact
  surface.
- **CDN origin checks**: if a CDN is configured to validate origin
  availability with HEAD against a page or API URL, point it at a `/static`
  or `/assets` asset (both mounts carry their production cache header) or at
  `GET /health` (use GET).

### Guard tests

`apps/gooseforum/app/http/routes/contract_head_http_test.go` asserts this
contract against the real `RegisterByGin` assembly over a real HTTP server:
the registered HEAD route set is exactly the static mounts + `/mcp`; `/static`
HEAD equals GET in status and headers (including the long-public
`Cache-Control`); `/assets` HEAD equals GET including the immutable
`Cache-Control`, probed against a manifest-emitted file and skipped without
`pnpm build`; missing assets on both mounts answer 404 with
`Cache-Control: no-store` (`TestHeadContractMissingAssetsAreNotCacheable`);
dynamic GET routes and write-only endpoints answer 404 to HEAD; and no body
ever reaches the wire on HEAD. The
tests pin `server.gzip` on, so the gzip assertions hold under any local
`[server].gzip` setting.

## Contract pipeline (Partial)

```
Go controller, route wrapper, and Gin middleware (current behavior)
   │  manually maintained operation descriptions
   ▼
packages/api-contract/openapi.yaml + paths/ + components/
   │  Redocly lint and bundle             │  openapi-typescript
   ▼                                      ▼
packages/api-contract/fixtures/      @gooseforum/client/openapi types
   │                                      │
   └──── route-level httptest assertions ──┴──── CI generated-output no-diff gate
```

- **Go behavior is the fact source**: controllers, route wrappers, middleware, and their `httptest`
  coverage establish the behavior being documented.
- **OpenAPI is the controlled protocol source**: each covered operation documents that behavior in a
  reviewable, consumable format. It is split into entry point, paths, and schemas but linted and bundled
  as one contract.
- **Generated Web types are artifacts**: `@gooseforum/client/openapi` exports only the generated OpenAPI
  types under `src/gen/`; it does not replace the existing hand-written page payload contracts or create
  a request client. CI regenerates the types and rejects an uncommitted diff.
- **Fixtures are representative wire samples**: committed JSON fixtures capture stable envelope shape,
  message codes, and rate-limit metadata. The route-level Go tests exercise real Gin route chains and
  assert the actual status, envelope, result shape, and `Retry-After` behavior against those fixtures.
- **Mobile/Dart generation is Planned**: no Dart generator or generated mobile artifact is maintained by
  this repository yet. Mobile response mirrors remain hand-maintained, and shared OpenAPI fixtures
  exercise their runtime deserialization where the mobile client consumes a controlled operation.
- **Route coverage is gated**: `TestRoutesSnapshot` (`apps/gooseforum/app/http/routes/routes_dump_test.go`)
  dumps every route `RegisterByGin` registers under the default config into
  `packages/api-contract/fixtures/routes-snapshot.json` (OIDC `/api/oauth/*` endpoints are excluded —
  they are only registered with `oidc.enabled=true` and are tracked in their own slice). After a route
  change, regenerate with `YOURTJ_UPDATE_ROUTES_SNAPSHOT=1 go test ./app/http/routes/ -run TestRoutesSnapshot`;
  `go test ./...` fails on snapshot drift. `scripts/check-route-coverage.mjs` (part of `pnpm run check`)
  then requires every snapshot route to be either an OpenAPI operation or listed in
  `packages/api-contract/route-coverage.json` (`excluded` for non-JSON-API routes such as SSR pages and
  static assets, `knownUncovered` with an owning slice for pending `/api` routes), rejects stale or
  dangling list entries and contract operations with no matching route, and regenerates the committed
  `packages/api-contract/coverage-matrix.md` — CI rejects an uncommitted matrix diff, same as the
  generated TypeScript types. Net effect: a new route that is neither contracted nor listed turns CI red.

Breaking-change comparison is not a current gate. Enable a base-versus-head bundled-spec breaking
gate (for example `oasdiff` against the bundled base and head specs) in a separate change; that gate
must compare the PR base and head contracts rather than a hand-maintained duplicate baseline. Route
coverage reached 100% of `/api` routes with issue #277, so both sides of a PR comparison now carry
complete operation coverage and the precondition for such a gate is met.

## Data model

- Migrations: upstream `app/migration` (Go migrations, run at startup/CLI); PostgreSQL is the
  default deployment database and SQLite the local development/test default; MySQL is not
  supported; the file db stays SQLite.
- Post content revisions: `post_revisions` is an append-only snapshot table (post_id, version,
  editor_id, content, rendered_html, process_status, created_at). Every content edit — first post
  (post_no = 1) and replies alike, by the author — appends a new version inside the edit
  transaction and updates `posts.last_editor_id` / `last_edited_at`; post creation seeds version 1
  (editor = author). A row lock serializes concurrent edits so (post_id, version) stays monotonic.
  History is read-only (`GET /api/forum/posts/revisions?postId=`): deleted/anonymized posts
  blank all version bodies, and blocked posts plus pending-review versions hide their bodies from
  non-moderators — the same visibility rules the post window applies. Permanent deletion and
  privacy erasure blank revision bodies so the snapshot table cannot bypass the deletion lifecycle.
- State machines: business lifecycles use explicit state machines (e.g. topic:
  draft/published/archived/deleted), not ambiguous boolean combinations (product principle 9).
- Soft/hard delete policy is decided with the database migration decision; record in the note.
- Agent model: `users.actor_type` (0 human / 1 bot) plus `agents` (user_id PK-join, token_prefix,
  token_hash, webhook_endpoint, enabled, created_by, last_used_at); `users.username` has one database
  unique index shared by human and bot accounts. The token hash is the only stored secret material,
  the prefix is a non-secret lookup key. Token rotation is a compare-and-swap on the current prefix
  (concurrent rotations fail loudly); disable clears the token hash, so re-enabling requires an
  explicit rotation. Rotation, disablement, and profile changes use column-scoped updates rather
  than saving stale snapshots; successful authentication touches `last_used_at` at most once per
  minute.
- Agent public API coverage: the six operations under `/api/v1/agent` (`me`, topic list/create,
  post list/create, search) are `Current` in the OpenAPI contract (`Agent` tag, `agentBearerAuth`
  security scheme, `paths/agent.yaml`, dedicated schemas and fixtures). The route-level contract
  tests assert all six operations plus the canonical `auth.required` 401 envelope shared by every
  failed Agent credential. Agent writes reuse the human topic/post rate limits; browser-only
  honeypot, captcha, and new-user cooldown gates are skipped.
- Agent mention parsing and webhook sending remain `Planned`; they are not part of the covered
  contract surface.
- SQL connections enable GORM error translation so uniqueness races map to stable domain errors. The
  structured GORM logger implements `ParamsFilter`; parameterized logging therefore keeps bind values
  out of rendered SQL instead of relying on an otherwise inert configuration flag.
- Search index sync is event-driven: topic publish/update/delete events keep Meilisearch documents in
  sync; the index is a rebuildable projection (`rebuild-search-index` CLI), not the only truth.
- Wiki paragraph search follows the same projection rule: `wiki_pages.para_anchors` is derived from
  rendered Wiki Markdown, and the `wiki_pages` Meilisearch index is rebuilt from the database or
  incrementally replaced per page. Public search applies the page visibility boundary both in the
  index filter and in the service aggregation defense check.
- Course aggregation & lineage (2026 课程沿革): `course.review_scope`（teacher/team/course 三档）
  与 `course.team_key`（教学团队键）驱动课评聚合口径——team 档在详情读取时按团队全部卡
  实时聚合评分/分布/教师名单（无独立投影表）；`offering.teaching_class_id` 与 `term` 构成
  唯一索引，是排课物化与历史数据包导入共享的 offering 定位键（物化为权威写入源，导入
  从属复用；两源均不写 `offering.status`）；`instructor.teacher_code` 落库工号供规则引擎
  跨学期匹配。`course_relations` 沿革表（from_course_id → to_course_id，relation_type
  EQUIVALENT/RENAMED_FROM/SPLIT_FROM/MERGED_FROM/RELATED，source rule/manual，status
  pending/approved/ignored/merged，evidence_json 证据快照；同 (from,to,type) 唯一）只表达
  语义、不参与课程身份。人工确认等价（EQUIVALENT/RENAMED_FROM）后 `MergeCourses` 物理
  合并：offering（评价/教师关联随行零丢失）与别名（冲突跳过并记录）迁移到 to 卡、from 卡
  隐藏、evidence_json 写入合并快照（`UndoMergeCourse` 按快照反向迁移恢复）；合并/撤销/审核
  均写 `opt_record` 审计。候选由确定性规则引擎 `courseservice/lineage` 产出：
  教学班级级 `course-lineage-scan` CLI（R1-R5，dry-run JSON，输入 pk_course_detail.id）
  与卡级 `course-lineage-seed` CLI（E1-E3，装配课程目录可见卡在 course.id 层面配对，
  默认 dry-run，`--write`/`--write-family` 幂等写 pending，不复活已处置关系），
  或管理端手动创建；审核面板
  （`/api/forum/moderation/course-relation-*`、`course-merge(-undo)`，OpenAPI 覆盖）按状态
  （pending/approved/ignored/merged）与类型（EQUIVALENT/RENAMED_FROM/SPLIT_FROM/
  MERGED_FROM/RELATED）过滤，支持批准/忽略/撤回处理决定（approved/ignored →
  pending；merged 只能走 `course-merge-undo`）/合并/撤销。

## Task queue & background workers

- `task_queue` rows carry a `type` string; workers poll by **type prefix** so task types never leak
  across handlers:
  - `email.*` (activation/reset_password; legacy `activation` / `reset_password` rows are whitelisted)
  - `export` (data export)
  - `import` (staged, checksum-addressed JSON import)
  - `file-migrate` (BLOB → object storage migration)
  - `topic-search.*`, `user-search.*`, `category-search.*`, and `course-search.*` (search projections)
- Task claiming is **atomic with a lease** (issue #138): a worker claims a row via a CAS update
  (`pending/retrying → running`, `RowsAffected = 1`), so concurrent workers/processes never execute
  the same task simultaneously from the queue's perspective. Each claim generates a fresh,
  non-reusable fencing token (`lease_token`); `processed_at` only tracks the lease start for
  expiry-based reclaim. Every write back — terminal state (`success/failed/retrying`), retry
  counter, deletion, and progress payload (`task_json`) — is fenced on the fencing token
  (`status = running AND lease_token = ?`), so a worker whose lease was reclaimed can neither
  overwrite the new owner's cursor nor flip its state. Workers periodically reclaim `running` tasks
  whose lease expired (crashed process), so no task stays stuck in `running` forever.
  **Delivery semantics are at-least-once, not exactly-once**: fencing protects DB state writes but
  cannot cancel a handler's external side effects once a lease is lost — the heartbeat only notices
  loss up to `leaseRenewInterval` (30s) later, a heartbeat `err` (DB blip) branch keeps the handler
  running, and email handlers take no context (`DialAndSend` is uninterruptible). A reclaimed task
  can therefore be executed by the stale worker and then again by its new owner; the fencing token
  only prevents the stale worker from writing terminal state over the new owner.
- Export and migration tasks update `task_json` with progress payloads (`processed/total/errorCount`,
  cursor `lastId`) so the admin panel can render live progress and resume after restarts.
- Import requests are bounded at 50 MiB and staged with mode `0600`; the task payload stores only the
  filename, format, and SHA-256. The worker rechecks the digest, runs row validation and invariant
  rebuilding in one transaction, and keeps the source file on failure. `POST
  /api/admin/data/import/tasks/{taskId}/replay` resets a failed task after the operator has fixed the
  cause. Identical uploads reuse the same task instead of overwriting a failed task's source.
- Search projection rows are transaction-bound for the main topic/category/wiki write paths. User
  profile, likes/follows, and content delete/restore event paths enqueue after the business commit;
  they therefore have a short crash window but remain idempotent, retryable, and rebuildable. A
  worker re-reads current database state before upserting or deleting the external document, so an
  unavailable Meilisearch instance only delays a rebuildable projection.
- Export files land in `data/export/` and are retained 7 days (daily cron cleanup).

## Config-driven features (pageConfig)

New page config types added with this admin backlog work (all persisted in `page_config`, cached in
`hotdataserve`, editable from the admin panel):

| pageType | Struct | Purpose |
|---|---|---|
| `storageSettings` | `pageConfig.StorageSettings` | provider local/s3, endpoint/bucket/region, bucket lookup (auto/dns/path), credentials, optional public URL prefix, optional `internalEndpoint` (S3 mode: routes server-side data-plane operations; empty or same host as `endpoint` → single client) |
| `termsOfService` | `pageConfig.TermsOfServiceConfig` | markdown ToS rendered at `/terms` |
| `securitySettings` (extended) | `pageConfig.SecurityAndRegistration` | + reservedUsernames / bannedUsernames / sensitiveWords / sensitiveAction (block\|review)。reserved 同时约束 username 与 nickname（归一化全等）；banned 含新增词冻结存量；sensitiveWords 经归一化子串扫描覆盖话题/回复/私信/课评/个人资料自由文本。内置默认词库见 `defaultconfig/pageconfig/security.json`（来源 fwwdn/sensitive-stop-words，Apache-2.0），存量空数组由 v27 数据迁移补齐，banned 默认恒为空（防误冻结） |
| `postingSettings` (extended) | `pageConfig.PostingContent` | `llms.enabled`, `llms.fullText`, and `llms.files` gate the public AI-readable projections |

Object storage addressing notes: Alibaba OSS and Tencent COS (buckets created after 2024-01-01)
require virtual-hosted style — use `bucketLookup: dns` with an explicit region; MinIO/R2 accept
`auto`/`path`. Endpoint may include a scheme; it is stripped before building the minio-go client
(`Secure` is derived from the scheme).

## Derived projections

| Projection | Source | Rebuildable |
|---|---|---|
| Search index | Meilisearch | ✅ full rebuild (rebuild-search-index CLI; reconciles with DB by removing documents missing from DB) |
| Counters (replies/likes) | DB aggregate or cache | ✅ recompute |
| Hot lists / feeds | derived queries | ✅ |
| Notification read/unread | user pointer table | ✅ |
| AI-readable exports (`llms.txt`, full text, per-topic Markdown) | published topics and normal, non-deleted posts in the DB | ✅ generated on demand; 10-second cache cleared by topic/reply/category events, direct clear on moderation/reply-edit/topic-category/unpublish paths, and relevant setting changes; full export capped at 5000 topics / 8 MiB / 30 s (truncated with marker) |

Principle: projections must be rebuildable from the fact source; never treat a projection as the only truth.

The AI-readable exports are public text representations, not JSON API operations in the controlled
OpenAPI surface. Their visibility boundary is the existing forum publication and moderation state:
draft, blocked, pending-review, soft-deleted, or first-post-blocked content is excluded. The index follows
the llms.txt Markdown structure (site heading, optional description, and a `Topics` link list); full-text
and per-topic documents preserve the stored Markdown source.

## Contract change discipline

- For an OpenAPI-covered operation, backend behavior, `openapi.yaml`, generated TypeScript output, and
  representative fixtures/route-level contract tests ship in the same PR.
- Operations outside the current coverage remain manually synchronized with their consumers until they
  are added to the controlled contract. `@gooseforum/client` must stay in sync with Go structs.
- The mobile client mirrors contracts into `apps/mobile/packages/core/lib/src/gen/*.dart` (a
  generated-artifact placeholder until the OpenAPI pipeline lands). Backend/TS contract changes that
  affect the mobile surface must update the Dart mirrors in the same PR; fixture contract tests
  (`core/test/fixtures`) back runtime deserialization.
- Automated Dart generation remains Planned; route coverage is a current contract gate as described
  above and is not a Planned capability.
- Docs status words updated in step (docs/README.md).
