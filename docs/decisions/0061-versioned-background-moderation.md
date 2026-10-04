# Versioned background moderation with editable rejection

## Status

Superseded by [0062](0062-moderation-notification-feedback.md)
Class: architecture

## Context and Problem Statement

Authors need an immediate, durable send acknowledgement. Moderation latency should not keep the editor
open, and a rejected submission must remain editable. Editing published content also requires two
simultaneous views: readers keep the approved version while its author sees the submitted revision.
A single overwritten post body cannot represent these views, regardless of how many status values it has.

## Decision Drivers

- Preserve Normal (0), Blocked (1) and Pending (2); Blocked is editable, not an account restriction.
- A successful send means the content and its processing task have been committed.
- Review exactly the submitted title, categories, body and gallery; stale outcomes never publish a newer draft.
- Keep public content stable during edit review and keep candidate-only images private.
- Reuse the single binary, existing task queue and post history, with SQLite and PostgreSQL support.

## Considered Options

- Add a rejected status while continuing to overwrite the public row.
- Introduce separate draft/review tables and a second content lifecycle.
- Reuse `post_revisions` for full immutable submissions with latest/approved pointers on `posts`.

## Decision Outcome

Choose full snapshots in `post_revisions`. The public topic/post rows remain the approved projection;
`latest_revision_id` selects the author's current submission and `published_revision_id` identifies the
approved snapshot. The three existing moderation statuses apply to each submission. Owner-only views
are composed on copies outside shared caches. Rejected candidates appear in content management, where
an author can read, edit/resubmit or delete them; a rejected edit leaves the old public projection intact.

A moderated save commits a revision, private attachment references and a `content-review` task together.
Jev and sensitive-word outcomes run in the worker. Human and automatic decisions share one transaction:
lock the post then topic, verify the latest pending revision and active lifecycle, then publish all fields
or reject that version. Approval is quiet; rejection creates one notification linking to content management.
`content-published` tasks deliver existing publication events and pushes after commit. The queue retries
infrastructure errors and recovers expired leases; content awaiting a model/human decision remains private.
SSE carries invalidation only, with authenticated REST reconciliation on reconnect.

This supersedes [0056](0056-ai-image-text-moderation.md)'s request-time/process-local execution contract.
Its vision evidence → Jev per-policy Noul probabilities → Go resolver, fail-closed model behavior,
provider privacy controls, policy calibration and shadow observation remain the supported inference design.
The settings UI offers shadow and background review; legacy `enforce` is accepted as a background-review
compatibility value. Draft saves are not reviewed, but publication is. Scope is forum topics, replies and
edits, including Agent/MCP writes; Wiki first posts remain owned by the Wiki sync system.

Legacy public posts receive a complete baseline from their current row when next edited. Migration 31
adopts active legacy pending rows in bounded batches and can be retried. It cannot reconstruct an old
approved body that the previous implementation already overwrote. Deletion/retention guards cover
revisions and their attachments; retained audit content follows [0021](0021-deletion-final-state-data-retention.md).
Newly submitted image references are private, but the pre-bind upload compatibility window is unchanged.

Wagtail's live-with-unpublished-changes model and revision API (official documentation, read 2026-10-04)
support separating publication from the latest revision. We adopt that separation, without importing its
page tree or multi-stage editorial workflow into a campus forum.

### Consequences

- Authors get quick send confirmation and a recoverable rejected submission; readers see stable approved content.
- No fourth moderation state, extra service or parallel review table is needed.
- More snapshot storage and owner-specific projections are necessary. Human review must send a revision ID.
- Queue polling adds up to five seconds before model work begins; a crashed running task can wait for the
  existing ten-minute lease expiry. Model errors increase the human queue rather than exposing content.
- Publication effects use the existing at-least-once event handlers. The rejection receipt itself is
  transactional and unique per terminal review; external push delivery is not exactly once.

## Pros and Cons of the Options

- Extra status only: fewer columns, but cannot keep old and new bodies simultaneously; rejected edits still
  overwrite public data. Reject.
- Separate review tables: clear isolation, but duplicates ownership, history and deletion rules. Reject.
- Existing revisions: reuses identity/history and supports atomic projection changes; requires full snapshots
  and explicit version checks. Choose.

## Links

- [Forum user behavior](../product/forum.md#governance-and-public-exports)
- [Storage and revision contract](../architecture/contracts-and-data.md#data-model)
- [Moderation operations](../operations/deployment.md#ai-图文审查issue-975)
- [Wagtail page statuses](https://guide.wagtail.org/en/concepts/page-status/)
- [Wagtail revision model](https://docs.wagtail.org/en/stable/reference/models.html#revision)
