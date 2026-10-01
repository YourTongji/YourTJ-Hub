# Flutter optical glass for mobile controls

## Status

Proposed
Class: architecture

## Context and Problem Statement

Mobile navigation, floating actions, search, composers and menus need a shared optical material.
A blur hidden behind an opaque fill cannot show the content beneath it. Independent native platform
views would split the existing Flutter navigation, gestures, text selection and modal focus model.
The app must remain readable when transparency, motion or contrast preferences change during editing.

## Decision Drivers

- Make functional layers visibly distinct while keeping posts and timetables readable.
- Preserve the retained navigation shell, draft controllers, account isolation and native editing.
- Keep a single material implementation with bounded rendering and no continuous animation.
- Offer explicit renderer and accessibility fallbacks without recreating content.

## Considered Options

- Keep opaque surfaces with background blur.
- Place native iOS Liquid Glass platform views behind Flutter controls.
- Render a shared Flutter optical material with an Impeller backdrop shader and solid/frosted fallbacks.

## Decision Outcome

Use `GfLiquidSurface` in the UI kit. Impeller applies Gaussian diffusion followed by a rounded lens
shader that displaces the actual backdrop near its rim. The rendering layer supplies scene bounds at
paint time so translated and scaled controls retain their lens geometry. The compiled shader program
is shared; mounted surfaces own and dispose their shader instances. No screenshots, GPU readback,
remote dependency or repeating ticker participates in the production material.

Regular glass carries navigation and search; stronger glass supports composers and sheets; a lighter menu role reveals
its context without weakening other controls. Profile selection uses a solid underline. Clear
controls use a dark veil over photographs. A clipped foreground keeps text and icons sharp. Highlights,
rims and external shadows define the control boundary. Post bodies and timetable cells remain ordinary
content surfaces. This is a custom Flutter treatment inspired by Liquid Glass, not Apple's native API
or a pixel-identical Telegram implementation. Component geometry is mobile-owned; semantic colors
continue to use the shared Web/mobile palette.

`ImageFilter.isShaderFilterSupported` gates the optical path. Other renderers and shader load failure
use Gaussian frost. High contrast, reduced motion/accessibility navigation, or Reduce Transparency
remove filtering and use an opaque fill. The app bridges iOS Reduce Transparency into `GfGlassSettings`
above the navigator, including live changes and resume refresh; a late initial response cannot override
a newer notification. Fallback changes leave text controllers, selection and focus mounted.

The shared component foundation in [0039](0039-native-gf-component-foundation.md) and retained window
model in [0037](0037-adaptive-mobile-reading-window.md) remain authoritative. Verification follows
[0047](0047-mobile-visual-acceptance.md): behavior/layout assertions, native Impeller integration,
and inspected device/simulator evidence outside Git. Simulator debug results do not establish device
frame-budget compliance.

## Pros and Cons of the Options

- Opaque blur is cheap to maintain, but cannot provide visible transmission or refraction.
- Native iOS material follows Apple's rendering, but introduces platform-view compositing and separate
  accessibility/gesture integration, and has no equivalent implementation on other renderers.
- Flutter optics preserve the current component and state owners across platforms. They require shader
  maintenance and device performance verification, and cannot claim Apple's proprietary rendering.
  Frost and solid fallbacks keep the product usable when optical rendering is unavailable.

## Links

- [Mobile material and geometry specification](../product/mobile-design-system.md#optical-control-material)
- [Mobile experience](../product/mobile-experience.md)
- [Apple material roles and accessibility](https://developer.apple.com/design/human-interface-guidelines/materials)
- [Flutter ImageFilter.shader requirements](https://api.flutter.dev/flutter/dart-ui/ImageFilter/ImageFilter.shader.html)
- [Flutter shader-filter support](https://api.flutter.dev/flutter/dart-ui/ImageFilter/isShaderFilterSupported.html)
