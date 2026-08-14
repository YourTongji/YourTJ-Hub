# Contracts, Data & Derived Projections

> Doc type: architecture
>
> Status: Active (contract pipeline `Partial`)
>
> Owner: Platform maintainers
>
> Last verified: 2026-08-12

## Contract status

The contract capability is **Partial**. The controlled OpenAPI 3.1 entry point is
`packages/api-contract/openapi.yaml`; it currently covers these operations only:

- `POST /api/login`;
- `GET /api/login-public-key`;
- `POST /api/auth/totp/verify`;
- `GET /api/user/totp/status`;
- `POST /api/user/totp/setup`;
- `POST /api/user/totp/enable`;
- `POST /api/user/totp/disable`;
- `POST /api/logout`;
- `POST /api/auth/oidc/exchange`;
- `POST /api/forum/topics/write`;
- `GET /api/user/sessions`;
- `POST /api/user/sessions/revoke`;
- `POST /api/user/sessions/revoke-all`;
- `GET /api/v1/agent/me`;
- `GET /api/v1/agent/topics` and `POST /api/v1/agent/topics`;
- `GET /api/v1/agent/topics/{topicId}/posts` and `POST /api/v1/agent/topics/{topicId}/posts`;
- `GET /api/v1/agent/search`.
- `GET /api/forum/courses` and `GET /api/forum/courses/{courseId}` (course catalog read endpoints, `security: []`);
- `GET /api/forum/courses/{courseId}/reviews` and `POST /api/forum/course-reviews`;
- `PATCH /api/forum/course-reviews/{reviewId}` and `DELETE /api/forum/course-reviews/{reviewId}`;
- `PUT /api/forum/course-reviews/{reviewId}/helpful`,
  `DELETE /api/forum/course-reviews/{reviewId}/helpful`, and
  `POST /api/forum/course-reviews/{reviewId}/reports`;
- `POST /api/forum/moderation/course-review-status`,
  `POST /api/forum/moderation/course-review-reports`, and
  `POST /api/forum/moderation/course-review-reveal`.

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

Breaking-change comparison is not a current gate. The `dev` base before this first coverage contains no
stable operations to compare, so a snapshot baseline would be redundant and misleading. Enable a
base-versus-head bundled-spec breaking gate in a separate change only when `dev` has stable operation
coverage on both sides of the comparison; that gate must compare the PR base and head contracts rather
than a hand-maintained duplicate baseline.

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

## Task queue & background workers

- `task_queue` rows carry a `type` string; workers poll by **type prefix** so task types never leak
  across handlers:
  - `email.*` (activation/reset_password; legacy `activation` / `reset_password` rows are whitelisted)
  - `export` (data export)
  - `file-migrate` (BLOB → object storage migration)
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
- Export files land in `data/export/` and are retained 7 days (daily cron cleanup).

## Config-driven features (pageConfig)

New page config types added with this admin backlog work (all persisted in `page_config`, cached in
`hotdataserve`, editable from the admin panel):

| pageType | Struct | Purpose |
|---|---|---|
| `storageSettings` | `pageConfig.StorageSettings` | provider local/s3, endpoint/bucket/region, bucket lookup (auto/dns/path), credentials, optional public URL prefix |
| `termsOfService` | `pageConfig.TermsOfServiceConfig` | markdown ToS rendered at `/terms` |
| `securitySettings` (extended) | `pageConfig.SecurityAndRegistration` | + reservedUsernames / bannedUsernames / sensitiveWords / sensitiveAction (block\|review) |
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
- Dart generation and full-route coverage are Planned; do not claim they are current contract gates.
- Docs status words updated in step (docs/README.md).
