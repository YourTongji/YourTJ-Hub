# home_widget 0.10.0 compatibility copy

This copy keeps the upstream Android, iOS, and Dart implementation. Its only source change is lowering
the iOS package and CocoaPods deployment metadata to 13.0 so the Flutter Runner can retain its iOS 13
minimum; WidgetKit calls in the plugin are availability-guarded, and the app's Widget Extension remains
iOS 14.0.

Upstream: https://github.com/ABausG/home_widget

Keep the platform metadata patch when updating this copy. Remove the local override when upstream
supports an iOS 13 Runner directly.
