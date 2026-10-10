# Personal sticker uploads require review before use

## Status

Accepted
Class: feature

## Context and Problem Statement

Personal sticker uploads become shared assets when their tokens are sent in messages or posts.
The existing upload path makes each new asset immediately selectable and resolvable. Issue #1071
requires asynchronous review while preserving old assets and message history.

## Decision Drivers

- Keep unreviewed images out of token resolution, pickers and message rendering.
- Reuse the existing durable task queue and AI image moderation service.
- Preserve old assets without retroactive review and retain rejected assets for sent history.
- Keep pending user-upload images outside the forum moderator queue.
- Give owners a clear status and allow site administrators to close AI review fallbacks.

## Considered Options

- Allow immediate use and disable an asset after a rejecting result.
- Block the upload request until synchronous AI moderation finishes.
- Store new uploads as pending, evaluate them in a worker and route undecided assets to site administrators.

## Decision Outcome

New personal sticker assets start as `pending` and disabled. A transaction creates the asset,
private-library membership, file reference and review task together. AI `allow` enables the asset;
`block` marks it rejected and leaves it disabled; `review`, unavailable moderation, and shadow mode
leave it pending. Existing assets default to approved so migration does not change their availability.

Only approved and enabled assets can be resolved or inserted. Owners can see pending or rejected
status in their library and can still remove or reorder those entries. Site administrators can
approve or reject pending personal uploads in the existing review queue. Forum moderators cannot
read or decide this queue. Manual decisions update the AI moderation record when one exists and are
always written to the existing operation log.

Conversation previews replace every sticker token, including unknown and unavailable tokens, with
the localized `[Animated sticker]` label. The stored message body and sent token remain unchanged.

## Pros and Cons of the Options

- Post-review disablement makes uploads immediately usable but exposes unreviewed images to other
  people and allows them to be collected before a block decision.
- Synchronous review keeps assets out of use but makes upload latency and AI availability part of
  the interactive request, and cannot close human-review outcomes.
- Asynchronous pre-review keeps uploads responsive and prevents premature use; undecided assets need
  a separate site-admin queue and may remain unavailable until someone reviews them.

## Links

- [Issue #1071](https://github.com/YourTongji/YourTJ-Hub/issues/1071)
- [Personal sticker library](0038-personal-sticker-library.md)
- [AI image and text moderation](0056-ai-image-text-moderation.md)
- [Versioned background moderation](0061-versioned-background-moderation.md)
