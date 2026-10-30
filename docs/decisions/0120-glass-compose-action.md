# Glass compose action and menu

## Status

Accepted
Class: feature

## Context and Problem Statement

The compact shell's dock and the wide rail float on one glass surface
(`GfNavigationSurface`), but the compose action beside them was a flat Material
`FloatingActionButton`, and its menu used opaque cards over a black scrim with a second flat FAB as
the close button. The floating layer therefore mixed two materials and two touch languages. The
action's place and behaviour (where it sits, how it follows hidden chrome, what the menu offers and
how it dismisses) are settled and must not change.

## Decision Drivers

- One floating-layer material and press language for the dock, the rail and the compose action.
- Glyphs and labels stay legible on any content, in light and dark themes (3:1 for the glyph).
- High contrast, reduced motion and accessible navigation keep the existing opaque fallback.
- No new dependency or rendering-engine requirement; position, keys and behaviour unchanged.

## Considered Options

1. Extend the shared `BackdropFilter` glass with a primary-tinted tone and a glass button that uses
   the dock's press spring.
2. Adopt a refraction renderer such as `liquid_glass_renderer` for true lens distortion.
3. Keep Material FABs and only make the menu cards translucent.

## Decision Outcome

Choose option 1.

- `GfNavigationSurface` gains `GfGlassTone.prominent`: the dock's blur and saturation under a
  primary tint (88% light, 84% dark, at least 3.3:1 for the glyph over white or black content), a
  top sheen, a rim lit at the top that brightens again along the bottom inside edge, as a thick lens
  focuses light, and an outer edge. The regular tone is unchanged. Without glass it is solid primary.
- `GfGlassButton` (ui_kit) is a capsule of that glass. A press swells the glass by the dock's 3 px
  with the dock's spring while the content dips slightly. A soft light gathers under the finger and
  follows it. Without glass a press tints the opaque surface. It has a keyboard focus ring,
  Enter/Space activation, and one merged button node with its tooltip or label.
- The root compose action is a prominent `GfGlassButton` in the same position. Menu items are
  regular-tone capsules that rise and grow from the action, nearest first, and leave together. The
  close button sits exactly over the action, visible from the first frame, and its plus turns into
  the close mark. The page behind is frosted (8 px blur) and dimmed less than a sheet (20% light,
  50% dark); the non-glass fallback keeps the theme scrim.

## Pros and Cons of the Options

### shared BackdropFilter glass (selected)

- Good: one material and spring for the whole floating layer; works on Skia and Impeller; no
  dependency; fallbacks and contrast are controlled by the existing tokens.
- Bad: no true refraction; the lens reading comes from sheen, rim and touch light only.

### refraction renderer

- Good: real lens distortion close to iOS 26.
- Bad: new dependency in pre-release, requires Impeller, unsupported on some targets; would make the
  action differ from the dock unless both migrate.

### translucent cards only

- Good: smallest change.
- Bad: the action stays a different material from the dock, which is the inconsistency to remove.

## Links

- [Mobile design system: navigation glass](../product/mobile-design-system.md)
- [Mobile experience: navigation and reading](../product/mobile-experience.md#navigation-and-reading)
- [Apple HIG: Materials](https://developer.apple.com/design/human-interface-guidelines/materials)
- [Material 3 FAB menu](https://m3.material.io/components/fab-menu)
