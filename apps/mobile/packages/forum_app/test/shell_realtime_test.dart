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

  @override
  Future<UnreadStatusPayload> getUnreadStatus() async =>
      const UnreadStatusPayload(notifications: false, messages: false);
}

void main() {
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
