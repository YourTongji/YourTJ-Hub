import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import 'package:core/core.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/topic_list.dart';
import 'fixtures/page_fixtures.dart';

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
    await tester.pumpWidget(
      MaterialApp.router(
        routerConfig: router,
        locale: const Locale('zh'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('alice'));
    await tester.pumpAndSettle();
    expect(find.text('profile 1'), findsOneWidget);
    router.pop();
    await tester.pumpAndSettle();
    await tester.tap(find.byType(GfAvatar).first);
    await tester.pumpAndSettle();
    expect(find.text('profile 1'), findsOneWidget);
    router.pop();
    await tester.pumpAndSettle();
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
      MaterialApp(
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
      MaterialApp(
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

  testWidgets('pinned summary expands without hiding regular rows or footer', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final topic = home.topics.first;
    var loads = 0;
    final controller = ScrollController();
    addTearDown(controller.dispose);
    await tester.pumpWidget(
      MaterialApp(
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
    );
    await tester.pumpAndSettle();
    expect(find.text('Pinned one'), findsNothing);
    expect(find.text('Pinned two'), findsNothing);
    expect(find.text('Regular topic'), findsOneWidget);
    expect(find.text('Announcement'), findsOneWidget);
    expect(find.text('置顶话题（2）'), findsOneWidget);
    final collapsedHeight = tester.getTopLeft(find.text('Regular topic')).dy;
    await tester.tap(find.text('置顶话题（2）'));
    await tester.pumpAndSettle();
    expect(find.text('Pinned one'), findsOneWidget);
    expect(find.text('Pinned two'), findsOneWidget);
    expect(
      tester.getTopLeft(find.text('Regular topic')).dy,
      greaterThan(collapsedHeight),
    );
    controller.jumpTo(1500);
    await tester.pumpAndSettle();
    controller.jumpTo(0);
    await tester.pumpAndSettle();
    expect(find.text('Pinned one'), findsOneWidget);
    await tester.tap(find.text('置顶话题（2）'));
    await tester.pumpAndSettle();
    expect(find.text('Pinned one'), findsNothing);
    expect(loads, 0);
  });

  testWidgets('ordered streams and card mode retain pinned rows in place', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final topic = home.topics.first;
    for (final mode in GfTopicFeedMode.values) {
      await tester.pumpWidget(
        MaterialApp(
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: GfTopicList(
              collapsePinned: mode == GfTopicFeedMode.card,
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

  testWidgets('pinned-only feeds and all locales keep the summary accessible', (
    tester,
  ) async {
    final home = parsePageProps<HomeProps>(parsePayload(homePayloadJson()))!;
    final semanticsHandle = tester.ensureSemantics();

    final topic = home.topics.first.copyWith(pinWeight: 1);
    tester.view.physicalSize = const Size(320, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    for (final locale in AppLocalizations.supportedLocales) {
      final l10n = await AppLocalizations.delegate.load(locale);
      await tester.pumpWidget(
        MaterialApp(
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
                topics: [topic],
                hasMore: false,
                onLoadMore: () {},
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      final label = find.text(l10n.homePinnedTopics(1));
      final semantics = tester.getSemantics(label).getSemanticsData();
      expect(semantics.flagsCollection.isExpanded, ui.Tristate.isFalse);
      final target = find
          .ancestor(of: label, matching: find.byType(InkWell))
          .first;
      expect(tester.getSize(target).height, greaterThanOrEqualTo(48));
      await tester.tap(label);
      await tester.pumpAndSettle();
      expect(find.text(topic.title), findsOneWidget);
      expect(find.text(l10n.publishArticle), findsOneWidget);
      expect(tester.takeException(), isNull);
    }
    semanticsHandle.dispose();
  });

  // issue #895：无标题瞬间的列表行要与 Web TopicRow/SSR 一致，始终有可识别的行标题。
  Future<void> pumpList(WidgetTester tester, TopicPayload topic) async {
    await tester.pumpWidget(
      MaterialApp(
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

      expect(find.text('(｀・ω・´)'), findsOneWidget);
    },
  );
}
