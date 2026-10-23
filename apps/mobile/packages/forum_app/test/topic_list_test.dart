import 'dart:ui' as ui;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/topic_list.dart';
import 'package:forum_app/src/widgets/user_profile_preview.dart';
import 'fixtures/page_fixtures.dart';
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Users extends UserRepository {
  _Users()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: MemoryTokenStorage(),
          baseUrl: 'https://example.test',
        ),
      );
  final cards = <int>[];
  @override
  Future<UserCardPayload> getUserCard(int userId) async {
    cards.add(userId);
    throw StateError('offline');
  }
}

void main() {
  testWidgets('author, image and body have distinct navigation targets', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final topic = home.topics.first.copyWith(
      images: ['https://example.test/one.png', 'https://example.test/two.png'],
    );
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => Scaffold(
            body: GfTopicList(
              loading: false,
              topics: [topic],
              feedMode: GfTopicFeedMode.card,
              hasMore: false,
              onLoadMore: () {},
            ),
          ),
        ),
        GoRoute(
          path: '/u/:id',
          builder: (_, state) =>
              Scaffold(body: Text('profile ${state.pathParameters['id']}')),
        ),
        GoRoute(
          path: '/p/:id',
          builder: (_, state) =>
              Scaffold(body: Text('topic ${state.pathParameters['id']}')),
        ),
      ],
    );
    addTearDown(router.dispose);
    final users = _Users();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          userRepositoryProvider.overrideWithValue(users),
          currentUserProvider.overrideWith((_) async => null),
        ],
        child: MaterialApp.router(
          routerConfig: router,
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('alice'));
    await tester.pumpAndSettle();
    expect(find.text('profile 1'), findsOneWidget);
    router.pop();
    await tester.pumpAndSettle();
    // Avatar opens the profile preview card in place, matching topic detail.
    await tester.tap(find.byType(GfAvatar).first);
    await tester.pumpAndSettle();
    expect(find.byType(UserProfilePreview), findsOneWidget);
    expect(users.cards, [1]);
    expect(router.state.uri.path, '/');
    Navigator.of(tester.element(find.byType(UserProfilePreview))).pop();
    await tester.pumpAndSettle();
    expect(find.byType(UserProfilePreview), findsNothing);
    await tester.tap(
      find.byWidgetPredicate(
        (w) =>
            w is Image &&
            w.frameBuilder != null &&
            w.image is ResizeImage &&
            (w.image as ResizeImage).imageProvider is NetworkImage &&
            ((w.image as ResizeImage).imageProvider as NetworkImage).url
                .endsWith('two.png'),
      ),
    );
    // The network image loader keeps animating in the test HTTP environment.
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    final viewer = tester.widget<GfImageViewer>(find.byType(GfImageViewer));
    expect(viewer.initialIndex, 1);
    expect(viewer.images, topic.images);
    expect(viewer.onSaveImage, isNotNull);
    expect(viewer.saveImageLabel, '保存图片');
    expect(viewer.onShareImage, isNotNull);
    router.pop();
    await tester.pumpAndSettle();
    await tester.tap(find.text(topic.title));
    await tester.pumpAndSettle();
    expect(find.text('topic 100'), findsOneWidget);
  });

  testWidgets('topic cards expose like and bookmark shortcuts', (tester) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    var topic = home.topics.first.copyWith(liked: false, bookmarked: false);
    bool? likeTarget;
    bool? bookmarkTarget;

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: StatefulBuilder(
              builder: (context, setState) => GfTopicList(
                loading: false,
                topics: [topic],
                feedMode: GfTopicFeedMode.card,
                hasMore: false,
                onLoadMore: () {},
                onLikeTopic: (_, target) async {
                  likeTarget = target;
                  setState(() => topic = topic.copyWith(liked: target));
                  return true;
                },
                onBookmarkTopic: (_, target) async {
                  bookmarkTarget = target;
                  setState(() => topic = topic.copyWith(bookmarked: target));
                  return true;
                },
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byTooltip('点赞'), findsOneWidget);
    expect(find.byTooltip('收藏'), findsOneWidget);
    expect(
      find.byWidgetPredicate((w) => w is GfSymbol && w.name == 'heart'),
      findsOneWidget,
    );
    expect(
      find.byWidgetPredicate((w) => w is GfSymbol && w.name == 'bookmark'),
      findsOneWidget,
    );

    await tester.tap(find.byTooltip('点赞'));
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('收藏'));
    await tester.pumpAndSettle();

    expect(likeTarget, isTrue);
    expect(bookmarkTarget, isTrue);
    expect(
      find.byWidgetPredicate((w) => w is GfSymbol && w.name == 'heart-filled'),
      findsOneWidget,
    );
    expect(
      find.byWidgetPredicate(
        (w) => w is GfSymbol && w.name == 'bookmark-filled',
      ),
      findsOneWidget,
    );

    await tester.tap(find.byTooltip('点赞'));
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('取消收藏'));
    await tester.pumpAndSettle();

    expect(likeTarget, isFalse);
    expect(bookmarkTarget, isFalse);
    expect(
      find.byWidgetPredicate((w) => w is GfSymbol && w.name == 'heart'),
      findsOneWidget,
    );
    expect(
      find.byWidgetPredicate((w) => w is GfSymbol && w.name == 'bookmark'),
      findsOneWidget,
    );
  });

  testWidgets('failed topic actions keep the unselected state', (tester) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    var topic = home.topics.first.copyWith(liked: false, bookmarked: false);
    var calls = 0;

    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: StatefulBuilder(
              builder: (context, setState) => GfTopicList(
                loading: false,
                topics: [topic],
                feedMode: GfTopicFeedMode.card,
                hasMore: false,
                onLoadMore: () {},
                onLikeTopic: (_, target) async {
                  calls++;
                  return false;
                },
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('点赞'));
    await tester.pumpAndSettle();

    expect(calls, 1);
    expect(
      find.byWidgetPredicate((w) => w is GfSymbol && w.name == 'heart'),
      findsOneWidget,
    );
    expect(
      find.byWidgetPredicate((w) => w is GfSymbol && w.name == 'heart-filled'),
      findsNothing,
    );
  });

  testWidgets('list mode uses the same pinned line without touching rows', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final topic = home.topics.first;
    var loads = 0;
    final controller = ScrollController();
    addTearDown(controller.dispose);
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          theme: gfThemeData(Brightness.light),
          home: Scaffold(
            body: GfTopicList(
              collapsePinned: true,
              loading: false,
              controller: controller,
              header: const Text('Announcement'),
              topics: [
                topic.copyWith(id: 1, title: 'Pinned one', pinWeight: 1),
                topic.copyWith(id: 2, title: 'Pinned two', pinWeight: 2),
                topic.copyWith(id: 3, title: 'Regular topic', pinWeight: 0),
                for (var i = 4; i <= 30; i++)
                  topic.copyWith(id: i, title: 'Topic $i', pinWeight: 0),
              ],
              hasMore: false,
              onLoadMore: () => loads++,
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final line = find.byKey(const ValueKey('pinned-strip-line'));
    expect(tester.getSize(line).height, lessThanOrEqualTo(40));
    expect(find.text('Pinned one'), findsOneWidget);
    expect(find.text('Pinned two'), findsNothing);
    expect(find.text('Regular topic'), findsOneWidget);
    expect(find.text('Announcement'), findsOneWidget);
    expect(find.text('2'), findsOneWidget);
    final collapsedHeight = tester.getTopLeft(find.text('Regular topic')).dy;
    await tester.tap(line);
    await tester.pumpAndSettle();
    expect(find.text('Pinned one'), findsOneWidget);
    expect(find.text('Pinned two'), findsOneWidget);
    // Pinned titles stay in the strip; no pinned rows join the list.
    expect(
      find.ancestor(
        of: find.text('Pinned two'),
        matching: find.byType(GfTopicRow),
      ),
      findsNothing,
    );
    expect(
      tester.getTopLeft(find.text('Regular topic')).dy,
      greaterThan(collapsedHeight),
    );
    controller.jumpTo(1500);
    await tester.pumpAndSettle();
    controller.jumpTo(0);
    await tester.pumpAndSettle();
    expect(find.text('Pinned two'), findsOneWidget);
    await tester.tap(line);
    await tester.pumpAndSettle();
    expect(find.text('Pinned two'), findsNothing);
    expect(loads, 0);
  });

  testWidgets('ordered streams retain pinned rows in place', (tester) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final topic = home.topics.first;
    for (final mode in GfTopicFeedMode.values) {
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: GfTopicList(
                feedMode: mode,
                loading: false,
                topics: [
                  topic.copyWith(id: 1, title: 'First regular', pinWeight: 0),
                  topic.copyWith(id: 2, title: 'Then pinned', pinWeight: 1),
                ],
                hasMore: false,
                onLoadMore: () {},
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Then pinned'), findsOneWidget);
      expect(find.text('置顶话题（1）'), findsNothing);
      expect(
        tester.getTopLeft(find.text('First regular')).dy,
        lessThan(tester.getTopLeft(find.text('Then pinned')).dy),
      );
    }
  });

  Future<String? Function()> pumpCardFeed(
    WidgetTester tester,
    List<TopicPayload> topics,
  ) async {
    String? opened;
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => Scaffold(
            body: GfTopicList(
              collapsePinned: true,
              feedMode: GfTopicFeedMode.card,
              loading: false,
              topics: topics,
              hasMore: false,
              onLoadMore: () {},
            ),
          ),
        ),
        GoRoute(
          path: '/p/:id',
          builder: (_, state) {
            opened = state.pathParameters['id'];
            return const Scaffold(body: Text('topic page'));
          },
        ),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(
          routerConfig: router,
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          theme: gfThemeData(Brightness.light),
        ),
      ),
    );
    await tester.pumpAndSettle();
    return () => opened;
  }

  testWidgets(
    'empty card uses localized title and short thought appears once',
    (tester) async {
      final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
      await pumpCardFeed(tester, [
        home.topics.first.copyWith(title: '', description: '', pinWeight: 0),
      ]);
      expect(find.text('无标题'), findsOneWidget);
      expect(find.text('(｀・ω・´)'), findsNothing);
      await pumpCardFeed(tester, [
        home.topics.first.copyWith(title: '', description: '失眠咯', pinWeight: 0),
      ]);
      expect(find.text('失眠咯'), findsOneWidget);
      expect(find.text('无标题'), findsNothing);
    },
  );

  testWidgets('media-only cards keep images and localized fallback', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final topic = home.topics.first.copyWith(
      title: ' ',
      description: ' ',
      pinWeight: 0,
      images: ['https://example.test/photo.png'],
    );
    for (final entry in {
      'zh': '无标题',
      'en': 'Untitled',
      'ja': '無題',
      'de': 'Ohne Titel',
    }.entries) {
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            locale: Locale(entry.key),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: GfTopicList(
                loading: false,
                topics: [topic],
                feedMode: GfTopicFeedMode.card,
                hasMore: false,
                onLoadMore: () {},
              ),
            ),
          ),
        ),
      );
      await tester.pump();
      final card = tester.widget<GfTopicCard>(find.byType(GfTopicCard));
      expect(card.title, entry.value);
      expect(card.description, isEmpty);
      expect(card.imageUrls, ['https://example.test/photo.png']);
    }
  });

  testWidgets('card mode folds pins into one announcement-style line', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final topic = home.topics.first;
    final opened = await pumpCardFeed(tester, [
      for (var i = 1; i <= 4; i++)
        topic.copyWith(id: i, title: 'Pinned $i', pinWeight: i),
      topic.copyWith(id: 9, title: 'Regular card', pinWeight: 0),
    ]);

    // Collapsed: pin, badge, first title and the count on a single line.
    final line = find.byKey(const ValueKey('pinned-strip-line'));
    expect(tester.getSize(line).height, lessThanOrEqualTo(40));
    expect(find.text('置顶'), findsOneWidget);
    expect(find.text('Pinned 1'), findsOneWidget);
    expect(find.text('Pinned 2'), findsNothing);
    expect(find.text('4'), findsOneWidget);
    expect(find.byType(GfTopicCard), findsOneWidget);

    await tester.tap(line);
    await tester.pumpAndSettle();
    for (var i = 1; i <= 4; i++) {
      expect(find.text('Pinned $i'), findsOneWidget);
    }
    expect(find.text('收起'), findsOneWidget);
    // Each unfolded title is led by its author's avatar under the pin tile.
    final row = find.byKey(const ValueKey('pinned-strip-2'));
    expect(
      find.descendant(of: row, matching: find.byType(GfAvatar)),
      findsOneWidget,
    );
    expect(tester.getSize(row).height, lessThanOrEqualTo(40));
    expect(find.byType(GfTopicCard), findsOneWidget);

    await tester.tap(find.text('Pinned 2'));
    await tester.pumpAndSettle();
    expect(opened(), '2');
  });

  testWidgets('a single pin opens straight from its line', (tester) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final topic = home.topics.first;
    final opened = await pumpCardFeed(tester, [
      topic.copyWith(id: 5, title: 'Only pin', pinWeight: 1),
      topic.copyWith(id: 9, title: 'Regular card', pinWeight: 0),
    ]);
    expect(find.text('1'), findsNothing);
    await tester.tap(find.text('Only pin'));
    await tester.pumpAndSettle();
    expect(opened(), '5');
  });

  testWidgets('pinned line fits narrow windows with large text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 640);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final topic = home.topics.first;
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          locale: const Locale('de'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: MediaQuery(
            data: const MediaQueryData(
              size: Size(320, 640),
              textScaler: TextScaler.linear(2),
            ),
            child: Scaffold(
              body: GfTopicList(
                collapsePinned: true,
                feedMode: GfTopicFeedMode.card,
                loading: false,
                topics: [
                  for (var i = 1; i <= 5; i++)
                    topic.copyWith(
                      id: i,
                      title: 'Ein sehr langer angehefteter Beitragstitel $i',
                      pinWeight: i,
                    ),
                ],
                hasMore: false,
                onLoadMore: () {},
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    // The badge yields its width to the title; the pin glyph remains.
    expect(find.text('Angeheftet'), findsNothing);
    final semantics = tester.ensureSemantics();
    expect(find.bySemanticsLabel(RegExp('^Angeheftet')), findsOneWidget);
    semantics.dispose();
    await tester.tap(find.byKey(const ValueKey('pinned-strip-line')));
    await tester.pumpAndSettle();
    expect(find.text('Einklappen'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('pinned-only feeds and all locales keep the line accessible', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final semanticsHandle = tester.ensureSemantics();

    final topic = home.topics.first;
    tester.view.physicalSize = const Size(320, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    for (final locale in AppLocalizations.supportedLocales) {
      final l10n = await AppLocalizations.delegate.load(locale);
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            key: ValueKey(locale),
            locale: locale,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: MediaQuery(
              data: const MediaQueryData(textScaler: TextScaler.linear(2)),
              child: Scaffold(
                body: GfTopicList(
                  collapsePinned: true,
                  loading: false,
                  topics: [
                    topic.copyWith(id: 1, title: 'Pinned one', pinWeight: 1),
                    topic.copyWith(id: 2, title: 'Pinned two', pinWeight: 2),
                  ],
                  hasMore: false,
                  onLoadMore: () {},
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final line = find.byKey(const ValueKey('pinned-strip-line'));
      final label = find.bySemanticsLabel(
        RegExp(RegExp.escape(l10n.homePinnedTopics(2))),
      );
      expect(
        tester
            .getSemantics(label)
            .getSemanticsData()
            .flagsCollection
            .isExpanded,
        ui.Tristate.isFalse,
      );
      await tester.tap(line);
      await tester.pumpAndSettle();
      expect(
        tester
            .getSemantics(label)
            .getSemanticsData()
            .flagsCollection
            .isExpanded,
        ui.Tristate.isTrue,
      );
      expect(find.text('Pinned two'), findsOneWidget);
      expect(find.text(l10n.announcementCollapseAction), findsOneWidget);
      expect(tester.takeException(), isNull);
    }
    semanticsHandle.dispose();
  });

  // issue #895：无标题瞬间的列表行要与 Web TopicRow/SSR 一致，始终有可识别的行标题。
  Future<void> pumpList(WidgetTester tester, TopicPayload topic) async {
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: GfTopicList(
              loading: false,
              topics: [topic],
              hasMore: false,
              onLoadMore: () {},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('list identifies question, moment and article types', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    for (final entry in {1: '提问', 2: '瞬间', 3: '文章'}.entries) {
      await pumpList(
        tester,
        home.topics.first.copyWith(contentType: entry.key),
      );
      expect(find.text(entry.value), findsOneWidget);
    }
  });

  testWidgets('untitled moment uses its excerpt as the row title', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final topic = home.topics.first.copyWith(
      title: '',
      description: '只有正文的瞬间摘要',
      contentType: 2,
    );
    await pumpList(tester, topic);

    // 摘要作为行标题（15px）出现一次，而不是作为 13px 的摘要行重复出现。
    expect(
      find.byWidgetPredicate(
        (Widget w) =>
            w is Text && w.data == '只有正文的瞬间摘要' && w.style?.fontSize == 15,
      ),
      findsOneWidget,
    );
    expect(
      find.byWidgetPredicate(
        (Widget w) =>
            w is Text && w.data == '只有正文的瞬间摘要' && w.style?.fontSize == 13,
      ),
      findsNothing,
    );
  });

  testWidgets(
    'untitled moment without excerpt still has an identifying title',
    (tester) async {
      final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
      final topic = home.topics.first.copyWith(
        title: '',
        description: '',
        contentType: 2,
      );
      await pumpList(tester, topic);

      expect(find.text('无标题'), findsOneWidget);
    },
  );

  testWidgets('every For You reason, recent included, stamps its seal', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final glyphs = {
      'following': 'user-round-check',
      'category': 'folder',
      'newreply': 'message-circle',
      'recent': 'clock',
    };
    Finder seal(String glyph) => find.byWidgetPredicate(
      (widget) =>
          widget is GfSymbol && widget.name == glyph && widget.size == 16,
    );
    Future<void> pump(String? reason) async {
      await tester.pumpWidget(
        ProviderScope(
          child: MaterialApp(
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: GfTopicList(
                loading: false,
                topics: [home.topics.first.copyWith(feedReason: reason)],
                feedMode: GfTopicFeedMode.card,
                hasMore: false,
                onLoadMore: () {},
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
    }

    for (final entry in glyphs.entries) {
      await pump(entry.key);
      expect(seal(entry.value), findsOneWidget, reason: entry.key);
    }
    // Feeds without a server reason (latest, categories) stay unstamped.
    await pump(null);
    for (final glyph in glyphs.values) {
      expect(seal(glyph), findsNothing, reason: glyph);
    }
  });

  testWidgets('list rows wire image avatar frames like the web stack', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final topic = home.topics.first.copyWith(
      participants: [
        UserBriefPayload(
          id: 7,
          username: 'bob',
          avatarUrl: '/static/pic/7.webp',
          decoration: const UserDecoration(
            avatarFrame: DecorationStyle(
              code: 'art',
              name: 'Art',
              kind: 'avatarFrame',
              assetUrl: '/static/decorations/art.webp',
              assetScale: 1.4,
              lightColors: ['#123456', '#654321'],
              darkColors: ['#abcdef', '#fedcba'],
            ),
          ),
        ),
      ],
    );
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: GfTopicList(
              loading: false,
              topics: [topic],
              feedMode: GfTopicFeedMode.list,
              hasMore: false,
              onLoadMore: () {},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // Web list rows render AvatarStack size="sm" (TopicRow.vue:222, h-6/w-6)
    // and pass the participant's whole avatarFrame through, so the app row
    // must hand over the artwork and scale without the colour ring.
    final avatar = tester.widget<GfAvatar>(find.byType(GfAvatar));
    expect(avatar.size, 24);
    expect(avatar.frameScale, 1.4);
    expect(avatar.frameAssetUrl, isNotEmpty);
    expect(avatar.frameColors, isEmpty);
  });

  testWidgets('anonymous authors keep their existing card targets', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final persona = home.topics.first.copyWith(
      author: home.topics.first.author.copyWith(publicUid: 'p-7'),
    );
    final legacy = home.topics.first.copyWith(
      id: 101,
      title: 'legacy anonymous',
      author: const UserBriefPayload(id: 0, username: 'ghost', avatarUrl: ''),
    );
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (_, _) => Scaffold(
            body: GfTopicList(
              loading: false,
              topics: [persona, legacy],
              feedMode: GfTopicFeedMode.card,
              hasMore: false,
              onLoadMore: () {},
            ),
          ),
        ),
        GoRoute(
          path: '/a/:uid',
          builder: (_, state) =>
              Scaffold(body: Text('persona ${state.pathParameters['uid']}')),
        ),
        GoRoute(
          path: '/p/:id',
          builder: (_, state) =>
              Scaffold(body: Text('topic ${state.pathParameters['id']}')),
        ),
      ],
    );
    addTearDown(router.dispose);
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp.router(
          routerConfig: router,
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
        ),
      ),
    );
    await tester.pumpAndSettle();
    final cards = find.byType(GfTopicCard);
    // A persona's avatar and name both open its independent profile.
    await tester.tap(
      find.descendant(of: cards.first, matching: find.byType(GfAvatar)),
    );
    await tester.pumpAndSettle();
    expect(router.state.uri.path, '/a/p-7');
    router.pop();
    await tester.pumpAndSettle();
    // A legacy anonymous avatar opens nothing, so the tap reaches the card.
    await tester.tap(
      find.descendant(of: cards.last, matching: find.byType(GfAvatar)),
    );
    await tester.pumpAndSettle();
    expect(find.byType(UserProfilePreview), findsNothing);
    expect(router.state.uri.path, '/p/101');
  });

  for (final mode in GfTopicFeedMode.values) {
    testWidgets('a topic deleted this session leaves the ${mode.name} feed', (
      tester,
    ) async {
      final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
      final kept = home.topics.first.copyWith(id: 101, title: 'kept topic');
      final container = ProviderContainer();
      addTearDown(container.dispose);
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: GfTopicList(
                loading: false,
                topics: [home.topics.first, kept],
                feedMode: mode,
                hasMore: false,
                onLoadMore: () {},
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('移动端测试话题'), findsOneWidget);
      container.read(removedTopicsProvider.notifier).add(100);
      await tester.pumpAndSettle();
      expect(find.text('移动端测试话题'), findsNothing);
      expect(find.text('kept topic'), findsOneWidget);
    });
  }

  testWidgets('feed cards outside GfTopicList drop a deleted topic too', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final container = ProviderContainer();
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: Builder(
              builder: (context) => ListView(
                children: [buildTopicFeedCard(context, home.topics.first)],
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byType(GfTopicCard), findsOneWidget);
    container.read(removedTopicsProvider.notifier).add(100);
    await tester.pumpAndSettle();
    expect(find.byType(GfTopicCard), findsNothing);
  });
}
