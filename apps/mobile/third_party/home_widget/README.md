# home_widget 0.10.0 compatibility copy

This copy keeps the upstream Android, iOS, and Dart implementation. Its platform metadata uses
an iOS 15 minimum, matching the Flutter Runner and Widget Extension. The local resource patch
bundles a privacy manifest in both SwiftPM and CocoaPods resources.

Upstream: https://github.com/ABausG/home_widget

Keep the deployment metadata and privacy-resource patches when updating this copy. Remove the
local override when upstream includes the privacy manifest in both package managers and its
minimum is compatible with the app's supported iOS versions.
