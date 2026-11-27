# Tiered public media caching with conditional reads

## Status

Accepted
Class: architecture

## Context and Problem Statement

`/file/img/*` served every readable image with `Cache-Control: no-store` and no validator, so Web and
the app downloaded and decoded unchanged avatars, stickers and post images again on every view. The
static and page gzip middleware also compressed already-compressed formats (`.webp`, `.avif`, `.ico`,
`.woff2`), adding `Vary: Accept-Encoding`; the Flutter media repository refuses to persist a response
with that header, so preset avatars never entered its disk cache. Avatars had a five-minute public
window ([0125](0125-public-avatar-cache.md)) but still paid a full transfer after expiry. Private
previews, pending content and withdrawn references must stay uncacheable and must not be reachable
through a validator.

## Decision Drivers

- Repeated views of unchanged public media should not transfer the body again.
- Withdrawal and moderation must bound how long a cached copy can outlive the current reference.
- Private, pending, denied and error responses must never gain a shared-cache or validator path.
- One policy for Web, the app and the reverse proxy; the single binary keeps serving `/file/img`.
- Already-compressed formats must not be gzip-encoded or vary on `Accept-Encoding`.

## Considered Options

1. Keep `no-store` everywhere and add client-side caching only in the app.
2. Tiered public freshness plus a strong `ETag` and `If-None-Match`/`304` revalidation; keep private
   paths `no-store`.
3. Long immutable caching for every public image, keyed by filename.

## Decision Outcome

Choose option 2.

- **Tiers.** Active avatars, active official stickers and `stickers/market/` assets use
  `public, max-age=86400` because their filenames are unique per upload and the bytes never change;
  one day is the accepted bound for a withdrawn avatar or sticker remaining in a fresh browser or
  shared cache. Other public content images (published topic/reply/chat/cover and legacy files
  without usage rows) use `public, max-age=300`, bounding withdrawal reuse to five minutes. Private
  previews, pending content, denials, errors and non-image attachments keep `private, no-store` /
  `no-store` and carry no `ETag`.
- **Validator.** A strong `ETag` is derived from the stored row's `id`, `file_size` and `updated_at`.
  A matching `If-None-Match` (weak comparison, comma lists and `*`) returns `304 Not Modified` with
  the same `Cache-Control`/`ETag` before any BLOB or object-storage read. Authorization is resolved
  before the validator, so a withdrawn reference answers `404` even when the client presents the old
  `ETag`.
- **Compression.** The static and view gzip middleware excludes `.png`, `.gif`, `.jpeg`, `.jpg`,
  `.webp`, `.avif`, `.ico` and `.woff2`, so these responses carry no `Content-Encoding` and no
  `Vary: Accept-Encoding`; text assets keep gzip.
- **Boundary.** The reverse proxy must preserve `Cache-Control`/`ETag`; a public bucket or
  `publicUrlPrefix` bypasses this route and needs its own controls. A shared cache that follows
  `public` extends the withdrawal window to every client it serves.

## Pros and Cons of the Options

- Option 1 preserves immediate reauthorization but leaves repeated transfers and conflicts with
  client-side HTTP caching; the app would need a second retention policy.
- Option 2 removes repeat body transfers through revalidation, keeps ordinary user-content withdrawal
  bounded to five minutes, and leaves private paths untouched. Avatars and stickers trade a longer
  withdrawal window for a full day without repeat validation; this supersedes the five-minute avatar
  window in [0125](0125-public-avatar-cache.md).
- Option 3 maximizes reuse but lets a withdrawn or moderated image stay visible for the cache
  lifetime in browsers, shared caches and CDNs; filename immutability does not cover withdrawal.

## Links

- [PR #1710](https://github.com/YourTongji/YourTJ-Hub/pull/1710)
- [Superseded avatar cache decision](0125-public-avatar-cache.md)
- [Sticker market caching](0115-sticker-market-subscriptions.md)
- [Object storage: image read authorization and cache policy](../operations/object-storage.md#image-read-authorization-and-cache-policy)
- [Public avatar behavior](../product/forum.md#public-avatar-caching)
