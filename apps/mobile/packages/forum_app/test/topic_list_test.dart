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
}
