import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/pages/drafts/drafts_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/app_refresh_indicator.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Pages extends PageRepository {
  _Pages(super.client);
  bool fail = false;
  int count = 1;
  Completer<PagePayload>? pending;
  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    if (fail) throw const NetworkException(fallbackMessage: 'offline');
    if (pending != null) return pending!.future;
    return parsePayload({
      'component': PageComponent.drafts,
      'url': '/drafts',
      'version': '1',
      'layout': minimalLayoutJson(),
      'meta': {'title': 'Drafts'},
      'props': {
        'total': count,
        'drafts': [
          for (var index = 0; index < count; index++)
            {
              'id': 42 + index,
              'title': '云端未完成内容',
              'description': '云端正文',
              'editUrl': '/publish?id=42',
              'replyCount': 0,
              'viewCount': 0,
              'processStatus': 0,
              'createdAt': '2026-09-24',
              'updatedAt': '2026-09-24',
              'categories': [],
            },
        ],
        'pagination': {
          'page': 1,
          'nextPage': 0,
          'hasNext': false,
          'nextUrl': '',
        },
      },
    });
  }
}

const _scope = 'http%3A%2F%2Ffake.local:1';
final _client = GfApiClient(
  dio: Dio(),
  tokenStorage: MemoryTokenStorage(),
  baseUrl: 'http://fake.local',
);

class _Content extends ContentRepository {
  _Content() : super(_client);
  final calls = <List<int>>[];
  String? contentType;
  bool fail = false;
  List<ContentDeletionResult> results = const [
    ContentDeletionResult(contentId: 42, success: true),
  ];
  Completer<List<ContentDeletionResult>>? pending;
  void Function()? onDelete;

  @override
  Future<List<ContentDeletionResult>> delete({
    required String contentType,
    required List<int> ids,
    String? password,
  }) async {
    this.contentType = contentType;
    calls.add(ids);
    if (fail) throw const NetworkException(fallbackMessage: 'offline');
    onDelete?.call();
    return pending == null ? results : pending!.future;
  }
}

const _article = LocalDraft(
  key: 'new-article',
  title: 'Campus notes',
  content: 'Library opening times',
  contentType: 3,
  topicId: 0,
  categories: [],
  images: [],
  updatedAt: 2,
);
const _reply = LocalDraft(
  key: 'reply-100',
  kind: DraftKind.reply,
  title: '回复中的话题',
  content: '本机回复正文',
  contentType: 2,
  topicId: 100,
  categories: [],
  images: [],
  updatedAt: 1,
);

Future<ProviderContainer> _mount(
  WidgetTester tester,
  WritingStore store, {
  bool settle = true,
  Locale locale = const Locale('zh'),
  double textScale = 1,
  ContentRepository? content,
  PageRepository? pages,
  Brightness brightness = Brightness.light,
}) async {
  final client = GfApiClient(
    dio: Dio(),
    tokenStorage: MemoryTokenStorage(),
    baseUrl: 'http://fake.local',
  );
  final router = GoRouter(
    initialLocation: '/drafts',
    routes: [
      GoRoute(path: '/drafts', builder: (_, _) => const DraftsPage()),
      GoRoute(
        path: '/publish',
        builder: (_, state) => Scaffold(
          body: Text('editing-${state.uri.queryParameters['local']}'),
        ),
      ),
      GoRoute(
        path: '/recycle-bin',
        builder: (_, _) => const Scaffold(body: Text('recycle-bin')),
      ),
      GoRoute(
        path: '/my-content',
        builder: (_, _) => const Scaffold(body: Text('content-management')),
      ),
      GoRoute(
        path: '/p/:id',
        builder: (_, state) =>
            Scaffold(body: Text('reply-topic-${state.pathParameters['id']}')),
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        apiClientProvider.overrideWithValue(client),
        pageRepositoryProvider.overrideWithValue(pages ?? _Pages(client)),
        writingStoreProvider.overrideWithValue(store),
        if (content != null)
          contentRepositoryProvider.overrideWithValue(content),
        currentUserProvider.overrideWith(
          (ref) async => const CurrentUser(id: 1, username: 'alice'),
        ),
      ],
      child: MaterialApp.router(
        routerConfig: router,
        theme: gfThemeData(brightness),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: locale,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(textScale)),
          child: child!,
        ),
      ),
    ),
  );
  if (settle) {
    await tester.pumpAndSettle();
  } else {
    await tester.pump();
  }
  return ProviderScope.containerOf(tester.element(find.byType(DraftsPage)));
}

