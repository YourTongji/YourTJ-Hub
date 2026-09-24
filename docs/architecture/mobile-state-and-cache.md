# Mobile state, events and cache boundaries

> Doc type: architecture specification
>
> Status: Draft
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-24

This specification defines `Planned` contracts and explicitly provisional `Decision needed` policies for the
[mobile interaction standard](../product/mobile-design-system.md). Existing behavior is recorded in
[mobile experience](../product/mobile-experience.md) and [campus](../product/campus.md). The campus snapshot policy is `Current` under accepted decision 0035; foreground event contracts
remain `Planned` until their implementation is integrated.

## State ownership

Separate authoritative server entities, page traversal state, recoverable user work and disposable
cache. A topic's interaction state is shared by numeric topic ID within an account scope; the ordered
topic IDs, cursor, scroll anchor, filter and request generation belong to each stream. Changing one
stream cannot replace another stream's response. Entities remain subject to server visibility and
permissions; a cached card does not authorize an action.

Every private asynchronous operation captures API origin, numeric account ID and a session epoch.
Responses, events and delayed storage writes may apply only to their captured scope. Private campus
operations also capture the binding version. Logout/account changes synchronously advance the active
session epoch first, then dispose event connections and cancel work, then install the replacement
account scope. A callback from a previous session cannot revive its
cache, submit its draft or mutate the next account's counters.

Refresh is a new data revision, not a request to empty the view. Pagination is single-flight per
stream, deduplicates stable item IDs and requires cursor/item progress. Query, sort and filter changes
start a new generation. Mutation results cannot be overwritten by earlier reads. Optimistic like,
bookmark and watch changes retain their own previous value and revision for conditional rollback.

## User work

New topic drafts use durable local IDs independent of content type and of server topic IDs. A server
draft ID and an edit target are metadata, not substitutes for that local identity. Existing recovery
slots must migrate without overwriting distinct drafts. Reply drafts identify their topic and reply
target; message drafts identify their conversation (or validated peer before creation).

Saving is ordered per draft. A delete/acknowledgement is fenced after earlier writes; a subsequent
newer revision survives. Local write failures remain visible and retain the in-memory document.
Switching accounts hides another account's drafts without adopting them. Explicit account erasure
removes its local work according to the account lifecycle. Ordinary cache clearing never deletes it.

The media queue owns temporary local files, upload state and remote attachment references. It does
not own the editor document or submit content. Attachment ordering is independent of completion
order. Cancellation and retry affect the selected attachment, and publishing validates queue state.
Restart recovery requires app-owned copies of picker files, with a bounded cleanup lifecycle; a
temporary picker URI alone is not a durable attachment.

## Foreground events and visible reads

`Planned`: a single foreground event connection per active forum session delivers authenticated,
user-scoped invalidation hints. REST remains authoritative for message content, notification rows and
unread counts. The transport selection and operational tradeoffs require a separate decision record
with implementation. Normal foreground delivery must not depend on fixed-interval polling.

The server publishes only after a successful business transaction. Events identify affected domains
and conversation IDs, not private message text or school information. Connections are bounded;
overflow or lost continuity requires an explicit resynchronization, not silent event loss. Reconnect
and foreground restoration reconcile counts and fetch cursor deltas once. Retries use jitter/backoff;
backgrounding closes the foreground connection. An unavailable stream has visible degraded state
and bounded fallback behavior. Credentials must not be placed in query strings or event payloads.
Revoked sessions lose access to the stream, including already-open connections.

`Planned`: read acknowledgement accepts a bounded set of incoming message IDs actually visible in
the foreground conversation. It is idempotent, validates membership and message ownership, and
updates per-message read state plus conversation unread counts in one transaction. It does not
infer visibility from the largest ID, latest download cursor, reaching the route, or opening a push.
The existing legacy mark-all endpoint retains its contract for older clients; the new client must not
send ignored extra parameters to an old server and assume precise read semantics.

New message delivery and read writes share a concurrency boundary so interleavings cannot lose unread
increments. Sending/reading failures cannot emit success events. Batch-read retries merge only
observed IDs in the same account/conversation. Clearing a badge optimistically must be reversible.
Scroll position is independent of delivery: while reading history, arrivals show a new-message
control and do not jump the viewport or mark offscreen messages read.

## Cache policy

`Planned`: reuse the existing local database and media-cache boundaries. Every snapshot carries schema
version, scope, fetched-at time, provenance and available data. A cache envelope must distinguish
unknown/absent data from a valid empty result. Read failures expose freshness; they do not fabricate a
successful fetch time. Hard expiry and user-visible staleness are separate policies.

| Data | Persistence and refresh contract |
|---|---|
| Public forum pages, course metadata and Wiki | Bounded disposable cache; conditional refresh where supported; explicit stale/offline presentation. Private fields retain account scope. |
| Conversation and notification cache | Private account scope; reconcile on foreground events; cache retrieval never acknowledges reads. |
| Campus name, calendar, official timetable | `Current`: private allowlisted device snapshot, additionally binding-scoped; complete successful refreshes replace it atomically; existing snapshots refresh by user action. |
| Campus grades, academic summaries, school notice bodies | Page-local by default; broader durable retention needs an explicit product/privacy decision. |
| Credentials | Existing secure session storage/server credential boundary only; never generic cache or draft storage. |
| Drafts, media pending publication, unsent chat, unsynced plans | Recoverable user work; excluded from cache eviction and clear-cache controls. |

`Current`: [0035](../decisions/0035-campus-device-snapshot-and-schedule-widgets.md) accepts the
allowlisted device snapshot and supersedes [0033](../decisions/0033-campus-foreground-memory-cache.md).
This is an explicit application storage policy. Campus API responses retain `private, no-store`;
generic HTTP caches must not persist them. School tokens remain server-side, and Web's private-data
lifecycle remains separate. Only `profile`, `calendar`, `timetable` and `today` enter the snapshot;
credentials, grades and school message bodies are excluded.

Campus displays last successful update, offline/stale status and refresh progress. Local clock changes
derive today's display from the stored calendar/timetable and stored adjustment rules where supported;
they do not repeatedly call school APIs. Missing/expired term coverage or rules produces an explicit
refresh need rather than a fabricated empty day. The forum binding-status endpoint reads the local
forum database; it must not be confused with expensive school-data fetching.

A refresh coordinator deduplicates requests by site/account/binding/dataset, prevents overlapping
manual refreshes and avoids requesting the same school calendar/timetable through both overview and
today endpoints. Temporary school/network failures retain the same identity's previous snapshot.
Confirmed logout, binding removal/replacement or account erasure removes private campus snapshots
and their projections. Binding uncertainty must not expose a different account's data. Offline
display is explicitly a device snapshot, not proof that remote authorization remains current.

Cache clearing uses category-specific deletion and a scope generation fence. Pending older fetches
cannot rewrite cleared records. Settings reports what will be cleared, what remains, progress and
partial failures; it does not report success until the requested deletion completes. Clearing during
offline use leads to an honest empty state. Retention limits apply by age and size without evicting
user work.

## Verification boundaries

State tests cover delayed responses, simultaneous refresh/pagination/mutations, account/site changes,
storage failure and write/delete ordering. Event tests cover transaction rollback, session revocation,
overflow, reconnect and background transitions. Read tests cover partially visible history, new
offscreen messages, another account's IDs, duplicate acknowledgements and send/read interleavings.
Cache tests cover process restart, binding changes, clear-during-fetch, stale/empty distinctions and
zero school requests from repeated navigation or local clock ticks. Model changes retain the required
SQLite and PostgreSQL verification; contract changes ship fixtures and generated clients together.
