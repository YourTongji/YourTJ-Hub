# Revision-bound moderation appeals

## Status

Accepted
Class: feature

## Context and Problem Statement

Authors can edit and resubmit rejected content, but cannot request a second human judgment of the
unchanged submission. Review needs a durable explanation and outcome without weakening immutable
revision fencing, anonymous identity protection or deletion retention.

## Decision Drivers

- Review the exact rejected snapshot; approval must never publish a newer edit.
- Preserve the existing Normal / Blocked / Pending content states and publication transaction.
- Make submission and review durable, with private author history and scoped moderator access.
- Keep anonymous authors' internal identity out of appeal responses and notifications.

## Considered Options

- Reopen rejected content into the ordinary pending queue and store the explanation in its review reason.
- Add a fourth content state for appeals.
- Store one appeal per immutable revision and reuse publication for a terminal human decision.

## Decision Outcome

Choose one durable appeal per revision. Appeals cover rejected forum topics and replies, including
rejected edits whose previous approved projection remains public; Wiki first posts, deletion and
account restrictions are excluded. Authors submit a nonblank explanation from content management.
There is no time limit while the latest rejected version remains active, and no second appeal of the
same version. Editing/resubmitting creates a different revision and leaves its predecessor's appeal
unavailable for publication.

An appeal has its own pending / approved / rejected / stale outcome. This is an audit workflow,
not a fourth moderation state. Approval goes through the publication transaction, rechecking live
ownership/lifecycle, latest revision and both live/candidate category scope. The appeal decision,
revision projection, image permissions, search task and result receipt commit together. Rejection
requires a reason and preserves the rejected candidate. Superseded or deleted targets cannot be
restored through an appeal. Original review reasons remain available in retained revision history.

The private history and moderator queue omit author identity fields. User explanations and review
reasons are plain text, shown only to the owner and authorized reviewers. Notifications use safe
review templates and content-management links; they do not include explanations or rejected original
text. Appeal rows and retained snapshots follow the existing governance retention policy.

A stale decision reuses the rejected-review notification template so the author still learns the
outcome. When the author has already edited and resubmitted, that notice can read like a rejection
of the new submission even though it only closes the superseded appeal; product support should
interpret it as "the appeal is closed and the latest submission follows the normal review flow".
A dedicated neutral template remains possible future work.

### Consequences

- Authors can challenge a decision without changing their words.
- Existing publication effects, visibility and three-state content semantics remain authoritative.
- A new audit table and migration are needed; repeated submissions of a version cannot generate a queue flood.
- Flutter mirrors the API payloads; its native appeal interface is outside this Web feature.

## Pros and Cons of the Options

- Reopen the ordinary queue: fewer fields, but loses the original decision and mixes resubmission with appeals;
  a separate reopen step also risks races with workers. Reject.
- Fourth content state: makes appeals explicit on content rows, but changes every visibility reader and cannot
  describe rejected edits that retain a public projection. Reject.
- Revision-bound audit: preserves the original snapshot and one shared publication path, with additional audit
  storage and scoped private reads. Choose.

## Links

- [Issue #1114](https://github.com/YourTongji/YourTJ-Hub/issues/1114)
- [Versioned background moderation](0061-versioned-background-moderation.md)
- [Review notification feedback](0062-moderation-notification-feedback.md)
- [Governance retention](0021-deletion-final-state-data-retention.md)
- [Forum behavior](../product/forum.md#governance-and-public-exports)