class _ControlledStore extends WritingStore {
  bool failRead = false;
  bool failDelete = false;
  bool failRestore = false;
  Completer<List<LocalDraft>>? pendingRead;
  @override
  Future<List<LocalDraft>> drafts(String scope) async {
    if (pendingRead != null) return pendingRead!.future;
    if (failRead) throw StateError('Storage unavailable');
    return super.drafts(scope);
  }

  @override
  Future<void> delete(
    String scope,
    String key, {
    bool Function()? isCurrent,
  }) async {
    if (failDelete) throw StateError('Delete failed');
    return super.delete(scope, key, isCurrent: isCurrent);
  }

  @override
  Future<bool> restoreIfAbsent(
    String scope,
    LocalDraft draft, {
    bool Function()? isCurrent,
  }) async {
    if (failRestore) throw StateError('Restore failed');
    return super.restoreIfAbsent(scope, draft, isCurrent: isCurrent);
  }
}

Future<void> _deleteArticle(WidgetTester tester) async {
  final row = find.byKey(const ValueKey('new-article'));
  await tester.ensureVisible(row);
  await tester.tap(
    find.descendant(of: row, matching: find.byTooltip('删除本机草稿')),
  );
  await tester.pumpAndSettle();
  await tester.tap(find.widgetWithText(TextButton, '删除本机草稿').last);
  await tester.pumpAndSettle();
}

