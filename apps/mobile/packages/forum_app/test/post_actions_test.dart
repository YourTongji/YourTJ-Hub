import 'dart:async';
import 'dart:ui' show SemanticsAction;

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/topic/post_actions.dart';
import 'package:forum_app/src/pages/topic/post_history_sheet.dart';
import 'package:forum_app/src/pages/topic/post_editor.dart';
import 'package:forum_app/src/widgets/share/share_image_preview.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/private_notes.dart';
import 'package:forum_app/src/widgets/stickers/sticker_library_state.dart';
import 'package:forum_app/src/widgets/stickers/sticker_picker.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';
import 'fixtures/sticker_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Posts extends PostRepository {
  _Posts(super.client);
  final likes = <int>[];
  final bookmarks = <int>[];
  final deleted = <int>[];
  final contents = <String>[];
  final cursors = <int>[];
  bool failEdit = true;

  /// 让点赞请求挂起,用于断言忙碌态的图标颜色。
  Completer<void>? likeGate;

  @override
  Future<bool> deletePost({required int postId}) async {
    expect(postId, 42);
    deleted.add(postId);
    return true;
  }

  @override
  Future<bool> likePost({required int postId, required int action}) async {
    expect(postId, 42);
    likes.add(action);
    await likeGate?.future;
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

/// The editor's input lives in the shared reply composer.
final editorField = find.descendant(
  of: find.byKey(const Key('post-edit-composer')),
  matching: find.byType(TextField),
);

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

  Future<void> pump(
    WidgetTester tester,
    _Posts repo,
    Widget child, {
    List<Override> overrides = const [],
  }) async {
    final container = ProviderContainer(
      overrides: [postRepositoryProvider.overrideWithValue(repo), ...overrides],
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
  testWidgets('like semantics include its action and visible count', (
    tester,
  ) async {
    final semantics = tester.ensureSemantics();
    try {
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
      final data = tester
          .getSemantics(find.byType(TextButton).first)
          .getSemanticsData();
      expect(data.label, 'Like\n3');
      expect(data.hasAction(SemanticsAction.tap), isTrue);
    } finally {
      semantics.dispose();
    }
  });

  testWidgets(
    'own pending reply is editable while public actions stay hidden',
    (tester) async {
      await pump(
        tester,
        repo(),
        PostActions(
          post: post().copyWith(processStatus: 2, isHidden: true),
          onChanged: () async {},
          onReply: null,
          onReport: () {},
        ),
      );
      await tester.tap(find.byType(GfPopupMenuButton<String>));
      await tester.pumpAndSettle();
      expect(find.text('Edit'), findsOneWidget);
      expect(find.text('Share'), findsNothing);
      expect(find.text('Generate share image'), findsNothing);
      await tester.tap(find.text('Edit'));
      await tester.pumpAndSettle();
      expect(find.text('Original reply'), findsOneWidget);
    },
  );

  testWidgets('more action uses the same muted and busy icon colors', (
    tester,
  ) async {
    final pending = Completer<void>();
    await pump(
      tester,
      repo()..likeGate = pending,
      PostActions(
        post: post(),
        onChanged: () async {},
        onReply: () {},
        onReport: () {},
      ),
    );
    final more = find.byWidgetPredicate(
      (widget) => widget is GfSymbol && widget.name == 'ellipsis',
    );
    final colors = GfTheme.colorsOf(tester.element(more));
    expect(tester.widget<GfSymbol>(more).color, colors.iconMuted);
    await tester.tap(find.byTooltip('Like'));
    await tester.pump();
    expect(
      tester.widget<GfSymbol>(more).color,
      colors.iconMuted.withValues(alpha: .38),
    );
    pending.complete();
    await tester.pumpAndSettle();
  });

  testWidgets('share actions open for an available post', (tester) async {
    await pump(
      tester,
      repo(),
      PostActions(
        post: post().copyWith(isOwnPost: false, isAnonymous: true),
        topicTitle: 'A topic title',
        onChanged: () async {},
        onReply: null,
        onReport: () {},
      ),
    );
    await tester.tap(find.byType(GfPopupMenuButton<String>));
    await tester.pumpAndSettle();
    expect(find.text('Share'), findsOneWidget);
    expect(find.text('Generate share image'), findsOneWidget);
    await tester.tap(find.text('Generate share image'));
    await tester.pumpAndSettle();
    expect(find.byType(ShareImageCard), findsOneWidget);
    expect(find.text('A topic title'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(seconds: 1));
  });

  for (final (state, removedPost) in <(String, PostPayload)>[
    ('hidden', post().copyWith(isHidden: true)),
    ('author deleted', post().copyWith(isAuthorDeleted: true)),
    ('moderator removed', post().copyWith(isModeratorRemoved: true)),
  ]) {
    testWidgets('share actions are absent for $state posts', (tester) async {
      await pump(
        tester,
        repo(),
        PostActions(
          post: removedPost.copyWith(isOwnPost: false),
          onChanged: () async {},
          onReply: null,
          onReport: () {},
        ),
      );
      await tester.tap(find.byType(GfPopupMenuButton<String>));
      await tester.pumpAndSettle();
      expect(find.text('Share'), findsNothing);
      expect(find.text('Generate share image'), findsNothing);
      expect(find.text('Report post'), findsNothing);
    });
  }

  testWidgets('own replies offer no report entry', (tester) async {
    await pump(
      tester,
      repo(),
      PostActions(
        post: post(),
        onChanged: () async {},
        onReply: null,
        onReport: () => fail('own reply reported'),
      ),
    );
    expect(find.byTooltip('Report post'), findsNothing);
    await tester.tap(find.byType(GfPopupMenuButton<String>));
    await tester.pumpAndSettle();
    expect(find.text('Report post'), findsNothing);
  });

  testWidgets(
    'share actions are absent when the containing topic is unavailable',
    (tester) async {
      await pump(
        tester,
        repo(),
        PostActions(
          post: post().copyWith(isOwnPost: false),
          topicAvailable: false,
          onChanged: () async {},
          onReply: null,
          onReport: () {},
        ),
      );
      await tester.tap(find.byType(GfPopupMenuButton<String>));
      await tester.pumpAndSettle();
      expect(find.text('Share'), findsNothing);
      expect(find.text('Generate share image'), findsNothing);
    },
  );

  testWidgets('share card uses public author data, never private notes', (
    tester,
  ) async {
    final author = const UserBriefPayload(
      id: 99,
      username: 'login-name',
      nickname: 'Public name',
      avatarUrl: '/file/img/avatar.png',
    );
    final widget = PrivateNotesScope(
      ownerId: 1,
      notes: const <int, PrivateNotePayload>{
        99: PrivateNotePayload(
          targetUserId: 99,
          username: 'login-name',
          note: 'My note',
        ),
      },
      child: PostActions(
        post: post().copyWith(
          isOwnPost: false,
          isAnonymous: false,
          author: author,
        ),
        onChanged: () async {},
        onReply: null,
        onReport: () {},
      ),
    );
    await pump(tester, repo(), widget);
    await tester.tap(find.byType(GfPopupMenuButton<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Generate share image'));
    await tester.pumpAndSettle();

    // Share images are exported artifacts. PrivateNotesScope must not change
    // their author label or leak the viewer's local contact notes.
    expect(find.text('Public name'), findsOneWidget);
    expect(find.text('My note'), findsNothing);
    expect(find.text('login-name'), findsNothing);
    final avatar = tester.widget<ShareImageNetworkImage>(
      find.byType(ShareImageNetworkImage).first,
    );
    expect(avatar.url, contains('/file/img/avatar.png'));
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(seconds: 1));
  });

  testWidgets('anonymous share card does not reveal author DTO fields', (
    tester,
  ) async {
    await pump(
      tester,
      repo(),
      PostActions(
        post: post().copyWith(
          isOwnPost: false,
          isAnonymous: true,
          author: const UserBriefPayload(
            id: 99,
            username: 'private-login',
            nickname: 'Private real name',
            avatarUrl: '/file/img/private-avatar.png',
          ),
        ),
        onChanged: () async {},
        onReply: null,
        onReport: () {},
      ),
    );
    await tester.tap(find.byType(GfPopupMenuButton<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Generate share image'));
    await tester.pumpAndSettle();

    final anonymousLabel = AppLocalizations.of(
      tester.element(find.byType(PostActions)),
    ).courseCopyAuthorAnonymousLabel;
    expect(find.text(anonymousLabel), findsOneWidget);
    expect(find.text('private-login'), findsNothing);
    expect(find.text('Private real name'), findsNothing);
    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is ShareImageNetworkImage &&
            widget.url.contains('private-avatar.png'),
      ),
      findsNothing,
    );
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(seconds: 1));
  });

  for (final size in [(320.0, 1.0), (390.0, 1.0), (220.0, 2.0)]) {
    testWidgets('reply actions align from the left at ${size.$1}/${size.$2}', (
      tester,
    ) async {
      await pump(
        tester,
        repo(),
        Center(
          child: SizedBox(
            width: size.$1,
            child: MediaQuery(
              data: MediaQueryData(textScaler: TextScaler.linear(size.$2)),
              child: PostActions(
                post: post().copyWith(isOwnPost: false, likeCount: 3),
                onChanged: () async {},
                onReply: () {},
                onReport: () {},
              ),
            ),
          ),
        ),
      );
      final actions = find.byType(PostActions);
      final bounds = tester.getRect(actions);
      final like = tester.getRect(find.byTooltip('Like'));
      expect(like.left, closeTo(bounds.left, .01));
      final glyphs = find.descendant(
        of: actions,
        matching: find.byType(GfSymbol),
      );
      expect(glyphs, findsNWidgets(4));
      final centers = [
        for (var index = 0; index < 4; index++)
          tester.getCenter(glyphs.at(index)),
      ];
      expect(centers.first.dx - bounds.left, 22);
      if (size.$2 == 1) {
        expect(like.height, 44);
        for (final center in centers) {
          expect(center.dy, closeTo(centers.first.dy, .01));
        }
      } else {
        final wrapped = find.descendant(
          of: actions,
          matching: find.byType(Wrap),
        );
        expect(tester.widget<Wrap>(wrapped).alignment, WrapAlignment.start);
      }
      final count = find.descendant(of: actions, matching: find.text('3'));
      expect(tester.getCenter(count).dy, closeTo(centers.first.dy, .01));
      expect(
        tester.widget<Text>(count).style?.color,
        GfTheme.colorsOf(tester.element(actions)).iconMuted,
      );
      expect(tester.takeException(), isNull);
    });
  }
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
    ]);
    expect(symbols.every((symbol) => symbol.size == 20), isTrue);
    final targets = <String, Rect>{};
    for (final tooltip in ['Like', 'Bookmark', 'Reply', 'More options']) {
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
    for (final label in ['Like', 'Bookmark', 'Reply']) {
      await tester.tapAt(targets[label]!.topLeft + const Offset(2, 2));
      await tester.pumpAndSettle();
    }
    expect(repository.likes, [1]);
    expect(repository.bookmarks, [1]);
    expect(replies, 1);
    await tester.tapAt(targets['More options']!.topLeft + const Offset(2, 2));
    await tester.pumpAndSettle();
    expect(find.text('Revision history'), findsOneWidget);
    // Reporting a reply is a floor action, so it lives in the floor menu.
    await tester.tap(find.text('Report post'));
    await tester.pumpAndSettle();
    expect(reports, 1);
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
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(seconds: 1));
  });
  testWidgets(
    'reply likes and bookmarks patch the floor in place without reloading',
    (tester) async {
      final repository = repo();
      var refreshes = 0;
      final patches = <PostPayload>[];
      await pump(
        tester,
        repository,
        PostActions(
          post: post(),
          onChanged: () async {
            refreshes++;
          },
          onPostPatched: patches.add,
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
      // 点赞/收藏只回传本楼,不触发整窗重载(issue #1316)。
      expect(refreshes, 0);
      expect(patches, hasLength(2));
      expect(patches.first.isLiked, isTrue);
      expect(patches.first.likeCount, 4);
      expect(patches.last.isBookmarked, isTrue);
    },
  );
  testWidgets('unliking keeps the count at zero and patches the floor', (
    tester,
  ) async {
    final repository = repo();
    final patches = <PostPayload>[];
    await pump(
      tester,
      repository,
      PostActions(
        // 计数为 0 却已点赞:取消赞时必须停在 0,不能写进负数(provider 与
        // payload 都要守住下界)。
        post: post().copyWith(isLiked: true, likeCount: 0),
        onChanged: () async {},
        onPostPatched: patches.add,
        onReply: () {},
        onReport: () {},
      ),
    );
    final container = ProviderScope.containerOf(
      tester.element(find.byType(PostActions)),
    );
    await tester.tap(find.byTooltip('Like'));
    await tester.pumpAndSettle();

    expect(repository.likes, [2]);
    expect(patches, hasLength(1));
    expect(patches.single.isLiked, isFalse);
    expect(patches.single.likeCount, 0);
    expect(container.read(postReturnStatesProvider)[42], (
      liked: false,
      bookmarked: false,
      likeCount: 0,
    ));
  });
  testWidgets('delete still reloads the comment window', (tester) async {
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
    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();

    expect(repository.deleted, [42]);
    expect(refreshes, 1);
  });
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
      await tester.enterText(editorField, 'Edited reply with **formatting**');
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(editorField).controller!.text,
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
      expect(editorField, findsNothing);
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

  testWidgets('reply editor docks the full reply composer above the keyboard', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 844);
    addTearDown(tester.view.reset);
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: MemoryTokenStorage(),
      baseUrl: 'https://example.test',
    );
    final library = StickerLibrary(ComposerStickerRepository(client));
    addTearDown(library.dispose);
    await pump(
      tester,
      repo(),
      PostActions(
        post: post(),
        onChanged: () async {},
        onReply: () {},
        onReport: () {},
      ),
      overrides: [
        stickerLibraryProvider.overrideWithValue(library),
        stickerCollectionProvider.overrideWith(
          (ref) =>
              StickerCollection(ComposerStickerRepository(client), library),
        ),
      ],
    );
    tester.view.viewInsets = const FakeViewPadding(bottom: 300);
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Edit'));
    await tester.pumpAndSettle();
    final composerFinder = find.byKey(const Key('post-edit-composer'));
    GfPostComposer composer() => tester.widget(composerFinder);
    // The same tools as replying in a topic.
    for (final tip in [
      composer().imageTooltip!,
      composer().stickerTooltip!,
      composer().hideKeyboardLabel!,
    ]) {
      expect(find.byTooltip(tip), findsOneWidget, reason: tip);
    }
    expect(tester.widget<TextField>(editorField).focusNode!.hasFocus, isTrue);
    // Save rests on the keyboard edge and the input takes the rest.
    final save = find.widgetWithText(GfButton, 'Save');
    final dock = tester.getBottomLeft(save).dy;
    expect(dock, lessThanOrEqualTo(844 - 300));
    expect(dock, greaterThan(844 - 300 - 40));
    expect(tester.getSize(editorField).height, greaterThan(240));

    Future<List<double>> keyboard(double from, double to) async {
      final trace = <double>[];
      for (var frame = 1; frame <= 10; frame++) {
        tester.view.viewInsets = FakeViewPadding(
          bottom: from + (to - from) * frame / 10,
        );
        await tester.pump(const Duration(milliseconds: 16));
        trace.add(tester.getBottomLeft(save).dy);
      }
      await tester.pump(const Duration(milliseconds: 400));
      await tester.pumpAndSettle();
      trace.add(tester.getBottomLeft(save).dy);
      return trace;
    }

    // Keyboard and stickers trade places without moving the action row.
    await tester.tap(find.byTooltip(composer().stickerTooltip!));
    await tester.pump();
    for (final y in await keyboard(300, 0)) {
      expect(y, closeTo(dock, 3), reason: 'keyboard → stickers');
    }
    expect(find.byType(StickerPicker), findsOneWidget);
    await tester.tap(find.byTooltip(composer().stickerTooltip!));
    await tester.pump();
    for (final y in await keyboard(0, 300)) {
      expect(y, closeTo(dock, 3), reason: 'stickers → keyboard');
    }
    expect(find.byType(StickerPicker), findsNothing);

    // A failed save reports beside Save and keeps the text.
    await tester.enterText(editorField, 'Edited once');
    await tester.tap(save);
    await tester.pumpAndSettle();
    final status = find.byWidget(composer().status!);
    expect(tester.getCenter(status).dy, closeTo(tester.getCenter(save).dy, 1));
    expect(
      tester.getTopRight(status).dx,
      lessThanOrEqualTo(tester.getTopLeft(save).dx),
    );
    expect(find.text('Edited once'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('back with the sticker panel open folds it before closing', (
    tester,
  ) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(390, 844);
    addTearDown(tester.view.reset);
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: MemoryTokenStorage(),
      baseUrl: 'https://example.test',
    );
    final library = StickerLibrary(ComposerStickerRepository(client));
    addTearDown(library.dispose);
    await pump(
      tester,
      repo(),
      PostActions(
        post: post(),
        onChanged: () async {},
        onReply: () {},
        onReport: () {},
      ),
      overrides: [
        stickerLibraryProvider.overrideWithValue(library),
        stickerCollectionProvider.overrideWith(
          (ref) =>
              StickerCollection(ComposerStickerRepository(client), library),
        ),
      ],
    );
    await tester.tap(find.byTooltip('More options'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Edit'));
    await tester.pumpAndSettle();
    final GfPostComposer composer = tester.widget(
      find.byKey(const Key('post-edit-composer')),
    );
    await tester.tap(find.byTooltip(composer.stickerTooltip!));
    await tester.pumpAndSettle();
    expect(find.byType(StickerPicker), findsOneWidget);
    final l10n = AppLocalizations.of(tester.element(find.byType(PostEditor)));
    // First back only folds the panel; the clean editor stays open.
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(find.byType(StickerPicker), findsNothing);
    expect(find.byType(PostEditor), findsOneWidget);
    expect(find.text(l10n.publishLeaveTitle), findsNothing);
    // Second back closes the unchanged editor without asking.
    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();
    expect(find.byType(PostEditor), findsNothing);
    expect(tester.takeException(), isNull);
  });

  for (final width in [390.0, 1120.0]) {
    testWidgets(
      'dirty reply editor opens full screen and confirms every close at $width',
      (tester) async {
        tester.view.devicePixelRatio = 1;
        tester.view.physicalSize = Size(width, 844);
        addTearDown(tester.view.reset);
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
        final editor = find.byType(PostEditor);
        // Full screen at every width, like editing a topic.
        final route = ModalRoute.of(tester.element(editor));
        expect(route, isA<MaterialPageRoute<bool>>());
        expect((route! as MaterialPageRoute<bool>).fullscreenDialog, isTrue);
        await tester.enterText(editorField, 'Keep my draft');
        // System back is held while the draft is dirty and asks first.
        final l10n = AppLocalizations.of(tester.element(editor));
        await tester.binding.handlePopRoute();
        await tester.pumpAndSettle();
        expect(find.text(l10n.publishLeaveTitle), findsOneWidget);
        await tester.tap(find.text(l10n.commonCancel));
        await tester.pumpAndSettle();
        expect(find.text('Keep my draft'), findsOneWidget);
        await tester.tap(find.byTooltip('Close'));
        await tester.pumpAndSettle();
        expect(find.text(l10n.publishLeaveTitle), findsOneWidget);
        await tester.tap(find.text(l10n.commonCancel));
        await tester.pumpAndSettle();
        expect(find.text('Keep my draft'), findsOneWidget);
        await tester.tap(find.byTooltip('Close'));
        await tester.pumpAndSettle();
        await tester.tap(find.text(l10n.publishDiscard));
        await tester.pumpAndSettle();
        expect(editorField, findsNothing);
        expect(tester.takeException(), isNull);
      },
    );
  }
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
