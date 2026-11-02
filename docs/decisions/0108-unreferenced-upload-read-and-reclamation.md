# Unreferenced upload reads and reclamation

## Status

Accepted
Class: bug-fix

## Context and Problem Statement

`/file/img/*` authorizes a read from the content references recorded in `file_usages`. A user upload
exists at a public URL before any post, reply or draft references it. Because files with no usage row
at all were treated as pre-migration legacy files and served publicly, an upload that never entered
content stayed readable forever and was never reclaimed. Combined with an unrestricted
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
  `sticker`, `decoration_asset` and `badge_asset` rows are public while live (personal stickers are
  `PENDING` until review approves them).

Content that stores only a URL also registers its own `ACTIVE` reference instead of relying on the
upload path: chat image/voice messages write `chat_message`/`chat_media` rows inside the send
transaction, and a profile cover writes a `user`/`cover` row on every writer. Both are restricted to
sender/user-owned `ready` files, mirroring the online ownership check, because an `ACTIVE` non-owner row
is a permanent anonymous read grant and a permanent reclamation pin; forwarded chat copies keep no
reference of their own and rely on the original message's, which is never deleted. Both were affected by the
same defect — invisible to readers after the gate landed and reclaimable by the job — and both are now
publicly readable and excluded from reclamation. A replaced cover drops its old row, so the previous
file becomes reclaimable, while saving the same cover URL again keeps the existing reference (a
co-manager editing another bot field must not revoke a cover somebody else uploaded). Chat media is
publicly readable by its URL and has no revocation path: messages are never deleted, so the reference
stays for the life of the file. Existing chat media and covers predate this decision, so app data
migration v37 backfills their references from `messages.content` and `users.profile_cover_url` before
the reclamation job can treat those files as orphans. The same migration fills missing references for
tracked uploads in topic galleries, post bodies and retained revisions, including configured
`publicUrlPrefix` URLs. It checks uploader ownership and content visibility: deleted content and
private revisions retain bytes without becoming public. Untracked legacy files keep their existing
read behavior. The backfill is idempotent and re-runnable.

A daily job reclaims ready uploads that still carry only their ownership row after
`OrphanUploadTTL` (7 days). Reference creation and reclamation serialize on that ownership row in the
main database. The job locks the row, re-checks all non-owner references and commits `RECLAIMING` before
deleting bytes or image variants from the separate file database/object store. A publisher that gets
the lock first keeps the file alive; one that arrives after the claim is rejected and its content
transaction rolls back. Topic/reply writes and attachment references commit together, including
moderation candidates. Resolving a URL before the claim does not bypass this check.

Successful deletion marks the ownership row `PURGED` and retains it as a tombstone. It denies late
references and prevents a reclaimed upload from falling back to legacy public access. Storage errors
leave the row `RECLAIMING` for retry; database errors preserve the bytes. Ownership rows whose file
metadata is already missing are also claimed and marked `PURGED`, so they do not fill the bounded
candidate window forever. This keeps a small main-database record for each reclaimed upload instead of
retaining its bytes. An expired attachment must be uploaded again before the editor can save it.

A `Referer` allowlist is not a control: it is forgeable and must exempt empty referrers, so it only
sheds load and is not a substitute for this gate; this decision does not add one. With `publicUrlPrefix`
configured, browsers read the object storage directly, so this read gate (and its cache headers) does
not apply to those URLs and the bucket's own access controls govern them. Image responses currently
send `Cache-Control: no-store` for every read (from the same hardening work), so no image is cached
publicly; restoring a public cache lifetime for content-referenced images is a separate bandwidth
trade-off that needs its own decision.

## Pros and Cons of the Options

- **Referer allowlist plus shorter cache.** Cheap to add at the edge, but a client-supplied header
  cannot authorize a read, empty referrers must stay allowed, and a shortened public cache still
  serves every copy that already left the origin. It reduces load at best.
- **Ownership-row gate and TTL reclamation.** Uses a row the upload path already writes, needs no
  deployment change, cannot be forged, and keeps migrated images readable. It gives up the bandwidth
  savings a referer check would provide and retains small tombstones. A publish racing reclamation
  either pins the file or fails atomically; an upload abandoned in an editor older than the TTL must
  be uploaded again.
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
