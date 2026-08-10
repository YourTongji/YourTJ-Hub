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
