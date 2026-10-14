# Mobile editor caret following and body-image height cap

## Status

Accepted
Class: feature

## Context and Problem Statement

The publish page hosts the article editor with `scrollable: false` inside the page's
`SingleChildScrollView`; the editor body never scrolls itself, and flutter_quill's own
caret reveal (`_showCaretOnScreen`) is a no-op in that shape because it requires
`scrollable: true` or an attached `scrollController`. A caret past the fold was therefore
only reachable by manual scrolling (#1122). The course-review sheet instead uses
`fill: true`, where the editor owns an internal scroll view and already follows the caret;
simple text bodies rely on Flutter's `EditableText`.

The same report flagged body images: `_ComposerImageBuilder` and `_MarkdownImageBuilder`
set only `BoxFit.contain` plus `cacheWidth`, so Quill's inline embed layout bounds the
width but leaves the height unbounded. A portrait photo renders at its full intrinsic
height at editor width, and the block height is what pushes the caret away in the first
place. The scaling strategy was left as a maintainer decision with three options.

## Decision Drivers

- The caret line stays visible while typing a long article, including when the software
  keyboard shrinks the viewport under a stationary caret.
- Fill-mode hosts keep their pinned toolbar and internal scrolling; only the host that
  hands scrolling to the page may drive the page.
- Body-image blocks stay predictable and close to the reading surfaces: the mobile reader
  caps images at `min(360px, 70vh)` and web prose uses `max-width: 100%` with
  `max-height: min(360px, 70vh)`.
- No new dependency, and behavior expressible in widget tests without a live keyboard.

## Considered Options

- Caret: drive the host scrollable directly from an editor-owned follower widget.
- Caret: rely on `RenderEditable.showOnScreen` propagation to the ancestor viewport.
- Caret: hand flutter_quill a `ScrollController` that has clients in non-fill hosts.
- Image: fill the available width, cap the height at a fraction of the viewport (chosen).
- Image: scale to width only, no height cap.
- Image: small centered thumbnail with tap-to-enlarge editing.

## Decision Outcome

Caret: `EditorCaretFollower` wraps the non-fill editor. On selection/document changes and
on viewport-metrics changes (keyboard open/close), it measures the caret line through
`EditorState.renderEditor`, converts it into the nearest vertical `Scrollable`'s
coordinates, and animates the shortest scroll that keeps the line inside the viewport with
a 12 px top/bottom inset (120 ms, `fastOutSlowIn` — the same feel as flutter_quill's own
follow). After a completed animation it re-measures once, because content above the caret
can grow a frame later (the publish page's draft-status row appears in a post-frame
callback). User drags interrupt the animation and are never overridden. The publish page
and the default non-fill `_QuillBody` use it; `fill: true` hosts keep flutter_quill's
internal scrolling, and the simple text body keeps `EditableText`'s built-in following.

Image: `EditorEmbedImage` renders every editor body image (composer embeds and Markdown
image embeds, both editor hosts): `ConstrainedBox(maxHeight: 0.6 × viewport height)` around
a `GfNetworkImage(fit: BoxFit.contain, cacheWidth: screen width × dpr)`, centered with a
tight height (an `Align` with `heightFactor: 1`). Images keep their aspect ratio, fill the
editor width when wider than it, and shrink under the cap when taller; small originals are
not upscaled by the layout. The reading views keep their existing `min(360px, 70vh)` cap.

## Pros and Cons of the Options

- Follower widget: deterministic on every platform and testable; adds one small stateful
  wrapper per editor and duplicates nothing that flutter_quill already does for fill mode.
  It must not fight manual scrolling, hence the focus gate, the interrupt check and the
  single settle re-measure.
- `showOnScreen` propagation: no extra code, but it depends on scroll physics
  (`allowImplicitScrolling`) and scrolling policy rather than the caret contract, and it
  cannot respect a margin or coalesce per frame.
- Attached `ScrollController`: impossible without giving the editor its own scroll view —
  exactly the layout change the publish page avoids (the page scrolls as one surface).
- Fill width + height cap: matches the reader/web intent and fixes both the block height
  and the caret distance; deliberately changes how existing drafts with portrait photos
  look after restore (they render shorter).
- Width-only scaling: leaves tall photos dominating the page and keeps caret jumps large.
- Thumbnail + tap-to-enlarge: smallest block height but adds a preview interaction the
  issue did not ask for and diverges from the reading surfaces.

## Links

- [Issue #1122](https://github.com/YourTongji/YourTJ-Hub/issues/1122).
- [Editor caret follower](../../apps/mobile/packages/forum_app/lib/src/widgets/editor/editor_caret_follow.dart).
- [Editor embed image](../../apps/mobile/packages/forum_app/lib/src/widgets/editor/editor_embed_image.dart).
- [Mobile experience — publishing](../product/mobile-experience.md).
