# Decide upload image type from file header bytes, not the declared extension

## Status

Accepted
Class: bug-fix

## Context and Problem Statement

Every image upload gate (multipart `POST /file/img-upload`, admin
`POST /api/admin/img-upload`, `POST /api/upload-avatar`, and the direct-upload
`/file/img-upload/complete`, plus the sticker-import and decoration-asset
services) sniffed the bytes and then required the sniffed/decoded type to equal
the type implied by the client-declared extension. A correctly decodable image
whose declared extension was wrong — `foo.png` containing JPEG bytes, a browser
reporting a swapped MIME, a `.jpeg`/`.jpg` spelling difference — was rejected with
`upload.image.invalidContent`. Issue #1393 reports this as a false rejection: the
bytes are the only trustworthy signal, and the declared extension is client input.

## Decision Drivers

- The rejections are wrong for legitimate files; the user cannot act on
  "content does not match its extension" when both the format and the bytes are
  fine.
- The security property that must survive is "no non-image bytes are ever
  stored", not "the extension equals the decoded format". Extension checks stay as
  a pre-filter so dangerous names are refused before the body is read.
- One spelling of the rule across all upload exits: the same bytes must give the
  same stable code and the same stored result everywhere.
- The stored object name decides the served `Content-Type` (issue #408 read
  path), so a name whose extension contradicts the bytes would serve the wrong
  type.
- No new dependency and no change to the single-binary deployment shape.

## Considered Options

1. Keep extension/type equality and ask clients to fix their declared extension.
2. Treat the bytes as authoritative: detect the type with `imagepolicy.
   DetectContentType`, accept any allowed detected type, and normalise the stored
   name/type to the detected type when it conflicts with the declaration.
3. Bytes authoritative, but keep the client-declared name in storage and rely on
   `assert_type` for the served `Content-Type`.

## Decision Outcome

Choose option 2. `imagepolicy.DetectContentType` is the single byte-level entry
point: it reads a bounded 512KiB header, reports `ErrEmptyImageContent` for zero
bytes, `ErrUnsupportedImageFormat` for known-but-undecodable formats
(HEIC/HEIF/AVIF/TIFF), and `ErrInvalidImageContent` when the header does not
decode or the sniffed signature disagrees with the decoded format; otherwise it
returns the canonical MIME (`image/jpeg|png|gif|webp|bmp`). `ExtForContentType`
maps that MIME to the canonical storage extension (`image/jpeg` → `.jpg`).

Each upload exit keeps the same order: declared filename and size, then the
declared extension against the effective allowlist (so `evil.svg` is still
refused before the body is read), then the detected type against the same
allowlist (`upload.extension.unsupported` when it is not allowed), and only then
storage. `upload.image.invalidContent` now means exactly "the bytes are not a
usable image"; an extension conflict is no longer an error.

Storage naming follows the detected type. Multipart, admin and avatar uploads
use their generated UUID key with the canonical extension. Every direct upload
publishes a fresh UUID key in the same directory: a type conflict uses the
canonical extension, while a matching declaration keeps equivalent spellings
such as `.jpeg`. The response is the authority for the final name.

For a pending upload, completion reads a snapshot bounded by the declared size plus one
byte and requires its length to equal the declared size. Header validation and
publication use those same bytes. Even a matching type needs a new key because
the client's original presigned POST can still overwrite the temporary key.

Before writing the copy, reserve a `copying` file metadata row. Publication is a
single file-database transaction: consume the original `pending` row into
`cleanup` and publish the same owner's `copying` row as `ready`. A source can be
consumed only once. Each attempt owns a different key, so a losing attempt cannot
overwrite or delete the winner's object. The copy keeps the source creation time
when published. `copying` and `cleanup` rows are excluded from user upload counts,
so normalization does not charge a second upload, including across midnight.

The publication transaction also stores the original init name in the final
row's indexed `direct_upload_source_name` column. The owner can repeat complete
with either name and recover the same result after a lost response, even after
the temporary row is cleaned up. Only complete uses this mapping; ordinary image
reads do not resolve the writable source name. Deleting the final file removes
the result. Existing ready rows keep the column's empty default and do not gain
an inferred mapping.

After publication, attempt to delete the old object but retain its cleanup row
until the upload expiry cutoff: an already issued presigned POST can still write
the old key again. The worker retries failed deletes and late writes. Any leftover copy remains tracked by its copying row. The existing two-hour cleanup worker scans pending, copying and cleanup
rows, deleting bytes before metadata; ready winners are excluded. A process crash
between copy and publication therefore retains a reclaimable record. Provider
errors never justify dropping the last cleanup record. File ready publication and
main-database upload-owner registration retain their existing separate-write
boundary, documented in the object-storage runbook.

Reads keep deriving `Content-Type` from the stored object name's extension,
without sniffing: after this decision a stored name's extension is the detected
type by construction, so the header and the bytes cannot disagree.

## Pros and Cons of the Options

### keep extension/type equality (rejected)

- Good: nothing changes; the stored name always matches its bytes.
- Bad: legitimate uploads keep failing with an error the user cannot fix, over a
  declaration the client controls anyway.

### bytes authoritative, normalise the stored name (selected)

- Good: one rule for all exits, expressed once in `imagepolicy`; the stored name
  still fully determines the served type; dangerous extensions and non-image
  bytes stay rejected.
- Good: direct-upload recovery does not depend on receiving the first success
  response, and replaying the original upload policy cannot change the published
  bytes.
- Bad: publishing a pending direct upload reads the full size-bounded snapshot
  and writes a copy, increasing bandwidth and concurrent memory use. Each attempt
  needs a tracked copy row and a file-db transaction. Recovery adds an indexed
  column through the file database's AutoMigrate; no new table is needed, but
  historical ready rows have no original-name mapping. Failed operations can
  retain temporary objects until the cleanup worker retries.

### bytes authoritative, keep the declared name (rejected)

- Good: no rename, no extra transfer.
- Bad: the served `Content-Type` would come from a name that contradicts the
  bytes, so a `.png`-named JPEG would be served as `image/png` with `nosniff`, and
  the stored name would stop being a reliable type statement.

## Links

- Related issues: #1393 (this decision), #408 (extension allowlist and bounded
  header validation), #966 (per-failure stable codes)
- [Object storage: direct post-image upload](../operations/object-storage.md)
- [Contract: files and uploads](../../packages/api-contract/paths/direct-image-upload.yaml)
