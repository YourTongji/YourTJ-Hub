# Repository-owned native Gf component foundation

## Status
Accepted
Class: architecture

## Context and Problem Statement

The mobile app needs a consistent visual language across messaging, replies, publishing, search
and account forms. A third-party component skin adds a separate theme and interaction surface
behind the existing Gf API. Native input capabilities and touch geometry need to remain under
repository control while each writing context uses an appropriate shape.

## Decision Drivers

- Preserve Flutter text selection, IME composition, autofill, focus and accessibility.
- Keep a single repository-owned component API and semantic palette.
- Distinguish short chat entry, form validation and long-form writing without inconsistent skins.
- Keep visual density independent of minimum touch targets and support larger system text.
- Remove the pre-release component dependency and its picker compatibility overrides.

## Considered Options

- Compose Flutter primitives inside the existing Gf component API.
- Retain TDesign and override its theme and individual component layouts.
- Rebuild every control with low-level gesture and painting implementations.

## Decision Outcome

Gf components compose Flutter primitives directly. The app does not depend on TDesign. Shared
ThemeData owns native input, selection, button, dialog and menu defaults; Gf components define
context-specific shapes and preserve their public APIs. Ordinary forms use filled rounded fields,
search/chat use capsules, replies use a growing field and flat toolbar, and publishing retains an
open writing canvas. Native buttons, dialogs and focus handling preserve keyboard interaction.

Semantic colors continue to come from the shared token baseline. Native component geometry is
owned by `ui_kit`; this decision does not change the Web/mobile token mirror. Runtime image and
SVG rendering remain specialized dependencies. Maintaining Gf component behavior and regression
coverage becomes a repository responsibility.

## Pros and Cons of the Options

- Flutter composition provides control of shape and state while retaining platform input and
  accessibility. It requires maintenance of the Gf layout and behavior wrappers.
- TDesign overrides reuse a broad component catalog, but retain an additional pre-release theme
  layer and can leave inconsistent behavior or default visuals at component boundaries.
- Custom painting offers complete visual control, but duplicates native focus, selection,
  semantics and input behavior for no product benefit.

## Links

- [UI kit implementation](../../apps/mobile/packages/ui_kit/README.md)
- [Mobile input behavior](../product/mobile-experience.md#input-and-component-surfaces)
- [Mobile interaction standard](../product/mobile-design-system.md)
