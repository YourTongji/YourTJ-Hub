# iOS 15 deployment floor for iPhone Duo

## Status

Accepted
Class: architecture

## Context and Problem Statement

Full-screen iPhone Duo presentation requires the iOS 27.1 SDK. Xcode 27.1 supports deployment
targets starting at iOS 15 and rejects the repository's iOS 13 Runner, iOS 14 widget target and
older CocoaPods target settings during a simulator build. Retaining those minimums prevents
building the Duo candidate with the supported toolchain.

## Decision Drivers

- Build the Flutter app with the SDK required for full-screen Duo presentation.
- Keep one supported source and deployment floor for the app, widget extension and local bridge.
- Make the loss of iOS 13/14 compatibility explicit.
- Preserve the existing Flutter navigation, state ownership and native widget behavior.

## Considered Options

- Use an iOS 15 minimum for the app and widget, and build Duo candidates with Xcode 27.1 or later.
- Retain iOS 13/14 minimums and build with an older Xcode.
- Maintain separate legacy and Duo build configurations and release lines.

## Decision Outcome

The Runner and ScheduleWidgets extension use iOS 15 in every build configuration. The local
home_widget SwiftPM and CocoaPods metadata use the same minimum. CocoaPods raises older pod
build minimums to 15 without lowering a dependency's higher requirement. The local home_widget
copy remains because it also owns privacy-manifest packaging; raising the minimum alone does
not justify removing that patch.

New builds do not support devices still running iOS 13/14. This replaces the compatibility goal
of keeping the iOS 13 Runner when selecting the Duo toolchain. The Flutter presentation and
retained-window composition in [0095](0095-adaptive-ios-interaction-surfaces.md) remain applicable;
this decision changes platform support, not their widget/state ownership.

Duo acceptance requires evidence from the actual simulator or device, including window transitions
and asymmetric safe areas. Synthetic size/inset regression tests do not establish native acceptance.
The release host must be checked for the required SDK; this decision does not assert that the
existing unpinned CI runner has already adopted it.

## Pros and Cons of the Options

- iOS 15 gives the supported new toolchain a consistent deployment floor and avoids parallel builds,
  but excludes iOS 13/14 from new releases and still requires native Duo validation.
- An older Xcode preserves the older build settings, but cannot provide the SDK behavior needed
  for Duo full-screen presentation.
- Separate release lines can retain legacy support, but duplicate build, dependency and release
  validation without a demonstrated user need for that ongoing maintenance.

## Links

- [Apple Xcode SDK and deployment support](https://developer.apple.com/xcode/system-requirements) —
  Xcode 27.1 supports iOS 15–27 deployment targets; consulted 2026-10-26.
- [Apple: Prepare your app for iPhone Duo](https://developer.apple.com/videos/play/tech-talks/111461/) —
  SDK, local geometry and safe-area requirements; consulted 2026-10-26.
- [Mobile experience](../product/mobile-experience.md#navigation-and-reading)
- [Mobile release requirements](../operations/mobile-releases.md#ios-build-requirements)
