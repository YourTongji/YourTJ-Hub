import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/notifications/notifications_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/app_refresh_indicator.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => 'session';
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

GfApiClient _client() => GfApiClient(dio: Dio(), tokenStorage: _Tokens());

class _Pages extends PageRepository {
  _Pages() : super(_client());
  @override
  Future<PagePayload> fetch(String path) async =>
      parsePayload(homePayloadJson());
}

class _Notifications extends NotificationRepository {
  _Notifications() : super(_client());
  final requests = <(String, int, Completer<NotificationListResponse>)>[];
  final markAll = <Completer<bool>>[];
  final markOne = <(int, Completer<bool>)>[];
  @override
  Future<NotificationListResponse> fetchNotifications({
    String filter = 'all',
    int cursor = 0,
    int limit = 20,
  }) {
    final result = Completer<NotificationListResponse>();
    requests.add((filter, cursor, result));
    return result.future;
  }

  @override
  Future<bool> markAllNotificationsRead() {
    final result = Completer<bool>();
    markAll.add(result);
    return result.future;
  }

  @override
  Future<bool> markNotificationRead({required int notificationId}) {
    final result = Completer<bool>();
    markOne.add((notificationId, result));
    return result.future;
  }
}

NotificationListResponse _page(
  String text, {
  int id = 1,
  bool more = false,
  bool isRead = false,
}) => NotificationListResponse(
  items: [
    NotificationPayload(
      id: id,
      eventType: 'system',
      isRead: isRead,
      createdAt: '2026-09-24',
      title: text,
      content: '',
      actor: const NotificationActorPayload(id: 1, username: 'Alice'),
      payload: const NotificationInnerPayload(actorId: 1),
    ),
  ],
  nextCursor: more ? id : 0,
  hasNext: more,
  unreadCount: 1,
);

Future<ProviderContainer> _mount(
  WidgetTester tester,
  _Notifications repo,
) async {
  final container = ProviderContainer(
    overrides: [
      notificationRepositoryProvider.overrideWithValue(repo),
      tokenStorageProvider.overrideWithValue(_Tokens()),
      pageRepositoryProvider.overrideWithValue(_Pages()),
    ],
  );
  addTearDown(container.dispose);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: const Locale('en'),
        home: const NotificationsPage(),
      ),
    ),
  );
  await tester.pump();
  return container;
}

