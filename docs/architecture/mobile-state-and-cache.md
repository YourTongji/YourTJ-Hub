# Mobile state, events and cache boundaries

> Doc type: architecture specification
>
> Status: Draft
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-30

This specification defines `Planned` contracts and explicitly provisional `Decision needed` policies for the
[mobile interaction standard](../product/mobile-design-system.md). Existing behavior is recorded in
[mobile experience](../product/mobile-experience.md) and [campus](../product/campus.md). The campus snapshot policy is `Current` under [0052](../decisions/0052-campus-daily-entry-refresh.md), foreground SSE under 0036,
and the implemented storage lifecycle under 0048. Contracts marked `Planned` remain outside those implementations.

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

`Current`: ordinary drafts, bounded search history and local schedule snapshots use
`user_work.sqlite`, with API origin and numeric account scope. Preference migration commits and
verifies the new record before legacy cleanup. Tombstones remain authoritative if cleanup fails.
Unassigned legacy plans remain recoverable through explicit ownership confirmation; a numeric account
ID alone cannot identify their original site. `clearAll` is a separate, generation-fenced reset API;
cache cleanup never calls it.

`Current`: private-message draft records use the existing secure-storage plugin. iOS uses a distinct
Keychain service with `AfterFirstUnlockThisDeviceOnly` and synchronization disabled; Android retains
the token store's native file/cipher configuration with separate draft keys, and excludes the secure
and legacy preferences files from backup/transfer. Legacy plaintext is removed only after secure
write/read-back succeeds. Ordered deletion markers prevent an old plaintext record from resurrecting
if cleanup fails. Product retention and backup limits are specified in
[mobile experience](../product/mobile-experience.md).

The media queue owns temporary local files, upload state and remote attachment references. It does
not own the editor document or submit content. Attachment ordering is independent of completion
order. Cancellation and retry affect the selected attachment, and publishing validates queue state.
Restart recovery requires app-owned copies of picker files, with a bounded cleanup lifecycle; a
temporary picker URI alone is not a durable attachment.

## Foreground events and visible reads

`Current`: a single foreground SSE event connection per active forum session delivers authenticated,
user-scoped invalidation hints. REST remains authoritative for message content, notification rows and
unread counts. The transport and operational tradeoffs are defined by
[0036](../decisions/0036-foreground-realtime-invalidation.md). Normal foreground delivery must not depend on fixed-interval polling.

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

`Current`: [0048](../decisions/0048-mobile-storage-lifecycle.md) defines the lifecycle boundary.
`AppDatabase` schema 2 stores disposable snapshots under `(origin, numeric account, language,
domain, resource key)`. Cache envelopes contain a document version, successful-save time, last access
and UTF-8 byte size. Valid empty snapshots are distinct from a miss; reads do not renew retention.
Malformed/unknown-version/future-dated records fail closed. Version 1 forum/chat rows have no reliable
identity and are discarded; the already scoped campus document is retained during verified encryption
migration. Migrations use versioned SQL independently of the Freezed generation toolchain.

| Data | Persistence and refresh contract |
|---|---|
| Visited home streams and published normal topics/reply windows | `Current`: account/origin/language-scoped reading projection, 7-day hard retention; cached content is identified while network validation restores current state. Layout viewer, email, permissions and unread totals are not persisted. Draft, processed and deleted topics are excluded. |
| Conversation list and synchronized messages | `Current`: 30-day retention, complete empty list replaces old membership, removed conversations lose their messages; empty message deltas do not erase history. At most 50 conversations per snapshot and 200 messages per conversation, with a device-wide byte limit. Cached reads never acknowledge messages. |
| Public images, GIFs and avatars | `Current`: one media owner, explicit HTTP eligibility, full URL and account/origin/language key; credentials are never attached. Private/no-store/no-cache and uncertain signed resources remain memory-only. |
| Notifications, course metadata/reviews and Wiki | `Partial`: existing visible-page memory behavior; no new generic disk projection is authorized. |
| Campus name, calendar, official timetable | `Current`: private allowlisted device snapshot, additionally binding-scoped; complete successful refreshes replace it atomically; previous-day snapshots refresh on Campus entry after binding verification, with manual refresh retained. |
| Campus grades, academic summaries, school notice bodies | Page-local by default; broader durable retention needs an explicit product/privacy decision. |
| Credentials | Existing secure session storage only; never generic cache or draft storage. A persisted local-revocation marker prevents an unsuccessfully deleted token from restoring a session. |
| Drafts, unsent chat and unsynchronized plans | `Current`: recoverable user work, excluded from eviction and clear-cache controls. Ordinary writing/plans use a separate transaction database; private-message drafts retain secure storage. |
| Picker/upload work in progress | `Partial`: existing attachment queue behavior; this cache lifecycle does not claim durable restoration of every picker URI or an offline upload outbox. |

`Current`: uploaded private-message image URLs and their idempotent send identifiers share the
device-bound private draft store, scoped by API origin, numeric account and peer. They are written
before chat/send and restored only for manual retry, then removed after acknowledgement. Clear-user-data
and account deletion also erase them and fence late upload callbacks; ordinary cache eviction does not.
Picker bytes and uploads whose URL has not yet returned remain outside this recovery guarantee.

