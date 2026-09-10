import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/content/content_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Content extends ContentRepository {
  _Content(super.client);
  final pending = Completer<UserContentPage>();
  int deletions = 0;
  @override
  Future<List<ContentDeletionResult>> delete({
    required String contentType,
    required List<int> ids,
    String? password,
  }) async {
    deletions++;
    return [
      for (final id in ids) ContentDeletionResult(contentId: id, success: true),
    ];
  }

  @override
  Future<UserContentPage> list({
    required String contentType,
    bool deleted = false,
    int cursor = 0,
  }) => pending.future;
}

void main() {
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
      expect(find.byType(AlertDialog), findsOneWidget);
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pump();
      await tester.tap(
        find
            .descendant(
              of: find.byType(AlertDialog),
              matching: find.text('Delete'),
            )
            .last,
      );
      await tester.pumpAndSettle();
      expect(repo.deletions, 0);
      expect(find.text('Previous account topic'), findsNothing);
    },
  );
}
