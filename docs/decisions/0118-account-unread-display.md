# Account-wide unread display preference

## Status

Proposed
Class: feature

## Context and Problem Statement

Members need to choose how strongly unread activity asks for attention while keeping the same messages
and read state. Web and App have navigation and conversation indicators, and old servers only provide
boolean totals. Push delivery already has independent account preferences and system consent.

## Decision Drivers

- One consistent choice for a member across Web and App.
- Display changes must not acknowledge messages or mute delivery.
- Account changes and late requests must not apply another session's preference.
- Existing unread reconciliation should provide exact totals without a second count request.

## Considered Options

- Store a global number/dot preference on the forum account, separate from delivery.
- Store a device-local preference in each client.
- Combine count visibility with push delivery groups or introduce per-category display preferences.

## Decision Outcome

Use the first option. A private `users.unread_badge_style` column defaults to `number`, and the
owner-only notification-preferences API accepts only `number` or `dot`. Saving invalidates cached unread
status and emits existing private invalidation hints. Status includes optional exact notification and
message totals; unknown totals render a dot. Muted conversations remain unread. Numeric labels cap
visually at `99+`; dot mode omits quantities from accessibility labels too.

Clients share their existing unread request and fence older preference reads after writes. Failed saves
restore the previous choice. The preference does not alter read acknowledgements, push delivery or OS
authorization, and does not add OS app-icon badges. A single preference gives up category-level choices
in exchange for a small, consistent setting.

## Pros and Cons of the Options

- **Account preference:** consistent across clients and recoverable on a new device; requires a private
  column, an authenticated API and session-scoped client reconciliation.
- **Device-local preference:** avoids a server write and allows each device to differ; unexpectedly
  resets on reinstall and cannot provide the requested account-wide behavior.
- **Delivery/category coupling:** offers more control in one panel; confuses silent display with missing
  delivery and multiplies settings without a current need.

## Links

- [Issue #1415](https://github.com/YourTongji/YourTJ-Hub/issues/1415).
- [Forum notification behavior](../product/forum.md#mentions-and-notifications).
- [Flutter Badge API](https://api.flutter.dev/flutter/material/Badge-class.html): distinguishes a small
  indicator without a label from a labelled/count badge; informs the shared component presentation.
- [Apple notification guidance](https://developer.apple.com/design/human-interface-guidelines/notifications):
  system notification consent and app-icon badges belong to system delivery; the in-app preference
  keeps that boundary separate.
