import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:integration_test/integration_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/storage/storage_providers.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:forum_app/src/pages/schedule/schedule_page.dart';
import '../test/schedule_config_ui_test.dart'
    show openScheduleConfig, chooseNewTerm, plansJson;

/// Real encrypted device storage; all repository reads use synthetic PK data.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  binding.framePolicy = LiveTestWidgetsFlutterBindingFramePolicy.fullyLive;
  testWidgets(
    'configuration confirmation preserves or commits every plan across languages and themes',
    (tester) async {
      AppConfig.validate();
      expect(AppConfig.isDev, isTrue);
      SharedPreferences.setPrefix('flutter.dev.');
      final bootstrap = ProviderContainer();
      addTearDown(bootstrap.dispose);
      await bootstrap.read(storageBootstrapProvider.future);
      for (final language in ['zh', 'en', 'ja', 'de']) {
        for (final brightness in Brightness.values) {
          final result = await openScheduleConfig(
            tester,
            locale: Locale(language),
            brightness: brightness,
            textScale: 2,
            width: 320,
          );
          final before = plansJson(result.store);
          final localizations = AppLocalizations.of(
            tester.element(find.byType(SchedulePage)),
          );
          await chooseNewTerm(tester);
          expect(
            find.text(localizations.scheduleConfigChangeTitle),
            findsOneWidget,
          );
          expect(tester.takeException(), isNull);
          if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
            await tester.pump(const Duration(milliseconds: 400));
            await binding.takeScreenshot(
              'schedule-config-$language-${brightness.name}',
            );
          }
          await tester.tap(find.text(localizations.commonCancel));
          await tester.pumpAndSettle();
          expect(plansJson(result.store), before);
          expect(result.store.state.majorSelected.calendarId, 119);
          await chooseNewTerm(tester);
          await tester.tap(
            find.text(localizations.scheduleConfigChangeConfirm),
          );
          await tester.pumpAndSettle();
          await result.store.configurationSettled;
          expect(result.store.state.majorSelected.calendarId, 120);
          expect(
            result.store.state.plans.every(
              (p) => p.stagedCourses.isEmpty && p.customEvents.isEmpty,
            ),
            isTrue,
          );
          // Reopen the actual encrypted records, not a copied in-memory payload.
          final reopened = ScheduleStoreNotifier();
          await reopened.ready;
          expect(reopened.state.majorSelected.calendarId, 120);
          expect(plansJson(reopened), plansJson(result.store));
          reopened.dispose();
          await tester.pumpWidget(const SizedBox());
          await tester.pumpAndSettle();
        }
      }
    },
  );
}