Future<void> _confirmCloudDelete(WidgetTester tester) async {
  final action = find.widgetWithText(TextButton, '删除云端草稿').first;
  await tester.ensureVisible(action);
  await tester.tap(action);
  await tester.pumpAndSettle();
  await tester.tap(find.widgetWithText(TextButton, '删除云端草稿').last);
  await tester.pumpAndSettle();
}

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));
  testWidgets('device and cloud drafts expose visible delete actions', (
    tester,
  ) async {
    final store = WritingStore();
    await store.save(_scope, _article);
    await _mount(tester, store);
    expect(find.widgetWithText(TextButton, '删除本机草稿'), findsOneWidget);
    expect(find.widgetWithText(TextButton, '删除云端草稿'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
  });
  testWidgets(
    'cloud deletion is confirmed and preserves every local identity',
    (tester) async {
      final store = WritingStore();
      final recovery = LocalDraft.fromJson({
        ..._article.toJson(),
        'key': 'server-42',
        'topicId': 42,
      });
      await store.save(_scope, _article);
      await store.save(_scope, recovery);
      final pages = _Pages(_client);
      final content = _Content()..onDelete = () => pages.count = 0;
      await _mount(tester, store, content: content, pages: pages);
      await tester.tap(find.widgetWithText(ChoiceChip, '云端草稿'));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.widgetWithText(TextButton, '删除云端草稿'));
      await tester.tap(find.widgetWithText(TextButton, '删除云端草稿'));
      await tester.pumpAndSettle();
      expect(find.textContaining('本机恢复副本会保留'), findsOneWidget);
      await tester.tap(find.widgetWithText(TextButton, '取消'));
      await tester.pumpAndSettle();
      expect(content.calls, isEmpty);
      await _confirmCloudDelete(tester);
      await tester.pumpAndSettle();
      expect(content.contentType, 'topic');
      expect(content.calls, [
        [42],
      ]);
      expect(find.byKey(const ValueKey('cloud-42')), findsNothing);
      expect(
        (await store.drafts(_scope)).map((draft) => draft.key),
        containsAll(['new-article', 'server-42']),
      );
      expect(find.text('云端草稿已移入回收站，本机副本已保留。'), findsOneWidget);
      await tester.tap(find.text('打开回收站'));
      await tester.pumpAndSettle();
      expect(find.text('recycle-bin'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  for (final failure in ['network', 'item', 'missing', 'wrong-id']) {
    testWidgets('cloud $failure failure keeps the row and device copy', (
      tester,
    ) async {
      final store = WritingStore();
      await store.save(_scope, _article);
      final content = _Content()
        ..fail = failure == 'network'
        ..results = switch (failure) {
          'missing' => [],
          'wrong-id' => [
            const ContentDeletionResult(contentId: 99, success: true),
          ],
          _ => [const ContentDeletionResult(contentId: 42, success: false)],
        };
      await _mount(tester, store, content: content);
      await _confirmCloudDelete(tester);
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('cloud-42')), findsOneWidget);
      expect(find.text('打开回收站'), findsNothing);
      expect(await store.drafts(_scope), hasLength(1));
      expect(content.calls, [
        [42],
      ]);
      await tester.pumpWidget(const SizedBox.shrink());
    });
  }
  testWidgets(
    'in-flight cloud deletion prevents duplicate actions and stale refresh refill',
    (tester) async {
      final pages = _Pages(_client);
      final oldPayload = await pages.fetch('/drafts');
      final content = _Content()
        ..pending = Completer<List<ContentDeletionResult>>();
      await _mount(tester, WritingStore(), content: content, pages: pages);
      final pendingRead = Completer<PagePayload>();
      pages.pending = pendingRead;
      final refreshing = tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.pump();
      await _confirmCloudDelete(tester);
      expect(
        tester
            .widget<TextButton>(find.widgetWithText(TextButton, '删除云端草稿'))
            .onPressed,
        isNull,
      );
      pages.pending = null;
      pages.count = 0;
      content.pending!.complete(content.results);
      await tester.pumpAndSettle();
      pendingRead.complete(oldPayload);
      await refreshing;
      await tester.pumpAndSettle();
      expect(content.calls, [
        [42],
      ]);
      expect(find.byKey(const ValueKey('cloud-42')), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  testWidgets(
    'session change cancels cloud confirmation and ignores late deletion acknowledgement',
    (tester) async {
      final content = _Content();
      final container = await _mount(tester, WritingStore(), content: content);
      await tester.tap(find.widgetWithText(TextButton, '删除云端草稿'));
      await tester.pumpAndSettle();
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pump();
      await tester.tap(find.widgetWithText(TextButton, '删除云端草稿'));
      await tester.pump();
      expect(content.calls, isEmpty);
      await tester.pumpWidget(const SizedBox.shrink());
      content.pending = Completer<List<ContentDeletionResult>>();
      final next = await _mount(tester, WritingStore(), content: content);
      await _confirmCloudDelete(tester);
      next.read(offlineCacheEpochProvider.notifier).invalidate();
      content.pending!.complete(content.results);
      await tester.pump();
      expect(find.text('打开回收站'), findsNothing);
      expect(find.byKey(const ValueKey('cloud-42')), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  testWidgets('100 draft window explains the bounded list', (tester) async {
    final pages = _Pages(_client)..count = 100;
    await _mount(tester, WritingStore(), pages: pages);
    expect(find.textContaining('此页最多加载最近 100 份云端草稿'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
  });
  testWidgets('new composition returns to a refreshed draft list', (
    tester,
  ) async {
    final store = WritingStore();
    await _mount(tester, store);
    final l10n = AppLocalizations.of(tester.element(find.byType(DraftsPage)));
    await tester.tap(find.byTooltip(l10n.navPublish));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('compose-3')));
    await tester.pumpAndSettle();
    expect(find.text('editing-null'), findsOneWidget);
    await store.save(_scope, _article);
    GoRouter.of(tester.element(find.text('editing-null'))).pop();
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('new-article')), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
  });
  testWidgets('a stale local refresh cannot resurrect a deleted row', (
    tester,
  ) async {
    final store = _ControlledStore();
    await store.save(_scope, _article);
    await _mount(tester, store);
    final pending = Completer<List<LocalDraft>>();
    store.pendingRead = pending;
    final refreshing = tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pump();
    store.pendingRead = null;
    await _deleteArticle(tester);
    pending.complete([_article]);
    await refreshing;
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('new-article')), findsNothing);
    expect(await store.drafts(_scope), isEmpty);
    await tester.pumpWidget(const SizedBox.shrink());
  });
  testWidgets(
    'delete confirmation, failure and undo preserve the full recovery copy',
    (tester) async {
      final store = _ControlledStore();
      await store.save(_scope, _article);
      await _mount(tester, store);
      await tester.tap(find.byTooltip('删除本机草稿'));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(TextButton, '取消'));
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey('new-article')), findsOneWidget);
      store.failDelete = true;
      await _deleteArticle(tester);
      expect(find.byKey(const ValueKey('new-article')), findsOneWidget);
      expect(find.text('撤销'), findsNothing);
      store.failDelete = false;
      await _deleteArticle(tester);
      expect(find.byKey(const ValueKey('new-article')), findsNothing);
      expect(await store.drafts(_scope), isEmpty);
      expect(find.text('撤销'), findsOneWidget);
      store.failRestore = true;
      await tester.tap(find.text('撤销'));
      await tester.pumpAndSettle();
      expect(find.text('撤销'), findsOneWidget);
      expect(find.byKey(const ValueKey('new-article')), findsNothing);
      store.failRestore = false;
      await tester.tap(find.text('撤销'));
      await tester.pumpAndSettle();
      expect(find.text('撤销'), findsNothing);
      expect((await store.drafts(_scope)).single.toJson(), _article.toJson());
      await tester.tap(find.text('Campus notes'));
      await tester.pumpAndSettle();
      expect(find.text('editing-new-article'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  testWidgets(
    'undo never replaces a newer copy and session changes remove recovery actions',
    (tester) async {
      final store = WritingStore();
      await store.save(_scope, _article);
      final container = await _mount(tester, store);
      await _deleteArticle(tester);
      final newer = LocalDraft.fromJson({
        ..._article.toJson(),
        'content': 'Updated body',
      });
      await store.save(_scope, newer);
      await tester.tap(find.text('撤销'));
      await tester.pumpAndSettle();
      expect(find.text('Updated body'), findsOneWidget);
      expect((await store.drafts(_scope)).single.content, 'Updated body');
      await _deleteArticle(tester);
      await tester.enterText(find.byType(TextField), 'private query');
      await tester.pump();
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pump();
      expect(find.text('撤销'), findsNothing);
      expect(find.text('Campus notes'), findsNothing);
      expect(
        tester.widget<TextField>(find.byType(TextField)).controller!.text,
        isEmpty,
      );
      expect(await store.drafts(_scope), isEmpty);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  testWidgets(
    'cloud and empty-result filters clear without network or lost local drafts',
    (tester) async {
      final store = WritingStore();
      await store.save(_scope, _article);
      await _mount(tester, store);
      await tester.tap(find.widgetWithText(ChoiceChip, '云端草稿'));
      await tester.pumpAndSettle();
      expect(find.text('Campus notes'), findsNothing);
      expect(find.text('云端未完成内容'), findsOneWidget);
      await tester.enterText(find.byType(TextField), 'not found');
      await tester.pumpAndSettle();
      expect(find.text('没有符合这些条件的草稿。'), findsOneWidget);
      await tester.tap(find.text('清空搜索与筛选'));
      await tester.pumpAndSettle();
      expect(find.text('Campus notes'), findsOneWidget);
      expect(find.text('当前显示 2 份草稿'), findsOneWidget);
      expect(await store.drafts(_scope), hasLength(1));
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  testWidgets(
    'local loading is not empty and read failures retain displayed copies',
    (tester) async {
      final store = _ControlledStore();
      await store.save(_scope, _article);
      final pending = Completer<List<LocalDraft>>();
      store.pendingRead = pending;
      await _mount(tester, store, settle: false);
      expect(find.text('未完成的创作会保存在本机，显示在这里。'), findsNothing);
      expect(find.byType(GfLoading), findsOneWidget);
      store.pendingRead = null;
      pending.complete([_article]);
      await tester.pumpAndSettle();
      store.failRead = true;
      await tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.pumpAndSettle();
      expect(find.text('Campus notes'), findsOneWidget);
      expect(find.text('重试'), findsOneWidget);
      store.failRead = false;
      await tester.tap(find.text('重试'));
      await tester.pumpAndSettle();
      expect(find.text('重试'), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  for (final brightness in Brightness.values) {
    testWidgets('cloud controls wrap at large German text in $brightness', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 700);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await _mount(
        tester,
        WritingStore(),
        locale: const Locale('de'),
        textScale: 1.6,
        brightness: brightness,
      );
      final l10n = AppLocalizations.of(tester.element(find.byType(DraftsPage)));
      final action = find.widgetWithText(TextButton, l10n.draftDeleteCloud);
      await tester.scrollUntilVisible(
        action,
        120,
        scrollable: find
            .descendant(
              of: find.byType(CustomScrollView),
              matching: find.byType(Scrollable),
            )
            .first,
      );
      await tester.pumpAndSettle();
      expect(tester.getSize(action).height, greaterThanOrEqualTo(44));
      expect(tester.takeException(), isNull);
      await tester.tap(action);
      await tester.pumpAndSettle();
      expect(find.text(l10n.draftDeleteCloudConfirm), findsOneWidget);
      expect(tester.takeException(), isNull);
      await tester.tap(find.widgetWithText(TextButton, l10n.commonCancel));
      await tester.pumpAndSettle();
      await tester.pumpWidget(const SizedBox.shrink());
    });
  }
  testWidgets('narrow large-text German drafts remain usable with a keyboard', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 700);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetViewInsets);
    final store = WritingStore();
    await store.save(_scope, _article);
    await _mount(tester, store, locale: const Locale('de'), textScale: 1.6);
    await tester.enterText(find.byType(TextField), 'Library');
    tester.view.viewInsets = const FakeViewPadding(bottom: 260);
    await tester.pumpAndSettle();
    final row = find.byKey(const ValueKey('new-article'));
    await tester.scrollUntilVisible(
      row,
      120,
      scrollable: find
          .descendant(
            of: find.byType(CustomScrollView),
            matching: find.byType(Scrollable),
          )
          .first,
    );
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    await tester.tap(
      find.descendant(
        of: row,
        matching: find.byWidgetPredicate(
          (widget) => widget is GfSymbol && widget.name == 'trash-2',
        ),
      ),
    );
    await tester.pumpAndSettle();
    final l10n = AppLocalizations.of(tester.element(find.byType(AlertDialog)));
    await tester.tap(
      find.widgetWithText(TextButton, l10n.draftDeleteLocal).last,
    );
    await tester.pumpAndSettle();
    expect(find.text(l10n.publishUndo), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.tap(find.text(l10n.publishUndo));
    await tester.pumpAndSettle();
    expect(await store.drafts(_scope), hasLength(1));
    await tester.pumpWidget(const SizedBox.shrink());
  });
  testWidgets(
    'search and reply filter find drafts without changing stored content',
    (tester) async {
      final store = WritingStore();
      await store.save(_scope, _article);
      await store.save(_scope, _reply);
      await _mount(tester, store);
      expect(find.byType(GfSearchField), findsOneWidget);
      await tester.enterText(find.byType(TextField), 'LIBRARY');
      await tester.pumpAndSettle();
      expect(find.text('Campus notes'), findsOneWidget);
      expect(find.text('回复中的话题'), findsNothing);
      expect(find.text('云端未完成内容'), findsNothing);
      await tester.tap(find.byTooltip('清空搜索'));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(ChoiceChip, '回复'));
      await tester.pumpAndSettle();
      expect(find.text('回复中的话题'), findsOneWidget);
      expect(find.text('Campus notes'), findsNothing);
      expect(await store.drafts(_scope), hasLength(2));
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
  testWidgets(
    'drafts share one scroll surface and failed refresh keeps cloud content',
    (tester) async {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemoryTokenStorage(),
        baseUrl: 'http://fake.local',
      );
      final pages = _Pages(client);
      final store = WritingStore();
      await store.save(
        writingScope('http://fake.local', 1),
        const LocalDraft(
          key: 'reply-100',
          kind: DraftKind.reply,
          topicId: 100,
          title: '回复中的话题',
          content: '本机回复正文',
          contentType: 2,
          categories: [],
          images: [],
          updatedAt: 1,
        ),
      );
      final router = GoRouter(
        initialLocation: '/drafts',
        routes: [
          GoRoute(path: '/drafts', builder: (_, _) => const DraftsPage()),
          GoRoute(
            path: '/p/:id',
            builder: (_, state) => Scaffold(
              body: Text('reply-topic-${state.pathParameters['id']}'),
            ),
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            apiClientProvider.overrideWithValue(client),
            pageRepositoryProvider.overrideWithValue(pages),
            writingStoreProvider.overrideWithValue(store),
            currentUserProvider.overrideWith(
              (ref) async => const CurrentUser(id: 1, username: 'alice'),
            ),
          ],
          child: MaterialApp.router(
            routerConfig: router,
            theme: gfThemeData(Brightness.light),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            locale: const Locale('zh'),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(CustomScrollView), findsOneWidget);
      expect(find.byType(ListView), findsNothing);
      expect(find.text('本机回复正文'), findsOneWidget);
      expect(find.text('云端未完成内容'), findsOneWidget);
      pages.fail = true;
      await tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.pumpAndSettle();
      expect(find.text('云端未完成内容'), findsOneWidget);
      expect(find.text('本机回复正文'), findsOneWidget);
      expect(find.text('重试'), findsOneWidget);
      pages.fail = false;
      await tester.tap(find.text('重试'));
      await tester.pumpAndSettle();
      expect(find.text('重试'), findsNothing);
      await tester.tap(find.text('回复中的话题'));
      await tester.pumpAndSettle();
      expect(find.text('reply-topic-100'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );
}
