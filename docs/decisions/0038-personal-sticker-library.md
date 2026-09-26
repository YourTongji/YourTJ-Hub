# Personal sticker libraries with stable shared assets

## Status
Accepted
Class: feature

## Context and Problem Statement

Members need to upload their own stickers and collect stickers shared in conversations or posts.
The official-only directory in [0030](0030-global-sticker-library.md) cannot represent a private,
ordered collection. Removing an item from a collection must not break another person's messages.
Stickers must also remain distinct from ordinary image attachments in the native app.

## Decision Drivers

- Preserve existing `[:sticker:name:]` content and official packs.
- Keep personal collections private while allowing recipients to render and collect shared content.
- Retain historical content when a member removes a collection entry or closes an account.
- Reuse authenticated upload ownership, image validation and the existing file usage lifecycle.
- Bound collection size, permanent storage and batch work; preserve PostgreSQL support.

## Considered Options

- Immutable shared sticker assets with private, ordered membership records.
- Put every personal upload in the global official directory.
- Store image URLs directly in a device-only collection.
- Delete the underlying asset whenever its uploader removes it from their library.

## Decision Outcome

Personal uploads create immutable assets with cryptographically random token names. Official
stickers retain administrator-controlled names and packs. The public directory enumerates only
official stickers; a bounded resolver accepts explicit token names, so a recipient can render and
collect a shared personal sticker without enumerating other people's collections. Knowing the
personal token or its shared file URL grants access to the image; this is shared content, not a
confidential file transport. User-specific display labels remain on private membership records.

A member's library contains at most 200 entries, ordered independently of the official directory.
Uploading reuses the existing authenticated image upload, requires the caller's own image file,
and accepts at most 4 MiB per image. Each account may create at most 1000 personal assets because
those assets are retained for historical rendering. Resolving or ordering accepts at most 200 names.
Disabled official items retain membership but cannot be resolved or inserted; permanent official
deletion removes its memberships. Membership writes serialize per account; collection is idempotent and reordering validates the
complete current membership. Removing a membership does not release the asset's `file_usage`.
Account closure removes memberships and prevents in-flight library writes while retaining shared
assets. Administrative official-library operations cannot mutate personal assets.

Native composition uses one Recent / My / Official picker in messages, replies and publishing.
Selection replaces text at the current caret and never sends automatically. Management supports
upload, private labels, ordering and removal; long-pressing a received sticker offers collection.
A dedicated inline renderer consumes taps without opening the ordinary image gallery. Normal
attachments keep their existing gallery interaction. Unknown or unavailable tokens remain readable
and resolvable content can be retried after a network failure.

This supersedes [0030](0030-global-sticker-library.md) for product scope, public lookup and personal
asset lifecycle. Its official pack licensing, server rendering exclusions, single-binary deployment,
standard image storage and administrator CRUD remain the underlying model.

## Pros and Cons of the Options

- Shared assets plus memberships preserve history, private ordering and cross-device collections;
  retained assets consume storage even after removal, so lifetime upload quotas are explicit.
- A global directory is simpler to query but exposes private collection intent and mixes personal
  uploads with curated official content.
- Device-only URLs cannot reliably synchronize, validate ownership or distinguish stickers from
  attachments, and lose the library when local storage is cleared.
- Deletion on removal reclaims storage promptly but destroys other people's shared content.

## Links

- [Official sticker model](0030-global-sticker-library.md)
- [Personal library service](../../apps/gooseforum/app/service/stickerservice/library.go)
- [Mobile behavior](../product/mobile-experience.md#sticker-library)
- [Contract center](../../packages/api-contract/openapi.yaml)
