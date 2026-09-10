import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/wiki/wiki_search_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'widget_test.dart' show MemoryTokenStorage;

class _Repository extends WikiRepository {
  _Repository()
    : super(GfApiClient(dio: Dio(), tokenStorage: MemoryTokenStorage()));
  final requests = <String, Completer<WikiSearchResult>>{};
  @override
  Future<WikiSearchResult> search(String query) =>
      (requests[query] = Completer()).future;
}

WikiSearchResult _result(String title) => WikiSearchResult.fromJson({
  'query': '选课',
  'searchUnavailable': false,
  'total': 1,
  'items': [
    {
      'title': title,
      'titleHit': false,
      'score': 0.61,
      'hitType': 'body',
      'path': '指南/选课',
      'anchors': ['s-2'],
      'namespace': '指南',
      'heading': '选课',
      'snippet': '<mark>选课</mark>建议',
    },
  ],
});

void main() {
  testWidgets(
    'search debounces, ignores stale responses, and opens the hit anchor',
    (tester) async {
      final repository = _Repository();
      final router = GoRouter(
        routes: [
          GoRoute(path: '/', builder: (_, _) => const WikiSearchPage()),
          GoRoute(
            path: '/wiki/:path(.*)',
            builder: (_, _) => const Scaffold(body: Text('destination')),
          ),
        ],
      );
      addTearDown(router.dispose);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [wikiRepositoryProvider.overrideWithValue(repository)],
          child: MaterialApp.router(
            routerConfig: router,
            theme: gfThemeData(Brightness.light),
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), 'old');
      await tester.pump(const Duration(milliseconds: 150));
      expect(repository.requests, isEmpty);
      await tester.pump(const Duration(milliseconds: 150));
      await tester.enterText(find.byType(TextField), 'new');
      await tester.pump(const Duration(milliseconds: 300));
      repository.requests['new']!.complete(_result('新结果'));
      await tester.pumpAndSettle();
      repository.requests['old']!.complete(_result('旧结果'));
      await tester.pumpAndSettle();
      expect(find.text('旧结果'), findsNothing);
      expect(find.text('选课建议'), findsOneWidget);
      await tester.tap(find.text('新结果'));
      await tester.pumpAndSettle();
      expect(Uri.decodeComponent(router.state.uri.path), '/wiki/指南/选课');
      expect(router.state.uri.fragment, 's-2');
    },
  );
}
