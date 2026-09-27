import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/campus/campus_memory_cache.dart';
import 'package:forum_app/src/pages/campus/campus_page.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/offline/campus_snapshot_store.dart';
import 'campus_native_test.dart' show campusTestApp;
import 'fixtures/campus_fixtures.dart';

class ControlledCampusRepository extends FakeCampusRepository {
  Completer<CampusStatus>? pendingStatus;
  Object? statusError;
  final errors = <String, Object>{};
  final pending = <String, Completer<CampusDataset>>{};
  int statusCalls = 0;
  @override
  Future<CampusStatus> status({CancelToken? cancelToken}) async {
    statusCalls++;
    final value = await super.status(cancelToken: cancelToken);
    if (statusError != null) throw statusError!;
    return pendingStatus == null ? value : pendingStatus!.future;
  }

  @override
  Future<CampusDataset> dataset(String key, {CancelToken? cancelToken}) async {
    final value = await super.dataset(key, cancelToken: cancelToken);
    if (errors[key] != null) throw errors[key]!;
    return pending[key]?.future ?? value;
  }
}

const failure = ApiException(fallbackMessage: 'Network unavailable');

void main() {
  test(
    'reuse waits for fresh binding status and never restores grades',
    () async {
      final repo = ControlledCampusRepository();
      final cache = CampusMemoryCache();
      addTearDown(cache.dispose);
      final first = CampusController(repo, cache: cache);
      await first.refresh();
      await first.loadTab('academics');
      first.dispose();
      final count = repo.requested.length;
      repo.pendingStatus = Completer();
      final second = CampusController(repo, cache: cache);
      addTearDown(second.dispose);
      final ready = second.refresh(reuseCache: true);
      expect(second.state.data, isEmpty);
      expect(repo.requested.length, count);
      repo.pendingStatus!.complete(testStatus);
      await ready;
      expect(repo.statusCalls, 2);
      expect(repo.requested.length, count);
      expect(
        second.state.data.keys,
        unorderedEquals({...campusPersistentKeys, 'messages'}),
      );
      await second.loadTab('academics');
      expect(repo.requested.where((key) => key == 'grades').length, 2);
    },
  );

  for (final kind in ['makeup', 'holiday']) {
    test(
      'cached today preserves server teaching-day semantics: $kind',
      () async {
        final base = campusFixture('today');
        final today = CampusDataset(
          key: 'today',
          status: kind == 'holiday' ? 'empty' : 'ready',
          updatedAt: '',
          metrics: [],
          columns: [],
          rows: [],
          series: [],
          events: kind == 'holiday' ? [] : base.events,
          teachingDay: CampusTeachingDay(
            date: base.teachingDay!.date,
            sourceDate: kind == 'holiday' ? '' : base.teachingDay!.sourceDate,
            kind: kind,
            label: '国庆节',
            sectionCount: 11,
          ),
        );
        final cache = CampusMemoryCache();
        addTearDown(cache.dispose);
        final repo = ControlledCampusRepository()..todayOverride = today;
        final first = CampusController(repo, cache: cache);
        await first.refresh();
        first.dispose();
        final second = CampusController(repo, cache: cache);
        addTearDown(second.dispose);
        await second.refresh(reuseCache: true);
        expect(second.state.data['today'], same(today));
        expect(repo.requested.where((key) => key == 'today').length, 1);
        expect(repo.requested, contains('timetable'));
      },
    );
  }

  test(
    'visible clock never turns cache expiry into background polling',
    () async {
      var now = DateTime.utc(2026, 9, 20, 1);
      final cache = CampusMemoryCache(now: () => now);
      addTearDown(cache.dispose);
      final repo = ControlledCampusRepository()..now = () => now;
      final controller = CampusController(repo, cache: cache);
      addTearDown(controller.dispose);
      await controller.refresh();
      final count = repo.requested.length;
      now = now.add(const Duration(minutes: 4));
      await controller.enterTab('today');
      expect(repo.requested.length, count);
      now = now.add(const Duration(minutes: 1));
      await controller.refreshVisible();
      expect(repo.requested.length, count);
    },
  );

  test(
    'manual refresh keeps last successful content when a request fails',
    () async {
      final repo = ControlledCampusRepository();
      final controller = CampusController(repo);
      addTearDown(controller.dispose);
      await controller.refresh();
      final profile = controller.state.data['profile'];
      repo.pendingStatus = Completer();
      repo.pending['profile'] = Completer();
      final refreshing = controller.refresh();
      expect(controller.state.data['profile'], same(profile));
      expect(controller.state.refreshing, isTrue);
      repo.pendingStatus!.complete(testStatus);
      await Future<void>.delayed(Duration.zero);
      expect(controller.state.data['profile'], same(profile));
      repo.pending['profile']!.completeError(failure);
      await refreshing;
      expect(controller.state.refreshing, isFalse);
      expect(controller.state.data['profile'], same(profile));
      expect(controller.state.errors['profile'], failure);
      expect(
        controller.cache.restore(testBinding.revision),
        contains('profile'),
      );
    },
  );

  test(
    'changed binding is cleared before replacement requests finish',
    () async {
      final repo = ControlledCampusRepository();
      final controller = CampusController(repo);
      addTearDown(controller.dispose);
      await controller.refresh();
      repo.current = const CampusStatus(
        enabled: true,
        candidate: null,
        binding: CampusBinding(
          maskedId: 'NE••••WW',
          boundAt: '',
          revision: 'replacement',
          needsAuthorization: false,
        ),
      );
      repo.pending['profile'] = Completer();
      final refreshing = controller.refresh(reuseCache: true);
      await Future<void>.delayed(Duration.zero);
      expect(controller.state.data, isNot(contains('profile')));
      repo.pending['profile']!.complete(campusFixture('profile'));
      await refreshing;
      expect(controller.state.status!.binding!.revision, 'replacement');
      expect(repo.requested.where((k) => k == 'profile').length, 2);
    },
  );

  for (final status in [
    const CampusStatus(enabled: true, binding: null, candidate: null),
    const CampusStatus(enabled: false, binding: testBinding, candidate: null),
    const CampusStatus(
      enabled: true,
      candidate: null,
      binding: CampusBinding(
        maskedId: 'DE••••MO',
        boundAt: '',
        revision: 'demo-revision',
        needsAuthorization: true,
      ),
    ),
  ]) {
    test(
      'unbound, disabled or expired status rejects cached data: $status',
      () async {
        final repo = ControlledCampusRepository();
        final controller = CampusController(repo);
        addTearDown(controller.dispose);
        await controller.refresh();
        repo.current = status;
        await controller.refresh(reuseCache: true);
        expect(controller.state.data, isEmpty);
        expect(controller.cache.restore(testBinding.revision), isEmpty);
      },
    );
  }

  test('failed status validation retains last successful data', () async {
    final repo = ControlledCampusRepository();
    final controller = CampusController(repo);
    addTearDown(controller.dispose);
    await controller.refresh();
    repo.statusError = failure;
    await controller.refresh(reuseCache: true);
    expect(controller.state.data, isNotEmpty);
    expect(controller.state.error, failure);
    expect(controller.cache.restore(testBinding.revision), isNotEmpty);
  });

  for (final error in [
    const ApiException(
      fallbackMessage: '',
      messageCode: 'campus.authorizationRequired',
    ),
    const ApiException(
      fallbackMessage: '',
      messageCode: 'campus.connectionChanged',
    ),
    const ApiException(fallbackMessage: '', statusCode: 401),
    const ApiException(
      fallbackMessage: '',
      statusCode: 403,
      messageCode: 'permission.userFrozen',
    ),
    const ApiException(
      fallbackMessage: '',
      statusCode: 403,
      messageCode: 'permission.resolveFailed',
    ),
    const ApiException(
      fallbackMessage: '',
      statusCode: 503,
      messageCode: 'campus.disabled',
    ),
  ]) {
    test(
      'identity error clears cache and fences other inflight responses: $error',
      () async {
        final repo = ControlledCampusRepository();
        final controller = CampusController(repo);
        addTearDown(controller.dispose);
        await controller.refresh();
        repo.pending['profile'] = Completer();
        repo.errors['messages'] = error;
        final refreshing = controller.refresh();
        await Future<void>.delayed(Duration.zero);
        expect(controller.state.data, isEmpty);
        repo.pending['profile']!.complete(campusFixture('profile'));
        await refreshing;
        expect(controller.state.data, isEmpty);
        expect(controller.cache.restore(testBinding.revision), isEmpty);
      },
    );
  }

  for (final code in ['auth.csrf.rejected', 'permission.emailRequired']) {
    test('non-identity 403 preserves binding and cached data: $code', () async {
      final repo = ControlledCampusRepository()
        ..confirmError = ApiException(
          fallbackMessage: '',
          statusCode: 403,
          messageCode: code,
        );
      final controller = CampusController(repo);
      addTearDown(controller.dispose);
      await controller.refresh();
      final before = controller.state.data;
      expect(
        await controller.change((c) => repo.confirm(cancelToken: c)),
        isFalse,
      );
      expect(controller.state.status?.binding?.revision, testBinding.revision);
      expect(controller.state.data, before);
      expect(controller.state.error, repo.confirmError);
      expect(
        controller.cache.restore(testBinding.revision).keys,
        unorderedEquals({...campusPersistentKeys, 'messages'}),
      );
    });
  }

  test('Shanghai midnight invalidates teaching data without polling', () async {
    var now = DateTime.utc(2026, 9, 20, 15, 59);
    final cache = CampusMemoryCache(now: () => now);
    addTearDown(cache.dispose);
    final repo = ControlledCampusRepository()..now = () => now;
    final controller = CampusController(repo, cache: cache);
    addTearDown(controller.dispose);
    await controller.refresh();
    now = now.add(const Duration(minutes: 2));
    await controller.refreshVisible();
    expect(controller.state.data, isNot(contains('today')));
    expect(repo.requested.where((k) => k == 'today').length, 1);
    expect(repo.requested.where((k) => k == 'calendar').length, 1);
    expect(repo.requested.where((k) => k == 'messages').length, 1);
  });

  test('request spanning midnight cannot cache an old daily dataset', () async {
    var now = DateTime.utc(2026, 9, 20, 15, 59);
    final cache = CampusMemoryCache(now: () => now);
    addTearDown(cache.dispose);
    final repo = ControlledCampusRepository()
      ..now = (() => now)
      ..pending['today'] = Completer();
    final controller = CampusController(repo, cache: cache);
    addTearDown(controller.dispose);
    final refreshing = controller.refresh();
    await Future<void>.delayed(Duration.zero);
    now = now.add(const Duration(minutes: 2));
    repo.pending['today']!.complete(
      campusFixture('today', now: now.subtract(const Duration(minutes: 2))),
    );
    await refreshing;
    expect(controller.state.data, isNot(contains('today')));
    expect(cache.restore(testBinding.revision), isNot(contains('today')));
    repo.pending.clear();
    await controller.refreshVisible();
    expect(controller.state.data['today'], isNull);
    await controller.refresh();
    expect(controller.state.data['today']?.teachingDay?.date, '2026-09-21');
    expect(repo.requested.where((k) => k == 'today').length, 2);
  });

  test(
    'backgrounding fences late data even before the view disposes',
    () async {
      final cache = CampusMemoryCache();
      addTearDown(cache.dispose);
      final repo = ControlledCampusRepository()
        ..pending['profile'] = Completer();
      final controller = CampusController(repo, cache: cache);
      addTearDown(controller.dispose);
      final refreshing = controller.refresh();
      await Future<void>.delayed(Duration.zero);
      cache.didChangeAppLifecycleState(AppLifecycleState.inactive);
      repo.pending['profile']!.complete(campusFixture('profile'));
      await refreshing;
      cache.didChangeAppLifecycleState(AppLifecycleState.resumed);
      expect(controller.state.data, isNot(contains('profile')));
      expect(cache.restore(testBinding.revision), isEmpty);
    },
  );

  test('unbind clears cache and a later controller must fetch again', () async {
    final cache = CampusMemoryCache();
    addTearDown(cache.dispose);
    final repo = ControlledCampusRepository();
    final controller = CampusController(repo, cache: cache);
    await controller.refresh();
    expect(
      await controller.change(
        (cancel) => repo.unbind(testBinding.revision, cancelToken: cancel),
      ),
      isTrue,
    );
    expect(controller.state.data, isEmpty);
    controller.dispose();
    repo.current = testStatus;
    final next = CampusController(repo, cache: cache);
    addTearDown(next.dispose);
    await next.refresh(reuseCache: true);
    expect(repo.requested.where((key) => key == 'profile').length, 2);
  });

  test('rejected or unavailable datasets never enter the reusable cache', () {
    final cache = CampusMemoryCache();
    addTearDown(cache.dispose);
    cache.restore(testBinding.revision);
    for (final key in ['grades', 'summary', 'cet', 'terms']) {
      cache.put(
        testBinding.revision,
        campusFixture(key),
        cache.now(),
        cache.generation,
      );
    }
    final unavailable = CampusDataset(
      key: 'messages',
      status: 'unavailable',
      updatedAt: '',
      metrics: [],
      columns: [],
      rows: [],
      events: [],
      series: [],
    );
    cache.put(testBinding.revision, unavailable, cache.now(), cache.generation);
    expect(cache.restore(testBinding.revision), isEmpty);
  });

  testWidgets('return from notice verifies status and reuses overview data', (
    tester,
  ) async {
    final repo = ControlledCampusRepository();
    await tester.pumpWidget(campusTestApp(repo));
    await tester.pumpAndSettle();
    final notice = find.text('图书馆开放时间调整（演示）').first;
    await tester.drag(find.byType(ListView), const Offset(0, -450));
    await tester.pumpAndSettle();
    await Scrollable.ensureVisible(tester.element(notice), alignment: 0.5);
    await tester.pumpAndSettle();
    await tester.tap(notice);
    await tester.pumpAndSettle();
    expect(find.byType(SelectableText), findsOneWidget);
    final requests = repo.requested.where((key) => key != 'detail').length;
    final statuses = repo.statusCalls;
    Navigator.of(tester.element(find.byType(SelectableText))).pop();
    await tester.pumpAndSettle();
    expect(find.textContaining('演示同学'), findsOneWidget);
    expect(repo.statusCalls, greaterThan(statuses));
    expect(repo.requested.where((key) => key != 'detail').length, requests);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  testWidgets('calendar failure still renders server-adjusted today only', (
    tester,
  ) async {
    final repo = ControlledCampusRepository()..errors['calendar'] = failure;
    await tester.pumpWidget(campusTestApp(repo));
    await tester.pumpAndSettle();
    expect(find.text('第四周周二的数学'), findsOneWidget);
    expect(find.textContaining('国庆补课'), findsOneWidget);
    expect(repo.requested, contains('timetable'));
    expect(find.textContaining('课程 '), findsNothing);
    expect(find.text('教学楼 A101 · 四平路 · 示例教师'), findsNothing);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
  });

  for (final boundary in ['background', 'session', 'expiry']) {
    testWidgets('hidden campus restores persistent data after $boundary', (
      tester,
    ) async {
      final visible = ValueNotifier(true);
      addTearDown(visible.dispose);
      final repo = ControlledCampusRepository();
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
      final container = ProviderScope.containerOf(
        tester.element(find.byType(CampusPage)),
      );
      final requests = repo.requested.length;
      visible.value = false;
      await tester.pumpAndSettle();
      switch (boundary) {
        case 'background':
          tester.binding.handleAppLifecycleStateChanged(
            AppLifecycleState.inactive,
          );
          await tester.pumpAndSettle();
          tester.binding.handleAppLifecycleStateChanged(
            AppLifecycleState.resumed,
          );
        case 'session':
          container.read(offlineCacheEpochProvider.notifier).invalidate();
        case 'expiry':
          await tester.pump(const Duration(minutes: 5));
      }
      visible.value = true;
      await tester.pump();
      for (
        var i = 0;
        i < 100 && container.read(campusControllerProvider).refreshing;
        i++
      ) {
        await tester.pump(const Duration(milliseconds: 10));
      }
      expect(container.read(campusControllerProvider).refreshing, isFalse);
      expect(repo.statusCalls, 2);
      expect(repo.requested.length, requests);
      await tester.pumpWidget(const SizedBox());
      await tester.pumpAndSettle();
    });
  }
}
