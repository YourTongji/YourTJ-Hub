import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/pages/settings/badge_display_dialog.dart';
import 'package:ui_kit/ui_kit.dart';
import '../test/badge_display_dialog_test.dart' show badge;

/// Synthetic badge artwork served from loopback; saves are local callbacks.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  binding.framePolicy = LiveTestWidgetsFlutterBindingFramePolicy.fullyLive;
  testWidgets(
    'badge disclosures preserve drafts, preview artwork and partial-save recovery',
    (tester) async {
      AppConfig.validate();
      expect(AppConfig.isDev, isTrue);
      final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
      server.listen((request) async {
        request.response.headers.contentType = ContentType('image', 'svg+xml');
        request.response.write(
          '<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 40 40"><path fill="${request.uri.path.contains('1') ? '#2563eb' : '#e11d48'}" d="M20 1 26 13 39 15 29 25 32 39 20 32 8 39 11 25 1 15 14 13Z"/></svg>',
        );
        await request.response.close();
      });
      final badges = List.generate(
        6,
        (i) => badge(i, iconUrl: 'http://127.0.0.1:${server.port}/$i.svg'),
      );
      const expectSections = bool.fromEnvironment(
        'YOURTJ_EXPECT_BADGE_SECTIONS',
        defaultValue: true,
      );
      try {
        for (final language in ['zh', 'en', 'ja', 'de']) {
          for (final brightness in Brightness.values) {
            final worn = <String>[];
            final displays = <List<String>>[];
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
                home: Builder(
                  builder: (context) => Scaffold(
                    body: TextButton(
                      onPressed: () => Navigator.of(context).push<bool>(
                        MaterialPageRoute(
                          builder: (_) => BadgeDisplayDialog(
                            badges: badges,
                            selected: [badges.first],
                            wearableBadges: badges.take(2).toList(),
                            wornBadgeCode: 'b0',
                            onWear: (code) async {
                              worn.add(code);
                            },
                            onSave: (codes) async {
                              displays.add(codes);
                              if (displays.length == 1) {
                                throw StateError('synthetic display failure');
                              }
                            },
                          ),
                        ),
                      ),
                      child: const Text('Edit'),
                    ),
                  ),
                ),
              ),
            );
            await tester.pumpAndSettle();
            await tester.tap(find.text('Edit'));
            await tester.pumpAndSettle();
            if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
              await tester.pump(const Duration(milliseconds: 400));
              await binding.takeScreenshot(
                'badge-sections-$language-${brightness.name}',
              );
            }
            if (expectSections) {
              expect(find.byType(RadioListTile<String>), findsNothing);
              expect(find.byType(Checkbox), findsNothing);
              final wearHeader = find.byKey(const Key('badge-wear-section'));
              final displayHeader = find.byKey(
                const Key('badge-display-section'),
              );
              await tester.tap(wearHeader);
              await tester.pumpAndSettle();
              final choice = find.byWidgetPredicate(
                (w) => w is RadioListTile<String> && w.value == 'b1',
              );
              await tester.ensureVisible(choice);
              await tester.pumpAndSettle();
              expect(
                find.descendant(
                  of: choice,
                  matching: find.byType(GfBadgeMedallion),
                ),
                findsOneWidget,
              );
              await tester.tap(choice);
              await tester.pumpAndSettle();
              if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
                await tester.pump(const Duration(milliseconds: 300));
                await binding.takeScreenshot(
                  'badge-wear-$language-${brightness.name}',
                );
              }
              await tester.ensureVisible(wearHeader);
              await tester.pumpAndSettle();
              await tester.tap(wearHeader);
              await tester.pumpAndSettle();
              await tester.ensureVisible(displayHeader);
              await tester.pumpAndSettle();
              await tester.tap(displayHeader);
              await tester.pumpAndSettle();
              final select = find.byKey(const Key('badge-select-b1'));
              await tester.scrollUntilVisible(
                select,
                200,
                scrollable: find.byType(Scrollable).first,
              );
              await tester.ensureVisible(select);
              await tester.pumpAndSettle();
              await tester.tap(select);
              await tester.pumpAndSettle();
              expect(worn, isEmpty);
              expect(displays, isEmpty);
              await tester.ensureVisible(displayHeader);
              await tester.pumpAndSettle();
              await tester.tap(displayHeader);
              await tester.pumpAndSettle();
              final l = AppLocalizations.of(
                tester.element(find.byType(BadgeDisplayDialog)),
              );
              await tester.tap(find.text(l.commonSave));
              await tester.pumpAndSettle();
              expect(worn, ['b1']);
              expect(displays, [
                ['b0', 'b1'],
              ]);
              expect(find.byType(BadgeDisplayDialog), findsOneWidget);
              expect(find.byType(Checkbox), findsNothing);
              if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
                await tester.pump(const Duration(milliseconds: 300));
                await binding.takeScreenshot(
                  'badge-retry-$language-${brightness.name}',
                );
              }
              await tester.tap(find.text(l.commonSave));
              await tester.pumpAndSettle();
              expect(worn, ['b1']);
              expect(displays.length, 2);
              expect(find.byType(BadgeDisplayDialog), findsNothing);
            }
            expect(tester.takeException(), isNull);
            await tester.pumpWidget(const SizedBox());
            await tester.pumpAndSettle();
          }
        }
      } finally {
        await server.close(force: true);
      }
    },
  );
}
