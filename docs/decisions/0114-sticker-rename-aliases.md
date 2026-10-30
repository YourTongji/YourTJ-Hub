# Keep renamed official sticker names as aliases

## Status

Accepted
Class: feature

## Context and Problem Statement

Content stores official stickers as the plain-text token `[:sticker:name:]` and expands it only when
rendering ([0030](0030-global-sticker-library.md)). An administrator who renames an official sticker
therefore breaks every post, reply and message that used the old name: the token no longer resolves
and readers see raw text. Issue #1428 asks that renaming keep existing content renderable.

## Decision Drivers

- Already published content keeps rendering after a rename, on Web, server HTML and Flutter.
- Keep the plain-text token storage; no rewrite of historical content.
- A previous name must not later point at a different sticker.
- New content uses the current name; administrators can see what an old name still serves.

## Considered Options

1. Record every previous name in a `sticker_aliases` table (unique name → sticker id) and resolve
   unknown names through it.
2. Rewrite stored content (posts, revisions, messages, drafts) to the new name on rename.
3. Make official sticker names immutable after creation.

## Decision Outcome

Choose option 1. Renaming an official sticker records the old name as an alias of the same sticker in
the same transaction. `Resolve` and `ResolveURLs` look up current names first and fall back to
enabled stickers reached through an alias, so every rendering path shares the behaviour. An alias
reserves its name: creating or renaming another sticker to it returns `admin.sticker.nameReserved`.
Renaming a sticker back to one of its own aliases releases that alias. Deleting a sticker removes its
aliases, so its old names stop resolving together with its current name. The admin list shows each
sticker's aliases, oldest first, and the edit dialog states that the previous name keeps working.
Pickers, directories and recent use only offer the current name.

## Pros and Cons of the Options

### alias table (selected)

- Good: no content rewrite; history, revisions and offline caches keep resolving; one lookup path.
- Bad: names stay reserved indefinitely, so the namespace only grows; a rename cannot free a name for
  another sticker without deleting the original sticker.

### rewrite stored content

- Good: no lasting indirection.
- Bad: touches post revisions, chat history and drafts across many tables; edits history, invalidates
  caches and cannot reach content already copied to clients or external sites.

### immutable names

- Good: simplest; no ambiguity.
- Bad: blocks a real administration need (fixing typos, consistent naming) that the issue requests.

## Links

- Issue #1428
- [0030: global sticker library](0030-global-sticker-library.md)
- [0115: sticker market subscriptions](0115-sticker-market-subscriptions.md)
- [Forum product: links, Markdown and stickers](../product/forum.md#links-markdown-and-stickers)
