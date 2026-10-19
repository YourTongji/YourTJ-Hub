# Material interaction surfaces on iOS

## Status

Accepted
Class: simplification

## Context and Problem Statement

[0095](0095-adaptive-ios-interaction-surfaces.md) made Gf menus, action sheets, dropdowns, radio
rows, simple confirmations, switches and sliders switch to Flutter Cupertino presentation on iOS.
After it shipped in TestFlight DEV builds, maintainers asked to roll this presentation back: the
app language and feed display-mode choices became Cupertino sheets that no longer matched the rest
of the app, and on wide windows (iPad) the anchored Cupertino menu was positioned from the trigger's
left edge and could cover neighbouring content, a likely source of the offset a forum user
reported.

## Decision Drivers

- Return every platform to the Material presentation used before 0095.
- Keep the tablet workspaces, reading windows and wide-window dialogs from 0095.
- Keep later product changes that build on the shared action values (for example feed display-mode
  glyphs).
- Keep the change reviewable and revertible.

## Considered Options

- Remove only the iOS branches inside the Gf components, keeping their Material paths and APIs.
- Revert the whole 0095 change set, including tablet workspaces.
- Keep Cupertino presentation and only fix wide-window menu placement.

## Decision Outcome

Gf components keep their API but render Material on every platform: `GfPopupMenuButton` is a
`PopupMenuButton`, `showGfMenu` uses `showMenu` at its anchor, `showGfActionSheet` uses the shared
Gf bottom sheet, `GfDropdownButton(FormField)` and `GfRadioListTile` wrap their Material
counterparts, and `showGfAlertDialog` / `GfAlertDialog` / `GfDialogAction` use the Gf `Dialog`
surface with Material buttons. Sliders and the course-review anonymity switch return to Material,
and the theme no longer carries a Cupertino override. The app language and feed display-mode
pickers return to their previous Material bottom sheets; the display-mode sheet keeps its leading
glyphs. Cupertino use that predates 0095 (iOS page transitions and back gesture, the iOS loading
indicator, Cupertino localizations, adaptive text-selection toolbars and existing `.adaptive`
switches) is unchanged.

The tablet workspace, reading-window and rail decisions in 0095 continue to apply.

## Pros and Cons of the Options

- Removing only the iOS branches keeps call sites and tablet layouts intact and makes the change
  small to review, but leaves inert anchor plumbing at a few call sites.
- Reverting all of 0095 also removes unrelated tablet work that maintainers want to keep.
- Fixing only placement keeps the Cupertino style that maintainers asked to remove.

## Links

- [0095](0095-adaptive-ios-interaction-surfaces.md) — iOS interaction surfaces rolled back here;
  tablet layouts retained
- [Mobile interaction standard](../product/mobile-design-system.md)
- [Mobile experience](../product/mobile-experience.md)
- [PR 1170](https://github.com/YourTongji/YourTJ-Hub/pull/1170)
