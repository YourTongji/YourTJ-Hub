# Private message forwarding uses bounded immutable snapshots

## Status
Superseded by [0044](0044-nested-private-message-history.md)
Class: feature

## Context and Problem Statement

Mobile chat needs individual and merged forwarding to several recipients. A recipient must be able
to read selected content without gaining access to its source conversation. Retries after a lost
acknowledgement must not deliver duplicate messages or inflate unread counts.

## Decision Drivers

- Source membership and every selected message are checked by the server.
- Forwarding remains subject to recipient blocks, writable accounts, sensitive-word checks and chat quotas.
- Existing clients can read a plain-text fallback; the single-binary deployment and database columns stay intact.
- Delivery has an explicit acknowledgement boundary for each recipient.

## Considered Options

- Store a bounded server-built copy in the existing message content field.
- Store references to source messages and authorize recipients to resolve them.
- Let clients compose arbitrary structured chat-history payloads through ordinary send.

## Decision Outcome

`POST /api/forum/chat/forward` accepts a source conversation, up to 50 unique message IDs, a recipient,
mode and client operation identity. It sorts the source IDs into conversation order and commits all
outgoing messages for one recipient in one transaction. Individual mode copies at most 10 message bodies (the default per-user send window);
merged mode stores a versioned JSON snapshot as message type 4. Nested merged records are flattened;
the snapshot has at most 50 entries and 64 KiB of encoded content. Ordinary send cannot create type 4.

A snapshot contains display names, optional public avatar URLs, original creation times, message types
and content. It contains no source conversation/message IDs or private contact notes. Avatars are
display-only and do not link to source profiles; older snapshots use circular placeholders. The URL is
copied, while image availability and later file replacement follow the existing public asset lifecycle. Names are display information, not
verified attribution. Copies have their own message lifetime; removing or changing the source does
not retract delivered copies. Reporting a forwarded message attributes the report to its forwarder
and discloses only that message's bounded readable content through the existing administrator workflow.
Sticker tokens still follow the sticker library's availability rules.

A canonical actor/source IDs/recipient/mode/client identity determines stable outgoing identities.
Retries acknowledge the stored copy without rebuilding it from current display names. Individual
forwarding spends the normal send quota once per outgoing message; merged forwarding spends once.
Clients submit recipients separately, show each outcome and retry only unacknowledged recipients
with the same identity. Mobile selection is capped at ten recipients and the unfinished queue lives
in the current account session's memory. Process restart loses that queue; starting a new operation
can duplicate a previously unacknowledged delivery. The abandon confirmation explains this boundary.

Reads include both a readable `content` fallback and an optional structured `forwarded` bundle.
Mobile and web render a compact card and an accessible avatar-and-bubble detail view. Mobile shares
the live conversation row and message renderer, including sticker and selectable-text behavior. Mobile back navigation exits
selection without losing the composer draft; leaving a partially completed target page keeps a
resume entry in the source conversation. Account changes dispose queued work and hide old detail pages.

## Pros and Cons of the Options

- Server-built copies preserve privacy boundaries and stable retry results, but consume storage and
  cannot track later source edits or deletion. Bounded snapshots limit this cost.
- Source references save duplication but create a new cross-conversation permission/lifetime model;
  rejected because recipients need only the selected copy.
- Client-built records are simple to send but cannot validate source membership or distinguish the
  server's snapshot format; rejected. Plain text remains user-authored and is not proof of authorship.

## Links

- [Mobile experience](../product/mobile-experience.md)
- [Contract and data architecture](../architecture/contracts-and-data.md)
- [Private message reports](0040-user-blocks-and-private-message-reports.md)
- [Forwarding contract](../../packages/api-contract/paths/forum-chat-forward.yaml)