The managed cache target is 256 MiB: media 192 MiB, forum projections 32 MiB, chat 16 MiB,
campus documents up to 4 MiB and 12 MiB of accounting headroom. These are cache limits, not the
application's installed size or total process memory. Category rows estimate payload sizes; the total
includes database/index/journal files and managed media. Work storage is reported separately.
The database owner checks allocated/free pages and physical files after committed batches: 4 MiB of
free pages or a 64-MiB physical watermark triggers reclamation and checkpointing. Legacy non-incremental
files are converted once. The 256-MiB value is a management target, not a byte-exact peak guarantee
during an atomic write or migration; campus snapshots and user work are never sacrificed to force it.
Media has a 14-day idle bound, a 32-MiB download limit and at most three active downloads. Flutter's
decoded-image cache targets 48 MiB; active animation codecs and other UI objects are additional memory.

Native cache and work files use different SQLite3MultipleCiphers keys held in secure storage. Native
opening checks cipher availability even in release mode, enables WAL and full synchronous commits,
and excludes the managed directory from backup. An unavailable work key is an error, never permission
to rebuild or erase the user's database. On opening an existing disposable cache, SQLite corruption,
failed integrity checks or an unreadable cipher rebuild only the cache and its migration sidecars.
Lock contention, disk/permission errors and unavailable cipher support propagate without deleting
files. User work is never rebuilt on either corruption or key failure. The public file cache lives in
the OS cache directory.
The explicit campus offline document shares the encrypted cache database in application support so
it retains its existing managed lifetime. Cache deletion cannot address `user_work.sqlite`.
On iOS, backup exclusion applies to the app-owned private directory and the widget App Group's
`Library/Preferences` directory, including future UserDefaults rewrites. The system-owned App Group
container root is never modified: physical devices reject its extended-attribute writes even when
the simulator permits them. Genuine backup-configuration failures still propagate to startup recovery.

`Current`: [0052](../decisions/0052-campus-daily-entry-refresh.md) supersedes
[0035](../decisions/0035-campus-device-snapshot-and-schedule-widgets.md) with daily entry refresh,
retaining its allowlisted device snapshot and Widget projection contract.
This is an explicit application storage policy. Campus API responses retain `private, no-store`;
generic HTTP caches must not persist them. School tokens remain server-side, and Web's private-data
lifecycle remains separate. Only `profile`, `calendar`, `timetable` and `today` enter the snapshot;
credentials, grades and school message bodies are excluded.

Campus displays last successful update, offline/stale status and refresh progress. Local clock changes
derive today's display from the stored calendar/timetable and stored adjustment rules where supported;
they do not repeatedly call school APIs. Missing/expired term coverage or rules produces an explicit
refresh need rather than a fabricated empty day. The forum binding-status endpoint reads the local
forum database; it must not be confused with expensive school-data fetching.

A refresh coordinator uses the complete snapshot's `committedAt` Shanghai date as the durable daily
refresh marker. On entry, a previous-day snapshot triggers the four allowlisted reads and the selected
section's datasets after binding verification. Same-day success survives process restart; failures keep
the old marker so a later entry or manual refresh can retry. Batches crossing Shanghai midnight cannot
commit yesterday's reads as today's successful update. The existing forced fresh reads after background
resume remain independent of this daily trigger.

The coordinator deduplicates requests by site/account/binding/dataset, prevents overlapping
automatic/manual refreshes and avoids requesting the same school calendar/timetable through both overview and
today endpoints. Temporary school/network failures retain the same identity's previous snapshot.
Confirmed logout, binding removal/replacement or account erasure removes private campus snapshots
and their projections. Binding uncertainty must not expose a different account's data. Offline
display is explicitly a device snapshot, not proof that remote authorization remains current.

Cache clearing uses category-specific deletion and a scope generation fence. Pending older fetches
cannot rewrite cleared records. Settings reports what will be cleared, what remains, progress and
partial failures; it does not report success until the requested deletion completes. Clearing during
offline use leads to an honest empty state. Retention limits apply by age and size without evicting
user work.

Session cache views receive the cleanup coordinator as an application-layer dependency; the database
provider only owns the database lifetime and never reads back into its coordinator. Captured views
retain that cleanup capability across a session-epoch change. Shared topic/chat views perform one
coordinated sweep before publishing the signed-out widget state; a failed deletion still prevents
new credentials from being committed. Regression coverage must exercise the production provider graph
and encrypted file-backed database, in addition to isolated owner tests.
Failed cleanup records its stage, exception type and code stack in local diagnostics; exception
messages and storage payloads are excluded so credentials and private content are not logged.

A destructive local reset first flushes and verifies a content-free `reset.intent` marker in the
backup-excluded private directory. This marker is outside cache databases, encryption keys and
preferences, so rebuilding a corrupt or keyless cache cannot lose the pending reset. Startup checks it
before opening cache projections; any existing marker, including an interrupted write, requires
recovery. The legacy cache journal remains readable for compatibility. Both markers are removed only
after every reset owner succeeds. A missing user-work key still blocks reset completion and preserves
the encrypted work file. Retrying can recover transient secure-storage access failures; a permanently
lost key cannot be reconstructed by reinstalling or creating a new key. Support must preserve the file
and must not promise an in-app recovery path or silently erase it.

## Verification boundaries

State tests cover delayed responses, simultaneous refresh/pagination/mutations, account/site changes,
storage failure and write/delete ordering. Event tests cover transaction rollback, session revocation,
overflow, reconnect and background transitions. Read tests cover partially visible history, new
offscreen messages, another account's IDs, duplicate acknowledgements and send/read interleavings.
Cache tests cover process restart, binding changes, clear-during-fetch, stale/empty distinctions and
zero school requests from repeated navigation or local clock ticks. Model changes retain the required
SQLite and PostgreSQL verification; contract changes ship fixtures and generated clients together.
