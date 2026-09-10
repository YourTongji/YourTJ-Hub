import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;

bool hasYourTjInk(img.Image image) => image.any(
  (pixel) =>
      pixel.r < 16 &&
      pixel.g >= 95 &&
      pixel.g <= 103 &&
      pixel.b >= 150 &&
      pixel.b <= 160 &&
      pixel.a == 255,
);

void main() {
  test(
    'icon generation preserves Xcode boolean settings and follows Web artwork',
    () {
      final project = File(
        'ios/Runner.xcodeproj/project.pbxproj',
      ).readAsStringSync();
      expect(
        project,
        isNot(
          contains(
            'ASSETCATALOG_COMPILER_GENERATE_SWIFT_ASSET_SYMBOL_EXTENSIONS = AppIcon',
          ),
        ),
      );
      final source =
          jsonDecode(File('assets/launcher/source.json').readAsStringSync())
              as Map;
      expect(
        sha256
            .convert(File(source['source'] as String).readAsBytesSync())
            .toString(),
        source['sha256'],
        reason: 'Regenerate launcher icons after changing the Web mark',
      );
    },
  );

  test(
    'every iOS launcher slot uses opaque YourTJ artwork at its declared size',
    () {
      final root = Directory('ios/Runner/Assets.xcassets/AppIcon.appiconset');
      final manifest =
          jsonDecode(File('${root.path}/Contents.json').readAsStringSync())
              as Map;
      for (final entry in manifest['images'] as List) {
        final icon = img.decodePng(
          File('${root.path}/${entry['filename']}').readAsBytesSync(),
        )!;
        final size =
            (double.parse((entry['size'] as String).split('x').first) *
                    double.parse(
                      (entry['scale'] as String).replaceAll('x', ''),
                    ))
                .round();
        expect(
          (icon.width, icon.height),
          (size, size),
          reason: entry['filename'],
        );
        expect(
          hasYourTjInk(icon),
          isTrue,
          reason: '${entry['filename']} still has placeholder artwork',
        );
        expect(
          icon.every((pixel) => pixel.a == 255),
          isTrue,
          reason: 'iOS icons must be opaque',
        );
      }
    },
  );

  test('Android legacy densities and adaptive launcher use the brand mark', () {
    const densities = {
      'mdpi': 48,
      'hdpi': 72,
      'xhdpi': 96,
      'xxhdpi': 144,
      'xxxhdpi': 192,
    };
    for (final entry in densities.entries) {
      final icon = img.decodePng(
        File(
          'android/app/src/main/res/mipmap-${entry.key}/ic_launcher.png',
        ).readAsBytesSync(),
      )!;
      expect((icon.width, icon.height), (entry.value, entry.value));
      expect(
        hasYourTjInk(icon),
        isTrue,
        reason: '${entry.key} still has placeholder artwork',
      );
    }
    final adaptive = File(
      'android/app/src/main/res/mipmap-anydpi-v26/ic_launcher.xml',
    ).readAsStringSync();
    expect(adaptive, contains('<adaptive-icon'));
    expect(adaptive, contains('ic_launcher_foreground'));
    expect(adaptive, contains('ic_launcher_monochrome'));
    expect(adaptive, contains('android:inset="16%"'));
  });
  test(
    'adaptive artwork fits the safe circle and supplies a matching monochrome mask',
    () {
      const directory = 'android/app/src/main/res/drawable-xxxhdpi';
      final foreground = img.decodePng(
        File('$directory/ic_launcher_foreground.png').readAsBytesSync(),
      )!;
      final monochrome = img.decodePng(
        File('$directory/ic_launcher_monochrome.png').readAsBytesSync(),
      )!;
      expect(
        (foreground.width, foreground.height),
        (monochrome.width, monochrome.height),
      );
      final safeRadius = foreground.width * 66 / 108 / 2;
      for (final pixel in foreground) {
        expect(monochrome.getPixel(pixel.x, pixel.y).a, pixel.a);
        if (pixel.a == 0) continue;
        final dx = (pixel.x + .5 - foreground.width / 2) * .68;
        final dy = (pixel.y + .5 - foreground.height / 2) * .68;
        expect(
          dx * dx + dy * dy <= safeRadius * safeRadius,
          isTrue,
          reason: 'Brand artwork must survive circular adaptive masks',
        );
      }
      expect(
        monochrome
            .where((pixel) => pixel.a > 0)
            .every((pixel) => pixel.r == pixel.g && pixel.g == pixel.b),
        isTrue,
      );
      final manifest = File(
        'android/app/src/main/AndroidManifest.xml',
      ).readAsStringSync();
      expect(manifest, contains('android:roundIcon="@mipmap/ic_launcher"'));
    },
  );
}
