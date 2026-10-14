# Notification policy and private chat push

## Status
Accepted
Class: feature

## Context and Problem Statement

Activity notifications, direct messages and official inbox mail have different owners and read facts.
System delivery needs consistent category semantics, explicit user control and safe lock-screen copy.
A direct reply can also mention its recipient without becoming two events. Existing historical rows
cannot reliably prove that second reason.

## Decision Drivers

- Preserve existing event types, pagination, unread facts and conversation authorization.
- Default to timely mentions, replies and private messages without noisy low-value activity.
- Keep private message bodies and unpublished moderation subjects out of third-party payloads.
- Apply account preferences across Web and native channels without changing OS consent.

## Considered Options

- Typed activity reasons and selective defaults, with private chat sources in the existing outbox.
- Enable every event by default and include message previews.
- Disable every category by default and require separate manual activation.

## Decision Outcome

Use six activity categories: all, mentions, replies, likes, following and system, independently of the
unread filter. New directly mentioned recipients carry `hasDirectMention`; this reason takes
precedence in presentation and category selection while retaining the legacy event type and one row.
Historical replies are not inferred or backfilled from content.

Mentions, replies and chat default to enabled after transport consent. Likes, follows, subscribed topic
and Wiki updates default to silent. Review approval/rejection default to enabled; badges, pending review
and ordinary system events remain silent until an explicit system-group opt-in. Account preferences
update only named groups, so changing chat cannot accidentally enable ordinary system messages.
Explicit opt-out, conversation mute and live access/read checks override defaults. OS authorization
and device registration remain separate prerequisites.

Chat tasks reference committed message IDs and recipients; they do not create activity notifications.
Their outbox rows participate in the message transaction and are invisible to workers before commit,
including an enclosing multi-message forward transaction. A savepoint isolates best-effort queue
failure from message persistence. Existing workers claim, recover and clean up channel tasks. Unique
source/channel/recipient keys suppress repeated enqueue; transport delivery remains best effort.
Chat copy contains a public sender name and the action, never the body or private contact aliases.
Recipients and source IDs travel with controlled routes; clients verify current account ownership and
re-authorize object access. A loaded, uncovered foreground chat suppresses only its own system alert.

Low-value hints share a short per-recipient/target/channel throttle without removing stored events.
Android uses versioned chat, mentions/replies, activity and system channels while retaining the legacy
channel and OS choices. iOS uses ordinary alerts and object grouping. Provider acceptance is not proof
of an OS banner; credentials, signed devices and manufacturer delivery remain `Partial`.

## Pros and Cons of the Options

- Typed reasons and selective defaults preserve useful alerts and privacy; require coordinated
  contracts, clients and a small additive schema change.
- Every event with previews maximizes immediate information but increases interruption and exposes
  private content on lock screens and to providers.
- All categories off minimizes interruption but hides high-value messages after a user grants push
  consent and increases setup work.

## Links

- [Issue 1120 specification](https://github.com/YourTongji/YourTJ-Hub/issues/1120)
- [Forum notification semantics](../product/forum.md#mentions-and-notifications)
- [Native provider decision](0019-native-push-providers.md)
- [First-login Android permission](0049-android-first-login-notification-permission.md)
- [Native delivery verification](../operations/mobile-releases.md#native-push-activation-and-verification)
