# Moderation notifications for human review and recoverable rejection

## Status

Accepted
Class: feature

## Context and Problem Statement

Background moderation acknowledges a durable submission immediately. Most automatic approvals need
no extra interruption, but a human queue may take longer and authors need a clear result. Rejected
submissions remain editable, so an inaccessible detail page does not provide a useful recovery path.

## Decision Drivers

- Keep ordinary successful publication quiet.
- Explain human review and close the loop when a reviewer decides.
- Let authors recover rejected submissions without exposing their original text in notifications.
- Preserve transactional receipts and the existing three moderation states.

## Considered Options

- Silence every approval and rely entirely on page refreshes.
- Notify every automated and human transition.
- Silence automatic approval, notify human handoff/results, and link rejection to content management.

## Decision Outcome

Choose the third option. This supersedes only the notification behavior in
[0061](0061-versioned-background-moderation.md); its immutable revisions, publication projection,
visibility, durable queue, version fencing and deployment design remain in effect.

Automatic approval is quiet. Transfer to a human creates one `review_pending` notification linking to
the pending topic/floor. Human approval creates `review_approved`; automatic or human rejection creates
`review_rejected`, linking to content management. Rejected subjects keep only their first and last
Unicode character around six asterisks. Notification title, body and preview fields contain no rejected
original text; content management retains the full candidate and reason for its author.

The handoff receipt, review timestamp and effects task commit together under the post lock. A pending
revision with a review timestamp has already entered the human queue; retries do not repeat the notice
or accept a late automated result. Human decisions still require the latest active pending revision.
Approval notifications do not replace the ordinary publication effects. Push delivery remains at least
once, while notification creation is transactional per transition.

## Pros and Cons of the Options

- Silence every approval: fewer messages, but leaves human review without closure. Reject.
- Notify every transition: explicit, but routine automatic successes create unnecessary noise. Reject.
- Notify human review/results and rejection: explains exceptional waits and provides recovery, at the
  cost of distinct notification events and localized copy. Choose.

## Links

- [Forum behavior](../product/forum.md#governance-and-public-exports)
- [Revision storage](../architecture/contracts-and-data.md#data-model)
- [Human approval requirement](https://github.com/YourTongji/YourTJ-Hub/pull/1043#discussion_r4177091985)
- [Human review handoff](https://github.com/YourTongji/YourTJ-Hub/pull/1043#discussion_r4177069182)
- [Rejection recovery wording](https://github.com/YourTongji/YourTJ-Hub/pull/1043#issuecomment-5978915752)
