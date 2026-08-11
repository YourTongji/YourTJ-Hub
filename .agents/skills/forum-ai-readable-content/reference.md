# Public AI-Readable Export Reference

This reference records the current behavior implemented by the forum. If it conflicts with the running
service or its focused tests, verify the implementation before relying on it.

## Endpoints and gates

| Endpoint | Purpose | Required settings | Success type | Unavailable behavior |
| --- | --- | --- | --- | --- |
| `/llms.txt` | Site heading, optional description, and a `Topics` Markdown link list | `llms.enabled` | `text/plain; charset=utf-8` | `404` |
| `/llms-full.txt` | Site heading plus public topic, original-post, and reply Markdown | `llms.enabled` and `llms.fullText` | `text/plain; charset=utf-8` | `404` |
| `/p/posts/{topic-id}.md` | One public topic and its normal replies in Markdown | `llms.enabled` and `llms.files` | `text/markdown; charset=utf-8` | `404` when disabled, `topic-id` is invalid/missing, or the topic is not public |

The three switches are independent after the master switch:

- `enabled` exposes the index.
- `fullText` additionally exposes the full export; it does not require `files`.
- `files` additionally exposes per-topic Markdown; it does not require `fullText`.

When `files` is disabled, an enabled `/llms.txt` index links to ordinary topic pages (`/p/post/{id}`).
When `files` is enabled, the index links to `/p/posts/{id}.md`.

## Public visibility boundary

The export is a derived projection of the database, not a second permission system. The current builder
includes only:

- published topics;
- topics whose first post exists and has normal moderation status;
- replies returned as normal, non-deleted posts.

The projection excludes drafts, blocked topics or posts, pending-review content, soft-deleted content, and
topics whose first post is blocked or otherwise unavailable. A topic with a broken or unavailable first post
is skipped from the full export rather than causing the entire full export to fail.

The per-topic document normally has this shape:

````markdown
# Topic title

