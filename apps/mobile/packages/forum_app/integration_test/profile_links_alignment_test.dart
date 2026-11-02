import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/pages/settings/profile_edit_dialog.dart';
import 'package:ui_kit/ui_kit.dart';
import '../test/pages_behavior_test.dart' show settingsPayloadJson;
import '../test/settings_editors_test.dart' show openProfileLinks;

/// Actual profile/link routes with synthetic account data and no API writes.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  binding.framePolicy = LiveTestWidgetsFlutterBindingFramePolicy.fullyLive;
  testWidgets(
    'profile link inputs align and remain usable with large localized text',
    (tester) async {
      AppConfig.validate();
      expect(AppConfig.isDev, isTrue);
      final user = SettingsUserPayload.fromJson(
        (settingsPayloadJson()['props'] as Map)['user'] as Map<String, dynamic>,
      ).copyWith(websiteName: 'Campus', website: 'https://example.test');
      var writes = 0;
      for (final language in ['zh', 'en', 'ja', 'de']) {
        for (final brightness in Brightness.values) {
          for (final width in [390.0, 320.0]) {
            final scale = width == 320 ? 2.0 : 1.0;
            await tester.pumpWidget(
              MaterialApp(
                theme: gfThemeData(brightness),
                locale: Locale(language),
                localizationsDelegates: AppLocalizations.localizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                builder: (context, child) => MediaQuery(
                  data: MediaQuery.of(context).copyWith(
                    size: Size(width, MediaQuery.sizeOf(context).height),
                    textScaler: TextScaler.linear(scale),
                  ),
                  child: Center(
                    child: SizedBox(width: width, child: child),
                  ),
                ),
                home: Builder(
                  builder: (context) => Scaffold(
                    body: TextButton(
                      onPressed: () => Navigator.of(context).push(
                        MaterialPageRoute<void>(
                          builder: (_) => ProfileEditPage(
                            user: user,
                            onSave: (_) async {
                              writes++;
                            },
                          ),
                        ),
                      ),
                      child: const Text('Open'),
                    ),
                  ),
                ),
              ),
            );
            await tester.pumpAndSettle();
            await tester.tap(find.text('Open'));
            await tester.pumpAndSettle();
            await openProfileLinks(tester);
            final l = AppLocalizations.of(
              tester.element(find.byKey(const Key('profile-links-done'))),
            );
            if (width == 390 &&
                const bool.fromEnvironment(
                  'YOURTJ_EXPECT_LINK_ALIGNMENT',
                  defaultValue: true,
                )) {
              for (final entry in {
                'websiteName': l.settingsWebsiteName,
                'website': l.profileEditWebsiteUrl,
                'github': 'GitHub',
              }.entries) {
                final field = find.byKey(ValueKey('profile-${entry.key}'));
                await tester.ensureVisible(field);
                await tester.pumpAndSettle();
                final input = find.descendant(
                  of: field,
                  matching: find.byType(EditableText),
                );
                final label = find
                    .byWidgetPredicate(
                      (w) => w is GfText && w.data == entry.value,
                    )
                    .last;
                expect(
                  tester.getCenter(input).dy,
                  closeTo(tester.getCenter(label).dy, 1),
                  reason: entry.key,
                );
              }
            }
            await tester.drag(
              find.byType(ListView).last,
              const Offset(0, 1800),
            );
            await tester.pumpAndSettle();
            expect(tester.takeException(), isNull);
            if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
              await tester.pump(const Duration(milliseconds: 300));
              await binding.takeScreenshot(
                'profile-links-$language-${brightness.name}-${width.toInt()}',
              );
            }
            await tester.drag(
              find.byType(ListView).last,
              const Offset(0, -1800),
            );
            await tester.pumpAndSettle();
            expect(tester.takeException(), isNull);
            await tester.tap(find.byKey(const Key('profile-links-done')));
            await tester.pumpAndSettle();
            expect(find.byType(ProfileEditPage), findsOneWidget);
            expect(writes, 0);
            await tester.pumpWidget(const SizedBox());
            await tester.pumpAndSettle();
          }
        }
      }
    },
  );
}
