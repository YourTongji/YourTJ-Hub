import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/content/content_page.dart';
import 'package:forum_app/src/pages/topic/post_actions.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';
import '../test/fixtures/page_fixtures.dart';

class _TokenStorage implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

class _Content extends ContentRepository {
  _Content(super.client);
  bool submitted = false;
  @override
  Future<UserContentPage> list({
    required String contentType,
    bool deleted = false,
    int cursor = 0,
  }) async => UserContentPage(
    items: [
      UserContentItem(
        id: 42,
        contentType: 'post',
        title: '校园生活中的一次讨论',
        content: '**完整正文**\n\n被拒的修改仍可编辑。',
        processStatus: submitted ? 2 : 1,
        reviewReason: submitted ? '' : '请修改正文后重新提交。',
        hasPublishedVersion: true,
      ),
    ],
    hasMore: false,
    nextCursorId: 0,
  );
}

class _Posts extends PostRepository {
  _Posts(super.client, this.content);
  final _Content content;
  @override
  Future<UpdatePostResult> updatePost({
    required int postId,
    required String content,
  }) async {
    this.content.submitted = true;
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

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  for (final brightness in [Brightness.light, Brightness.dark]) {
    testWidgets('native rejected content recovery ${brightness.name}', (
      tester,
    ) async {
      final client = GfApiClient(
        dio: Dio(),
        tokenStorage: _TokenStorage(),
        baseUrl: 'http://127.0.0.1',
      );
      final repo = _Content(client);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            contentRepositoryProvider.overrideWithValue(repo),
            postRepositoryProvider.overrideWithValue(_Posts(client, repo)),
          ],
          child: MaterialApp(
            theme: gfThemeData(brightness),
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const ContentPage(),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.textContaining('请修改正文后重新提交'), findsOneWidget);
      await binding.takeScreenshot('content-review-${brightness.name}');
      await tester.tap(find.byType(PopupMenuButton<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.text('修改并重新提交'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField), '修改后的回复正文');
      await binding.takeScreenshot('content-review-edit-${brightness.name}');
      await tester.tap(find.widgetWithText(TextButton, '修改并重新提交'));
      await tester.pumpAndSettle();
      expect(find.textContaining('审核中'), findsOneWidget);
      expect(find.byType(AlertDialog), findsNothing);
      expect(tester.takeException(), isNull);

      // Detail actions retain the author's editor while the reply is pending.
      final detail = TopicDetailProps.fromJson(
        topicDetailPayloadJson()['props'] as Map<String, dynamic>,
      );
      final pendingPost = detail.postStream.posts.first.copyWith(
        id: 42,
        postNo: 2,
        isOwnPost: true,
        isHidden: true,
        processStatus: 2,
        content: '修改后的回复正文',
      );
      await tester.pumpWidget(
        ProviderScope(
          key: UniqueKey(),
          overrides: [
            postRepositoryProvider.overrideWithValue(_Posts(client, repo)),
          ],
          child: MaterialApp(
            theme: gfThemeData(brightness),
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: PostActions(
                post: pendingPost,
                onChanged: () async {},
                onReply: null,
                onReport: () {},
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(PopupMenuButton<String>));
      await tester.pumpAndSettle();
      await binding.takeScreenshot('pending-edit-menu-${brightness.name}');
      await tester.tap(find.text('编辑'));
      await tester.pumpAndSettle();
      expect(find.text('修改后的回复正文'), findsOneWidget);
      await binding.takeScreenshot('pending-edit-sheet-${brightness.name}');
      expect(tester.takeException(), isNull);
    });
  }
}
