import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/local/writing_store.dart';
import 'package:forum_app/src/pages/search/search_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

class _SearchCall {
  _SearchCall(this.query, this.scope, this.page);
  final String query;
  final String scope;
  final int page;
  final result = Completer<SearchPageProps>();
}

class _SearchRepository extends TopicRepository {
  _SearchRepository()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: _Tokens(),
          baseUrl: 'http://fake.local',
        ),
      );
  final calls = <_SearchCall>[];
  @override
  Future<SearchPageProps> search({
    required String query,
    String scope = '',
    int page = 1,
    CancelToken? cancelToken,
  }) {
    final call = _SearchCall(query, scope, page);
    calls.add(call);
    return call.result.future;
  }
}

TopicPayload _topic(int id) => TopicPayload(
  id: id,
  title: 'Topic $id',
  description: 'Topic description',
  url: '/p/$id',
  author: const UserBriefPayload(id: 1, username: 'alice', avatarUrl: ''),
  participants: const [],
  categories: const [],
  replyCount: 0,
  viewCount: 1,
  pinWeight: 0,
  processStatus: 0,
  activityText: '',
  lastUpdateTime: '',
);

SearchPageProps _result(
  _SearchCall call, {
  int topics = 0,
  int users = 0,
  int categories = 0,
  int totalPages = 1,
  List<String>? failedScopes,
  bool unavailable = false,
}) => SearchPageProps(
  query: call.query,
  scope: call.scope,
  topics: List.generate(topics, (i) => _topic((call.page - 1) * 10 + i + 1)),
  users: List.generate(
    users,
    (i) => UserSearchPayload(
      id: i + 1,
      username: 'user$i',
      nickname: 'User $i',
      avatarUrl: '',
      bio: 'User bio',
    ),
  ),
  categories: List.generate(
    categories,
    (i) => CategorySearchPayload(
      id: i + 1,
      name: 'Category $i',
      slug: 'cat$i',
      icon: '#',
      color: '#123456',
      desc: 'Category description',
    ),
  ),
  courses: const [],
  total: topics * totalPages,
  usersTotal: users,
  categoriesTotal: categories,
  coursesTotal: 0,
  totalPages: totalPages,
  pagination: PaginationPayload(
    page: call.page,
    nextPage: call.page + 1,
    hasNext: call.page < totalPages,
    nextUrl: '',
  ),
  failedScopes: failedScopes,
  searchUnavailable: unavailable,
);

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  Future<_SearchRepository> pumpSearch(
    WidgetTester tester, {
    Locale locale = const Locale('en'),
  }) async {
    final repository = _SearchRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          topicRepositoryProvider.overrideWithValue(repository),
          writingScopeProvider.overrideWith((ref) async => 'test:guest'),
        ],
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: locale,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const SearchPage(),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'original');
    await tester.testTextInput.receiveAction(TextInputAction.search);
    await tester.pump();
    return repository;
  }

  testWidgets(
    'scope controls remain available while loading and after failure',
    (tester) async {
      final repo = await pumpSearch(tester);
      expect(find.text('Users'), findsOneWidget);
      await tester.tap(find.text('Users'));
      await tester.pump();
      expect(repo.calls.last.scope, 'users');
      repo.calls.last.result.completeError(StateError('offline'));
      await tester.pumpAndSettle();
      expect(find.text('Topics'), findsOneWidget);
      await tester.tap(find.text('Topics'));
      await tester.pump();
      expect(repo.calls.last.scope, 'topics');
      repo.calls.last.result.complete(_result(repo.calls.last, topics: 1));
      repo.calls.first.result.complete(_result(repo.calls.first, users: 1));
      await tester.pumpAndSettle();
      expect(find.text('Topic 1'), findsOneWidget);
      expect(find.text('User 0'), findsNothing);
    },
  );

  testWidgets(
    'scope switch uses submitted query and ignores an older scope response',
    (tester) async {
      final repo = await pumpSearch(tester);
      repo.calls.first.result.complete(_result(repo.calls.first, topics: 1));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'unsent edit');
      await tester.tap(find.text('Users').first);
      await tester.pump();
      final users = repo.calls.last;
      expect(users.query, 'original');
      await tester.tap(find.text('Categories').first);
      await tester.pump();
      final categories = repo.calls.last;
      expect(categories.scope, 'categories');
      categories.result.complete(_result(categories, categories: 1));
      users.result.complete(_result(users, users: 1));
      await tester.pumpAndSettle();
      expect(find.text('Category 0'), findsOneWidget);
      expect(find.text('User 0'), findsNothing);
      expect(
        tester.widget<TextField>(find.byType(TextField)).controller!.text,
        'unsent edit',
      );
    },
  );

  testWidgets(
    'scoped totals are not presented as zero results in other scopes',
    (tester) async {
      final repo = await pumpSearch(tester);
      repo.calls.first.result.complete(_result(repo.calls.first, topics: 1));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Users').first);
      await tester.pump();
      repo.calls.last.result.complete(
        _result(repo.calls.last, users: 1).copyWith(usersTotal: 50),
      );
      await tester.pumpAndSettle();
      expect(find.text('0'), findsNothing);
      expect(find.text('1 shown · 50 matches'), findsOneWidget);
    },
  );

  for (final scope in ['topics', 'users', 'categories']) {
    testWidgets('$scope results build only rows near the viewport', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(390, 700);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final repo = await pumpSearch(tester);
      repo.calls.first.result.complete(
        _result(
          repo.calls.first,
          topics: scope == 'topics' ? 10 : 0,
          users: scope == 'users' ? 30 : 0,
          categories: scope == 'categories' ? 30 : 0,
        ),
      );
      await tester.pumpAndSettle();
      final last = switch (scope) {
        'topics' => 'Topic 10',
        'users' => 'User 29',
        _ => 'Category 29',
      };
      expect(find.text(last), findsNothing);
      if (scope == 'topics') {
        expect(find.byType(GfTopicRow).evaluate().length, lessThan(10));
      }
    });
  }

  testWidgets('topic pagination preserves the other aggregate groups', (
    tester,
  ) async {
    final repo = await pumpSearch(tester);
    repo.calls.first.result.complete(
      _result(
        repo.calls.first,
        topics: 1,
        users: 1,
        categories: 1,
        totalPages: 2,
      ),
    );
    await tester.pump();
    await tester.pump();
    // The short initial result brings the topic footer into the viewport.
    final more = repo.calls.last;
    expect(more.page, 2);
    expect(more.scope, 'topics');
    more.result.complete(_result(more, topics: 1, totalPages: 2));
    await tester.pumpAndSettle();
    expect(find.text('User 0'), findsOneWidget);
    expect(find.text('Topic 1'), findsOneWidget);
    expect(find.text('Topic 11'), findsOneWidget);
    await tester.scrollUntilVisible(
      find.text('Category 0'),
      200,
      scrollable: find
          .descendant(
            of: find.byType(ListView),
            matching: find.byType(Scrollable),
          )
          .first,
    );
    expect(find.text('Category 0'), findsOneWidget);
  });

  testWidgets(
    'a failed request retries the submitted query after further typing',
    (tester) async {
      final repo = await pumpSearch(tester);
      repo.calls.first.result.completeError(StateError('offline'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'not submitted');
      await tester.tap(find.text('Retry'));
      await tester.pump();
      expect(repo.calls.last.query, 'original');
      repo.calls.last.result.complete(_result(repo.calls.last, topics: 1));
      await tester.pumpAndSettle();
      expect(find.text('Topic 1'), findsOneWidget);
    },
  );

  testWidgets(
    'unavailable scope can switch and partial failures use type labels',
    (tester) async {
      final repo = await pumpSearch(tester);
      repo.calls.first.result.complete(
        _result(repo.calls.first, unavailable: true),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('Topics'));
      await tester.pump();
      repo.calls.last.result.complete(
        _result(
          repo.calls.last,
          topics: 1,
          failedScopes: ['users', 'categories'],
        ),
      );
      await tester.pumpAndSettle();
      expect(
        find.text('Search unavailable: Users, Categories'),
        findsOneWidget,
      );
      expect(find.text('Topic 1'), findsOneWidget);
    },
  );

  testWidgets(
    'unavailable topic page keeps prior results and retries the same page',
    (tester) async {
      final repo = await pumpSearch(tester);
      repo.calls.first.result.complete(
        _result(repo.calls.first, topics: 1, totalPages: 2),
      );
      await tester.pump();
      await tester.pump();
      expect(repo.calls.last.page, 2);
      repo.calls.last.result.complete(
        _result(repo.calls.last, unavailable: true),
      );
      await tester.pumpAndSettle();
      expect(find.text('Topic 1'), findsOneWidget);
      await tester.tap(find.text('Retry'));
      await tester.pump();
      expect(repo.calls.last.page, 2);
      repo.calls.last.result.complete(
        _result(repo.calls.last, topics: 1, totalPages: 2),
      );
      await tester.pumpAndSettle();
      expect(find.text('Topic 11'), findsOneWidget);
    },
  );

  testWidgets('clear while changing scope discards all outstanding responses', (
    tester,
  ) async {
    final repo = await pumpSearch(tester);
    await tester.tap(find.text('Users'));
    await tester.pump();
    await tester.tap(find.byTooltip('Clear search'));
    await tester.pump();
    for (final call in repo.calls) {
      call.result.complete(_result(call, users: 1));
    }
    await tester.pumpAndSettle();
    expect(find.text('User 0'), findsNothing);
    expect(find.text('All'), findsNothing);
    await tester.enterText(find.byType(TextField), 'next');
    await tester.testTextInput.receiveAction(TextInputAction.search);
    await tester.pump();
    expect(repo.calls.last.scope, '');
    expect(repo.calls.last.page, 1);
    repo.calls.last.result.complete(_result(repo.calls.last, categories: 1));
    await tester.pumpAndSettle();
  });

  testWidgets(
    'session invalidation resets scope and rejects its pending response',
    (tester) async {
      final repo = await pumpSearch(tester);
      await tester.tap(find.text('Users'));
      await tester.pump();
      final container = ProviderScope.containerOf(
        tester.element(find.byType(SearchPage)),
      );
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pump();
      for (final call in repo.calls) {
        call.result.complete(_result(call, users: 1));
      }
      await tester.pumpAndSettle();
      expect(find.text('User 0'), findsNothing);
      expect(find.text('All'), findsNothing);
      await tester.enterText(find.byType(TextField), 'new session');
      await tester.testTextInput.receiveAction(TextInputAction.search);
      await tester.pump();
      expect(repo.calls.last.scope, '');
      repo.calls.last.result.complete(_result(repo.calls.last, topics: 1));
      await tester.pumpAndSettle();
      expect(find.text('Topic 1'), findsOneWidget);
    },
  );

  testWidgets('late topic pagination cannot append to a different scope', (
    tester,
  ) async {
    final repo = await pumpSearch(tester);
    repo.calls.first.result.complete(
      _result(repo.calls.first, topics: 1, totalPages: 2),
    );
    await tester.pump();
    await tester.pump();
    final more = repo.calls.last;
    expect(more.page, 2);
    await tester.tap(find.text('Users'));
    await tester.pump();
    repo.calls.last.result.complete(_result(repo.calls.last, users: 1));
    more.result.complete(_result(more, topics: 1, totalPages: 2));
    await tester.pumpAndSettle();
    expect(find.text('User 0'), findsOneWidget);
    expect(find.text('Topic 11'), findsNothing);
  });

  for (final language in ['zh', 'en', 'ja', 'de']) {
    for (final width in [320.0, 390.0, 768.0, 1024.0]) {
      testWidgets('search types at 200% in $language on $width', (
        tester,
      ) async {
        tester.view.physicalSize = Size(width, 900);
        tester.view.devicePixelRatio = 1;
        tester.platformDispatcher.textScaleFactorTestValue = 2;
        addTearDown(tester.view.reset);
        addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
        final repo = await pumpSearch(tester, locale: Locale(language));
        repo.calls.first.result.complete(
          _result(repo.calls.first, users: 1, topics: 1, categories: 1),
        );
        await tester.pumpAndSettle();
        await tester.scrollUntilVisible(
          find.text('Category 0'),
          200,
          scrollable: find
              .descendant(
                of: find.byType(ListView),
                matching: find.byType(Scrollable),
              )
              .first,
        );
        expect(find.text('Category 0'), findsOneWidget);
        expect(tester.takeException(), isNull);
      });
    }
  }
}
