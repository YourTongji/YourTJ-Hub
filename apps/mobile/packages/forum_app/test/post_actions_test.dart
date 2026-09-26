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
  testWidgets('reply controls wrap with uniform glyphs at enlarged text', (
    tester,
  ) async {
    final repository = repo();
    var replies = 0;
    var reports = 0;
    await pump(
      tester,
      repository,
      Center(
        child: SizedBox(
          width: 220,
          child: MediaQuery(
            data: const MediaQueryData(textScaler: TextScaler.linear(2)),
            child: PostActions(
              post: post().copyWith(isOwnPost: false, likeCount: 98765),
              onChanged: () async {},
              onReply: () => replies++,
              onReport: () => reports++,
            ),
          ),
        ),
      ),
    );
    expect(tester.takeException(), isNull);
    final actions = find.byType(PostActions);
    expect(
      find.descendant(of: actions, matching: find.byType(Wrap)),
      findsOneWidget,
    );
    final symbols = tester.widgetList<GfSymbol>(
      find.descendant(of: actions, matching: find.byType(GfSymbol)),
    );
    expect(symbols.map((symbol) => symbol.name), [
      'heart',
      'bookmark',
      'corner-down-left',
      'ellipsis',
      'flag',
    ]);
    expect(symbols.every((symbol) => symbol.size == 20), isTrue);
    final targets = <String, Rect>{};
    for (final tooltip in [
      'Like',
      'Bookmark',
      'Reply',
      'More options',
      'Report post',
    ]) {
      // Material 3 places an IconButton's Tooltip around its 40px surface,
      // inside the padded touch target. Measure the button, not that surface.
      final control = tooltip == 'Like'
          ? find.byTooltip(tooltip)
          : find.byWidgetPredicate(
              (widget) => widget is IconButton && widget.tooltip == tooltip,
            );
      expect(control, findsOneWidget);
      final target = tester.getRect(control);
      targets[tooltip] = target;
      expect(target.height, greaterThanOrEqualTo(44));
      expect(target.width, greaterThanOrEqualTo(44));
      expect(target.left, greaterThanOrEqualTo(tester.getRect(actions).left));
      expect(target.right, lessThanOrEqualTo(tester.getRect(actions).right));
    }
    // The outer margin must activate the action, not merely reserve layout.
    for (final label in ['Like', 'Bookmark', 'Reply', 'Report post']) {
      await tester.tapAt(targets[label]!.topLeft + const Offset(2, 2));
      await tester.pumpAndSettle();
    }
    expect(repository.likes, [1]);
    expect(repository.bookmarks, [1]);
    expect(replies, 1);
    expect(reports, 1);
    await tester.tapAt(targets['More options']!.topLeft + const Offset(2, 2));
    await tester.pumpAndSettle();
    expect(find.text('Revision history'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
  testWidgets(
    'reply return handoff preserves both successful interaction fields',
    (tester) async {
      final repository = repo();
      await pump(
        tester,
        repository,
        PostActions(
          post: post(),
          onChanged: () async {},
          onReply: () {},
          onReport: () {},
        ),
      );
      final container = ProviderScope.containerOf(
        tester.element(find.byType(PostActions)),
      );
      await tester.tap(find.byTooltip('Like'));
      await tester.pumpAndSettle();
      expect(container.read(postReturnStatesProvider)[42], (
        liked: true,
        bookmarked: false,
        likeCount: 4,
      ));
      await tester.tap(find.byTooltip('Bookmark'));
      await tester.pumpAndSettle();
      expect(container.read(postReturnStatesProvider)[42], (
        liked: true,
        bookmarked: true,
        likeCount: 4,
      ));
    },
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
      await tester.tap(find.byTooltip('Like'));
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
    expect(find.byTooltip('Like'), findsNothing);
    expect(find.byTooltip('Reply'), findsNothing);
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    expect(find.text('Edit'), findsNothing);
    expect(find.text('Delete'), findsNothing);
    expect(find.text('Revision history'), findsOneWidget);
  });

  testWidgets(
    'dirty reply sheet blocks outside and drag dismissal and confirms close',
    (tester) async {
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
      await tester.tap(find.text('Edit'));
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byKey(const Key('post-edit-content')),
        'Keep my draft',
      );
      await tester.tapAt(const Offset(10, 10));
      await tester.pumpAndSettle();
      expect(find.text('Keep my draft'), findsOneWidget);
      await tester.dragFrom(
        tester.getTopLeft(find.byType(BottomSheet)) + const Offset(80, 8),
        const Offset(0, 500),
      );
      await tester.pumpAndSettle();
      expect(find.text('Keep my draft'), findsOneWidget);
      await tester.tap(find.byTooltip('Close'));
      await tester.pumpAndSettle();
      final l10n = AppLocalizations.of(
        tester.element(find.byType(PostActions)),
      );
      expect(find.text(l10n.publishLeaveTitle), findsOneWidget);
      await tester.tap(find.text(l10n.commonCancel));
      await tester.pumpAndSettle();
      expect(find.text('Keep my draft'), findsOneWidget);
      await tester.tap(find.byTooltip('Close'));
      await tester.pumpAndSettle();
      await tester.tap(find.text(l10n.publishDiscard));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('post-edit-content')), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );
  testWidgets('history follows older cursor and represents masked snapshots', (
    tester,
  ) async {
    final repository = repo();
    await pump(tester, repository, const PostHistorySheet(postId: 42));
    await tester.tap(find.byType(ExpansionTile).first);
    await tester.pumpAndSettle();
    expect(find.text('This version is unavailable'), findsOneWidget);
    // Short histories fill the viewport automatically using the older cursor.
    expect(repository.cursors, [0, 2]);
    expect(find.byType(ExpansionTile), findsNWidgets(2));
    expect(find.text('Load more'), findsNothing);
  });
}
