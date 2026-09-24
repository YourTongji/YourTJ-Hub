import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/navigation/tab_scroll_registry.dart';
import 'package:flutter/material.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'package:ui_kit/ui_kit.dart';

import 'campus_native_test.dart' show campusTestApp;
import 'fixtures/campus_fixtures.dart';

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

class _DelayedGrades extends FakeCampusRepository {
  Completer<CampusDataset>? grades;
  @override
  Future<CampusDataset> dataset(String key, {CancelToken? cancelToken}) {
    if (key == 'grades' && grades != null) return grades!.future;
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
    expect(find.byType(TextField), findsNothing);
    visible.value = true;
    await tester.pumpAndSettle();
    expect(tester.widget<GfTabBar>(find.byType(GfTabBar)).selected, 'messages');
    expect(tester.widget<TextField>(_search).controller!.text, '图书馆');
    expect(find.text('校园文化节报名开始（演示）'), findsNothing);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
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
      final container = ProviderScope.containerOf(
        tester.element(find.byType(CampusPage)),
      );
      unawaited(
        container
            .read(tabScrollRegistryProvider)
            .scrollToTop(GfShellDestination.campus),
      );
      await tester.pumpAndSettle();
      expect(_list(tester).offset, 0);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    },
  );

  for (final boundary in ['background', 'session', 'binding']) {
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
      if (boundary == 'background') {
        tester.binding.handleAppLifecycleStateChanged(
          AppLifecycleState.inactive,
        );
        await tester.pumpAndSettle();
        tester.binding.handleAppLifecycleStateChanged(
          AppLifecycleState.resumed,
        );
      } else {
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
      }
      await tester.pumpAndSettle();
      expect(tester.widget<GfTabBar>(find.byType(GfTabBar)).selected, 'today');
      await _select(tester, 'messages');
      expect(tester.widget<TextField>(_search).controller!.text, isEmpty);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }

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
      await tester.pumpAndSettle();
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
      await tester.pumpAndSettle();
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
        await tester.pumpAndSettle();
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
      final list = find.byType(ListView).first;
      if (input == 'drag') {
        await tester.drag(list, const Offset(0, -100));
      } else {
        await tester.sendEventToBinding(
          PointerScrollEvent(
            position: tester.getCenter(list),
            scrollDelta: const Offset(0, 100),
          ),
        );
      }
      await tester.pump(const Duration(seconds: 1));
      final selectedOffset = _list(tester).offset;
      expect(selectedOffset, lessThan(300));
      repo.grades!.complete(campusFixture('grades'));
      await tester.pumpAndSettle();
      expect(_list(tester).offset, closeTo(selectedOffset, 1));
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }
}
