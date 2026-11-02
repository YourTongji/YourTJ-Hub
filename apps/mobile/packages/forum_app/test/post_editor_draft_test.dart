import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/pages/topic/post_editor.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Posts extends PostRepository {
  _Posts(super.client, this.beforeWrite);
  final Future<void> Function() beforeWrite;
  Object? failure = const UnauthorizedException();
  int calls = 0;
  @override
  Future<UpdatePostResult> updatePost({
    required int postId,
    required String content,
  }) async {
    calls++;
    await beforeWrite();
    if (failure != null) throw failure!;
    return UpdatePostResult(
      id: postId,
      content: content,
      renderedContent: '',
      updatedAt: '',
    );
  }
}

class _Store extends WritingStore {
  bool fail = false;
  Completer<void>? pending;
  @override
  Future<void> save(
    String scope,
    LocalDraft draft, {
    bool Function()? isCurrent,
  }) async {
    if (fail) throw StateError('disk unavailable');
    await pending?.future;
    await super.save(scope, draft, isCurrent: isCurrent);
  }
}

final _field = find.descendant(
  of: find.byKey(const Key('post-edit-composer')),
  matching: find.byType(TextField),
);
Future<({ProviderContainer container, _Posts posts})> _open(
  WidgetTester tester,
  WritingStore store, {
  int owner = 1,
}) async {
  final client = GfApiClient(
    dio: Dio(),
    tokenStorage: MemoryTokenStorage(),
    baseUrl: 'https://example.test',
  );
  final posts = _Posts(client, () async {});
  final container = ProviderContainer(
    overrides: [
      apiClientProvider.overrideWithValue(client),
      postRepositoryProvider.overrideWithValue(posts),
      writingStoreProvider.overrideWithValue(store),
      currentUserProvider.overrideWith(
        (ref) async => CurrentUser(id: owner, username: 'author'),
      ),
    ],
  );
  addTearDown(container.dispose);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp(
        theme: gfThemeData(Brightness.light),
        locale: const Locale('en'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(
          builder: (context) => Scaffold(
            body: TextButton(
              onPressed: () => showPostEditor(
                context,
                postId: 42,
                content: 'original reply',
              ),
              child: const Text('Open editor'),
            ),
          ),
        ),
      ),
    ),
  );
  await tester.tap(find.text('Open editor'));
  await tester.pumpAndSettle();
  return (container: container, posts: posts);
}

