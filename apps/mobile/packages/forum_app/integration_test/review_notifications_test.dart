import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/notifications/notifications_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import '../test/fixtures/page_fixtures.dart';

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => 'session';
  @override
  Future<void> write(String token) async {}
  @override
  Future<void> clear() async {}
}

GfApiClient _client() => GfApiClient(dio: Dio(), tokenStorage: _Tokens());

class _Pages extends PageRepository {
  _Pages() : super(_client());
  @override
  Future<PagePayload> fetch(String path, {CancelToken? cancelToken}) async =>
      parsePayload(homePayloadJson());
}

class _Notifications extends NotificationRepository {
  _Notifications() : super(_client());
  @override
  Future<NotificationListResponse> fetchNotifications({
    String filter = 'all',
    int cursor = 0,
    int limit = 20,
    CancelToken? cancelToken,
  }) async => NotificationListResponse(
    items: [
      for (final entry in ['Pending', 'Approved', 'Rejected'].indexed)
        NotificationPayload(
          id: entry.$1 + 1,
          eventType: 'review_${entry.$2.toLowerCase()}',
          isRead: false,
          createdAt: '2026-10-04T08:00:00Z',
          title: '',
          content: '',
          actor: const NotificationActorPayload(id: 0, username: ''),
          payload: NotificationInnerPayload(
            actorId: 0,
            templateKey: 'notifications.templates.review${entry.$2}',
            topicId: 42,
            postNo: 1,
            topicTitle: entry.$2 == 'Rejected' ? '校******论' : '校园生活中的一次讨论',
          ),
        ),
    ],
    nextCursor: 0,
    hasNext: false,
    unreadCount: 3,
  );
  @override
  Future<bool> markNotificationRead({required int notificationId}) async =>
      true;
}

void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  for (final brightness in [Brightness.light, Brightness.dark]) {
    testWidgets('review notifications and recovery links ${brightness.name}', (
      tester,
    ) async {
      final router = GoRouter(
        routes: [
          GoRoute(path: '/', builder: (_, _) => const NotificationsPage()),
          GoRoute(
            path: '/my-content',
            builder: (_, _) => const Scaffold(body: Text('内容管理入口')),
          ),
          GoRoute(
            path: '/p/42',
            builder: (_, state) => Scaffold(
              body: Text('待审详情 ${state.uri.queryParameters['postNo']}'),
            ),
          ),
        ],
      );
      await tester.pumpWidget(
        ProviderScope(
          key: UniqueKey(),
          overrides: [
            notificationRepositoryProvider.overrideWithValue(_Notifications()),
            tokenStorageProvider.overrideWithValue(_Tokens()),
            pageRepositoryProvider.overrideWithValue(_Pages()),
          ],
          child: MaterialApp.router(
            routerConfig: router,
            theme: gfThemeData(brightness),
            locale: const Locale('zh'),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      await tester.pumpAndSettle();
      final rows = find.byType(GfNotificationRow);
      expect(rows, findsNWidgets(3));
      final widgets = tester.widgetList<GfNotificationRow>(rows).toList();
      expect(widgets[0].title, '你的内容正在等待人工审核');
      expect(widgets[1].title, '你的内容已通过审核，现在所有人都能看到它');
      expect(widgets[2].subtitle, contains('校******论'));
      expect(widgets[2].subtitle, contains('前往内容管理自查修改后重新提交'));
      expect(widgets[2].subtitleMaxLines, isNull);
      final guidance = find.text(widgets[2].subtitle);
      final paragraph = tester.renderObject<RenderParagraph>(
        find.descendant(of: guidance, matching: find.byType(RichText)),
      );
      expect(paragraph.didExceedMaxLines, isFalse);
      expect(paragraph.overflow, TextOverflow.visible);
      expect(tester.takeException(), isNull);
      await binding.takeScreenshot('review-notifications-${brightness.name}');
      await tester.tap(find.textContaining('你的内容正在等待人工审核'));
      await tester.pumpAndSettle();
      expect(find.text('待审详情 1'), findsOneWidget);
      router.pop();
      await tester.pumpAndSettle();
      await tester.tap(find.textContaining('你的内容未通过审核'));
      await tester.pumpAndSettle();
      expect(find.text('内容管理入口'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
      router.dispose();
    });
  }
}
