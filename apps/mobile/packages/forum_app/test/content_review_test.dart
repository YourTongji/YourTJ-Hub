import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/content/content_page.dart';
import 'package:forum_app/src/pages/topic/post_edit_sheet.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/realtime/realtime_updates.dart';
import 'package:ui_kit/ui_kit.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Posts extends PostRepository {
  _Posts(super.client);
  bool fail = true;
  final attempts = <String>[];
  @override
  Future<UpdatePostResult> updatePost({
    required int postId,
    required String content,
  }) async {
    expect(postId, 42);
    attempts.add(content);
    if (fail) throw const ApiException(fallbackMessage: 'Save failed');
    return UpdatePostResult(
      id: postId,
      content: content,
      renderedContent: '',
      updatedAt: '',
      pendingReview: true,
      checking: true,
    );
  }
}

class _Content extends ContentRepository {
  _Content(super.client, this.posts);
  final _Posts posts;
  @override
  Future<UserContentPage> list({
    required String contentType,
    bool deleted = false,
    int cursor = 0,
  }) async => UserContentPage(
    items: contentType != 'post'
        ? []
        : [
            UserContentItem(
              id: 42,
              contentType: 'post',
              title: 'Topic',
              content: 'Rejected body',
              processStatus: posts.fail ? 1 : 2,
            ),
          ],
    hasMore: false,
    nextCursorId: 0,
  );
}

void main() {
  test(
    'content invalidation preserves other counters and reconnect reconciles content',
    () {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      final notifier = container.read(realtimeInvalidationsProvider.notifier);
      notifier.chat(9);
      notifier.content();
      var state = container.read(realtimeInvalidationsProvider);
      expect(state.contentRevision, 1);
      expect(state.chatConvId, 9);
      notifier.resync();
      state = container.read(realtimeInvalidationsProvider);
      expect(state.contentRevision, 2);
      expect(state.chatConvId, 0);
    },
  );

  testWidgets('rejected reply edits survive failure and can be resubmitted', (
    tester,
  ) async {
    final repo = _Posts(
      GfApiClient(
        dio: Dio(),
        tokenStorage: MemoryTokenStorage(),
        baseUrl: 'https://example.test',
      ),
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          postRepositoryProvider.overrideWithValue(repo),
          contentRepositoryProvider.overrideWithValue(
            _Content(
              GfApiClient(
                dio: Dio(),
                tokenStorage: MemoryTokenStorage(),
                baseUrl: 'https://example.test',
              ),
              repo,
            ),
          ),
        ],
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
    await tester.tap(find.text('Replies'));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(PopupMenuButton<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Edit and resubmit'));
    await tester.pumpAndSettle();
    expect(find.byType(PostEditSheet), findsOneWidget);
    expect(find.byTooltip('Add image'), findsOneWidget);
    expect(find.text('Rejected body'), findsOneWidget);
    await tester.enterText(find.byType(TextField), 'Corrected body');
    await tester.tap(find.widgetWithText(GfButton, 'Save'));
    await tester.pumpAndSettle();
    expect(find.text('Corrected body'), findsOneWidget);
    expect(find.byType(PostEditSheet), findsOneWidget);
    await tester.tap(find.byTooltip('Close'));
    await tester.pumpAndSettle();
    expect(find.text('Keep your work?'), findsOneWidget);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(find.text('Corrected body'), findsOneWidget);
    repo.fail = false;
    await tester.tap(find.widgetWithText(GfButton, 'Save'));
    await tester.pumpAndSettle();
    expect(find.byType(PostEditSheet), findsNothing);
    expect(find.text('Under review'), findsOneWidget);
    expect(repo.attempts, ['Corrected body', 'Corrected body']);
    expect(tester.takeException(), isNull);
  });
}