void main() {
  testWidgets(
    'reply edits are durably recoverable before session-expiring HTTP',
    (tester) async {
      final store = WritingStore();
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: MemoryTokenStorage(),
        baseUrl: 'https://example.test',
      );
      List<LocalDraft>? atRequest;
      final posts = _Posts(client, () async {
        atRequest = await store.drafts(writingScope('https://example.test', 1));
      });
      final container = ProviderContainer(
        overrides: [
          apiClientProvider.overrideWithValue(client),
          postRepositoryProvider.overrideWithValue(posts),
          writingStoreProvider.overrideWithValue(store),
          currentUserProvider.overrideWith(
            (ref) async => const CurrentUser(id: 1, username: 'alice'),
          ),
        ],
      );
      addTearDown(container.dispose);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            locale: const Locale('en'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const PostEditor(postId: 42, content: 'original reply'),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final field = find.descendant(
        of: find.byKey(const Key('post-edit-composer')),
        matching: find.byType(TextField),
      );
      await tester.enterText(field, 'final uncommitted edit');
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();
      expect(posts.calls, 1);
      expect(atRequest, hasLength(1));
      expect(atRequest!.single.content, 'final uncommitted edit');
      // The recovery copy belongs to A and the current site, never another account.
      expect(
        await store.drafts(writingScope('https://example.test', 2)),
        isEmpty,
      );
      expect(
        await store.drafts(writingScope('https://another.test', 1)),
        isEmpty,
      );
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets(
    'a storage failure blocks HTTP and keeps input for a manual retry',
    (tester) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(320, 844);
      tester.platformDispatcher.textScaleFactorTestValue = 2;
      addTearDown(tester.view.reset);
      addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
      final store = _Store()..fail = true;
      final result = await _open(tester, store);
      await tester.enterText(_field, 'never discard this input');
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();
      expect(result.posts.calls, 0);
      expect(
        tester.widget<TextField>(_field).controller!.text,
        'never discard this input',
      );
      expect(
        find.text(
          AppLocalizations.of(
            tester.element(find.byType(PostEditor)),
          ).draftLocalSaveFailed,
        ),
        findsOneWidget,
      );
      final failure = find.text(
        AppLocalizations.of(
          tester.element(find.byType(PostEditor)),
        ).draftLocalSaveFailed,
      );
      final failureBox = tester.getRect(failure);
      expect(failureBox.width, greaterThan(100));
      expect(failureBox.height, greaterThan(10));
      expect(tester.takeException(), isNull);
      store.fail = false;
      result.posts.failure = null;
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();
      expect(result.posts.calls, 1);
      expect(find.byType(PostEditor), findsNothing);
      await tester.pump(const Duration(seconds: 2));
      expect(
        await store.drafts(writingScope('https://example.test', 1)),
        isEmpty,
      );
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets(
    'same-account reopen restores edits without replay; discard is durable',
    (tester) async {
      final store = WritingStore();
      final first = await _open(tester, store);
      await tester.enterText(_field, 'recover my reply edit');
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();
      expect(first.posts.calls, 1);
      first.container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pumpWidget(const SizedBox());
      final second = await _open(tester, store, owner: 2);
      expect(
        tester.widget<TextField>(_field).controller!.text,
        'original reply',
      );
      expect(second.posts.calls, 0);
      second.container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pumpWidget(const SizedBox());
      final restored = await _open(tester, store);
      expect(
        tester.widget<TextField>(_field).controller!.text,
        'recover my reply edit',
      );
      expect(restored.posts.calls, 0);
      await tester.tap(find.byTooltip('Close'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(
        (await store.drafts(
          writingScope('https://example.test', 1),
        )).single.content,
        'recover my reply edit',
      );
      await tester.tap(find.byTooltip('Close'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Discard changes'));
      await tester.pumpAndSettle();
      expect(find.byType(PostEditor), findsNothing);
      await tester.pump(const Duration(seconds: 2));
      expect(
        await store.drafts(writingScope('https://example.test', 1)),
        isEmpty,
      );
      expect(restored.posts.calls, 0);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets(
    'changing session during preflight cancels the disk write and HTTP',
    (tester) async {
      final store = _Store()..pending = Completer<void>();
      final result = await _open(tester, store);
      await tester.enterText(_field, 'old account edit');
      await tester.tap(find.text('Save'));
      await tester.pump();
      result.container.read(offlineCacheEpochProvider.notifier).invalidate();
      store.pending!.complete();
      await tester.pumpAndSettle();
      expect(result.posts.calls, 0);
      expect(
        await store.drafts(writingScope('https://example.test', 1)),
        isEmpty,
      );
      expect(
        await store.drafts(writingScope('https://example.test', 2)),
        isEmpty,
      );
      await tester.pumpWidget(const SizedBox());
    },
  );
  testWidgets('401 editor cleanup cannot pop the replacement login route', (
    tester,
  ) async {
    final store = WritingStore();
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: MemoryTokenStorage(),
      baseUrl: 'https://example.test',
    );
    late GoRouter router;
    late ProviderContainer container;
    final posts = _Posts(client, () async {
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      router.go('/login');
    });
    router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (context, _) => Scaffold(
            body: TextButton(
              onPressed: () => showPostEditor(
                context,
                postId: 42,
                content: 'original reply',
              ),
              child: const Text('Open editor'),
            ),
          ),
        ),
        GoRoute(
          path: '/login',
          builder: (_, _) => const Scaffold(body: Text('replacement login')),
        ),
      ],
    );
    addTearDown(router.dispose);
    container = ProviderContainer(
      overrides: [
        apiClientProvider.overrideWithValue(client),
        postRepositoryProvider.overrideWithValue(posts),
        writingStoreProvider.overrideWithValue(store),
        currentUserProvider.overrideWith(
          (ref) async => const CurrentUser(id: 1, username: 'author'),
        ),
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp.router(
          routerConfig: router,
          theme: gfThemeData(Brightness.light),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Open editor'));
    await tester.pumpAndSettle();
    await tester.enterText(_field, 'preserve me across 401');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(router.state.uri.path, '/login');
    expect(find.text('replacement login'), findsOneWidget);
    expect(posts.calls, 1);
    expect(
      (await store.drafts(
        writingScope('https://example.test', 1),
      )).single.content,
      'preserve me across 401',
    );
    await tester.pumpWidget(const SizedBox());
  });
}
