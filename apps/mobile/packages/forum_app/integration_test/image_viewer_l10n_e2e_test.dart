import 'dart:async';
import 'dart:convert';
import 'dart:ui' show Tristate;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:image/image.dart' as img;
import 'package:integration_test/integration_test.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/pages/settings/settings_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/push/push_service.dart';
import 'package:forum_app/src/router.dart';
import 'package:forum_app/src/theme_mode.dart';
import 'package:forum_app/src/widgets/markdown_view.dart';
import '../test/pages_behavior_test.dart' show MemTokenStorage;

// Full GfApp + production Markdown gallery navigation, real local image upload
// and HTTP decoding using the existing session for unreferenced uploads.
// The fixture route uses the production Markdown-to-viewer handler; no topic is published.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  testWidgets(
    'native gallery labels, selected positions and original size in four locales',
    (tester) async {
      expect([
        'localhost',
        '127.0.0.1',
        '::1',
      ], contains(Uri.parse(AppConfig.apiBaseUrl).host));
      final semantics = tester.ensureSemantics();
      final locale = ValueNotifier(const Locale('zh'));
      final container = ProviderContainer(
        overrides: [
          tokenStorageProvider.overrideWithValue(
            MemTokenStorage(const String.fromEnvironment('YOURTJ_TEST_TOKEN')),
          ),
          pushBootstrapProvider.overrideWith((_) {}),
        ],
      );
      try {
        Future<void> until(bool Function() ready) async {
          final end = DateTime.now().add(const Duration(seconds: 45));
          while (!ready() && DateTime.now().isBefore(end)) {
            await tester.pump(const Duration(milliseconds: 100));
          }
          expect(ready(), isTrue);
          await tester.pumpAndSettle();
          expect(tester.takeException(), isNull);
        }

        appRouter.go('/settings');
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: ValueListenableBuilder<Locale>(
              valueListenable: locale,
              builder: (_, value, _) => GfApp(locale: value),
            ),
          ),
        );
        await until(() => find.byType(SettingsPage).evaluate().isNotEmpty);
        final urls = <String>[];
        for (var index = 0; index < 2; index++) {
          final image = img.Image(width: 600, height: 360);
          for (final pixel in image) {
            pixel.setRgb(
              index == 0 ? 45 : 190,
              60 + pixel.x * 150 ~/ 600,
              80 + pixel.y * 150 ~/ 360,
            );
          }
          final path = await container
              .read(fileRepositoryProvider)
              .uploadImage(
                bytes: img.encodePng(image),
                filename: 'duo-viewer-$index.png',
              );
          urls.add(Uri.parse(AppConfig.apiBaseUrl).resolve(path).toString());
        }
        final markdown =
            'Local native gallery acceptance fixture with two synthetic gradient images.\n\n![one](${urls[0]})\n\n![two](${urls[1]})';
        unawaited(
          Navigator.of(
            tester.element(find.byType(SettingsPage)),
            rootNavigator: true,
          ).push<void>(
            MaterialPageRoute(
              builder: (_) => Scaffold(
                appBar: AppBar(title: const Text('Duo gallery acceptance')),
                body: GfMarkdownView(
                  key: const Key('duo-gallery'),
                  data: markdown,
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        final gallery = find.byKey(const Key('duo-gallery'));
        for (final url in urls) {
          await precacheImage(
            GfMediaScope.imageProvider(tester.element(gallery), url),
            tester.element(gallery),
          );
        }
        await tester.pumpAndSettle();
        for (final language in ['zh', 'en', 'ja', 'de']) {
          for (final brightness in Brightness.values) {
            locale.value = Locale(language);
            container
                .read(themeModeProvider.notifier)
                .setMode(
                  brightness == Brightness.dark
                      ? ThemeMode.dark
                      : ThemeMode.light,
                );
            await tester.pumpAndSettle();
            await tester.tap(
              find.descendant(of: gallery, matching: find.byType(Image)).first,
            );
            await until(() => find.byType(GfImageViewer).evaluate().isNotEmpty);
            final viewer = find.byType(GfImageViewer);
            final l = AppLocalizations.of(tester.element(viewer));
            final original = find.byTooltip(l.imageViewerOriginalSize);
            expect(original.hitTestable(), findsOneWidget);
            final rail = tester.getSemantics(
              find.bySemanticsLabel(l.imageViewerThumbnails),
            );
            expect(rail.label, l.imageViewerThumbnails);
            expect(find.byTooltip(l.topicShare).hitTestable(), findsOneWidget);
            await tester.tap(original);
            await tester.pumpAndSettle();
            expect(find.byTooltip(l.imageViewerFitPreview), findsOneWidget);
            await tester.tap(
              find.byKey(const Key('gf-image-viewer-thumbnail-1')),
            );
            await tester.pumpAndSettle();
            final second = tester.getSemantics(
              find.byKey(const Key('gf-image-viewer-thumbnail-semantic-1')),
            );
            final first = tester.getSemantics(
              find.byKey(const Key('gf-image-viewer-thumbnail-semantic-0')),
            );
            expect(second.label, l.imageViewPosition(2, 2));
            expect(second.flagsCollection.isSelected, Tristate.isTrue);
            expect(first.flagsCollection.isSelected, Tristate.isFalse);
            expect(find.text('2 / 2'), findsOneWidget);
            expect(original, findsOneWidget);
            // Evidence from the actual native Semantics tree, not screenshot OCR.
            debugPrint(
              'VIEWER_AX ${jsonEncode({'language': language, 'theme': brightness.name, 'rail': rail.label, 'first': first.label, 'firstSelected': first.flagsCollection.isSelected.toString(), 'second': second.label, 'secondSelected': second.flagsCollection.isSelected.toString(), 'originalSize': tester.getSemantics(original).label})}',
            );
            if (language == 'zh' &&
                const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
              await tester.longPress(original);
              await tester.pump(const Duration(milliseconds: 700));
              await binding.takeScreenshot(
                '1443-viewer-$language-${brightness.name}',
              );
            }
            await tester.drag(
              find.byKey(const Key('gf-image-viewer-page-swipe-area')),
              const Offset(350, 0),
            );
            await tester.pumpAndSettle();
            expect(find.text('1 / 2'), findsOneWidget);
            final close = MaterialLocalizations.of(
              tester.element(viewer),
            ).closeButtonTooltip;
            await tester.tap(find.byTooltip(close));
            await until(() => viewer.evaluate().isEmpty);
            expect(gallery, findsOneWidget);
          }
        }
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump(const Duration(milliseconds: 600));
        await tester.pumpAndSettle();
      } finally {
        semantics.dispose();
        locale.dispose();
        container.dispose();
      }
    },
    skip: !const bool.fromEnvironment('YOURTJ_TEST_LOCAL_IMAGE_VIEWER'),
  );
}