void main() {
  testWidgets(
    'single read retains unread on failure and rejects an older read response',
    (tester) async {
      final repo = _Notifications();
      await _mount(tester, repo);
      repo.requests.first.$3.complete(_page('One notification'));
      await tester.pumpAndSettle();
      var row = tester.widget<GfNotificationRow>(
        find.byType(GfNotificationRow),
      );
      expect(row.onMarkRead, isNotNull);
      row.onMarkRead!();
      row.onMarkRead!();
      await tester.pump();
      expect(repo.markOne, hasLength(1));
      expect(
        tester
            .widget<GfNotificationRow>(find.byType(GfNotificationRow))
            .onMarkRead,
        isNull,
      );
      repo.markOne.single.$2.completeError(StateError('single read failed'));
      await tester.pumpAndSettle();
      row = tester.widget<GfNotificationRow>(find.byType(GfNotificationRow));
      expect(row.unread, isTrue);
      expect(find.textContaining('single read failed'), findsWidgets);

      final refresh = tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      row.onMarkRead!();
      repo.markOne.last.$2.complete(true);
      await tester.pump();
      repo.requests.last.$3.complete(_page('One notification'));
      await refresh;
      await tester.pumpAndSettle();
      expect(
        tester.widget<GfNotificationRow>(find.byType(GfNotificationRow)).unread,
        isFalse,
      );
    },
  );

  testWidgets(
    'pagination stays in its generation and cannot pollute a new filter',
    (tester) async {
      final repo = _Notifications();
      await _mount(tester, repo);
      repo.requests.first.$3.complete(_page('All first', more: true));
      await tester.pump();
      await tester.pump();
      expect(repo.requests.last.$2, 1);
      final pagination = repo.requests.last.$3;
      await tester.tap(find.text('Unread'));
      await tester.pump();
      repo.requests.last.$3.complete(_page('Unread chosen', id: 20));
      await tester.pumpAndSettle();
      pagination.complete(_page('Old page', id: 2));
      await tester.pumpAndSettle();
      expect(find.text('Unread chosen'), findsOneWidget);
      expect(find.text('Old page'), findsNothing);
    },
  );

  testWidgets('no-progress pagination pauses automatic requests with retry', (
    tester,
  ) async {
    final repo = _Notifications();
    await _mount(tester, repo);
    repo.requests.first.$3.complete(_page('First', more: true));
    await tester.pump();
    await tester.pump();
    repo.requests.last.$3.complete(_page('First', more: true));
    await tester.pumpAndSettle();
    final footer = tester.widget<GfListFooter>(find.byType(GfListFooter));
    final l = AppLocalizations.of(
      tester.element(find.byType(NotificationsPage)),
    );
    expect(footer.error, l.commonLoadFailed);
    expect(repo.requests, hasLength(2));
    await tester.pump(const Duration(seconds: 1));
    expect(repo.requests, hasLength(2));
    footer.onLoadMore();
    repo.requests.last.$3.complete(_page('Next', id: 2));
    await tester.pumpAndSettle();
    expect(find.text('Next'), findsOneWidget);
  });

  testWidgets('server-confirmed read clears a failed read retry', (
    tester,
  ) async {
    final repo = _Notifications();
    await _mount(tester, repo);
    repo.requests.first.$3.complete(_page('One notification'));
    await tester.pumpAndSettle();
    tester
        .widget<GfNotificationRow>(find.byType(GfNotificationRow))
        .onMarkRead!();
    repo.markOne.single.$2.completeError(
      const ApiException(fallbackMessage: 'read timed out'),
    );
    await tester.pumpAndSettle();
    expect(find.text('Retry'), findsOneWidget);
    final refresh = tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    repo.requests.last.$3.complete(_page('One notification', isRead: true));
    await refresh;
    await tester.pumpAndSettle();
    expect(
      tester.widget<GfNotificationRow>(find.byType(GfNotificationRow)).unread,
      isFalse,
    );
    expect(find.text('Retry'), findsNothing);
  });

  testWidgets(
    'failed refresh preserves pagination failure until explicit retry',
    (tester) async {
      final repo = _Notifications();
      await _mount(tester, repo);
      repo.requests.first.$3.complete(_page('First', more: true));
      await tester.pump();
      await tester.pump();
      repo.requests.last.$3.completeError(
        const ApiException(fallbackMessage: 'page failed'),
      );
      await tester.pumpAndSettle();
      expect(
        tester.widget<GfListFooter>(find.byType(GfListFooter)).error,
        'Failed to load',
      );
      final refresh = tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      repo.requests.last.$3.completeError(
        const ApiException(fallbackMessage: 'refresh failed'),
      );
      await refresh;
      await tester.pump();
      await tester.pump();
      expect(
        tester.widget<GfListFooter>(find.byType(GfListFooter)).error,
        'Failed to load',
      );
      expect(repo.requests, hasLength(3));
      await tester.pump(const Duration(seconds: 1));
      expect(repo.requests, hasLength(3));
      tester.widget<GfListFooter>(find.byType(GfListFooter)).onLoadMore();
      expect(repo.requests.last.$2, 1);
      repo.requests.last.$3.complete(_page('Next', id: 2));
      await tester.pumpAndSettle();
      expect(find.text('Next'), findsOneWidget);
      expect(find.text('First'), findsOneWidget);
    },
  );

  testWidgets(
    'account changes hide old rows and ignore pending read acknowledgements',
    (tester) async {
      final repo = _Notifications();
      final container = await _mount(tester, repo);
      repo.requests.first.$3.complete(_page('Account one'));
      await tester.pumpAndSettle();
      tester
          .widget<GfNotificationRow>(find.byType(GfNotificationRow))
          .onMarkRead!();
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pump();
      expect(find.text('Account one'), findsNothing);
      repo.requests.last.$3.complete(_page('Account two'));
      await tester.pumpAndSettle();
      repo.markOne.single.$2.complete(true);
      await tester.pumpAndSettle();
      expect(
        tester.widget<GfNotificationRow>(find.byType(GfNotificationRow)).unread,
        isTrue,
      );
      expect(find.text('Account two'), findsOneWidget);
    },
  );

  testWidgets('late previous filter cannot replace the active notifications', (
    tester,
  ) async {
    final repo = _Notifications();
    await _mount(tester, repo);
    await tester.tap(find.text('Unread'));
    await tester.pump();
    repo.requests.last.$3.complete(_page('Unread current'));
    await tester.pumpAndSettle();
    repo.requests.first.$3.complete(_page('All stale'));
    await tester.pumpAndSettle();
    expect(find.text('Unread current'), findsOneWidget);
    expect(find.text('All stale'), findsNothing);
  });

  testWidgets('failed refresh retains rows and exposes failure', (
    tester,
  ) async {
    final repo = _Notifications();
    await _mount(tester, repo);
    repo.requests.first.$3.complete(_page('Keep this notification'));
    await tester.pumpAndSettle();
    final refresh = tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pump();
    repo.requests.last.$3.completeError(StateError('offline'));
    await refresh;
    await tester.pumpAndSettle();
    expect(find.text('Keep this notification'), findsOneWidget);
    expect(find.textContaining('offline'), findsOneWidget);
  });

  testWidgets('mark all is single-flight and failures are visible', (
    tester,
  ) async {
    final repo = _Notifications();
    await _mount(tester, repo);
    repo.requests.first.$3.complete(_page('Still unread'));
    await tester.pumpAndSettle();
    final button = tester.widget<GfIconButton>(
      find.widgetWithIcon(GfIconButton, Icons.done_all_rounded),
    );
    button.onPressed!();
    button.onPressed!();
    await tester.pump();
    expect(repo.markAll, hasLength(1));
    repo.markAll.single.completeError(StateError('read failed'));
    await tester.pumpAndSettle();
    expect(find.textContaining('read failed'), findsOneWidget);
    expect(
      tester.widget<GfNotificationRow>(find.byType(GfNotificationRow)).unread,
      isTrue,
    );
  });
}
