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
- Combine count visibility with push delivery groups or calculate category totals separately in each client.
- Filter badge totals in the server using a separate account category selection.

## Decision Outcome

Use account preferences with server-side category filtering. A private `users.unread_badge_style`
column defaults to `number`. The owner-only notification-preferences API accepts `number` or `dot`
and a `badgeCategories` array drawn from replies, mentions, likes, following and system. A nullable
`users.unread_badge_categories` text column stores JSON: NULL means all; an explicit empty array means
none. Partial writes update only supplied non-null columns, so old style-only clients and concurrent
style/category changes preserve unrelated choices. Saving invalidates cached unread status and emits
existing private invalidation hints.

Status includes style, categories and optional exact notification and message totals. Notification
flags/counts apply the same category predicates as the notification center; directly mentioned replies
belong to mentions, unknown event types to system. Latest unread ID/type remain unfiltered for
independent browser delivery. Unknown totals render a dot. Muted conversations remain unread. Numeric
labels cap visually at `99+`; dot mode omits quantities from accessibility labels too.

Clients share their existing unread request and fence older preference reads after writes. Failed saves
restore the previous choice. The preference does not alter read acknowledgements, push delivery or OS
authorization, and does not add OS app-icon badges. The server owns the category predicates and
aggregation so Web, SSR and App agree; private-message badges and notification lists/read state are
outside this filter.

## Pros and Cons of the Options

- **Account preference:** consistent across clients and recoverable on a new device; requires a private
  column, an authenticated API and session-scoped client reconciliation.
- **Device-local preference:** avoids a server write and allows each device to differ; unexpectedly
  resets on reinstall and cannot provide the requested account-wide behavior.
- **Delivery coupling:** reduces fields but confuses silent display with missing delivery and cannot
  express independent badge/push choices.
- **Client-side category totals:** permits local combinations but expands the status payload and
  duplicates aggregation in Web/App/SSR; the server already owns classification and private refresh.
- **Server filtering:** one shared count and taxonomy, with a separate small account field; requires
  adding categories to the existing preference/status contract and upgrading the account schema.

## Links

- [Issue #1576](https://github.com/YourTongji/YourTJ-Hub/issues/1576).
- [Discourse notification menu](https://meta.discourse.org/t/understanding-the-notifications-menu-in-discourse/3059):
  reviewed 2026-11-04; category tabs and retained unread entries support separating category presentation
  from read acknowledgement. This does not establish Discourse support for the same badge preference.
- [Issue #1415](https://github.com/YourTongji/YourTJ-Hub/issues/1415).
- [Forum notification behavior](../product/forum.md#mentions-and-notifications).
- [Flutter Badge API](https://api.flutter.dev/flutter/material/Badge-class.html): distinguishes a small
  indicator without a label from a labelled/count badge; informs the shared component presentation.
- [Apple notification guidance](https://developer.apple.com/design/human-interface-guidelines/notifications):
  system notification consent and app-icon badges belong to system delivery; the in-app preference
  keeps that boundary separate.
