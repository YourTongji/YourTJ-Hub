# Unreferenced upload reads and reclamation

## Status

Accepted
Class: bug-fix

## Context and Problem Statement

`/file/img/*` authorizes a read from the content references recorded in `file_usages`. A user upload
exists at a public URL before any post, reply or draft references it. Because files with no usage row
at all were treated as pre-migration legacy files and served publicly, an upload that never entered
content stayed readable forever, was cacheable, and was never reclaimed. Combined with an unrestricted
`Referer`, that made the image service a free image host that bypassed content moderation
([issue #1365](https://github.com/YourTongji/YourTJ-Hub/issues/1365)).

Files migrated from the previous platform have no usage rows: their markdown only stores URLs. Failing
every unreferenced file closed would 404 images inside old posts, so the read path needs a signal that
separates "a new upload nobody referenced yet" from "a legacy file we never tracked".

## Decision Drivers

- Close the moderation bypass: unreferenced uploads must not be anonymously readable.
- Keep images embedded in pre-migration posts readable.
- Authorize with server-side facts that cannot be forged by the client.
- Work for in-editor previews: embedded image requests carry no `Authorization` header on the web, so
  authorization must also accept the same-origin session cookie.
- Allow a normal edit-then-publish interval before an upload is treated as abandoned.
- Bound storage and origin bandwidth instead of carrying every abandoned upload forever.

## Considered Options

- Edge `Referer` allowlist plus a shorter `Cache-Control` for image responses.
- `upload_owner` ownership row as the authoritative signal: a tracked upload with no public content
  reference is readable only by its uploader, and is reclaimed after a TTL.
- Fail every file without a content reference closed, including files with no usage rows.

## Decision Outcome

Use the ownership row as the authority. Every user upload path records `upload_owner` (S3 direct
upload `complete`, and the proxied multipart upload), and `GET /file/img/*` fails closed for a file
that is tracked but not publicly referenced:

- a pending content reference authorizes the existing review previews;
- an upload with ownership but no content reference is served only to its uploader's session
  (`Cache-Control: private, no-store`), anonymous and other users get 404;
- a file with no usage row at all keeps the previous public behavior, which is what keeps images inside
  pre-migration posts working; managed assets are untouched by this gate because their own `avatar`,
  `sticker`, `decoration_asset` and `badge_asset` rows are ACTIVE.

A daily job removes ready uploads that still carry only their ownership row after
`OrphanUploadTTL` (7 days), re-checking content references immediately before deleting bytes, image
variants and the ownership row. `file_usages` lives in the main database while `file_data` lives in the
separate file database, so the job first drops ownership rows whose file row no longer exists;
otherwise a restored or reset file database could fill the candidate window with stale rows and the
job would silently stop reclaiming anything.

A `Referer` allowlist is not a control: it is forgeable and must exempt empty referrers, so it only
sheds load and is not a substitute for this gate; this decision does not add one. Image responses
currently send `Cache-Control: no-store` for every read (from the same hardening work), so no image is
cached publicly; restoring a public cache lifetime for content-referenced images is a separate
bandwidth trade-off that needs its own decision.

## Pros and Cons of the Options

- **Referer allowlist plus shorter cache.** Cheap to add at the edge, but a client-supplied header
  cannot authorize a read, empty referrers must stay allowed, and a shortened public cache still
  serves every copy that already left the origin. It reduces load at best.
- **Ownership-row gate and TTL reclamation.** Uses a row the upload path already writes, needs no
  deployment change, cannot be forged, and keeps migrated images readable. It gives up the bandwidth
  savings a referer check would provide, and a publish racing the TTL boundary can lose its image
  (accepted and marked in code); an upload abandoned in an editor older than the TTL is gone.
- **Fail every unreferenced file closed.** The strictest rule, but it breaks images inside
  pre-migration posts, which carry no usage row at all, without any data to tell them apart.

## Links

- [Issue #1365](https://github.com/YourTongji/YourTJ-Hub/issues/1365)
- [PR #1366](https://github.com/YourTongji/YourTJ-Hub/pull/1366) introduced the read gate this record
  codifies
- [Object storage operations](../operations/object-storage.md) documents the cleanup and the legacy
  boundary
- [Forum product](../product/forum.md) documents the user-visible read behavior
- [0021](0021-deletion-final-state-data-retention.md) keeps deleted-content bytes as the final retention
  state; this record only reclaims uploads that never entered content
