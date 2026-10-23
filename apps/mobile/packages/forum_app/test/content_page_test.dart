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
      for (final id in ids) ContentDeletionResult(contentId: id, success: true),
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

void main() {
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
