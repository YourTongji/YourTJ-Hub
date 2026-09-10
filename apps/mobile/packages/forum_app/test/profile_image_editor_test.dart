import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/profile_image_editor.dart';
import 'package:forum_app/src/widgets/profile_image_crop.dart';

ProfileCropSource stripes() {
  final image = img.Image(width: 30, height: 10);
  for (final pixel in image) {
    pixel.setRgb(
      pixel.x < 10 ? 255 : 0,
      pixel.x >= 10 && pixel.x < 20 ? 255 : 0,
      pixel.x >= 20 ? 255 : 0,
    );
  }
  return prepareProfileCrop(img.encodePng(image));
}

void main() {
  test(
    'dragged selection exports the visible pixels and clamps at image edges',
    () {
      final source = stripes();
      final centered = ProfileCropGeometry(
        source: const Size(30, 10),
        viewport: const Size(100, 100),
        zoom: 1,
        offset: Offset.zero,
      );
      final moved = ProfileCropGeometry(
        source: const Size(30, 10),
        viewport: const Size(100, 100),
        zoom: 1,
        offset: const Offset(-999, 999),
      );
      expect(centered.selection, const Rect.fromLTWH(10, 0, 10, 10));
      expect(moved.selection, const Rect.fromLTWH(20, 0, 10, 10));
      final avatar = img.decodeWebP(
        exportProfileCrop(
          ProfileCropExport(source, moved.selection, cover: false),
        ),
      )!;
      expect((avatar.width, avatar.height), (300, 300));
      final pixel = avatar.getPixel(150, 150);
      expect([pixel.r, pixel.g, pixel.b], [0, 0, 255]);
      final cover = img.decodeWebP(
        exportProfileCrop(
          ProfileCropExport(source, centered.selection, cover: true),
        ),
      )!;
      expect((cover.width, cover.height), (1600, 320));
    },
  );

  test('camera orientation is normalized before crop coordinates are used', () {
    final image = img.Image(width: 30, height: 10);
    image.exif.imageIfd.orientation = 6;
    final source = prepareProfileCrop(img.encodeJpg(image));
    expect((source.width, source.height), (10, 30));
    expect(
      () => prepareProfileCrop(Uint8List.fromList([1, 2, 3])),
      throwsFormatException,
    );
  });

  testWidgets(
    'failed upload retains the crop and successful retry closes the editor',
    (tester) async {
      final attempts = <Uint8List>[];
      final navigator = GlobalKey<NavigatorState>();
      await tester.pumpWidget(
        MaterialApp(
          navigatorKey: navigator,
          theme: gfThemeData(Brightness.light),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const Scaffold(body: Text('Profile')),
        ),
      );
      navigator.currentState!.push(
        MaterialPageRoute<void>(
          builder: (_) => ProfileImageEditor(
            source: stripes(),
            cover: false,
            onSave: (bytes) async {
              attempts.add(bytes);
              if (attempts.length == 1) throw StateError('offline');
            },
          ),
        ),
      );
      await tester.pumpAndSettle();
      // Reveal the blue stripe at the right edge. The crop remains blue on retry.
      await tester.timedDrag(
        find.byKey(const Key('profile-image-crop')),
        const Offset(-1000, 0),
        const Duration(seconds: 1),
      );
      await tester.pumpAndSettle();
      Future<void> saveAndWait(int count) async {
        await tester.runAsync(() async {
          await tester.tap(find.text('Save'));
          final deadline = DateTime.now().add(const Duration(seconds: 10));
          while (attempts.length < count && DateTime.now().isBefore(deadline)) {
            await Future<void>.delayed(const Duration(milliseconds: 20));
          }
        });
        await tester.pumpAndSettle();
        expect(attempts.length, count);
      }

      await saveAndWait(1);
      expect(find.byType(ProfileImageEditor), findsOneWidget);
      expect(find.text('Reset position'), findsOneWidget);
      await saveAndWait(2);
      expect(attempts.last, attempts.first);
      final avatar = img.decodeWebP(attempts.last)!;
      expect(avatar.getPixel(150, 150).b, 255);
      expect(find.byType(ProfileImageEditor), findsNothing);
      expect(find.text('Profile'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
