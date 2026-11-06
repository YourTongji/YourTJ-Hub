import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  final harmony = TargetPlatform.values.where(
    (platform) => platform.name == 'ohos',
  );

  test(
    'HarmonyOS preserves the existing mobile theme and text metrics',
    () {
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      for (final brightness in Brightness.values) {
        debugDefaultTargetPlatformOverride = TargetPlatform.android;
        final expected = gfThemeData(brightness);
        debugDefaultTargetPlatformOverride = harmony.single;
        final actual = gfThemeData(brightness);
        expect(actual.platform, expected.platform);
        expect(actual.colorScheme, expected.colorScheme);
        expect(actual.textTheme, expected.textTheme);
        expect(actual.inputDecorationTheme, expected.inputDecorationTheme);
      }
      for (final width in [320.0, 390.0, 820.0]) {
        expect(
          GfAppTextScaler.baselineFor(
            platform: harmony.single,
            shortestSide: width,
          ),
          GfAppTextScaler.baselineFor(
            platform: TargetPlatform.android,
            shortestSide: width,
          ),
        );
      }
    },
    skip: harmony.isEmpty ? 'Requires Flutter OH platform support' : false,
  );
}
