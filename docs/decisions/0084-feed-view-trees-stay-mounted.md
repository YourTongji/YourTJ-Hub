# Feed list and card views stay mounted and switch by display

## Status

Accepted
Class: architecture

## Context and Problem Statement

The home feed's list/card switch measured INP 263 ms (153 ms processing + 109 ms presentation,
issue #1164) on a 20-topic first page. `TopicList.vue` rendered the table rows (`TopicRow`) and the
cards (`TopicFeedPreview` + `TopicCardActions`) as mutually exclusive `v-if`/`v-else` branches. The
two branches produce different vnode types at the same keys, so Vue cannot reuse anything across
them: every switch unmounts one whole tree and mounts the other (measured 0/20 row nodes surviving,
55 ms of script on the first switch to card, 25-33 ms on later switches) including each row's
component state, hover-preview registration and Teleport cleanup.

Both views must stay: the table is the desktop reading shape, the card is the mobile default
(`readFeedMode` returns `card` below the `lg` breakpoint) and the explicit target of the switch
control; the hover preview popover only exists on table rows.

## Decision Drivers

- Switching must not rebuild the list: the 20-row rebuild is the bulk of the INP processing time.
- First paint must not regress: the home page already has an open LCP problem (#1163); keeping both
  trees mounted unconditionally raised mount cost from 60 ms to 101 ms in the fixture.
- Row lifecycle side effects (hover-preview registration, popover teardown, pending timers) must not
  leak or survive a switch into the hidden tree.
- Feed telemetry (`feed-telemetry`) and For You anchors (`for-you-sessions`) query `[data-feed-id]`
  in document order; duplicate row nodes would make them match `display:none` nodes whose rects
  are all zero.
- List-only pages (search, category, moderation, user profiles) have no switch control and must not
  pay for a second tree.

## Considered Options

- Keep the exclusive branches; only lazy-load card images and slim the row components.
- Keep both trees mounted, switch with `v-show`, prebuild the second tree while idle (chosen).
- Wrap each topic in one keyed container component that switches layout inside.
- Accept the current behavior and record it as a known limitation.

## Decision Outcome

`TopicList` renders both view trees once and switches them with `v-show`. `tableMounted` /
`cardMounted` gate each tree with `v-if`; the tree that is not the initial mode mounts later — on a
`switchable` list, during a `requestIdleCallback` (timeout 2000 ms, `setTimeout` 300 ms fallback,
cancelled on unmount) so first paint still renders only the active view; otherwise when the mode
first changes, which is the lazy fallback for a click before prebuild finishes.

The hidden tree drops `data-feed-id` / `data-feed-trace` (`feedMode`-conditional bindings), so
telemetry and For You anchors only ever see the visible rows. Mode changes call
`closeAllTopicPreviews()` (new export in `topic-hover-preview.ts`) because hiding rows does not
unmount them. `HomePage` passes `switchable`; other list pages keep the default and render a single
tree.

Measured on the fixture (`count=20`, Playwright Chromium, headed): first switch to card after an
expanded hover preview — script 42.5 → 13.0 ms, task 50.5 → 20.5 ms; steady-state switches
22-29 → 7-16 ms; row nodes surviving the switch 0/20 → 20/20 (headless, software rasterization:
script 56 → 24 ms, task 134-150 → 87-100 ms). First mount of a home list stays at ~60-70 ms
because the card tree is built while idle, not at mount.

## Pros and Cons of the Options

- Both trees mounted (chosen): switching only patches `display`, so the list DOM, row component
  instances and their hover-preview registration survive; the cost is a permanently doubled list
  DOM (~818 extra nodes on the home page) plus the idle prebuild work, and the hidden tree needs
  the attribute strip so telemetry/anchors keep their single-match semantics.
- Exclusive branches with lazy images / slimmer rows: smaller diff and no duplicate DOM, but the
  measured root cause (unmount + mount of the whole tree, ~25 ms steady script) stays, and every
  row's preview registration and Teleport teardown still runs on each switch.
- One keyed container per topic with layout switched inside: keeps one component instance per
  topic, but the inner table/card structures still diff to different vnodes, so most of the rebuild
  remains; it also adds a wrapper level between the list container and the row grid.
- Accept as known limitation: the home page is the highest-traffic entry point and the switch is a
  primary control; recording it as a limit leaves a measured 263 ms interaction in place.

## Links

- [Issue #1164](https://github.com/YourTongji/YourTJ-Hub/issues/1164).
- [TopicList.vue](../../apps/gooseforum/resource/src/site/components/TopicList.vue).
- [Topic hover preview](../../apps/gooseforum/resource/src/runtime/topic-hover-preview.ts).
- [Feed telemetry](../../apps/gooseforum/resource/src/runtime/feed-telemetry.ts).
- [Regression test](../../apps/gooseforum/resource/test/topic-list-view-switch.test.ts).
