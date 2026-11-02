import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/topic/agent_reply_setting.dart';
import 'package:forum_app/src/pages/topic/topic_actions.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/share/share_image_preview.dart';
import 'package:go_router/go_router.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Topics extends TopicRepository {
  _Topics(super.client);
  final deleted = <int>[];
  final replySettings = <bool>[];
  bool failReplySetting = false;
  @override
  Future<void> updateAgentReplies({
    required int topicId,
    required bool disabled,
  }) async {
    expect(topicId, 100);
    if (failReplySetting) throw StateError("save failed");
    replySettings.add(disabled);
  }

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
    bool fromSource = false,
    bool withFirstPost = true,
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
      initialLocation: fromSource ? '/source' : '/topic',
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => const Scaffold(body: Text('Home')),
        ),
        GoRoute(
          path: '/source',
          builder: (_, _) => const Scaffold(body: Text('Source')),
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
                  canManageAgentReplies: own,
                  canModerateTopic: moderator,
                ),
              ),
              onChanged: onChanged ?? () async {},
              firstPost: withFirstPost ? props.postStream.posts.first : null,
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
    if (fromSource) {
      router.push('/topic');
      await tester.pumpAndSettle();
    }
    return (container: container, router: router, repo: repo);
  }

  Future<void> menu(WidgetTester tester, String action) async {
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    await tester.tap(find.text(action));
    await tester.pumpAndSettle();
  }

  Future<_Topics> pumpReplySetting(
    WidgetTester tester, {
    required bool canManage,
    bool disabled = false,
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
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: AgentReplySetting(
              topicId: 100,
              disabled: disabled,
              canManage: canManage,
              onChanged: onChanged ?? () async {},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    return repo;
  }

  testWidgets(
    'author switches robot replies off and refreshes; readers have no control',
    (tester) async {
      var refreshes = 0;
      final repo = await pumpReplySetting(
        tester,
        canManage: true,
        onChanged: () async {
          refreshes++;
        },
      );
      expect(find.text('Allow bot replies'), findsOneWidget);
      await tester.tap(find.byType(Switch));
      await tester.pumpAndSettle();
      expect(repo.replySettings, [true]);
      expect(refreshes, 1);
      await pumpReplySetting(tester, canManage: false);
      expect(find.byType(Switch), findsNothing);
      expect(find.byKey(const Key('agent-replies-notice')), findsNothing);
      await pumpReplySetting(tester, canManage: false, disabled: true);
      expect(find.byKey(const Key('agent-replies-notice')), findsOneWidget);
    },
  );

  testWidgets('failed robot reply setting keeps its state and permits retry', (
    tester,
  ) async {
    var refreshes = 0;
    final repo = await pumpReplySetting(
      tester,
      canManage: true,
      onChanged: () async {
        refreshes++;
      },
    );
    repo.failReplySetting = true;
    await tester.tap(find.byType(Switch));
    await tester.pumpAndSettle();
    expect(refreshes, 0);
    expect(repo.replySettings, isEmpty);
    expect(find.text('Bad state: save failed'), findsOneWidget);
    expect(tester.widget<Switch>(find.byType(Switch)).value, isTrue);
    repo.failReplySetting = false;
    await tester.tap(find.byType(Switch));
    await tester.pumpAndSettle();
    expect(repo.replySettings, [true]);
    expect(refreshes, 1);
    expect(find.text('Bad state: save failed'), findsNothing);
  });

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
  testWidgets('delete requires confirmation and pops back to the source list', (
    tester,
  ) async {
    final h = await pump(tester, fromSource: true);
    await menu(tester, 'Delete');
    expect(h.repo.deleted, isEmpty);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(h.repo.deleted, isEmpty);
    expect(h.container.read(removedTopicsProvider), isEmpty);
    await menu(tester, 'Delete');
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(h.repo.deleted, [100]);
    expect(h.router.state.uri.path, '/source');
    expect(find.text('Source'), findsOneWidget);
    // The page beneath drops the topic in place instead of reloading.
    expect(h.container.read(removedTopicsProvider), {100});
  });
  testWidgets('a deep-linked topic with nothing beneath returns home', (
    tester,
  ) async {
    final h = await pump(tester);
    await menu(tester, 'Delete');
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(h.repo.deleted, [100]);
    expect(h.router.state.uri.path, '/');
    expect(h.container.read(removedTopicsProvider), {100});
  });
  testWidgets('pending topic still opens the author editor', (tester) async {
    await pump(tester, status: 2);
    await menu(tester, 'Edit');
    expect(find.text('Editor 100'), findsOneWidget);
  });
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
    expect(h.container.read(removedTopicsProvider), isEmpty);
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
  testWidgets('every topic-level action lives in the app-bar menu', (
    tester,
  ) async {
    await pump(tester, own: false);
    final l10n = AppLocalizations.of(tester.element(find.byType(TopicActions)));
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    for (final label in [
      'Share',
      'Generate share image',
      'Revision history',
      'Report post',
    ]) {
      expect(find.text(label), findsOneWidget, reason: label);
    }
    await tester.tap(find.text('Report post'));
    await tester.pumpAndSettle();
    expect(find.text(l10n.contentReport), findsOneWidget);
    expect(find.byType(GfDropdownButtonFormField<String>), findsOneWidget);
  });
  testWidgets('authors manage but do not report their own topic', (
    tester,
  ) async {
    await pump(tester);
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    for (final label in [
      'Share',
      'Generate share image',
      'Edit',
      'Revision history',
      'Delete',
    ]) {
      expect(find.text(label), findsOneWidget, reason: label);
    }
    expect(find.text('Report post'), findsNothing);
  });
  testWidgets('topic share image renders the opening post', (tester) async {
    await pump(tester, own: false);
    await menu(tester, 'Generate share image');
    expect(find.byType(ShareImageCard), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(ShareImageCard),
        matching: find.text('移动端测试话题'),
      ),
      findsOneWidget,
    );
    expect(
      find.descendant(
        of: find.byType(ShareImageCard),
        matching: find.textContaining('/p/post/100'),
      ),
      findsOneWidget,
    );
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(seconds: 1));
  });
  testWidgets('without the opening post the menu omits its image and history', (
    tester,
  ) async {
    await pump(tester, own: false, withFirstPost: false);
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    expect(find.text('Share'), findsOneWidget);
    expect(find.text('Report post'), findsOneWidget);
    expect(find.text('Generate share image'), findsNothing);
    expect(find.text('Revision history'), findsNothing);
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
    await tester.pump(const Duration(seconds: 8));
  });
}
