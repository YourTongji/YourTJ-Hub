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
  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    if (fail) throw const NetworkException(fallbackMessage: 'offline');
    return parsePayload({
      'component': PageComponent.drafts,
      'url': '/drafts',
      'version': '1',
      'layout': minimalLayoutJson(),
      'meta': {'title': 'Drafts'},
      'props': {
        'total': 1,
        'drafts': [
          {
            'id': 42,
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
        pageRepositoryProvider.overrideWithValue(_Pages(client)),
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
  await tester.tap(find.widgetWithText(TextButton, '删除本机草稿'));
  await tester.pumpAndSettle();
}

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));
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
    await tester.tap(find.widgetWithText(TextButton, l10n.draftDeleteLocal));
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
