# Topic content and comments panes

## Status

Accepted
Class: feature

## Context and Problem Statement

Readers on an expanded Duo want to refer to a topic's original post while reading its comments.
A single combined list makes them scroll between the two. Unlike Home's independent topic streams,
these two regions share one subject and are useful together.

## Decision Drivers

- Keep the original post available while browsing its discussion.
- Give the post more reading space than each comment.
- Preserve reply drafts and the active reading position across folding.
- Keep phone and enlarged-text reading comfortable without device-name checks.

## Considered Options

- Keep the combined post-and-comments list at every width.
- Place the original post and comments in independently scrolling panes when space permits.
- Open comments in a temporary sheet over the original post.

## Decision Outcome

Use a bounded topic workspace with the original post on the left and comments on the right.
The comments pane owns reply composition and the floor picker. Back and the discussion actions use
the native side-pocket glass capsule when available, sharing its existing geometry and fallback rules.
The shared header and a quiet separator
connect both panes without presenting them as unrelated cards. Narrow windows and enlarged text use
the existing combined reading flow. Layout changes map the active pane's relative scroll offset and
retain the same reply controller, target and draft. Comments retain their sorting, pagination,
permissions and submission behavior.

This refines the topic reading window from [0037](0037-adaptive-mobile-reading-window.md).
Home retains the single-stream composition in [0110](0110-single-column-home-reading.md).
The implementation thresholds and bounds belong to the mobile experience reference.

## Pros and Cons of the Options

- A combined list has one scroll position and fits compact screens, but prevents simultaneous
  reference to the original post and discussion.
- Independent panes support that reference and keep each region readable. They require mapping the
  active reading position when folded; pixel offsets cannot guarantee the identical text line after
  text wraps to a different width.
- A temporary sheet gives comments more room on a phone, but covers the source material and adds an
  extra open/close action on a window that already has enough space for both.

## Links

- [Mobile experience](../product/mobile-experience.md#navigation-and-reading)
- [Mobile layout standard](../product/mobile-design-system.md#navigation-and-windows)
- [Topic implementation](../../apps/mobile/packages/forum_app/lib/src/pages/topic/topic_page.dart)
- [Duo adaptation PR and acceptance evidence](https://github.com/YourTongji/YourTJ-Hub/pull/1407)
