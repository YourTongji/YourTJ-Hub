import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:drift/native.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'package:forum_app/src/pages/campus/campus_helpers.dart';
import 'package:forum_app/src/pages/campus/campus_data_views.dart';
import 'package:forum_app/src/pages/campus/campus_private_surface.dart';
import 'package:forum_app/src/pages/campus/campus_message_page.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';
import 'package:forum_app/src/pages/campus/campus_memory_cache.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/widgets/account_drawer.dart';
import 'package:forum_app/src/widgets/campus_shortcuts.dart';
import 'package:forum_app/src/widgets/schedule_time_grid.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/campus_fixtures.dart';
import 'fixtures/page_fixtures.dart';

class CampusPkRepository extends PkRepository {
  CampusPkRepository()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: CampusMemoryTokenStorage(),
          baseUrl: 'https://forum.example',
        ),
      );
  @override
  Future<SectionTimesPayload?> sectionTimes() async => null;
}

class UnavailableCampusRepository extends FakeCampusRepository {
  @override
  Future<CampusStatus> status({CancelToken? cancelToken}) async =>
      throw const ApiException(fallbackMessage: 'School unavailable');
}

Widget campusTestApp(
  FakeCampusRepository repository, {
  Widget child = const CampusPage(),
  double scale = 1,
  Brightness brightness = Brightness.light,
  Locale locale = const Locale('zh'),
  bool signedIn = true,
}) {
  final database = AppDatabase(NativeDatabase.memory());
  addTearDown(database.close);
  return ProviderScope(
    overrides: [
      offlineDatabaseProvider.overrideWithValue(database),
      campusRepositoryProvider.overrideWithValue(repository),
      pkRepositoryProvider.overrideWithValue(CampusPkRepository()),
      currentUserProvider.overrideWith(
        (ref) async =>
            signedIn ? const CurrentUser(id: 1, username: 'demo') : null,
      ),
      accountLayoutProvider.overrideWith(
        (ref) async => LayoutPayload.fromJson(minimalLayoutJson()),
      ),
    ],
    child: MaterialApp(
      theme: gfThemeData(brightness),
      locale: locale,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      builder: (context, child) => MediaQuery(
        data: MediaQuery.of(
          context,
        ).copyWith(textScaler: TextScaler.linear(scale)),
        child: child!,
      ),
      home: child,
    ),
  );
}

