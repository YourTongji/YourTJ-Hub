// Campus landing cleanup (#1372 D): no Explore duplicate, identity and
// offline data live in Campus settings, and only the Campus tab uses the
// graduation cap.
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/l10n/app_localizations_de.dart';
import 'package:forum_app/l10n/app_localizations_en.dart';
import 'package:forum_app/l10n/app_localizations_ja.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/pages/campus/campus_connection.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'package:forum_app/src/pages/campus/campus_settings_page.dart';
import 'package:forum_app/src/pages/settings/campus_cache_clear_tile.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/campus_shortcuts.dart';
import 'package:ui_kit/ui_kit.dart';

import 'campus_native_test.dart' show campusTestApp;
import 'fixtures/campus_fixtures.dart';

class _Courses extends CourseRepository {
  _Courses()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: CampusMemoryTokenStorage(),
          baseUrl: 'https://forum.example',
        ),
      );
  final calls = <bool>[];
  @override
  Future<CourseListResultPayload> list({
    String? keyword,
    List<String> departments = const [],
    List<String> terms = const [],
    List<String> campuses = const [],
    List<String> instructors = const [],
    bool onlyWithReviews = false,
    String sortBy = '',
    int page = 1,
    int size = 20,
  }) async {
    calls.add(onlyWithReviews);
    return const CourseListResultPayload(
      list: [
        CourseSummaryPayload(
          id: 7,
          name: '高等数学（演示）',
          primaryCode: '100001',
          department: '数学科学学院',
          creditX10: 50,
          ratingAvg: 4.6,
        ),
      ],
      page: 1,
      size: 3,
      total: 1,
      hasNext: false,
    );
  }
}

AppLocalizations _l(WidgetTester tester) =>
    AppLocalizations.of(tester.element(find.byType(CampusPage)));

void main() {
  testWidgets('campus tabs keep school content and use distinct names', (
    tester,
  ) async {
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    final bar = tester.widget<GfTabBar>(find.byType(GfTabBar));
    expect(bar.tabs.map((tab) => tab.value), [
      'today',
      'timetable',
      'academics',
      'messages',
      'calendars',
    ]);
    expect(bar.tabs.map((tab) => tab.label), [
      '今天',
      '学校课表',
      '学业记录',
      '学校通知',
      '校历',
    ]);
    expect(find.text('身份连接'), findsNothing);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  test('campus and planner names are distinct in every locale', () {
    for (final l in <AppLocalizations>[
      AppLocalizationsZh(),
      AppLocalizationsEn(),
      AppLocalizationsJa(),
      AppLocalizationsDe(),
    ]) {
      expect(l.campusTimetable, isNot(l.scheduleTitle), reason: l.localeName);
      expect(l.campusMessages, isNot(l.navMessages), reason: l.localeName);
    }
    final zh = AppLocalizationsZh();
    expect(zh.campusTimetable, '学校课表');
    expect(zh.campusMessages, '学校通知');
    expect(zh.scheduleTitle, '选课排课');
    expect(zh.schedulerWebTitle, isNot(contains('排课器')));
  });

  testWidgets('campus settings hold identity connection and offline data', (
    tester,
  ) async {
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    final l = _l(tester);
    expect(find.byTooltip('探索校园'), findsNothing);
    final gear = find.byTooltip(l.campusSettings);
    expect(gear, findsOneWidget);
    expect(
      find.descendant(of: gear, matching: find.byType(GfSymbol)),
      findsOneWidget,
    );
    expect(
      tester
          .widget<GfSymbol>(
            find.descendant(of: gear, matching: find.byType(GfSymbol)),
          )
          .name,
      'settings',
    );
    await tester.tap(gear);
    await tester.pumpAndSettle();
    expect(find.byType(CampusSettingsPage), findsOneWidget);
    expect(find.text(l.campusSettings), findsWidgets);
    expect(find.byType(CampusConnection), findsOneWidget);
    expect(find.text(testBinding.maskedId), findsOneWidget);
    await tester.ensureVisible(find.byType(CampusCacheClearTile));
    expect(find.byType(CampusCacheClearTile), findsOneWidget);
    expect(tester.takeException(), isNull);
    Navigator.of(tester.element(find.byType(CampusSettingsPage))).pop();
    await tester.pumpAndSettle();
    expect(find.byType(CampusPage), findsOneWidget);
    expect(tester.widget<GfTabBar>(find.byType(GfTabBar)).selected, 'today');
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('guests see shortcuts, sign-in and course picks only', (
    tester,
  ) async {
    final courses = _Courses();
    await tester.pumpWidget(
      campusTestApp(
        FakeCampusRepository(),
        signedIn: false,
        overrides: [courseRepositoryProvider.overrideWithValue(courses)],
      ),
    );
    await tester.pumpAndSettle();
    final l = _l(tester);
    expect(find.byType(CampusShortcuts), findsOneWidget);
    expect(find.text(l.authLoginTitle), findsOneWidget);
    expect(find.byTooltip('探索校园'), findsNothing);
    expect(find.byTooltip(l.campusSettings), findsNothing);
    await tester.ensureVisible(find.text('高等数学（演示）'));
    expect(find.text(l.campusCoursesTitle), findsOneWidget);
    expect(courses.calls, [true]);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('today keeps the course picks Explore used to show', (
    tester,
  ) async {
    final courses = _Courses();
    await tester.pumpWidget(
      campusTestApp(
        FakeCampusRepository(),
        overrides: [courseRepositoryProvider.overrideWithValue(courses)],
      ),
    );
    await tester.pumpAndSettle();
    final picks = find.text('高等数学（演示）');
    await tester.scrollUntilVisible(
      picks,
      200,
      scrollable: find.byType(Scrollable).first,
    );
    expect(picks, findsOneWidget);
    expect(find.text(_l(tester).campusCoursesTitle), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('graduation cap is reserved for the Campus tab', (tester) async {
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    final tiles = tester
        .widgetList<GfIconTile>(
          find.descendant(
            of: find.byType(CampusShortcuts),
            matching: find.byType(GfIconTile),
          ),
        )
        .map((tile) => tile.symbol);
    expect(tiles, hasLength(3));
    expect(tiles, isNot(contains('graduation-cap')));
    expect(
      tester
          .widgetList<GfSymbol>(find.byType(GfSymbol))
          .where((s) => s.name.startsWith('graduation-cap')),
      isEmpty,
    );
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('iPad campus modules do not reuse the Campus tab glyph', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(1200, 900);
    addTearDown(tester.view.reset);
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    final rows = tester.widgetList<GfSettingRow>(
      find.descendant(
        of: find.byKey(const Key('campus-module-directory')),
        matching: find.byType(GfSettingRow),
      ),
    );
    expect(rows, hasLength(5));
    expect(rows.map((row) => row.symbol), isNot(contains('graduation-cap')));
    expect(find.byKey(const Key('campus-module-connection')), findsNothing);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
}
