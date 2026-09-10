# YourTJ launcher artwork

The source is the same cat/YourTJ vector mark used by Web:
`apps/gooseforum/resource/src/site/assets/logo.svg`. `source.json` records its digest so the native
icon regression check detects unsynchronized brand changes. The horizontal in-app wordmark is
not a launcher icon.

Run from `apps/mobile/packages/forum_app`:

```bash
# macOS: brew install librsvg; Linux: install librsvg2-bin.
# Keep the generator isolated: its cli_util constraint differs from workspace Melos.
dart pub global activate flutter_launcher_icons 0.14.4
dart run tool/generate_launcher_artwork.dart
flutter test test/launcher_icons_test.dart
```

The wrapper rasterizes the original vector, centers the complete mark on a 1024 px canvas, creates
opaque white-background and transparent/monochrome masters, then applies `flutter_launcher_icons.yaml`.
It preserves Xcode project settings because generator 0.14.4 misidentifies new ASSETCATALOG boolean
settings as icon names. Use the wrapper instead of invoking that generator directly.

- iOS exports every AppIcon slot, including the opaque 1024 px App Store icon. Corners are masked
  by the OS, not pre-rounded into the artwork.
- Android exports all five legacy densities, adaptive foreground/background and Android 13 themed
  monochrome layers. The 16% foreground inset keeps the mark inside the adaptive safe circle;
  both normal and round launcher references resolve to this icon.
- Masters live outside Flutter's bundled asset list; they are build/design inputs only.

An icon change requires rebuilding/installing the app and a new Apple build for store distribution.
Changing repository PNGs does not replace the icon in an already uploaded or installed binary.
