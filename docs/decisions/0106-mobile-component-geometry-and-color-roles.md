# Mobile component geometry and derived color roles

## Status
Proposed
Class: architecture

## Context and Problem Statement

The native Gf foundation owns controls, but pages and components repeat radius, opacity, spacing and
text-size values. Shared Web tokens alone do not express grouped mobile forms, floating surfaces or
small controls with larger touch targets. A visual change needs one set of mobile roles without
forking the shared palette or replacing Flutter input and focus behavior.

## Decision Drivers

- Preserve the shared palette mirror and runtime site colors.
- Keep native selection, IME, autofill, focus and Material interaction behavior.
- Make grouped forms, reading surfaces and overlays recognizable in both themes.
- Separate painted control size from minimum touch targets and text scaling.
- Preview component compositions without production accounts or API data.

## Considered Options

- Add mobile geometry and derived color roles inside the existing ui_kit.
- Change the shared Web/mobile token values to serve every native surface.
- Continue per-page styling or adopt a second component framework.

## Decision Outcome

Use a mobile role layer in gf_design.dart with spacing, radius, control, glyph and elevation roles,
a named UI typography ramp, and color roles derived from the active GfColors. Shared palette tokens
and their Web mirrors remain unchanged. Material defaults and Gf wrappers consume the same roles.
Semantic foregrounds account for the soft fill and neutral host surface; export-card themes keep
separate contrast rules. Rich reading keeps its own size preference.

Inset form groups use GfListSection. A standalone gallery demonstrates actual components and synthetic
sample screens, with route-selectable theme and scroll state. The gallery supplements the existing
production-page simulator acceptance; no screenshot baselines are introduced.

## Pros and Cons of the Options

- A ui_kit layer makes mobile styling consistent and keeps existing Flutter behavior. It adds a small
  vocabulary beside the Web token mirror, so contributors must choose the appropriate layer.
- Changing shared tokens reduces vocabulary but changes Web geometry and cannot express native-only
  grouping or surface roles without affecting unrelated consumers.
- Per-page styling is initially cheap but leaves inconsistent states and repeated fixes. A second
  framework introduces another theme and input contract behind the Gf API.

## Links

- [Native Gf foundation](0039-native-gf-component-foundation.md)
- [Visual acceptance](0047-mobile-visual-acceptance.md)
- [Mobile component roles](../product/mobile-design-system.md#mobile-component-roles)
- [Component gallery](../../apps/mobile/packages/ui_kit/example/README.md)
