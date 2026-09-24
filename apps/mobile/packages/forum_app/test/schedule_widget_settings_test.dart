import 'package:core/core.dart';
import 'package:drift/native.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/campus_widget/schedule_widget_bridge.dart';
import 'package:forum_app/src/campus_widget/schedule_widget_projection.dart';
import 'package:forum_app/src/app_config.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/offline/campus_snapshot_store.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/settings/schedule_widget_settings_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/campus_fixtures.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('widget settings requires one-time privacy acknowledgement', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          locale: Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ScheduleWidgetSettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(GfAlertDialog), findsOneWidget);
    await tester.tap(find.text('确认'));
    await tester.pumpAndSettle();
    expect(
      (await SharedPreferences.getInstance()).getBool(
        'schedule_widget_privacy_ack',
      ),
      isTrue,
    );
  });

  testWidgets('refresh includes tomorrow classes from the calendar rules', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({
      'schedule_widget_privacy_ack': true,
    });
    final database = AppDatabase(NativeDatabase.memory());
    addTearDown(database.close);
    final now = DateTime.parse('2026-09-22T00:00:00Z');
    final scope = CampusCacheScope(
      site: Uri.parse(
        AppConfig.apiBaseUrl.isNotEmpty
            ? AppConfig.apiBaseUrl
            : GfApiClient.defaultBaseUrl,
      ).origin,
      accountId: 7,
    );
    final store = CampusSnapshotStore(database, now: () => now);
    final data = {
      for (final key in campusPersistentKeys) key: campusFixture(key, now: now),
    };
    data['calendar'] = _withCalendarBounds(data['calendar']!);
    await store.write(scope, 'binding', data, committedAt: now);
    final bridge = RecordingWidgetBridge();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          currentUserProvider.overrideWith(
            (ref) async => const CurrentUser(id: 7, username: ''),
          ),
          campusSnapshotStoreProvider.overrideWithValue(store),
          campusRepositoryProvider.overrideWithValue(FakeCampusRepository()),
          scheduleWidgetBridgeProvider.overrideWithValue(bridge),
        ],
        child: const MaterialApp(
          locale: Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ScheduleWidgetSettingsPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('从离线课表刷新'));
    await tester.pumpAndSettle();

    final days = (bridge.projection!.toJson()['days']! as List)
        .cast<Map<String, dynamic>>();
    expect(days[1]['date'], '2026-09-23');
    expect(
      (days[1]['courses']! as List).cast<Map<String, dynamic>>().map(
        (course) => course['name'],
      ),
      contains('课程 3（演示）'),
    );
  });
}

CampusDataset _withCalendarBounds(CampusDataset source) => CampusDataset(
  key: source.key,
  status: source.status,
  updatedAt: source.updatedAt,
  metrics: [
    ...source.metrics,
    const CampusMetric(label: '学期开始', value: '2026-09-21', unit: ''),
    const CampusMetric(label: '学期结束', value: '2027-01-18', unit: ''),
  ],
  columns: source.columns,
  rows: source.rows,
  events: source.events,
  series: source.series,
  messages: source.messages,
  teachingDay: source.teachingDay,
);

class RecordingWidgetBridge extends ScheduleWidgetBridge {
  ScheduleWidgetProjection? projection;

  @override
  Future<void> write(ScheduleWidgetProjection projection) async {
    this.projection = projection;
  }
}
