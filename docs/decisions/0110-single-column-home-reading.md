# Single-column Home reading

## Status

Accepted
Class: simplification

## Context and Problem Statement

Home's wide layout places a second Hot topic list and search entry beside the active
feed. On an unfolded Duo, the two lists compete for attention and repeat controls.
Readers need a coherent stream when they fold or unfold the window; Hot already has
an entry in the persistent sort tabs.

## Decision Drivers

- Prioritize the active stream and readable line lengths.
- Keep search, categories, Hot and reading position available across window changes.
- Preserve side-pocket glass navigation and independent system safe areas.
- Avoid device-name checks and additional feed requests solely for decoration.

## Considered Options

- Keep two columns with a smaller Hot card and one search entry.
- Use one centered, bounded feed at every window width.
- Stretch one feed across the entire expanded window.

## Decision Outcome

Home uses one centered reading column up to 720 logical pixels wide. Its header,
sort tabs, category row and stream share that column; a neutral canvas fills the
outer gutters. Search appears once in the header, and Hot uses its retained sort.
There is no auxiliary list or width-triggered Hot prefetch. Centering consumes the
part of each system side inset covered by the gutter, so controls receive only the
inset that reaches the column. Compact edge-to-edge reading beneath the glass
capsule continues to apply.

This replaces Home's composition in
[0095](0095-adaptive-ios-interaction-surfaces.md). Its other window rules remain:
settings, Messages, course and Wiki workspaces retain their contextual panes;
pushed pages keep bounded reading windows; navigation stays independent of scroll.
[0100](0100-material-interaction-surfaces-on-ios.md) continues to own Material
transient surfaces. No routes, feed ordering or account permissions change.

## Pros and Cons of the Options

- A smaller Hot card retains simultaneous discovery, but still creates two competing
  topic streams and an extra request when the window expands.
- One bounded stream preserves the phone reading model and removes duplicate
  presentation. It deliberately uses side space as breathing room instead of
  showing more topics at once.
- A full-width stream fills all available space, but spreads metadata and text
  across long rows and makes the layout change more disruptive.

## Links

- [Apple layout guidance](https://developer.apple.com/design/human-interface-guidelines/layout)
  — hierarchy, alignment and adaptable content; consulted 2026-10-26.
- [Mobile experience](../product/mobile-experience.md#navigation-and-reading)
- [Adaptive reading window](0037-adaptive-mobile-reading-window.md)
- [Duo adaptation PR](https://github.com/YourTongji/YourTJ-Hub/pull/1407)
