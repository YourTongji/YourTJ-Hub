import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/topic/post_actions.dart';
import 'package:forum_app/src/pages/topic/post_history_sheet.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Posts extends PostRepository {
  _Posts(super.client);
  final likes = <int>[];
  final bookmarks = <int>[];
  final contents = <String>[];
  final cursors = <int>[];
  bool failEdit = true;
  @override
  Future<bool> likePost({required int postId, required int action}) async {
    expect(postId, 42);
    likes.add(action);
    return true;
  }

  @override
  Future<bool> bookmarkPost({required int postId, required int action}) async {
    expect(postId, 42);
    bookmarks.add(action);
    return true;
  }

  @override
  Future<UpdatePostResult> updatePost({
    required int postId,
    required String content,
  }) async {
    expect(postId, 42);
    contents.add(content);
    if (failEdit) {
      failEdit = false;
      throw StateError('retry');
    }
    return UpdatePostResult(
      id: 42,
      postNo: 2,
      content: content,
      renderedContent: content,
      updatedAt: '2026-09-07T00:00:00Z',
    );
  }

  @override
  Future<PostRevisionPage> revisions({
    required int postId,
    int beforeVersion = 0,
  }) async {
    cursors.add(beforeVersion);
    return PostRevisionPage(
      postId: postId,
      versions: [
        PostRevision(
          version: beforeVersion == 0 ? 2 : 1,
          editor: const UserBriefPayload(id: 0, username: '', avatarUrl: ''),
          content: '',
          renderedHTML: '',
          processStatus: 1,
          createdAt: '2026-09-07',
        ),
      ],
      hasMore: beforeVersion == 0,
      beforeVersion: beforeVersion == 0 ? 2 : 0,
    );
  }
}

void main() {
  PostPayload post() {
    final props = topicDetailPayloadJson()['props'] as Map<String, dynamic>;
    final first =
        (props['postStream']['posts'] as List).first as Map<String, dynamic>;
    return PostPayload.fromJson(first).copyWith(
      id: 42,
      postNo: 2,
      isOwnPost: true,
      isLiked: false,
      isBookmarked: false,
      likeCount: 3,
      content: 'Original reply',
    );
  }

  Future<void> pump(WidgetTester tester, _Posts repo, Widget child) async {
    final container = ProviderContainer(
      overrides: [postRepositoryProvider.overrideWithValue(repo)],
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
          home: Scaffold(body: child),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  _Posts repo() => _Posts(
    GfApiClient(
      dio: Dio(),
      tokenStorage: MemoryTokenStorage(),
      baseUrl: 'https://example.test',
    ),
  );
  testWidgets('share platform failure is handled without losing the page', (
    tester,
  ) async {
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
    await pump(
      tester,
      repo(),
      PostActions(
        post: post(),
        onChanged: () async {},
        onReply: () {},
        onReport: () {},
      ),
    );
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Share'));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(find.byType(PostActions), findsOneWidget);
    await tester.pump(const Duration(seconds: 4));
  });
  testWidgets(
    'reply likes and bookmarks call the API and refresh the server state',
    (tester) async {
      final repository = repo();
      var refreshes = 0;
      await pump(
        tester,
        repository,
        PostActions(
          post: post(),
          onChanged: () async {
            refreshes++;
          },
          onReply: () {},
          onReport: () {},
        ),
      );
      await tester.tap(find.byIcon(Icons.favorite_border));
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip('Bookmark'));
      await tester.pumpAndSettle();
      expect(repository.likes, [1]);
      expect(repository.bookmarks, [1]);
      expect(refreshes, 2);
    },
  );
  testWidgets(
    'failed reply edit retains text for retry and closes only on success',
    (tester) async {
      final repository = repo();
      var refreshes = 0;
      await pump(
        tester,
        repository,
        PostActions(
          post: post(),
          onChanged: () async {
            refreshes++;
          },
          onReply: () {},
          onReport: () {},
        ),
      );
      await tester.tap(find.byTooltip('More options'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Edit'));
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byKey(const Key('post-edit-content')),
        'Edited reply with **formatting**',
      );
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();
      expect(
        tester
            .widget<TextField>(find.byKey(const Key('post-edit-content')))
            .controller!
            .text,
        'Edited reply with **formatting**',
      );
      expect(refreshes, 0);
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();
      expect(repository.contents, [
        'Edited reply with **formatting**',
        'Edited reply with **formatting**',
      ]);
      expect(refreshes, 1);
      expect(find.byKey(const Key('post-edit-content')), findsNothing);
    },
  );
  testWidgets('removed replies expose neither editing nor engagement', (
    tester,
  ) async {
    await pump(
      tester,
      repo(),
      PostActions(
        post: post().copyWith(isAuthorDeleted: true),
        onChanged: () async {},
        onReply: () {},
        onReport: () {},
      ),
    );
    expect(find.byIcon(Icons.favorite_border), findsNothing);
    expect(find.byTooltip('Reply'), findsNothing);
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    expect(find.text('Edit'), findsNothing);
    expect(find.text('Delete'), findsNothing);
    expect(find.text('Revision history'), findsOneWidget);
  });
  testWidgets('history follows older cursor and represents masked snapshots', (
    tester,
  ) async {
    final repository = repo();
    await pump(tester, repository, const PostHistorySheet(postId: 42));
    await tester.tap(find.byType(ExpansionTile).first);
    await tester.pumpAndSettle();
    expect(find.text('This version is unavailable'), findsOneWidget);
    await tester.tap(find.text('Load more'));
    await tester.pumpAndSettle();
    expect(repository.cursors, [0, 2]);
    expect(find.byType(ExpansionTile), findsNWidgets(2));
    expect(find.text('Load more'), findsNothing);
  });
}
