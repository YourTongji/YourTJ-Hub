import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/navigation/tab_scroll_registry.dart';
import 'package:forum_app/src/navigation/reading_chrome.dart';
import 'package:flutter/material.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'package:ui_kit/ui_kit.dart';

import 'campus_native_test.dart' show campusTestApp;
import 'fixtures/campus_fixtures.dart';
import 'test_font_helpers.dart' show loadTestFonts;

Finder get _search => find.descendant(
  of: find.byType(GfSearchField),
  matching: find.byType(TextField),
);

Future<void> _select(WidgetTester tester, String tab) async {
  final bar = tester.widget<GfTabBar>(find.byType(GfTabBar));
  bar.onSelected(tab);
  await tester.pumpAndSettle();
}

ScrollController _list(WidgetTester tester) => tester
    .widget<ListView>(
      find
          .descendant(
            of: find.byType(GfScrollToTop),
            matching: find.byType(ListView),
          )
          .first,
    )
    .controller!;

ScrollController _listForTab(WidgetTester tester, String tab) => tester
    .widget<ListView>(find.byKey(PageStorageKey<String>('campus-list-$tab')))
    .controller!;

class _DelayedGrades extends FakeCampusRepository {
  Completer<CampusDataset>? grades;
  @override
  Future<CampusDataset> dataset(String key, {CancelToken? cancelToken}) {
    if (key == 'grades' && grades != null) return grades!.future;
    return super.dataset(key, cancelToken: cancelToken);
  }
}

class _LongMessages extends FakeCampusRepository {
  @override
  Future<CampusDataset> dataset(String key, {CancelToken? cancelToken}) async {
    final data = await super.dataset(key, cancelToken: cancelToken);
    if (key != 'messages') return data;
    return CampusDataset(
      key: data.key,
      status: data.status,
      updatedAt: data.updatedAt,
      metrics: data.metrics,
      columns: data.columns,
      rows: data.rows,
      events: data.events,
      series: data.series,
      messages: [for (var i = 0; i < 40; i++) ...data.messages],
      teachingDay: data.teachingDay,
    );
  }
}

class _DelayedResume extends FakeCampusRepository {
  Completer<CampusStatus>? statusResponse;
  Completer<CampusDataset>? messagesResponse;
  Completer<CampusDataset>? timetableResponse;

  @override
  Future<CampusStatus> status({CancelToken? cancelToken}) {
    final response = statusResponse;
    if (response != null) return response.future;
    return super.status(cancelToken: cancelToken);
  }

  @override
  Future<CampusDataset> dataset(String key, {CancelToken? cancelToken}) {
    if (key == 'messages' && messagesResponse != null) {
      return messagesResponse!.future;
    }
    if (key == 'timetable' && timetableResponse != null) {
      return timetableResponse!.future;
    }
    return super.dataset(key, cancelToken: cancelToken);
  }
}

