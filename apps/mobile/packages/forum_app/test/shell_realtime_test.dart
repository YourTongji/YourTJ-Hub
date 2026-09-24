import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/navigation/route_visibility.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/realtime/realtime_updates.dart';
import 'package:forum_app/src/router.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'pages_behavior_test.dart' show MemTokenStorage, NoopCache;

class _UnreadRepository extends NotificationRepository {
  _UnreadRepository(super.client);

  int calls = 0;
  Completer<UnreadStatusPayload>? gate;

  @override
  Future<UnreadStatusPayload> getUnreadStatus() async {
    calls++;
    return gate?.future ??
        const UnreadStatusPayload(notifications: false, messages: false);
  }
}

void main() {
  testWidgets('domain and unread hints fetch the badge only once', (
    tester,
  ) async {
    final storage = MemTokenStorage();
    await storage.write('accepted-token');
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    final unread = _UnreadRepository(client);
    final eventsReady = Completer<void>();
    final streamDone = Completer<void>();
    var connects = 0;
    final cache = NoopCache();
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(storage),
        apiClientProvider.overrideWithValue(client),
        notificationRepositoryProvider.overrideWithValue(unread),
        offlineTopicCacheProvider.overrideWithValue(cache),
        offlineChatCacheProvider.overrideWithValue(cache),
        realtimeConnectProvider.overrideWithValue((
          token,
          cancel,
          onActivity,
        ) async {
          connects++;
          Stream<ForumSseFrame> events() async* {
            yield const ForumSseFrame(
              event: 'hello',
              data:
                  '{"version":1,"heartbeatSeconds":15,"resync":true,"capabilities":{"visibleRead":true}}',
            );
            await eventsReady.future;
            yield const ForumSseFrame(
              event: 'chat.changed',
              data: '{"convId":1,"change":"received"}',
            );
            yield const ForumSseFrame(event: 'unread.changed', data: '{}');
            await streamDone.future;
          }

          return ForumRealtimeConnection(events(), cancel);
        }),
      ],
    );
    addTearDown(container.dispose);
    final router = GoRouter(
      observers: [VisibilityRouteObserver()],
      routes: [
        StatefulShellRoute.indexedStack(
          builder: (_, _, shell) => GfShell(navigationShell: shell),
          branches: [
            StatefulShellBranch(
              routes: [
                GoRoute(
                  path: '/',
                  builder: (_, _) => const Scaffold(body: Text('home')),
                ),
              ],
            ),
          ],
        ),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp.router(
          routerConfig: router,
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pumpAndSettle();
    for (
      var i = 0;
      i < 8 && (!container.read(realtimeHealthyProvider) || connects == 0);
      i++
    ) {
      await tester.pump();
    }
    expect(routeIsUncovered(tester.element(find.byType(GfShell))), isTrue);
    expect(connects, 1);
    expect(container.read(realtimeHealthyProvider), isTrue);
    final before = unread.calls;
    final beforeRevision = container
        .read(realtimeInvalidationsProvider)
        .chatRevision;
    unread.gate = Completer<UnreadStatusPayload>();
    eventsReady.complete();
    await tester.pump();
    await tester.pump();
    expect(
      container.read(realtimeInvalidationsProvider).chatRevision,
      beforeRevision + 1,
    );
    expect(unread.calls, before + 1);
    unread.gate!.complete(
      const UnreadStatusPayload(notifications: false, messages: false),
    );
    await tester.pump();
    await tester.pump();
    expect(unread.calls, before + 1);
    await tester.pumpWidget(const SizedBox.shrink());
    streamDone.complete();
  });

  testWidgets('guest shell reconnects after login returns to the same route', (
    tester,
  ) async {
    final storage = MemTokenStorage();
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    var connects = 0;
    final cache = NoopCache();
    final container = ProviderContainer(
      overrides: [
        tokenStorageProvider.overrideWithValue(storage),
        apiClientProvider.overrideWithValue(client),
        notificationRepositoryProvider.overrideWithValue(
          _UnreadRepository(client),
        ),
        offlineTopicCacheProvider.overrideWithValue(cache),
        offlineChatCacheProvider.overrideWithValue(cache),
        realtimeConnectProvider.overrideWithValue((
          token,
          cancel,
          onActivity,
        ) async {
          expect(token, 'accepted-token');
          connects++;
          return ForumRealtimeConnection(
            const Stream<ForumSseFrame>.empty(),
            cancel,
          );
        }),
      ],
    );
    addTearDown(container.dispose);
    final router = GoRouter(
      observers: [VisibilityRouteObserver()],
      routes: [
        StatefulShellRoute.indexedStack(
          builder: (_, _, shell) => GfShell(navigationShell: shell),
          branches: [
            StatefulShellBranch(
              routes: [
                GoRoute(
                  path: '/',
                  builder: (_, _) => const Scaffold(body: Text('home')),
                ),
              ],
            ),
          ],
        ),
        GoRoute(
          path: '/login',
          builder: (_, _) => const Scaffold(body: Text('login')),
        ),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp.router(
          routerConfig: router,
          theme: gfThemeData(Brightness.light),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(connects, 0);

    final shellScaffold = tester.state<ScaffoldState>(
      find.byType(Scaffold).first,
    );
    shellScaffold.openDrawer();
    await tester.pumpAndSettle();
    expect(shellDrawerOpen.value, isTrue);
    shellScaffold.closeDrawer();
    await tester.pumpAndSettle();
    expect(shellDrawerOpen.value, isFalse);

    router.push('/login');
    await tester.pumpAndSettle();
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    await storage.write('accepted-token');
    router.pop();
    await tester.pumpAndSettle();
    expect(find.text('home'), findsOneWidget);
    expect(connects, 1);
    await tester.pumpWidget(const SizedBox.shrink());
  });
}
