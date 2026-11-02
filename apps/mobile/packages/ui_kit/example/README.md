# Gf component gallery

A standalone developer preview of the production `ui_kit` components, with synthetic sample screens.
It has no forum session, production API client or stored user data. Sample photos use public image
services; offline failures use each component's ordinary fallback.

From this directory:

```sh
flutter create --platforms=ios --project-name ui_kit_gallery --org de.yourtj .
flutter pub get
flutter run -d <device-id>
flutter run -d <device-id> --route '/settings?theme=dark&y=640'
```

The index lists foundation, buttons, inputs, feedback, lists, overlays and content sections,
plus the sample screens. `theme=light|dark` and `y=<logical-pixel-offset>` select capture states.
The theme toggle remains available in the index. Native preview runners are generated locally and
ignored by Git; other platforms can be generated with `flutter create --platforms=macos,android .`.

Use the real forum pages for workflow acceptance: sample screens demonstrate component composition
and do not prove API, navigation or save/retry behavior. Keep screenshots outside Git, following
[mobile verification](../../../../../docs/development/testing.md).