void main() {
  testWidgets('campus tab return preserves selected notice search', (
    tester,
  ) async {
    final visible = ValueNotifier(true);
    addTearDown(visible.dispose);
    await tester.pumpWidget(
      campusTestApp(
        FakeCampusRepository(),
        child: ValueListenableBuilder<bool>(
          valueListenable: visible,
          builder: (_, active, _) =>
              TickerMode(enabled: active, child: const CampusPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await _select(tester, 'messages');
    await tester.enterText(_search, '图书馆');
    await tester.pumpAndSettle();
    visible.value = false;
    await tester.pumpAndSettle();
    expect(find.byType(GfTabBar), findsOneWidget);
    expect(find.byType(GfSkeleton), findsWidgets);
    expect(find.byType(TextField), findsNothing);
    visible.value = true;
    await tester.pumpAndSettle();
    expect(tester.widget<GfTabBar>(find.byType(GfTabBar)).selected, 'messages');
    expect(tester.widget<TextField>(_search).controller!.text, '图书馆');
    expect(find.text('校园文化节报名开始（演示）'), findsNothing);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('campus navigation survives app backgrounding', (tester) async {
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    await _select(tester, 'messages');
    await tester.enterText(_search, '图书馆');
    await tester.pumpAndSettle();

    for (final state in [
      AppLifecycleState.inactive,
      AppLifecycleState.hidden,
      AppLifecycleState.paused,
    ]) {
      tester.binding.handleAppLifecycleStateChanged(state);
      await tester.pumpAndSettle();
      expect(find.byType(GfTabBar), findsOneWidget);
      expect(find.byType(GfSkeleton), findsWidgets);
      expect(find.byType(TextField), findsNothing);
    }

    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pumpAndSettle();

    expect(tester.widget<GfTabBar>(find.byType(GfTabBar)).selected, 'messages');
    expect(tester.widget<TextField>(_search).controller!.text, '图书馆');
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets(
    'campus keeps its shell and skeleton until fresh resume data arrives',
    (tester) async {
      final repo = _DelayedResume();
      await tester.pumpWidget(campusTestApp(repo));
      await tester.pumpAndSettle();
      await _select(tester, 'messages');
      await tester.enterText(_search, '图书馆');
      await tester.pumpAndSettle();
      final campusTitle = AppLocalizations.of(
        tester.element(find.byType(CampusPage)),
      ).campusTitle;

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      await tester.pumpAndSettle();
      expect(find.text('校园文化节报名开始（演示）'), findsNothing);
      expect(find.text(campusTitle), findsOneWidget);
      expect(find.byType(GfTabBar), findsOneWidget);

      repo.statusResponse = Completer<CampusStatus>();
      repo.messagesResponse = Completer<CampusDataset>();
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.hidden);
      await tester.pump();
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      await tester.pump();
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.byType(GfTabBar), findsOneWidget);
      expect(
        tester.widget<GfTabBar>(find.byType(GfTabBar)).selected,
        'messages',
      );
      expect(find.byType(GfSkeleton), findsWidgets);
      expect(find.byType(TextField), findsNothing);
      expect(find.text('校园文化节报名开始（演示）'), findsNothing);

      repo.statusResponse!.complete(testStatus);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.byType(GfSkeleton), findsWidgets);
      expect(find.text('校园文化节报名开始（演示）'), findsNothing);

      repo.messagesResponse!.complete(campusFixture('messages'));
      await tester.pumpAndSettle();
      expect(tester.widget<TextField>(_search).controller!.text, '图书馆');
      expect(find.text('图书馆开放时间调整（演示）'), findsWidgets);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );

  for (final succeeds in [true, false]) {
    testWidgets('current section resumes before unrelated data: $succeeds', (
      tester,
    ) async {
      final repo = _DelayedResume();
      await tester.pumpWidget(campusTestApp(repo));
      await tester.pumpAndSettle();
      await _select(tester, 'messages');
      _list(tester).jumpTo(120);
      await tester.pump();

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      await tester.pumpAndSettle();
      repo.statusResponse = Completer<CampusStatus>();
      repo.messagesResponse = Completer<CampusDataset>();
      repo.timetableResponse = Completer<CampusDataset>();
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pump();
      expect(_search, findsNothing);
      repo.statusResponse!.complete(testStatus);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      expect(_search, findsNothing);

      if (succeeds) {
        repo.messagesResponse!.complete(campusFixture('messages'));
      } else {
        repo.messagesResponse!.completeError(
          const ApiException(fallbackMessage: 'Messages unavailable'),
        );
      }
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
      final container = ProviderScope.containerOf(
        tester.element(find.byType(CampusPage)),
      );
      final state = container.read(campusControllerProvider);
      expect(state.fetching, contains('timetable'));
      expect(state.snapshot, isNull); // Never commit an incomplete snapshot.
      expect(state.errors.containsKey('messages'), !succeeds);
      final currentSectionVisible = _search.evaluate().isNotEmpty;
      final restoredOffset = _list(tester).offset;

      repo.timetableResponse!.complete(campusFixture('timetable'));
      await tester.pumpAndSettle();
      expect(container.read(campusControllerProvider).snapshot, isNotNull);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
      expect(currentSectionVisible, isTrue);
      if (succeeds) expect(restoredOffset, closeTo(120, 1));
    });
  }

  testWidgets('large-text timetable skeleton keeps its labels unclipped', (
    tester,
  ) async {
    await loadTestFonts(tester);
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(320, 568);
    addTearDown(tester.view.reset);
    final repo = _DelayedResume();
    await tester.pumpWidget(
      campusTestApp(repo, scale: 2, locale: const Locale('de')),
    );
    await tester.pumpAndSettle();
    await _select(tester, 'timetable');
    final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    await tester.pumpAndSettle();
    repo.statusResponse = Completer<CampusStatus>();
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pump();
    final paragraph = tester.renderObject<RenderParagraph>(
      find.descendant(
        of: find.text(l.scheduleTimeAxis),
        matching: find.byType(RichText),
      ),
    );
    final painter = TextPainter(
      text: paragraph.text,
      textDirection: paragraph.textDirection,
      textScaler: paragraph.textScaler,
    )..layout(maxWidth: paragraph.size.width);
    final naturalHeight = painter.height;
    final availableHeight = paragraph.size.height;
    painter.dispose();
    repo.statusResponse!.complete(testStatus);
    await tester.pumpAndSettle();
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
    expect(naturalHeight, lessThanOrEqualTo(availableHeight));
  });

  testWidgets('campus sections retain independent search text', (tester) async {
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    await _select(tester, 'messages');
    await tester.enterText(_search, '图书馆');
    await _select(tester, 'academics');
    await tester.ensureVisible(_search);
    await tester.enterText(_search, '数学');
    await _select(tester, 'messages');
    expect(tester.widget<TextField>(_search).controller!.text, '图书馆');
    await _select(tester, 'academics');
    expect(tester.widget<TextField>(_search).controller!.text, '数学');
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('campus timetable keeps selected week after changing section', (
    tester,
  ) async {
    await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
    await tester.pumpAndSettle();
    await _select(tester, 'timetable');
    await tester.tap(
      find.byTooltip(
        MaterialLocalizations.of(
          tester.element(find.byType(CampusPage)),
        ).nextPageTooltip,
      ),
    );
    await tester.pumpAndSettle();
    await _select(tester, 'messages');
    await _select(tester, 'timetable');
    final l = AppLocalizations.of(tester.element(find.byType(CampusPage)));
    expect(find.text(l.scheduleWeekN(2)), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
  testWidgets(
    'campus restores independent scroll positions after delayed data',
    (tester) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final visible = ValueNotifier(true);
      addTearDown(visible.dispose);
      final repo = _DelayedGrades();
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
      await _select(tester, 'academics');
      _list(tester).jumpTo(450);
      await tester.pumpAndSettle();
      final academicOffset = _list(tester).offset;
      expect(academicOffset, greaterThan(300));
      await _select(tester, 'messages');
      expect(_list(tester).offset, 0);
      _list(tester).jumpTo(120);
      await tester.pumpAndSettle();
      final noticeOffset = _list(tester).offset;
      await _select(tester, 'academics');
      expect(_list(tester).offset, closeTo(academicOffset, 1));
      visible.value = false;
      await tester.pumpAndSettle();
      repo.grades = Completer();
      visible.value = true;
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      expect(
        tester.widget<GfTabBar>(find.byType(GfTabBar)).selected,
        'academics',
      );
      repo.grades!.complete(campusFixture('grades'));
      await tester.pumpAndSettle();
      expect(_list(tester).offset, closeTo(academicOffset, 1));
      await _select(tester, 'messages');
      expect(_list(tester).offset, closeTo(noticeOffset, 1));
      final messageList = find.byKey(
        const PageStorageKey<String>('campus-list-messages'),
      );
      final scrollHandle = tester
          .widget<GfScrollToTop>(
            find
                .ancestor(of: messageList, matching: find.byType(GfScrollToTop))
                .first,
          )
          .controller!;
      expect(scrollHandle.isAttached, isTrue);
      final container = ProviderScope.containerOf(
        tester.element(find.byType(CampusPage)),
      );
      final scrollToTop = container
          .read(tabScrollRegistryProvider)
          .scrollToTop(GfShellDestination.campus);
      await tester.pumpAndSettle();
      await scrollToTop;
      expect(_list(tester).offset, 0);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );

  testWidgets('hidden chrome swipes retain each campus section scroll offset', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final visible = ValueNotifier(true);
    addTearDown(visible.dispose);
    final repo = _LongMessages();
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
    await tester.pump();
    await tester.pump(const Duration(seconds: 1));
    final container = ProviderScope.containerOf(
      tester.element(find.byType(CampusPage)),
    );
    Future<void> selectSection(String tab) async {
      tester.widget<GfTabBar>(find.byType(GfTabBar)).onSelected(tab);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      await tester.pump();
      expect(tester.widget<GfTabBar>(find.byType(GfTabBar)).selected, tab);
    }

    Future<void> swipeTo(int index) async {
      GfTabSwipeProgressScope.onTabSelectedOf(
        tester.element(find.byType(GfTabBar)),
      )!(index);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      await tester.pump();
    }

    Future<void> waitForList(String tab) async {
      final finder = find.byKey(PageStorageKey<String>('campus-list-$tab'));
      for (
        var attempt = 0;
        attempt < 20 && finder.evaluate().isEmpty;
        attempt++
      ) {
        await tester.pump(const Duration(milliseconds: 100));
      }
      expect(finder, findsOneWidget);
    }

    await selectSection('academics');
    await waitForList('academics');
    await selectSection('messages');
    await waitForList('messages');

    container.read(readingChromeProvider).update(48, 48);
    await tester.pump(const Duration(milliseconds: 300));
    await tester.drag(
      find.byKey(const PageStorageKey<String>('campus-list-messages')),
      const Offset(0, -50),
    );
    await tester.pump();
    final messageOffset = _listForTab(tester, 'messages').offset;
    expect(messageOffset, greaterThan(0));

    await selectSection('academics');
    await tester.drag(
      find.byKey(const PageStorageKey<String>('campus-list-academics')),
      const Offset(0, -400),
    );
    await tester.pump();
    final academicOffset = _listForTab(tester, 'academics').offset;
    expect(academicOffset, greaterThan(100));

    await swipeTo(3);
    expect(tester.widget<GfTabBar>(find.byType(GfTabBar)).selected, 'messages');
    await waitForList('messages');
    expect(_listForTab(tester, 'messages').offset, closeTo(messageOffset, 1));

    await swipeTo(2);
    expect(
      tester.widget<GfTabBar>(find.byType(GfTabBar)).selected,
      'academics',
    );
    await waitForList('academics');
    expect(_listForTab(tester, 'academics').offset, closeTo(academicOffset, 1));
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  for (final boundary in ['session', 'binding']) {
    testWidgets('campus navigation clears at $boundary boundary', (
      tester,
    ) async {
      final repo = FakeCampusRepository();
      final visible = ValueNotifier(true);
      addTearDown(visible.dispose);
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
      await _select(tester, 'messages');
      await tester.enterText(_search, '图书馆');
      final container = ProviderScope.containerOf(
        tester.element(find.byType(CampusPage)),
      );
      visible.value = false;
      await tester.pumpAndSettle();
      if (boundary == 'session') {
        container.read(offlineCacheEpochProvider.notifier).invalidate();
      } else {
        repo.current = const CampusStatus(
          enabled: true,
          candidate: null,
          binding: CampusBinding(
            maskedId: 'NEW',
            boundAt: '',
            revision: 'changed',
            needsAuthorization: false,
          ),
        );
      }
      visible.value = true;
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      await tester.pump();
      expect(tester.widget<GfTabBar>(find.byType(GfTabBar)).selected, 'today');
      await _select(tester, 'messages');
      expect(tester.widget<TextField>(_search).controller!.text, isEmpty);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }

  testWidgets('first campus binding resets an observed unbound section', (
    tester,
  ) async {
    final repo = FakeCampusRepository()
      ..current = const CampusStatus(
        enabled: true,
        binding: null,
        candidate: null,
      );
    await tester.pumpWidget(campusTestApp(repo));
    await tester.pumpAndSettle();
    await _select(tester, 'connection');
    final container = ProviderScope.containerOf(
      tester.element(find.byType(CampusPage)),
    );
    final controller = container.read(campusControllerProvider.notifier);
    await controller.refresh();
    await tester.pumpAndSettle();
    expect(
      tester.widget<GfTabBar>(find.byType(GfTabBar)).selected,
      'connection',
    );
    await repo.confirm();
    await controller.refresh();
    await tester.pumpAndSettle();
    expect(tester.widget<GfTabBar>(find.byType(GfTabBar)).selected, 'today');
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('failed refresh retains notice filter and position', (
    tester,
  ) async {
    final repo = FakeCampusRepository();
    await tester.pumpWidget(campusTestApp(repo));
    await tester.pumpAndSettle();
    await _select(tester, 'messages');
    await tester.enterText(_search, '演示');
    _list(tester).jumpTo(90);
    await tester.pumpAndSettle();
    final offset = _list(tester).offset;
    repo.todayError = const ApiException(fallbackMessage: 'Unavailable');
    final container = ProviderScope.containerOf(
      tester.element(find.byType(CampusPage)),
    );
    await container.read(campusControllerProvider.notifier).refresh();
    await tester.pumpAndSettle();
    expect(tester.widget<TextField>(_search).controller!.text, '演示');
    expect(_list(tester).offset, closeTo(offset, 1));
    expect(find.text('图书馆开放时间调整（演示）'), findsWidgets);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
  for (final locale in ['zh', 'en', 'ja', 'de']) {
    testWidgets('campus query survives route cover at 200% in $locale', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(
        campusTestApp(FakeCampusRepository(), locale: Locale(locale), scale: 2),
      );
      await tester.pumpAndSettle();
      await _select(tester, 'messages');
      await tester.enterText(_search, '图书馆');
      await tester.pumpAndSettle();
      final navigator = Navigator.of(tester.element(find.byType(CampusPage)));
      unawaited(
        navigator.push<void>(
          MaterialPageRoute(
            builder: (_) => const Scaffold(body: Text('Public campus tool')),
          ),
        ),
      );
      await tester.pumpAndSettle();
      navigator.pop();
      await tester.pumpAndSettle();
      expect(
        tester.widget<GfTabBar>(find.byType(GfTabBar)).selected,
        'messages',
      );
      expect(tester.widget<TextField>(_search).controller!.text, '图书馆');
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }

  testWidgets(
    'authorization loss clears filters without trapping connection tab',
    (tester) async {
      final repo = FakeCampusRepository();
      await tester.pumpWidget(campusTestApp(repo));
      await tester.pumpAndSettle();
      await _select(tester, 'messages');
      await tester.enterText(_search, '图书馆');
      final container = ProviderScope.containerOf(
        tester.element(find.byType(CampusPage)),
      );
      container
          .read(campusControllerProvider.notifier)
          .invalidateForError(
            const ApiException(
              messageCode: 'campus.authorizationRequired',
              fallbackMessage: 'Authorize',
            ),
          );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      await tester.pump();
      expect(tester.widget<GfTabBar>(find.byType(GfTabBar)).selected, 'today');
      await _select(tester, 'connection');
      expect(
        tester.widget<GfTabBar>(find.byType(GfTabBar)).selected,
        'connection',
      );
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );
  for (final boundary in ['site', 'logout']) {
    testWidgets('mounted campus forgets navigation after $boundary', (
      tester,
    ) async {
      final repo = FakeCampusRepository();
      await tester.pumpWidget(campusTestApp(repo));
      await tester.pumpAndSettle();
      final pageState = tester.state(find.byType(CampusPage));
      final container = ProviderScope.containerOf(
        tester.element(find.byType(CampusPage)),
      );
      await _select(tester, 'messages');
      await tester.enterText(_search, '图书馆');
      await tester.pumpWidget(
        campusTestApp(
          boundary == 'site' ? FakeCampusRepository() : repo,
          signedIn: boundary != 'logout',
          database: container.read(offlineDatabaseProvider),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      await tester.pump();
      expect(tester.state(find.byType(CampusPage)), same(pageState));
      if (boundary == 'logout') {
        container.invalidate(currentUserProvider);
        await tester.pumpAndSettle();
        expect(find.byType(GfTabBar), findsNothing);
        await tester.pumpWidget(
          campusTestApp(
            repo,
            database: container.read(offlineDatabaseProvider),
          ),
        );
        container.invalidate(currentUserProvider);
        await tester.pumpAndSettle();
      }
      expect(tester.widget<GfTabBar>(find.byType(GfTabBar)).selected, 'today');
      await _select(tester, 'messages');
      expect(tester.widget<TextField>(_search).controller!.text, isEmpty);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }

  testWidgets('campus timetable restores week across bottom destinations', (
    tester,
  ) async {
    final visible = ValueNotifier(true);
    addTearDown(visible.dispose);
    await tester.pumpWidget(
      campusTestApp(
        FakeCampusRepository(),
        child: ValueListenableBuilder<bool>(
          valueListenable: visible,
          builder: (_, active, _) =>
              TickerMode(enabled: active, child: const CampusPage()),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await _select(tester, 'timetable');
    final element = tester.element(find.byType(CampusPage));
    final l = AppLocalizations.of(element);
    await tester.tap(
      find.byTooltip(MaterialLocalizations.of(element).nextPageTooltip),
    );
    await tester.pumpAndSettle();
    visible.value = false;
    await tester.pumpAndSettle();
    visible.value = true;
    await tester.pumpAndSettle();
    expect(find.text(l.scheduleWeekN(2)), findsOneWidget);
    await tester.tap(find.text(l.scheduleCurrentWeek));
    await tester.pumpAndSettle();
    expect(find.text(l.scheduleWeekN(1)), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });
  for (final error in [
    const ApiException(statusCode: 401, fallbackMessage: 'Expired'),
    const ApiException(
      messageCode: 'campus.connectionChanged',
      fallbackMessage: 'Changed',
    ),
    const ApiException(
      messageCode: 'permission.userFrozen',
      fallbackMessage: 'Frozen',
    ),
  ]) {
    testWidgets(
      'identity rejection ${error.messageCode ?? error.statusCode} clears operation state',
      (tester) async {
        await tester.pumpWidget(campusTestApp(FakeCampusRepository()));
        await tester.pumpAndSettle();
        await _select(tester, 'messages');
        await tester.enterText(_search, '图书馆');
        final container = ProviderScope.containerOf(
          tester.element(find.byType(CampusPage)),
        );
        final controller = container.read(campusControllerProvider.notifier);
        controller.invalidateForError(error);
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 300));
        await tester.pump();
        expect(
          tester.widget<GfTabBar>(find.byType(GfTabBar)).selected,
          'today',
        );
        await controller.refresh();
        await tester.pumpAndSettle();
        await _select(tester, 'messages');
        expect(tester.widget<TextField>(_search).controller!.text, isEmpty);
        await tester.pumpWidget(const SizedBox());
        await tester.pumpAndSettle();
      },
    );
  }
  for (final input in ['drag', 'wheel']) {
    testWidgets('$input cancels restoration while section data is pending', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final visible = ValueNotifier(true);
      addTearDown(visible.dispose);
      final repo = _DelayedGrades();
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
      await _select(tester, 'academics');
      _list(tester).jumpTo(450);
      await tester.pumpAndSettle();
      visible.value = false;
      await tester.pumpAndSettle();
      repo.grades = Completer();
      visible.value = true;
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      final list = find
          .descendant(
            of: find.byType(GfScrollToTop),
            matching: find.byType(ListView),
          )
          .first;
      final beforeInputOffset = _list(tester).offset;
      if (input == 'drag') {
        await tester.drag(list, const Offset(0, 100));
      } else {
        await tester.sendEventToBinding(
          PointerScrollEvent(
            position: tester.getCenter(list),
            scrollDelta: const Offset(0, -100),
          ),
        );
      }
      await tester.pump(const Duration(seconds: 1));
      final selectedOffset = _list(tester).offset;
      expect(selectedOffset, lessThan(beforeInputOffset));
      repo.grades!.complete(campusFixture('grades'));
      await tester.pumpAndSettle();
      expect(_list(tester).offset, closeTo(selectedOffset, 1));
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }
}
