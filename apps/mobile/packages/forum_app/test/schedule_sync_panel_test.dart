import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/schedule/schedule_sync_panel.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:forum_app/src/schedule/schedule_sync.dart';
import 'schedule_sync_test.dart' show Tokens, Transport, plan, item;

class CapturingSync extends ScheduleSyncController {
  CapturingSync(ScheduleStoreNotifier store)
    : super(
        transport: Transport(),
        tokenStorage: Tokens(),
        store: store,
        readUserId: () async => 7,
      );
  Map<String, String>? choice;
  @override
  Future<void> resolveConflict(String id, Map<String, String> choices) async {
    choice = Map.of(choices);
  }
}

void main() {
  testWidgets(
    'narrow-screen panel chooses one field and exposes recovery drafts',
    (tester) async {
      SharedPreferences.setMockInitialValues({});
      final store = ScheduleStoreNotifier();
      await store.ready;
      final sync = CapturingSync(store);
      addTearDown(sync.dispose);
      addTearDown(store.dispose);
      tester.view.physicalSize = const Size(320, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      sync.conflicts.value = [
        PlanSyncConflict('p', plan(), plan(), item(plan(), 2), [
          const PkPlanMergeConflict(['name'], 'Local title', 'Remote title'),
        ]),
      ];
      await tester.pumpWidget(
        ProviderScope(
          overrides: [scheduleSyncControllerProvider.overrideWithValue(sync)],
          child: MaterialApp(
            locale: const Locale('en'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const Scaffold(
              body: SingleChildScrollView(child: ScheduleSyncPanel()),
            ),
          ),
        ),
      );
      expect(
        tester.widget<FilledButton>(find.byType(FilledButton)).onPressed,
        isNull,
      );
      await tester.tap(find.text('Use cloud'));
      await tester.pump();
      await tester.tap(find.text('Merge and save'));
      await tester.pump();
      expect(sync.choice, {'["name"]': 'remote'});
      sync.conflicts.value = [];
      sync.drafts.value = {'p': plan()};
      await tester.pump();
      expect(find.text('Recovery drafts'), findsOneWidget);
      expect(find.text('Restore as new plan'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