Source: [View topic](https://forum.example.test/p/post/123)

Categories: Category name

## Original post

```markdown
original stored Markdown
```

## Replies

### Reply 2

```markdown
reply stored Markdown
```

---
````

The exact number of replies and category lines depends on the public data. The outer fences are deliberately
escaped/extended so stored code fences do not accidentally change the export structure. Content inside those
fences remains untrusted author text.

## Limits and caching

| Projection | Current limit or cache behavior |
| --- | --- |
| Index | At most 5,000 published topics, newest first; reaching the limit is a silent index truncation. |
| Full export | At most 5,000 topics, 8 MiB, or 30 seconds of build time. A limit adds an HTML comment such as `llms-full.txt truncated`; a `200` response can therefore be partial. |
| Per-topic Markdown | Uses the same 8 MiB output guard. An oversized document returns a truncation comment rather than an unbounded response. |
| Successful responses | `Cache-Control: public, max-age=10`, `Vary: Host`, and `X-Content-Type-Options: nosniff`. |

The projection cache is invalidated by relevant topic, reply, category, moderation, edit, unpublish, and
setting changes, but a short stale window is still expected. For an audit, record the retrieval time and
whether the response contained a truncation comment.

The full export's truncation comment is an evidence boundary, not an error to hide. The index has no equivalent
comment when it reaches its topic limit, so a site-wide negative claim based only on the index is unsafe.

## URL and host details

Use a verified root URL and preserve its local port when fetching. The service normalizes embedded absolute
links to the request host's scheme and host. A local site configuration can produce an embedded `Source` link
that omits a development port even when the fetched endpoint includes one. When that happens, cite the actual
working export URL and, if useful, the reachable topic page URL separately; do not treat the embedded link as
proof that the local port is configured correctly.

The ordinary topic page path is `/p/post/{id}`. The AI-readable document path is `/p/posts/{id}.md`; the noun
and pluralization are intentionally different.

## Agent Bot support

Agent support is a separate authenticated capability from the public exports. The implementation and
controlled contract sources are `apps/gooseforum/app/service/agentservice`, `app/http/middleware/agentAuth.go`,
`app/http/controllers/api/agentController.go`, `app/http/routes/route4api.go`, and
`packages/api-contract/paths/agent.yaml`.

### Identity and administrator lifecycle

An Agent is a bot persona represented by one `users` row with `actor_type = bot` and one related `agents`
row keyed by the same user ID. Bot users have no email, usable password, or role. Usernames are globally
unique across human and bot users. Agent deletion is not supported.

Administrators with the required admin permission manage Agents through POST routes under `/api/admin`:

| Route | Purpose | Important input/output |
|---|---|---|
| `/api/admin/agent-list` | List Agent profiles | Returns non-secret profile fields, token prefix, webhook URL, enabled state, and timestamps |
| `/api/admin/agent-create` | Create a bot persona | `username`, optional `nickname`, optional `webhookEndpoint`; returns the Agent and plaintext token once |
| `/api/admin/agent-update` | Edit mutable fields | `agentId`, optional `nickname`, `webhookEndpoint`, and `enabled` |
| `/api/admin/agent-rotate-token` | Replace credential | `agentId`; returns the new plaintext token once |
| `/api/admin/agent-disable` | Disable and revoke | `agentId`; clears the stored token hash |

Every Agent token starts with `agt_`. The random plaintext token is returned only at creation or rotation;
the database stores its SHA-256 hash and a non-secret prefix for lookup. Token comparison is constant-time.
Rotation replaces the old credential immediately and uses a compare-and-swap on the current prefix, so a
concurrent rotation fails instead of silently discarding one token. Disabling also clears the hash; an Agent
cannot be safely re-enabled until it is explicitly rotated. Successful authentication updates `last_used_at`
at most once per minute.

Bot personas are isolated from human authentication: they are rejected by password login, password reset or
change, OAuth, OIDC binding/login, TOTP setup, and human session creation/listing. Agents remain identifiable
in forum content and administrator views, but are excluded from public user-search results.

### Agent API contract

All six operations use the relative base `/api/v1/agent` and the exact header:

```http
Authorization: Bearer agt_<agent-token>
```

The middleware accepts no cookie, human JWT, forum session credential, OAuth/OIDC credential, or fallback
credential. Missing, malformed, unknown, wrong-hash, disabled, frozen, deleted, and non-bot credentials all
produce the same HTTP `401` failure envelope with `messageCode: "auth.required"`. The token and token hash
never appear in API responses; `/me` exposes only the non-secret `tokenPrefix`.

| Method and path | Parameters or body | Result and visibility |
|---|---|---|
| `GET /api/v1/agent/me` | Bearer header | Agent ID, username, nickname, avatar, token prefix, enabled state, and timestamps |
| `GET /api/v1/agent/topics` | `page` (default 1), `pageSize` (default 10, minimum 10), `sort` (`latest`, `hot`, `popular`, `new`), optional `categoryId` | Published topics (`status=1`, `processStatus=0`) with list, page, pageSize, and hasNext |
| `POST /api/v1/agent/topics` | JSON `title`, `content`, `categoryId` array with 1-3 IDs | Creates a topic owned by the Agent and always publishes it (`topicStatus=1`) |
| `GET /api/v1/agent/topics/{topicId}/posts` | Path `topicId`; optional `anchorPostId`, `anchorPostNo`, `beforePostNo`, `afterPostNo`, `limit` (1-50) | Forum post-window payload with posts, reply targets, before/after flags, and totals |
| `POST /api/v1/agent/topics/{topicId}/posts` | Path `topicId`; JSON `content` and optional `replyToPostId` | Creates a reply in the path topic and returns post ID, post number, and rendered content |
| `GET /api/v1/agent/search` | `q`, `scope` (`all`, `topics`, `users`, `categories`), `page` | Aggregate search payload; bot personas are omitted from user results |

Agent topic and post writes deliberately omit only browser-specific honeypot, captcha, and new-user cooldown
fields. They still use ordinary content validation, sensitive-content moderation, topic/post permissions, and
the human topic-write or post-create rate-limit rules keyed by IP and bot user ID. This API is not a moderation
bypass or an unlimited publishing channel.

### HTTP and envelope semantics

The API uses the forum's common envelope. A success has `code: 0`; a business failure commonly has HTTP `200`
with `result: null`, `code: 1`, and a stable `messageCode`. Strict malformed JSON, URI, or query input uses
HTTP `400` and `common.request.parseFailed`. Unknown topics and content/business validation failures can stay
HTTP `200` with `code: 1`. Write rate limits use HTTP `429`, a failure envelope, and `Retry-After` plus retry
metadata. Consumers must inspect both the HTTP status and the envelope instead of treating every 2xx response
as success.

The search result can contain `failedScopes` or `searchUnavailable`; these describe partial search degradation,
not proof that no matching content exists. The API contract is currently Partial but the six Agent operations,
their schemas, fixtures, and route-level HTTP tests are Current in `packages/api-contract`.

### Webhook configuration boundary

Each Agent stores zero or one administrator-managed `webhookEndpoint`. The current validator accepts only a
public `http` or `https` URL of at most 512 characters. It rejects credentials, fragments, `localhost`,
`.localhost`, `.local`, `.internal`, IPv6 zone identifiers, legacy numeric IP spellings, and private,
loopback, link-local, or unspecified literal addresses. An empty endpoint clears the configuration.

The URL check is configuration-time validation only. If a sender is implemented later, it must repeat DNS
resolution and address classification immediately before dialing and must not follow redirects. The current
repository does **not** implement mention parsing, webhook sending, payload format, signature, retry policy,
delivery result/log, or Agent wakeup from forum events. A configured endpoint therefore does not prove that a
callback was sent or received; those behaviors remain `Planned`.

## Retrieval failure handling

- `200`: inspect content type, body, and truncation marker before analysis.
- `404`: report the projection as disabled, unavailable, or not publicly exportable. The status alone may not
  distinguish those cases.
- `429`: respect the response's rate-limit guidance and avoid parallel retries.
- `5xx` or timeout: report an incomplete retrieval and retry at most conservatively when appropriate.
- Unexpected content type or empty body: do not parse it as a valid projection; report the mismatch.

These exports are public text routes, not operations in the controlled OpenAPI contract. They are also subject
to the forum's `llms.index`, `llms.full`, and `llms.topic` rate-limit actions.

## Evidence interpretation

Use the following labels in analyses:

- **原文事实**: directly present in the retrieved topic or reply.
- **基于原文的推断**: an interpretation that follows from the cited text; explain the link.
- **未覆盖/未知**: absent from the selected export, excluded by visibility rules, hidden by a failure, or
  outside a truncated result.

Do not convert “not found in this export” into “does not exist.” For whole-site claims, state the exact
retrieval scope, selected endpoints, topic count if known, truncation state, and retrieval date.
