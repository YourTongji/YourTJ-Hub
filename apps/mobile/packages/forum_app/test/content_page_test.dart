import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/content/content_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/router.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Content extends ContentRepository {
  _Content(super.client);
  final pending = Completer<UserContentPage>();
  int deletions = 0;
  bool requirePassword = false;
  final passwords = <String?>[];
  final listTypes = <String>[];
  final failIds = <int>{};
  @override
  Future<List<ContentDeletionResult>> delete({
    required String contentType,
    required List<int> ids,
    String? password,
  }) async {
    deletions++;
    passwords.add(password);
    if (requirePassword && password == null) {
      throw const ApiException(
        messageCode: 'content.batchDelete.confirmRequired',
        fallbackMessage: 'Password confirmation required',
      );
    }
    return [
      for (final id in ids)
        ContentDeletionResult(
          contentId: id,
          success: !failIds.contains(id),
          message: failIds.contains(id) ? 'delete rejected' : '',
        ),
    ];
  }

  @override
  Future<UserContentPage> list({
    required String contentType,
    bool deleted = false,
    int cursor = 0,
  }) {
    listTypes.add(contentType);
    return pending.future;
  }
}

class _RestoringContent extends ContentRepository {
  _RestoringContent(super.client, this.item, {this.fail = false, this.gate});
  final UserContentItem item;
  final bool fail;
  final Completer<void>? gate;
  bool restored = false;
  @override
  Future<UserContentPage> list({
    required String contentType,
    bool deleted = false,
    int cursor = 0,
  }) async => UserContentPage(
    items: deleted != restored ? [item] : [],
    hasMore: false,
    nextCursorId: 0,
  );
  @override
  Future<void> restore(UserContentItem item) async {
    await gate?.future;
    if (fail) throw StateError('restore rejected');
    restored = true;
  }
}

