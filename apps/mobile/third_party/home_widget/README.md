# home_widget 0.10.0 compatibility copy

This is a runtime subset of upstream
[`home_widget-v0.10.0`](https://github.com/ABausG/home_widget/tree/9f2ff6e2cce03b02f66d3aaac1f0f276c5ce636a/packages/home_widget).
The mobile workspace selects this copy with its `home_widget` dependency override.
The import in [`27373070`](https://github.com/YourTongji/YourTJ-Hub/commit/27373070a4f0aca1709fafc07ede15c779b5eec6)
keeps the plugin source, platform metadata, and license rather than the upstream examples,
tests, and repository scaffolding.

## Local patches

| Patch | Where and why |
| --- | --- |
| Standalone dependency override | The vendored `pubspec.yaml` omits upstream monorepo `resolution: workspace` so Flutter resolves this package as a local dependency here. The import made this change. |
| iOS 15 deployment metadata | The podspec and SwiftPM manifest set iOS 15, matching the current Runner and Widget Extension targets. See [`f1cde41a`](https://github.com/YourTongji/YourTJ-Hub/commit/f1cde41a). |
| Apple privacy resources | `PrivacyInfo.xcprivacy` is local ([`b26838f6`](https://github.com/YourTongji/YourTJ-Hub/commit/b26838f6)); SwiftPM processes it and CocoaPods bundles it separately from Swift sources. |
| Older iOS reload compatibility | Where WidgetKit is unavailable, `updateWidget` returns `false` rather than a Flutter error, preserving the cache-clear/login flow that originally needed iOS 13 compatibility. The Swift change and Dart API comment are from [`c127ccaf`](https://github.com/YourTongji/YourTJ-Hub/commit/c127ccaf). This fallback does not lower the current deployment target. |
| Android `mergeAndArmNext` | `HomeWidgetScheduler.mergeAndArmNext` unions new times into the stored schedule instead of replacing it, so the NextClass receiver can contribute countdown display boundaries without dropping the bridge's full schedule (including the +1ms stale switch). Added for [#1648](https://github.com/YourTongji/YourTJ-Hub/pull/1648); upstream `schedule` remains replace-semantics. |

The Android scheduler, its Dart scheduling/cancellation/exactness APIs, the Glance receiver's
state refresh, and the Android `setAppGroupId` no-op are already part of this upstream tag.
They are not local patches, except for `mergeAndArmNext` listed above. The app's timetable
widgets rely on that scheduler, so upstream updates must still preserve the app-side
registration and scheduling contract.

## Update checklist

- Diff the new upstream package against this runtime subset. Preserve the local patches above
  unless equivalent upstream behavior is verified; keep the upstream license.
- Check both SwiftPM and CocoaPods privacy-resource packaging, iOS deployment compatibility,
  and the older-system reload fallback covered by `native_widget_structure_test.dart`.
- Verify Android timetable-widget scheduling and Glance updates with the new version. Review
  any receiver behavior changes separately with device evidence.
- Keep the local override until the upstream package satisfies these requirements. Update this
  inventory and upstream pin when the copy changes.
