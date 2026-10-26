# Adaptive iOS interaction surfaces

## Status

Superseded by [0110](0110-single-column-home-reading.md)
Class: architecture

The successor replaces only Home's two-column composition and retains the other
window-layout rules. [0100](0100-material-interaction-surfaces-on-ios.md) owns the
Material presentation of transient surfaces.

## Context and Problem Statement

The Flutter app shares Gf components across Android and iOS, but Material popup menus, dropdowns
and confirmations also appear on iPhone and iPad. A wide window needs bounded actions near their
trigger, while a compact window needs an accessible short action surface. Pushed reading and form
pages need readable line lengths without replacing route or editor state. A tablet workspace also
needs useful navigation beside its content, rather than a wider copy of a phone page.

## Decision Drivers

- Preserve action values, permissions, form validation, cancellation and draft recovery.
- Follow current window geometry, including iPad split windows and rotation.
- Retain Flutter input, focus, accessibility and the existing Gf theme.
- Avoid UI dependencies, a native app rewrite or a higher minimum iOS version.

## Considered Options

- Adapt the existing Gf API with Flutter Cupertino primitives and bounded page composition.
- Keep Material presentation on every platform and only adjust spacing.
- Replace the app's interaction surfaces with UIKit or a third-party platform UI framework.

## Decision Outcome

Gf owns platform presentation. iOS short action menus and single choices use Cupertino actions with
an independent cancellation action, a visible current choice and explicit destructive actions. Wide
windows place bounded menus near their trigger; compact windows present action sheets. Android keeps
Material controls and the same action values. Simple confirmations use Cupertino dialog routes and
content on iOS, and answer only through their own actions: a barrier tap never cancels on either
platform, so a confirmation cannot dismiss itself by accident. Complex credential and editing forms explicitly retain the bounded Gf dialog surface,
keyboard avoidance and their existing save/error semantics.

Pushed reading and form routes use a centered reading window without an additional navigator. The
wrapper remains in the widget tree across width changes, consumes horizontal safe areas once and
gives the page its actual column width. Home keeps category tabs above its retained feed and uses
the right column for search and existing Hot topics; settings uses a directory beside its current section.
Messages pairs its directory with the selected chat, and course and Wiki selectors stay beside their content.
These workspaces, search and publishing can use
1120 pixels, while individual reading surfaces retain their reading bounds. Compact windows or
enlarged text fall back to a single content pane. This supersedes the universal 720-pixel Home
bound in [0037](0037-adaptive-mobile-reading-window.md); its retained navigator and 600-pixel
root navigation breakpoint continue to apply. The root rail displays destination labels and measures
them at the current system text size. It uses a 220-pixel sidebar with icon-and-label rows when the
window is at least 1100 pixels wide and text scale is no greater than 1.4; otherwise its width grows
from 72 to 160 pixels before labels wrap. Text is not scaled down. Login fills the window with a
continuous background while its form stays bounded. Image viewing, the
scheduler and authenticated management keep their full viewport. The component foundation in
[0039](0039-native-gf-component-foundation.md) and visual acceptance in
[0047](0047-mobile-visual-acceptance.md) continue to apply.

This uses Flutter's Cupertino appearance, not native UIKit presentation or a Liquid Glass API.

## Pros and Cons of the Options

- Gf adaptation preserves the existing caller boundary and platform input, but the repository owns
  trigger placement, cancellation and layout regression coverage.
- Material spacing adjustments have the smallest initial diff, but leave the platform interaction
  mismatch and wide-window presentation unresolved.
- UIKit or a third-party UI framework can provide more platform-specific surfaces, but duplicates
  state/focus ownership or adds a dependency without solving a missing business capability.

## Links

- [Apple popovers](https://developer.apple.com/design/human-interface-guidelines/popovers/) — wide
  versus compact presentation; consulted 2026-10-13.
- [Apple action sheets](https://developer.apple.com/design/human-interface-guidelines/action-sheets)
  — cancellation and destructive action semantics; consulted 2026-10-13.
- [Flutter CupertinoActionSheet](https://api.flutter.dev/flutter/cupertino/CupertinoActionSheet-class.html)
- [Apple Designed for iPad](https://developer.apple.com/videos/play/wwdc2020/10206/) —
  navigation beside content and compact-window fallback; consulted 2026-10-13.
- [Flutter adaptive design](https://docs.flutter.dev/ui/adaptive-responsive/best-practices) —
  use available window space for content and navigation; consulted 2026-10-13.
- [Flutter adaptive dialogs](https://api.flutter.dev/flutter/material/showAdaptiveDialog.html)
- [Mobile interaction standard](../product/mobile-design-system.md)
- [Mobile experience](../product/mobile-experience.md)
- [Issue 1165](https://github.com/YourTongji/YourTJ-Hub/issues/1165)
- [0100](0100-material-interaction-surfaces-on-ios.md) — the iOS Cupertino interaction surfaces in
  this record were rolled back to Material; the tablet layouts continue to apply
