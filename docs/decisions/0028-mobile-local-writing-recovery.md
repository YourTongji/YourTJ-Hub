# Mobile local writing recovery

## Status

Accepted
Class: feature

## Context and Problem Statement

A mobile editor needs to retain partial text before the title and classification required by the
server are complete, including when network access fails. Search history also needs a small,
clearable device-local store. A local recovery snapshot must remain distinct from a server draft.

## Decision Drivers

- Preserve unfinished writing without changing the publishing API or its validation.
- Separate sites and accounts on shared devices; never use a session token as a storage key.
- Keep storage failure visible and order deletion after in-flight saves.
- Preserve existing Markdown, category and uploaded-image contracts.

## Considered Options

- Autosave through the existing server draft endpoint.
- Add a new backend draft model and synchronization protocol.
- Keep account-scoped local recovery snapshots in app-private preferences.

## Decision Outcome

Use local preference snapshots keyed by API origin, numeric account ID and editor slot. A slot is
one new-composition entry type or an existing topic. The editor debounces writes and flushes when
leaving or becoming inactive. The draft list exposes local recovery separately from server drafts.
Server acknowledgement or explicit discard removes the snapshot after pending writes settle.
Recent search strings use the same account namespace with a separate bounded list and clear action.
Logging out hides local writing; account closure attempts to remove the owner's local records.

Message delivery state stays in a session-local outbox. It supports manual retries and is discarded
at a session boundary. It does not claim durable offline messaging or server-side idempotency.

## Pros and Cons of the Options

### Existing server draft endpoint

- Reuses server storage and works across devices.
- Cannot preserve incomplete required fields or work without a network connection.

### New backend synchronization protocol

- Can support cross-device partial writing and explicit conflict resolution.
- Requires a new contract, storage lifecycle and conflict UI beyond local recovery needs.

### Account-scoped device preferences

- Works with incomplete fields and offline, using an existing mobile dependency.
- Keeps tokens out of the data store and separates records by stable identity.
- Does not provide cross-device recovery, application-level encryption or a guarantee against
  termination before the latest write completes. Serialization and explicit save feedback are required.

## Links

- [Mobile publishing behavior](../product/mobile-experience.md#publishing)
- [Writing store implementation](../../apps/mobile/packages/forum_app/lib/src/local/writing_store.dart)
- [Server topic-writing contract](../../packages/api-contract/openapi.yaml)
