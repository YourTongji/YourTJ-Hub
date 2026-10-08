import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/topic/post_actions.dart';
import 'package:forum_app/src/pages/topic/topic_actions.dart';
import 'package:ui_kit/ui_kit.dart';
import 'fixtures/page_fixtures.dart';

void main() {
  for (final topicMenu in [true, false]) {
    testWidgets(
      'private identity management is absent from ${topicMenu ? 'topic' : 'reply'} menu',
      (tester) async {
        final json = topicDetailPayloadJson()['props'] as Map<String, dynamic>;
        final author = {
          'id': 0,
          'username': '躲进云里的猫',
          'avatarUrl': '/a/${'a' * 32}/avatar.svg',
          'publicUid': 'a' * 32,
          'kind': 'persona',
          'profileUrl': '/a/${'a' * 32}',
        };
        json['topic']['author'] = author;
        final props = TopicDetailProps.fromJson(json);
        final post = PostPayload.fromJson({
          ...json['postStream']['posts'][0] as Map<String, dynamic>,
          'author': author,
          'canModerate': true,
        });
        await tester.pumpWidget(
          ProviderScope(
            child: MaterialApp(
              theme: gfThemeData(Brightness.light),
              locale: const Locale('zh'),
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: Scaffold(
                body: topicMenu
                    ? TopicActions(
                        props: props.copyWith(
                          permissions: props.permissions.copyWith(
                            canModerateTopic: true,
                          ),
                        ),
                        firstPostId: post.id,
                        onChanged: () async {},
                      )
                    : PostActions(
                        post: post,
                        onChanged: () async {},
                        onReply: () {},
                        onReport: () {},
                      ),
              ),
            ),
          ),
        );
        await tester.tap(find.byType(PopupMenuButton<String>));
        await tester.pumpAndSettle();
        expect(find.text('管理匿名身份'), findsNothing);
        expect(find.byType(PopupMenuItem<String>), findsWidgets);
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pumpAndSettle();
      },
    );
  }
}
