import 'dart:io';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/settings/profile_edit_dialog.dart';
import 'package:ui_kit/ui_kit.dart';
import '../test/pages_behavior_test.dart' show settingsPayloadJson;

/// Production profile menus on the real native surface; synthetic local image
/// bytes and picker callbacks keep this acceptance separate from OS photo access.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  binding.framePolicy = LiveTestWidgetsFlutterBindingFramePolicy.fullyLive;
  testWidgets(
    'native profile menus preserve leading, selection and cancellation',
    (tester) async {
      final bytes = img.encodePng(
        img.Image(width: 8, height: 8)..clear(img.ColorRgb8(72, 108, 144)),
      );
      final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
      addTearDown(() => server.close(force: true));
      server.listen((request) async {
        request.response.headers.contentType = ContentType('image', 'png');
        request.response.add(bytes);
        await request.response.close();
      });
      final imageUrl = 'http://127.0.0.1:${server.port}/fixture.png';
      var converted = false;
      for (final language in ['zh', 'en', 'ja', 'de']) {
        for (final brightness in Brightness.values) {
          var picks = 0, writes = 0;
          final user = SettingsUserPayload.fromJson(
            (settingsPayloadJson()['props'] as Map)['user']
                as Map<String, dynamic>,
          ).copyWith(avatarUrl: imageUrl, profileCoverUrl: imageUrl);
          await tester.pumpWidget(
            MaterialApp(
              theme: gfThemeData(brightness),
              locale: Locale(language),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(context).copyWith(
                  size: Size(320, MediaQuery.sizeOf(context).height),
                  textScaler: const TextScaler.linear(2),
                ),
                child: Center(child: SizedBox(width: 320, child: child)),
              ),
              home: ProfileEditPage(
                user: user,
                onSave: (_) async {
                  writes++;
                },
                onPickImage: (cover, source) async {
                  expect(cover, isFalse);
                  expect(source, ProfileImageSource.gallery);
                  picks++;
                  return null;
                },
                onSaveAvatar: (_) async {
                  writes++;
                  return imageUrl;
                },
                onSavePresetAvatar: (_) async {
                  writes++;
                },
                onSaveCover: (_) async {
                  writes++;
                  return '';
                },
              ),
            ),
          );
          await tester.pumpAndSettle();
          if (Platform.isAndroid && !converted) {
            await binding.convertFlutterSurfaceToImage();
            converted = true;
          }
          await tester.ensureVisible(
            find.byKey(const Key('profile-edit-avatar')),
          );
          await tester.tap(find.byKey(const Key('profile-edit-avatar')));
          await tester.pumpAndSettle();
          for (final action in ['gallery', 'camera', 'preset']) {
            final label = find.byKey(ValueKey('profile-sheet-avatar-$action'));
            expect(label, findsOneWidget);
            final row = find.ancestor(
              of: label,
              matching: find.byType(ListTile),
            );
            expect(
              find.descendant(of: row, matching: find.byType(GfSymbol)),
              findsOneWidget,
            );
            expect(label.hitTestable(), findsOneWidget);
          }
          expect(writes, 0);
          expect(picks, 0);
          expect(tester.takeException(), isNull);
          await tester.pump(const Duration(milliseconds: 400));
          if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
            await binding.takeScreenshot(
              'menu-avatar-$language-${brightness.name}',
            );
          }
          await tester.tap(
            find.byKey(const ValueKey('profile-sheet-avatar-gallery')),
          );
          await tester.pumpAndSettle();
          expect(picks, 1);
          expect(writes, 0);
          await tester.ensureVisible(
            find.byKey(const Key('profile-edit-cover')),
          );
          await tester.tap(find.byKey(const Key('profile-edit-cover')));
          await tester.pumpAndSettle();
          final remove = find.byKey(
            const ValueKey('profile-sheet-cover-remove'),
          );
          expect(remove.hitTestable(), findsOneWidget);
          final removeRow = find.ancestor(
            of: remove,
            matching: find.byType(ListTile),
          );
          expect(
            find.descendant(of: removeRow, matching: find.byType(GfSymbol)),
            findsOneWidget,
          );
          await tester.pump(const Duration(milliseconds: 400));
          if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
            await binding.takeScreenshot(
              'menu-cover-$language-${brightness.name}',
            );
          }
          final sheet = tester.getRect(find.byType(BottomSheet));
          await tester.tapAt(Offset(sheet.center.dx, sheet.top - 16));
          await tester.pumpAndSettle();
          expect(remove, findsNothing);
          expect(find.byType(ProfileEditPage), findsOneWidget);
          expect(writes, 0);
          expect(picks, 1);
          expect(tester.takeException(), isNull);
          await tester.pumpWidget(const SizedBox());
          await tester.pumpAndSettle();
        }
      }
    },
  );
}
