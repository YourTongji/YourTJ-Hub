# Transactional forum publication points

## Status

Accepted
Class: architecture

## Context and Problem Statement

Forum content and legacy points belong to the same database. Publication must not acknowledge
accepted content while depending on an in-memory consumer to create its reward. Process loss
between content commit and event consumption otherwise leaves no durable reward intent.

## Decision Drivers

- A successful first publication must durably include its local reward.
- Existing source keys, bot exclusions, wiki exclusions, and reply-deletion tombstones remain authoritative.
- Keep the single-binary runtime and avoid an additional worker protocol for same-database writes.

## Considered Options

- Commit the local ledger and balance changes in the content transaction.
- Persist a reward outbox and deliver it through a retrying worker.
- Keep in-memory delivery and add periodic reconciliation.

## Decision Outcome

Use `pointservice.RewardPublicationTx` from direct publication, draft publication, and both revision
and legacy approval transactions. A ledger error rolls back publication. The existing `topic:<id>`
and `post:<id>` source keys make retries and older queued events idempotent. Pending content, bots,
and wiki first posts remain excluded; anonymous replies use their stored numeric owner.

Existing event consumers remain for compatibility, but new publication does not rely on them.
This decision covers legacy forum-local points, not XP or cross-platform credit. It does not infer
historical missing rewards or make the separate content-restoration path atomic.

## Pros and Cons of the Options

### Content transaction

- Provides a single commit boundary without another delivery state machine.
- Reuses the existing ledger, unique source keys, and deletion locking rules.
- Adds ledger/balance work to publication latency; a ledger failure also rejects publication.

### Durable outbox

- Allows independent retry and suits a future remote settlement owner.
- Introduces pending delivery, worker retries, and monitoring even though both current writes share a database.

### In-memory delivery with reconciliation

- Keeps the publication transaction shorter.
- Leaves a reward gap until repair and requires reconstructing intent from mutable historical content.

## Links

- [Local reward implementation](../../apps/gooseforum/app/service/pointservice/publication.go)
- [Publication approval transaction](../../apps/gooseforum/app/service/publicationservice/review.go)
- [Product boundary](../product/credit-and-escrow.md#current-boundary)
