# Foreground updates use a process-local SSE invalidation stream

## Status
Accepted
Class: architecture

## Context and Problem Statement

Foreground chat and notification views need to notice changes promptly. Periodic client polling delays updates and repeats reads while nothing changes. The forum currently serves each environment from one Go process, but chat and notification data remains authoritative in PostgreSQL and the existing REST APIs.

## Decision Drivers

- Deliver prompt foreground updates without transporting private message bodies or maintaining a second authoritative state.
- Keep the single-binary deployment and bound open connections, queued events, and database work.
- Recover correctly from disconnects, queue overflow, process restarts, and missed hints.
- State clearly when a deployment with multiple serving processes needs additional infrastructure.

## Considered Options

- A process-local Server-Sent Events stream carrying owner-scoped invalidation hints, followed by REST reconciliation.
- More frequent REST polling from every foreground client.
- A WebSocket channel carrying full chat and notification state.
- A shared broker with durable event IDs and replay across serving processes.

## Decision Outcome

`GET /api/forum/events` uses SSE for one-way, owner-scoped `chat.changed`, `notifications.changed`, and `unread.changed` hints. Every connection begins with `hello` and `resync: true`; clients then reconcile through REST. Frames have no message body, preview, unread count, or replay ID. The stream is never the source of truth. A bounded queue disconnects a slow consumer so reconnection causes a fresh REST reconciliation rather than silently discarding hints.

The hub is local to the serving process. The five-per-user and 10,000-per-process limits bound subscriptions; normal heartbeats do not query the database. Session validity is checked at handshake and on an independent five-minute cadence, while authenticated REST requests enforce validity on their own path. A forum database served by multiple Go processes needs a shared invalidation transport before this prompt-delivery contract applies across them. Main and dev are separate single-process environments.

## Pros and Cons of the Options

- Process-local SSE: one-way delivery fits invalidations, works with the current binary, and avoids idle REST reads. It cannot deliver cross-process hints and has no replay guarantee; REST resync is required after every connection.
- Frequent polling: simple to deploy but causes repeated reads and an update delay even when clients are foregrounded.
- Full-state WebSockets: support bidirectional communication, but add connection and state synchronization complexity without removing the need for REST recovery.
- Shared durable broker: supports cross-process fan-out and replay, but adds an operational dependency and event-retention model that the current single-process serving topology does not need.

## Links

- [Realtime HTTP contract](../../packages/api-contract/paths/forum-events.yaml)
- [Contracts and data architecture](../architecture/contracts-and-data.md)
- [Deployment shape and stream operation](../operations/deployment.md)
- [Process-local hub](../../apps/gooseforum/app/service/realtimeservice/hub.go)
