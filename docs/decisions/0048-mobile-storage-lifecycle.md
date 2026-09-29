# Mobile storage lifecycle and durable user work

## Status

Accepted
Class: architecture

## Context and Problem Statement

A device contains rebuildable forum and message snapshots, public images, an explicitly allowlisted
campus snapshot, unfinished user work and credentials. A single “clear data” operation cannot assign
all of them the same retention or deletion semantics. Unscoped topic/chat rows and independent media
loaders also make account isolation, byte accounting and clear-during-fetch behavior unreliable.
Preferences do not provide the transaction boundary required for recovery documents and schedule plans.

## Decision Drivers

- Keep drafts, unsent messages and unsynchronized plans out of automatic cache eviction.
- Isolate API origin, numeric account ID and representation; reject delayed work across session/clear boundaries.
- Make cleanup bounded, observable, retryable and safe after interruption.
- Minimize persisted private data and respect HTTP storage directives.
- Reuse Flutter, Riverpod, Dio and Drift without introducing a full offline synchronization engine.

## Considered Options

- Add a clear button over the existing independent stores.
- Add a global HTTP cache interceptor.
- Share lifecycle rules across domain-owned repositories and separate recoverable work from cache.
- Make a full local synchronization database authoritative for all application state.

## Decision Outcome

Use domain-owned repositories with a shared lifecycle coordinator. Stable cache identity includes API
origin, numeric account and language. Session epochs and category-specific generations are runtime
fences, captured before I/O and checked before applying UI state or committing data. A cleanup intent
is durable before deletion; every failed owner remains suspended until retry succeeds. Cache clearing
does not change authentication or erase user work. Explicit local reset additionally signs out, erases
local work and restores preferences; it never deletes remote content or school bindings. Startup
recovery and an unfinished reset block business routes, preventing new work from being created inside
a pending destructive operation.

`cache.sqlite` holds versioned, disposable business projections and the campus allowlist. It is
physically separate from `user_work.sqlite`. Both native databases use SQLite3MultipleCiphers through
Drift, distinct secure-storage keys, full synchronous commits and backup-excluded application support
storage. Cipher availability is checked in release builds; there is no plaintext fallback. A missing
work key preserves the unreadable database. The public media repository owns its cache directory,
index, HTTP policy, download bounds and byte eviction; UI components use one injected image loader.

The forum business projection may retain visited published, normal, non-deleted topic bodies and
reply windows, and visited home streams in their account scope. It excludes layout viewer identity,
email, permission claims, unread totals and private page chrome. A restored page is read-only until
network validation. Chat projections retain synchronized message history for the scoped account;
reading one is never a read acknowledgement. This explicit application policy does not permit caching
raw `no-store` HTTP responses. Unapproved data domains remain memory-only. Campus retains the exact
allowlist and binding lifecycle of [0035](0035-campus-device-snapshot-and-schedule-widgets.md).

User-work migration copies legacy records into a transaction, verifies the stored value and establishes
authority before removing the legacy copy. Deletion tombstones prevent resurrection. Legacy schedules
without a trustworthy site identity remain recoverable but unassigned until the user confirms their
source and destination. Private-message drafts retain their existing device-bound secure-storage owner.
This supersedes the preferences persistence choice in [0028](0028-mobile-local-writing-recovery.md);
the distinction between local recovery and server drafts remains.

Versioned SQL migrations retain the existing Drift runtime without coupling its code generator to the
workspace's Freezed generator. Media uses the existing Dio transport and a single managed file index:
this permits checking generation and HTTP eligibility before atomic commit without a second hidden
file-cache policy. The concrete limits and UI semantics live in the linked architecture/product specs.

## Pros and Cons of the Options

### Clear button over independent stores

- Small change with a visible entry point.
- Does not establish identity, retention, write fences, recoverability or complete space accounting.

### Global HTTP cache interceptor

- Simple transport integration and conditional-response reuse where supported.
- Cannot distinguish unfinished work, authoritative empty snapshots or permission-dependent content;
  persisting the existing page/campus `no-store` responses would violate transport policy.

### Shared lifecycle with domain repositories

- Preserves business ownership and existing campus/event boundaries while making deletion testable.
- Separates durable work from disposable data and supports incremental domain coverage.
- Requires native encryption verification, explicit migration and independent-owner failure handling.

### Full local synchronization engine

- Can support broad offline writes and cross-device conflict resolution.
- Requires server versions, idempotency and conflict semantics beyond reading cache and local recovery.

## Links

- [Mobile state and cache boundaries](../architecture/mobile-state-and-cache.md)
- [Mobile product behavior](../product/mobile-experience.md)
- [Drift encrypted native databases](https://drift.simonbinder.eu/platforms/encryption/)
- [Flutter offline-first repository patterns](https://docs.flutter.dev/app-architecture/design-patterns/offline-first)
- [HTTP caching requirements](https://www.rfc-editor.org/rfc/rfc9111.html)
- [Shared preferences persistence limitations](https://pub.dev/packages/shared_preferences)
