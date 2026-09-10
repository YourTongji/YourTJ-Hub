import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/topic/topic_actions.dart';
import 'package:forum_app/src/providers.dart';
import 'package:go_router/go_router.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Topics extends TopicRepository {
  _Topics(super.client);
  final deleted = <int>[];
  final moderated = <bool>[];
  @override
  Future<void> deleteTopic({required int topicId}) async =>
      deleted.add(topicId);
  @override
  Future<void> moderate({required int topicId, required bool ban}) async {
    expect(topicId, 100);
    moderated.add(ban);
  }
}

void main() {
  Future<({ProviderContainer container, GoRouter router, _Topics repo})> pump(
    WidgetTester tester, {
    bool own = true,
    bool moderator = false,
    int status = 0,
    Future<void> Function()? onChanged,
  }) async {
    final repo = _Topics(
      GfApiClient(
        dio: Dio(),
        tokenStorage: MemoryTokenStorage(),
        baseUrl: 'https://example.test',
      ),
    );
    final container = ProviderContainer(
      overrides: [topicRepositoryProvider.overrideWithValue(repo)],
    );
    final props = TopicDetailProps.fromJson(
      topicDetailPayloadJson()['props'] as Map<String, dynamic>,
    );
    final router = GoRouter(
      initialLocation: '/topic',
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => const Scaffold(body: Text('Home')),
        ),
        GoRoute(
          path: '/publish',
          builder: (_, state) =>
              Scaffold(body: Text('Editor ${state.uri.queryParameters['id']}')),
        ),
        GoRoute(
          path: '/topic',
          builder: (_, _) => Scaffold(
            body: TopicActions(
              props: props.copyWith(
                topic: props.topic.copyWith(processStatus: status),
                permissions: props.permissions.copyWith(
                  isOwnTopic: own,
                  canModerateTopic: moderator,
                ),
              ),
              onChanged: onChanged ?? () async {},
              firstPostId: 9001,
            ),
          ),
        ),
      ],
    );
    addTearDown(router.dispose);
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp.router(
          routerConfig: router,
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();
    return (container: container, router: router, repo: repo);
  }

  Future<void> menu(WidgetTester tester, String action) async {
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    await tester.tap(find.text(action));
    await tester.pumpAndSettle();
  }

  testWidgets('own topic opens its editor and refreshes after returning', (
    tester,
  ) async {
    var refreshes = 0;
    final h = await pump(
      tester,
      onChanged: () async {
        refreshes++;
      },
    );
    await menu(tester, 'Edit');
    expect(find.text('Editor 100'), findsOneWidget);
    h.router.pop();
    await tester.pumpAndSettle();
    expect(refreshes, 1);
  });
  testWidgets(
    'delete requires confirmation and returns home only after API success',
    (tester) async {
      final h = await pump(tester);
      await menu(tester, 'Delete');
      expect(h.repo.deleted, isEmpty);
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(h.repo.deleted, isEmpty);
      await menu(tester, 'Delete');
      await tester.tap(find.text('Confirm'));
      await tester.pumpAndSettle();
      expect(h.repo.deleted, [100]);
      expect(h.router.state.uri.path, '/');
    },
  );
  testWidgets('session change cancels a pending topic deletion', (
    tester,
  ) async {
    final h = await pump(tester);
    await menu(tester, 'Delete');
    h.container.read(offlineCacheEpochProvider.notifier).state++;
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(h.repo.deleted, isEmpty);
    expect(h.router.state.uri.path, '/topic');
  });
  testWidgets('readers cannot edit or moderate somebody else’s topic', (
    tester,
  ) async {
    await pump(tester, own: false);
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    expect(find.text('Edit'), findsNothing);
    expect(find.text('Delete'), findsNothing);
    expect(find.text('Hide content'), findsNothing);
    expect(find.text('Revision history'), findsOneWidget);
  });
  for (final status in [0, 1]) {
    testWidgets('moderation uses server status $status and refreshes', (
      tester,
    ) async {
      var refreshes = 0;
      final h = await pump(
        tester,
        own: false,
        moderator: true,
        status: status,
        onChanged: () async {
          refreshes++;
        },
      );
      final context = tester.element(find.byType(TopicActions));
      final l10n = AppLocalizations.of(context);
      await menu(
        tester,
        status == 0 ? l10n.topicModerateBan : l10n.topicModerateUnban,
      );
      await tester.tap(find.text('Confirm'));
      await tester.pumpAndSettle();
      expect(h.repo.moderated, [status == 0]);
      expect(refreshes, 1);
    });
  }
  testWidgets('failed native topic sharing stays on the page', (tester) async {
    const channel = MethodChannel('dev.fluttercommunity.plus/share');
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, (
      _,
    ) async {
      throw PlatformException(code: 'unavailable');
    });
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        channel,
        null,
      ),
    );
    await pump(tester);
    await menu(tester, 'Share');
    expect(tester.takeException(), isNull);
    expect(find.byType(TopicActions), findsOneWidget);
    await tester.pump(const Duration(seconds: 4));
  });
}
