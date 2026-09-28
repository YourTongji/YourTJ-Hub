# Nested private-message history preserves copied cards

## Status
Accepted
Class: feature

## Context and Problem Statement

Flattening an existing merged record removes the selected history card and its sender context.
Readers cannot distinguish messages from the selected conversation from messages inside a copied
record. Original authors' avatar snapshots must remain associated with their own messages.

## Decision Drivers

- Preserve the selected message's presentation, including independently openable history cards.
- Keep copies independent of source-conversation permissions and mutable profiles.
- Bound parsing, rendering, moderation and storage across the complete copied tree.
- Keep legacy snapshots and readable text fallback compatible.

## Considered Options

- Flatten every selected history into ordinary messages.
- Retain a bounded tree of copied history cards.
- Link to the original history instead of copying it.

## Decision Outcome

This supersedes [0043](0043-private-message-forward-snapshots.md). Its server-built immutable-copy,
membership, interaction permission, moderation, idempotency, per-recipient transaction, retry queue
and single-binary storage decisions are retained. The snapshot remains version 1 with a readable
text fallback. Existing flat snapshots remain valid.

Merged forwarding stores each selected message as one entry. A selected type-4 history retains its
own sender name, avatar URL and creation time, and an optional `forwarded` field contains the copied
child bundle. Type 4 requires this child; ordinary entries cannot carry one. No source message IDs,
conversation IDs, account identity handles or private contact notes are exposed. Names remain
display-only. Inner avatars are copied unchanged; absent avatars use placeholders, never a nickname
lookup. Public asset lifecycle and deletion semantics remain unchanged.

The complete snapshot is limited to four bundle levels, fifty entries including history cards, and
64 KiB encoded bytes. Both parsing and encoding enforce these bounds. Moderation and text fallback
traverse all child messages. Oversized copies fail atomically. Individual forwarding copies a valid
existing card without adding a bundle level. Flutter and Web render child entries as cards with their
own detail views and allow returning to the enclosing history.

## Pros and Cons of the Options

- Flattening is simpler but removes the selected card's boundaries and sender context; rejected.
- A bounded copy tree preserves that context and supports nested detail views at the cost of
  recursive validation and explicit depth/count limits; selected.
- Source links save storage but introduce cross-conversation access and lifetime coupling; rejected.

## Links

- [Previous snapshot decision](0043-private-message-forward-snapshots.md)
- [Mobile experience](../product/mobile-experience.md)
- [Contract and data architecture](../architecture/contracts-and-data.md)
- [Forwarding contract](../../packages/api-contract/paths/forum-chat-forward.yaml)