void main() {
  testWidgets(
    'topic deletion does not hide server-visible replies from owned content',
    (tester) async {
      final repo = _Content(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'https://example.test',
        ),
      );
      repo.pending.complete(
        const UserContentPage(
          items: [
            UserContentItem(
              id: 200,
              topicId: 100,
              contentType: 'post',
              title: 'Retained discussion reply',
            ),
            UserContentItem(
              id: 100,
              topicId: 999,
              contentType: 'post',
              title: 'Unrelated reply with colliding ID',
            ),
          ],
          hasMore: false,
          nextCursorId: 0,
        ),
      );
      final container = ProviderContainer(
        overrides: [contentRepositoryProvider.overrideWithValue(repo)],
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
            home: const ContentPage(initialType: 'post'),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Retained discussion reply'), findsOneWidget);
      container.read(removedTopicsProvider.notifier).add(100);
      await tester.pumpAndSettle();
      expect(find.text('Retained discussion reply'), findsOneWidget);
      expect(find.text('Unrelated reply with colliding ID'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  for (final kind in ['topic', 'post']) {
    for (final fail in [false, true]) {
      testWidgets(
        'restore $kind fail=$fail reconciles only acknowledged topic visibility',
        (tester) async {
          final item = UserContentItem(
            id: 100,
            topicId: kind == 'post' ? 200 : null,
            contentType: kind,
            title: 'Restorable item',
            canRestore: true,
          );
          final repo = _RestoringContent(
            GfApiClient(
              dio: Dio(),
              tokenStorage: MemoryTokenStorage(),
              baseUrl: 'https://example.test',
            ),
            item,
            fail: fail,
          );
          final container = ProviderContainer(
            overrides: [contentRepositoryProvider.overrideWithValue(repo)],
          );
          addTearDown(container.dispose);
          container.read(removedTopicsProvider.notifier).add(100);
          Widget app(bool deleted) => UncontrolledProviderScope(
            container: container,
            child: MaterialApp(
              theme: gfThemeData(Brightness.light),
              locale: const Locale('en'),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: ContentPage(
                key: ValueKey(deleted),
                deleted: deleted,
                initialType: kind,
              ),
            ),
          );
          await tester.pumpWidget(app(true));
          await tester.pumpAndSettle();
          expect(
            find.text('Restorable item'),
            findsOneWidget,
            reason: 'deleted topics remain available in the recycle bin',
          );
          await tester.tap(find.byType(GfPopupMenuButton<String>));
          await tester.pumpAndSettle();
          await tester.tap(find.text('Restore'));
          await tester.pumpAndSettle();
          expect(
            container.read(removedTopicsProvider).contains(100),
            fail || kind == 'post',
          );
          if (!fail) {
            expect(find.text('Restorable item'), findsNothing);
            await tester.pumpWidget(app(false));
            await tester.pumpAndSettle();
            expect(
              find.text('Restorable item'),
              findsOneWidget,
              reason:
                  'the restored content is visible again without signing out',
            );
          }
          expect(tester.takeException(), isNull);
        },
      );
    }
  }
  testWidgets(
    'old account restore acknowledgement cannot clear a new account deletion',
    (tester) async {
      final gate = Completer<void>();
      final repo = _RestoringContent(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'https://example.test',
        ),
        const UserContentItem(
          id: 100,
          contentType: 'topic',
          title: 'Restorable item',
          canRestore: true,
        ),
        gate: gate,
      );
      final container = ProviderContainer(
        overrides: [contentRepositoryProvider.overrideWithValue(repo)],
      );
      addTearDown(container.dispose);
      container.read(removedTopicsProvider.notifier).add(100);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            locale: const Locale('en'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ContentPage(deleted: true),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(GfPopupMenuButton<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Restore'));
      await tester.pump();
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      container.read(removedTopicsProvider.notifier).add(100);
      gate.complete();
      await tester.pumpAndSettle();
      expect(container.read(removedTopicsProvider), contains(100));
      expect(tester.takeException(), isNull);
    },
  );

  for (final scale in [1.0, 1.5, 1.51, 2.0]) {
    testWidgets('content keeps width and text-size boundaries at $scale', (
      tester,
    ) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(899, 900);
      tester.platformDispatcher.textScaleFactorTestValue = scale;
      addTearDown(tester.view.reset);
      addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
      final repo = _Content(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'https://example.test',
        ),
      );
      repo.pending.complete(
        const UserContentPage(items: [], hasMore: false, nextCursorId: 0),
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [contentRepositoryProvider.overrideWithValue(repo)],
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            locale: const Locale('en'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ContentPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final sidebar = find.byKey(const Key('content-type-sidebar'));
      expect(sidebar, findsNothing);
      for (final width in [900.0, 1120.0]) {
        tester.view.physicalSize = Size(width, 900);
        await tester.pumpAndSettle();
        expect(sidebar, scale <= 1.5 ? findsOneWidget : findsNothing);
        expect(repo.listTypes, hasLength(1));
        expect(tester.takeException(), isNull);
      }
    });
  }

  for (final deleted in [false, true]) {
    testWidgets(
      'wide content keeps type, selection and scroll across resizing deleted=$deleted',
      (tester) async {
        await tester.binding.setSurfaceSize(const Size(1120, 800));
        addTearDown(() => tester.binding.setSurfaceSize(null));
        final repo = _Content(
          GfApiClient(
            dio: Dio(),
            tokenStorage: MemoryTokenStorage(),
            baseUrl: 'https://example.test',
          ),
        );
        repo.pending.complete(
          UserContentPage(
            items: [
              for (var i = 1; i <= 30; i++)
                UserContentItem(
                  id: i,
                  contentType: 'post',
                  title: 'Owned content $i',
                ),
            ],
            hasMore: false,
            nextCursorId: 0,
          ),
        );
        final path = deleted ? '/recycle-bin' : '/my-content';
        final router = GoRouter(
          initialLocation: path,
          routes: [
            appRouter.configuration.routes.whereType<GoRoute>().singleWhere(
              (route) => route.path == path,
            ),
          ],
        );
        addTearDown(router.dispose);
        await tester.pumpWidget(
          ProviderScope(
            overrides: [contentRepositoryProvider.overrideWithValue(repo)],
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
        expect(find.byKey(const Key('content-type-sidebar')), findsOneWidget);
        await tester.tap(find.text('Replies'));
        await tester.pumpAndSettle();
        if (!deleted) {
          await tester.tap(find.byType(Checkbox).at(1));
          await tester.pump();
        }
        final list = tester.element(find.byType(ListView));
        final scrollable = find.descendant(
          of: find.byType(ListView),
          matching: find.byType(Scrollable),
        );
        await tester.drag(find.byType(ListView), const Offset(0, -350));
        await tester.pumpAndSettle();
        final offset = tester
            .state<ScrollableState>(scrollable)
            .position
            .pixels;
        final requests = repo.listTypes.length;
        await tester.binding.setSurfaceSize(const Size(500, 800));
        await tester.pumpAndSettle();
        expect(find.byKey(const Key('content-type-sidebar')), findsNothing);
        expect(tester.element(find.byType(ListView)), same(list));
        expect(
          tester.state<ScrollableState>(scrollable).position.pixels,
          closeTo(offset, .1),
        );
        expect(
          tester
              .widget<GfSegmented<String>>(find.byType(GfSegmented<String>))
              .selected,
          'post',
        );
        if (!deleted) expect(find.text('Selected 1 / 50'), findsOneWidget);
        await tester.binding.setSurfaceSize(const Size(1120, 800));
        await tester.pumpAndSettle();
        expect(find.byKey(const Key('content-type-sidebar')), findsOneWidget);
        expect(repo.listTypes.length, requests);
        expect(tester.takeException(), isNull);
      },
    );
  }
  testWidgets(
    'content confirmation offers a labeled password field and keyboard submit',
    (tester) async {
      final repo = _Content(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'https://example.test',
        ),
      )..requirePassword = true;
      repo.pending.complete(
        const UserContentPage(
          items: [
            UserContentItem(id: 1, contentType: 'topic', title: 'Owned topic'),
          ],
          hasMore: false,
          nextCursorId: 0,
        ),
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [contentRepositoryProvider.overrideWithValue(repo)],
          child: MaterialApp(
            theme: gfThemeData(Brightness.light),
            locale: const Locale('en'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ContentPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(Checkbox).first);
      await tester.pump();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();
      await tester.tap(
        find
            .descendant(
              of: find.byType(GfAlertDialog),
              matching: find.text('Delete'),
            )
            .last,
      );
      await tester.pumpAndSettle();
      final field = tester.widget<TextField>(find.byType(TextField));
      final l = AppLocalizations.of(tester.element(find.byType(ContentPage)));
      expect(field.decoration?.labelText, l.contentPassword);
      expect(field.autofillHints, contains(AutofillHints.password));
      expect(field.autocorrect, isFalse);
      expect(field.enableSuggestions, isFalse);
      await tester.enterText(find.byType(TextField), ' secret with spaces ');
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pumpAndSettle();
      expect(repo.passwords, [null, ' secret with spaces ']);
      expect(find.byType(AlertDialog), findsNothing);
    },
  );

  testWidgets('account boundary discards an in-flight private content list', (
    tester,
  ) async {
    final repo = _Content(
      GfApiClient(
        dio: Dio(),
        tokenStorage: MemoryTokenStorage(),
        baseUrl: 'https://example.test',
      ),
    );
    final container = ProviderContainer(
      overrides: [contentRepositoryProvider.overrideWithValue(repo)],
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
          home: const ContentPage(deleted: true),
        ),
      ),
    );
    await tester.pump();
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    repo.pending.complete(
      const UserContentPage(
        items: [
          UserContentItem(
            id: 1,
            contentType: 'topic',
            title: 'Previous account private trash',
          ),
        ],
        hasMore: false,
        nextCursorId: 0,
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Previous account private trash'), findsNothing);
  });
  testWidgets(
    'account change while confirming deletion cannot submit old content IDs',
    (tester) async {
      final repo = _Content(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'https://example.test',
        ),
      );
      repo.pending.complete(
        const UserContentPage(
          items: [
            UserContentItem(
              id: 1,
              contentType: 'topic',
              title: 'Previous account topic',
            ),
          ],
          hasMore: false,
          nextCursorId: 0,
        ),
      );
      final container = ProviderContainer(
        overrides: [contentRepositoryProvider.overrideWithValue(repo)],
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
            home: const ContentPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(Checkbox).first);
      await tester.pump();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();
      expect(find.byType(GfAlertDialog), findsOneWidget);
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pump();
      await tester.tap(
        find
            .descendant(
              of: find.byType(GfAlertDialog),
              matching: find.text('Delete'),
            )
            .last,
      );
      await tester.pumpAndSettle();
      expect(repo.deletions, 0);
      expect(find.text('Previous account topic'), findsNothing);
    },
  );

  testWidgets(
    'successful batch topic deletion tombstones the topic in removedTopicsProvider',
    (tester) async {
      final repo = _Content(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'https://example.test',
        ),
      );
      repo.pending.complete(
        const UserContentPage(
          items: [
            UserContentItem(id: 7, contentType: 'topic', title: 'Owned topic'),
          ],
          hasMore: false,
          nextCursorId: 0,
        ),
      );
      final container = ProviderContainer(
        overrides: [contentRepositoryProvider.overrideWithValue(repo)],
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
            home: const ContentPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(Checkbox).first);
      await tester.pump();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();
      await tester.tap(
        find
            .descendant(
              of: find.byType(GfAlertDialog),
              matching: find.text('Delete'),
            )
            .last,
      );
      await tester.pumpAndSettle();
      expect(container.read(removedTopicsProvider), contains(7));
    },
  );

  testWidgets(
    'failed batch topic deletion leaves removedTopicsProvider untouched',
    (tester) async {
      final repo = _Content(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'https://example.test',
        ),
      )..failIds.add(7);
      repo.pending.complete(
        const UserContentPage(
          items: [
            UserContentItem(id: 7, contentType: 'topic', title: 'Owned topic'),
          ],
          hasMore: false,
          nextCursorId: 0,
        ),
      );
      final container = ProviderContainer(
        overrides: [contentRepositoryProvider.overrideWithValue(repo)],
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
            home: const ContentPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(Checkbox).first);
      await tester.pump();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();
      await tester.tap(
        find
            .descendant(
              of: find.byType(GfAlertDialog),
              matching: find.text('Delete'),
            )
            .last,
      );
      await tester.pumpAndSettle();
      expect(container.read(removedTopicsProvider), isNot(contains(7)));
    },
  );

  testWidgets(
    'deleted topic leaves owned content and cannot return in a stale response',
    (tester) async {
      final repo = _Content(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'https://example.test',
        ),
      );
      final container = ProviderContainer(
        overrides: [contentRepositoryProvider.overrideWithValue(repo)],
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
            home: const ContentPage(),
          ),
        ),
      );
      await tester.pump();
      container.read(removedTopicsProvider.notifier).add(100);
      repo.pending.complete(
        const UserContentPage(
          items: [
            UserContentItem(
              id: 100,
              contentType: 'topic',
              title: 'deleted topic',
              excerpt: '',
              createdAt: '',
            ),
            UserContentItem(
              id: 101,
              contentType: 'topic',
              title: 'kept topic',
              excerpt: '',
              createdAt: '',
            ),
          ],
          hasMore: false,
          nextCursorId: 0,
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('deleted topic'), findsNothing);
      expect(find.text('kept topic'), findsOneWidget);
      container.read(removedTopicsProvider.notifier).add(101);
      await tester.pumpAndSettle();
      expect(find.text('kept topic'), findsNothing);
      expect(find.byType(GfEmpty), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('untitled moment uses its excerpt as the primary content line', (
    tester,
  ) async {
    final repo = _Content(
      GfApiClient(
        dio: Dio(),
        tokenStorage: MemoryTokenStorage(),
        baseUrl: 'https://example.test',
      ),
    );
    repo.pending.complete(
      const UserContentPage(
        items: [
          UserContentItem(
            id: 7,
            contentType: 'topic',
            title: '',
            excerpt: '只有正文的瞬间摘要',
            createdAt: '2026-09-28 10:00:00',
          ),
        ],
        hasMore: false,
        nextCursorId: 0,
      ),
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [contentRepositoryProvider.overrideWithValue(repo)],
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: const Locale('en'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const ContentPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final tile = tester.widget<ListTile>(find.byType(ListTile).first);
    expect((tile.title as Text).data, '只有正文的瞬间摘要');
    expect((tile.subtitle as Text).data, '2026-09-28 10:00:00');
  });

  // Profile streams open management on their own filter (issue #1372 H3).
  for (final (query, expected) in [
    ('?type=post', 'post'),
    ('?type=topic', 'topic'),
    ('?type=unknown', 'topic'),
    ('', 'topic'),
  ]) {
    testWidgets('my-content$query preselects the $expected filter', (
      tester,
    ) async {
      final repo = _Content(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'https://example.test',
        ),
      );
      repo.pending.complete(
        const UserContentPage(items: [], hasMore: false, nextCursorId: 0),
      );
      final router = GoRouter(
        initialLocation: '/my-content$query',
        routes: [
          appRouter.configuration.routes.whereType<GoRoute>().singleWhere(
            (route) => route.path == '/my-content',
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [contentRepositoryProvider.overrideWithValue(repo)],
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

      expect(repo.listTypes, [expected]);
      expect(
        tester
            .widget<GfSegmented<String>>(find.byType(GfSegmented<String>))
            .selected,
        expected,
      );
    });
  }
}
