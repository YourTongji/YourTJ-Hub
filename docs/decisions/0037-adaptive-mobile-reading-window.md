# Adaptive mobile reading windows

## Status
Accepted
Class: architecture

## Context and Problem Statement

The four-root shell in [0012](0012-unified-mobile-reading-navigation.md) hides root controls during
downward reading. Wide and short windows need a different space tradeoff: a bottom bar consumes
scarce height, while unrestricted content width creates long reading lines. Resizing must not replace
the retained route, editor or scroll state.

## Decision Drivers

- Keep Home, Campus, Notifications and Messages directly reachable with consistent Lucide icons.
- Preserve useful vertical reading space and bounded line length.
- Preserve drafts, branch state, accessibility and unread meaning across window changes.
- Keep navigation scrolling independent from reading gestures.

## Considered Options

- A responsive bottom bar or persistent side rail with a bounded reading column.
- The same scroll-aware bottom bar at every width.
- An always-visible bottom bar at every width.

## Decision Outcome

Below 600 logical pixels, use the scroll-aware bottom navigation from 0012. At 600 pixels and above,
use a persistent 72-pixel side rail with the same destinations, symbols and unread state. Bound forum,
notification and conversation columns to 720 pixels and Campus to 1120 pixels. Keep one retained
navigator structure across both layouts. Rail scrolling must not update reading-header visibility;
page headers and compose actions retain their existing reading behavior.

Menus anchor to their source column and preserve its system safe-area offset. Each rail action exposes
its name, selected state and activation; persistent navigation follows the active route in semantics
order. Pushed details and editors remain above the shell. The avatar drawer, flat surfaces, shared
colors, Lucide icons and reading hierarchy from 0012 remain the navigation model. Domain-specific
identity, campus data and management decisions continue to govern their own boundaries.

This supersedes [0012](0012-unified-mobile-reading-navigation.md), specifically replacing its universal
root-control hiding rule with window-dependent navigation. It does not introduce a second backend or
change the single-binary forum deployment.

## Pros and Cons of the Options

- Responsive rail: preserves height and stable navigation on wide windows, at the cost of a narrow
  horizontal strip and a second layout to verify. Bounded columns can leave intentional empty gutters.
- Universal scroll-aware bottom bar: one arrangement, but loses vertical space before it hides and
  gives wide windows unnecessarily long reading lines without separate bounds.
- Permanent bottom bar: predictable navigation, but consumes the same scarce height in compact and
  wide windows and abandons the compact reading-space behavior.

## Links

- [Implemented mobile navigation](../product/mobile-experience.md#navigation-and-reading)
- [Mobile layout standard](../product/mobile-design-system.md#navigation-and-windows)
- [Adaptive window implementation](../../apps/mobile/packages/forum_app/lib/src/navigation/reading_window.dart)
- [Window and safe-area regression coverage](../../apps/mobile/packages/forum_app/test/adaptive_shell_test.dart)