void main() {
  testWidgets('public campus tools remain available when school status fails', (
    tester,
  ) async {
    await tester.pumpWidget(campusTestApp(UnavailableCampusRepository()));
    await tester.pumpAndSettle();
    expect(find.byType(CampusShortcuts), findsOneWidget);
    expect(
      find.text(
        AppLocalizations.of(
          tester.element(find.byType(CampusPage)),
        ).campusUnavailable,
      ),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  for (final signedIn in [false, true]) {
    for (final bound in [false, true]) {
      testWidgets(
        'campus tools appear directly with signedIn=$signedIn bound=$bound',
        (tester) async {
          final repo = FakeCampusRepository();
          if (!bound) {
            repo.current = const CampusStatus(
              enabled: true,
              binding: null,
              candidate: null,
            );
          }
          await tester.pumpWidget(campusTestApp(repo, signedIn: signedIn));
          await tester.pumpAndSettle();
          expect(find.byType(CampusShortcuts), findsOneWidget);
          final l = AppLocalizations.of(
            tester.element(find.byType(CampusPage)),
          );
          for (final label in [
            l.campusCourseReviews,
            l.scheduleTitle,
            l.wikiTitle,
          ]) {
            expect(
              find.descendant(
                of: find.byType(CampusShortcuts),
                matching: find.text(label),
              ),
              findsOneWidget,
            );
          }
          expect(tester.takeException(), isNull);
          await tester.pumpWidget(const SizedBox());
          await tester.pumpAndSettle();
        },
      );
    }
  }

  test(
    'home loads only home datasets; stale refresh response cannot restore old identity',
    () async {
      final repo = FakeCampusRepository();
      final controller = CampusController(repo);
      await controller.refresh();
      expect(repo.requested.toSet(), {
        'profile',
        'calendar',
        'timetable',
        'messages',
        'today',
      });
      await controller.loadTab('academics');
      expect(repo.requested, containsAll(['grades', 'summary', 'cet']));
      controller.tab = 'today';
      repo.pendingProfile = Completer();
      final old = controller.refresh();
      await Future<void>.delayed(Duration.zero);
      // An identity mutation must preempt a refresh; repeated refresh taps coalesce.
      await controller.change(
        (token) => repo.unbind(testBinding.revision, cancelToken: token),
      );
      repo.pendingProfile!.complete(campusFixture('profile'));
      await old;
      expect(controller.state.status?.binding, isNull);
      expect(controller.state.data, isEmpty);
      controller.dispose();
      expect(repo.cancellations.every((c) => c.isCancelled), isTrue);
    },
  );
  test('failed confirmation preserves binding and surfaces failure', () async {
    final repo = FakeCampusRepository()
      ..confirmError = const ApiException(
        fallbackMessage: 'test',
        messageCode: 'campus.identityUnavailable',
      );
    final controller = CampusController(repo);
    await controller.refresh();
    expect(
      await controller.change((c) => repo.confirm(cancelToken: c)),
      isFalse,
    );
    expect(controller.state.status?.binding?.revision, testBinding.revision);
    expect(controller.state.error, isA<ApiException>());
    controller.dispose();
  });
  test('expired school date waits for explicit refresh', () async {
    final base = campusFixture('today');
    final repo = FakeCampusRepository()
      ..todayOverride = CampusDataset(
        key: 'today',
        status: 'empty',
        updatedAt: '',
        metrics: [],
        columns: [],
        rows: [],
        events: [],
        series: [],
        teachingDay: CampusTeachingDay(
          date: '2000-01-01',
          sourceDate: '',
          kind: 'holiday',
          label: '旧假期',
          sectionCount: 11,
        ),
      );
    final controller = CampusController(repo);
    await controller.refresh();
    repo.requested.clear();
    repo.todayOverride = base;
    await controller.refreshVisible();
    expect(repo.requested, isEmpty);
    expect(controller.state.data['today'], isNull);
    await controller.refresh();
    expect(
      controller.state.data['today']?.teachingDay?.date,
      campusDateKey(DateTime.now()),
    );
    controller.dispose();
  });
  test(
    'midnight on weekly tab invalidates daily data without polling',
    () async {
      var now = DateTime.utc(2026, 9, 20, 15, 59);
      final cache = CampusMemoryCache(now: () => now);
      addTearDown(cache.dispose);
      final repo = FakeCampusRepository()..now = () => now;
      final controller = CampusController(repo, cache: cache);
      await controller.refresh();
      await controller.loadTab('timetable');
      repo.requested.clear();
      now = now.add(const Duration(minutes: 2));
      await controller.refreshVisible();
      expect(repo.requested, isEmpty);
      expect(controller.state.data.containsKey('today'), isFalse);
      expect(controller.state.data.containsKey('calendar'), isFalse);
      controller.dispose();
    },
  );
  for (final code in ['campus.rulesUnavailable', 'campus.rulesInvalid']) {
    testWidgets('holiday is explained and $code hides old classes', (
      tester,
    ) async {
      final repo = FakeCampusRepository()
        ..todayOverride = CampusDataset(
          key: 'today',
          status: 'empty',
          updatedAt: '',
          metrics: [],
          columns: [],
          rows: [],
          events: [],
          series: [],
          teachingDay: CampusTeachingDay(
            date: campusDateKey(DateTime.now()),
            sourceDate: '',
            kind: 'holiday',
            label: '国庆节',
            sectionCount: 11,
          ),
        );
      await tester.pumpWidget(campusTestApp(repo));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('国庆节：今天放假停课。'));
      expect(find.text('国庆节：今天放假停课。'), findsOneWidget);
      final container = ProviderScope.containerOf(
        tester.element(find.byType(CampusPage)),
      );
      repo.todayError = ApiException(
        fallbackMessage: 'Rules unavailable',
        messageCode: code,
      );
      await container.read(campusControllerProvider.notifier).refresh();
      await tester.pumpAndSettle();
      final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
      expect(find.text(l.campusRulesUnavailable), findsOneWidget);
      expect(find.text('国庆节：今天放假停课。'), findsNothing);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }
  test('school clock and grid preserve distinct records and filter weeks', () {
    expect(campusNow(DateTime.parse('2026-09-19T18:00:00Z')).day, 20);
    final events = campusFixture('timetable').events;
    expect(campusCoursesForWeek(events, 2).length, 1);
    final grid = campusGrid([events.first, events.first], 1);
    expect(grid.cellCourses[0][0].length, 2);
    expect(grid.rowHeights.length, 11);
  });
  testWidgets(
    'home shows greeting/latest five notices; grades load only on academic tab',
    (tester) async {
      final repo = FakeCampusRepository();
      await tester.pumpWidget(campusTestApp(repo));
      await tester.pumpAndSettle();
      expect(find.textContaining('演示同学'), findsOneWidget);
      expect(
        tester.getTopLeft(find.text('今日课表')).dy,
        lessThan(
          tester
              .getTopLeft(
                find.descendant(
                  of: find.byType(ListView),
                  matching: find.text('校园消息'),
                ),
              )
              .dy,
        ),
      );
      expect(repo.requested, contains('today'));
      await tester.ensureVisible(find.text('第四周周二的数学'));
      expect(find.text('第四周周二的数学'), findsOneWidget);
      expect(find.textContaining('国庆补课'), findsOneWidget);
      await tester.ensureVisible(find.text('学业记录').first);
      expect(find.text('综合 GPA'), findsNothing);
      expect(repo.requested, isNot(contains('grades')));
      await tester.tap(find.text('学业记录').first);
      await tester.pumpAndSettle();
      expect(find.text('综合 GPA'), findsOneWidget);
      expect(repo.requested, contains('grades'));
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
      expect(repo.cancellations.every((c) => c.isCancelled), isTrue);
    },
  );
  for (final brightness in Brightness.values) {
    testWidgets(
      'native grid and academics fit narrow enlarged ${brightness.name} layout',
      (tester) async {
        tester.view.physicalSize = const Size(320, 640);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.reset);
        final repo = FakeCampusRepository();
        await tester.pumpWidget(
          campusTestApp(
            repo,
            brightness: brightness,
            scale: 2,
            locale: const Locale('de'),
          ),
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
        await tester.ensureVisible(find.text(l.campusTimetable));
        await tester.tap(find.text(l.campusTimetable));
        await tester.pumpAndSettle();
        final grid = tester.widget<ScheduleTimeGrid>(
          find.byType(ScheduleTimeGrid),
        );
        expect(grid.onTapEmptyCell, isNull);
        expect(tester.takeException(), isNull);
        await tester.ensureVisible(find.text(l.campusAcademics).first);
        await tester.tap(find.text(l.campusAcademics).first);
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
        await tester.pumpAndSettle();
      },
    );
  }
  testWidgets('tab exit disposes requests and private name', (tester) async {
    final visible = ValueNotifier(true);
    addTearDown(visible.dispose);
    final repo = FakeCampusRepository();
    await tester.pumpWidget(
      campusTestApp(
        repo,
        child: ValueListenableBuilder<bool>(
          valueListenable: visible,
          builder: (_, active, _) =>
              TickerMode(enabled: active, child: const CampusPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.textContaining('演示同学'), findsOneWidget);
    visible.value = false;
    await tester.pumpAndSettle();
    expect(find.textContaining('演示同学'), findsNothing);
    expect(repo.cancellations.every((c) => c.isCancelled), isTrue);
  });
  testWidgets('quick campus tab return reuses data after status verification', (
    tester,
  ) async {
    final visible = ValueNotifier(true);
    addTearDown(visible.dispose);
    final repo = FakeCampusRepository();
    await tester.pumpWidget(
      campusTestApp(
        repo,
        child: ValueListenableBuilder<bool>(
          valueListenable: visible,
          builder: (_, active, _) =>
              TickerMode(enabled: active, child: const CampusPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final requests = List<String>.of(repo.requested);
    visible.value = false;
    await tester.pumpAndSettle();
    expect(find.textContaining('演示同学'), findsNothing);
    visible.value = true;
    await tester.pumpAndSettle();
    expect(find.textContaining('演示同学'), findsOneWidget);
    expect(repo.requested, requests);
    expect(find.text('第四周周二的数学'), findsOneWidget);
    expect(find.textContaining('国庆补课'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
  testWidgets(
    'notification reads selectable plain text and clears on app background',
    (tester) async {
      final repo = FakeCampusRepository();
      await tester.pumpWidget(
        campusTestApp(
          repo,
          child: const CampusMessagePage(id: '9000000000000000'),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(SelectableText), findsOneWidget);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      await tester.pumpAndSettle();
      expect(find.byType(SelectableText), findsNothing);
      expect(repo.cancellations.every((c) => c.isCancelled), isTrue);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pumpAndSettle();
      expect(find.byType(SelectableText), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );
  testWidgets('notice permission update confirms then resumes reading', (
    tester,
  ) async {
    final repo = FakeCampusRepository()
      ..messageError = const ApiException(
        fallbackMessage: 'test',
        messageCode: 'campus.messageAuthorizationRequired',
      )
      ..current = const CampusStatus(
        enabled: true,
        binding: testBinding,
        candidate: CampusCandidate(
          maskedId: 'DE••••MO',
          mode: 'reauthorize',
          expiresAt: '2099-01-01T00:00:00Z',
        ),
      );
    await tester.pumpWidget(
      campusTestApp(
        repo,
        child: const CampusMessagePage(id: '9000000000000000'),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('确认更新授权'));
    await tester.pumpAndSettle();
    expect(repo.confirmed, 1);
    expect(find.byType(SelectableText), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
  testWidgets(
    'course details drop the prior identity when the session changes',
    (tester) async {
      final repo = FakeCampusRepository();
      await tester.pumpWidget(
        campusTestApp(
          repo,
          child: Scaffold(
            body: CampusPrivateSurface(
              builder: (_) => const CampusCourseDetails(week: 1, index: 0),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final container = ProviderScope.containerOf(
        tester.element(find.byType(CampusCourseDetails)),
      );
      await container
          .read(campusControllerProvider.notifier)
          .loadTab('timetable');
      await tester.pumpAndSettle();
      expect(find.text('课程 1（演示）'), findsOneWidget);
      repo.current = const CampusStatus(
        enabled: true,
        binding: null,
        candidate: null,
      );

      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pumpAndSettle();
      expect(find.text('课程 1（演示）'), findsNothing);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );
}
