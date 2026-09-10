import 'dart:io';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:core/core.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/widgets/topic_list.dart';
import 'fixtures/page_fixtures.dart';

void main() {
  setUpAll(() async {
    final loader = FontLoader('Roboto');
    loader.addFont(
      Future.value(
        ByteData.sublistView(
          File('test/assets/fonts/Roboto-Regular.ttf').readAsBytesSync(),
        ),
      ),
    );
    await loader.load();
  });
  Future<void> pump(
    WidgetTester tester, {
    required List<TopicPayload> topics,
    ScrollController? controller,
    Future<bool> Function(TopicPayload, bool)? like,
    Future<bool> Function(TopicPayload, bool)? bookmark,
    double scale = 1,
  }) async {
    tester.view.physicalSize = const Size(320, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      MaterialApp(
        theme: ThemeData(fontFamily: 'Roboto'),
        locale: const Locale('zh'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(
            context,
          ).copyWith(textScaler: TextScaler.linear(scale)),
          child: child!,
        ),
        home: Scaffold(
          body: GfTopicList(
            loading: false,
            topics: topics,
            controller: controller,
            feedMode: GfTopicFeedMode.card,
            hasMore: false,
            onLoadMore: () {},
            onLikeTopic: like,
            onBookmarkTopic: bookmark,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  TopicPayload topic() {
    final t = parsePageProps<HomeProps>(
      parsePayload(homePayloadJson()),
    )!.topics.first;
    return t.copyWith(
      liked: false,
      bookmarked: false,
      title: 'A',
      description: '',
      images: [],
      firstImageUrl: '',
      categories: [],
      author: t.author.copyWith(nickname: 'A', username: 'A', avatarUrl: ''),
      activityText: '',
      replyCount: 100,
      viewCount: 10000,
    );
  }

  testWidgets('metrics fit phone with enlarged text without actions', (
    tester,
  ) async {
    await pump(tester, topics: [topic()], scale: 2);
    expect(tester.takeException(), isNull);
  });
  testWidgets('metrics fit phone with enlarged text with actions', (
    tester,
  ) async {
    await pump(
      tester,
      topics: [topic()],
      scale: 2,
      like: (_, _) async => true,
      bookmark: (_, _) async => true,
    );
    expect(tester.takeException(), isNull);
  });
}
