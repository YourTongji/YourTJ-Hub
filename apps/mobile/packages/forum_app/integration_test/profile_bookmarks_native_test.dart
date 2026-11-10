import 'dart:async';
import 'dart:io';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/current_user.dart';
import 'package:forum_app/src/pages/profile/profile_page.dart';
import 'package:forum_app/src/pages/profile/profile_tabs.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/status_views.dart';
import 'package:integration_test/integration_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../test/fixtures/page_fixtures.dart';

class _MemoryTokens implements TokenStorage {
  @override
  Future<String?> read() async => null;
  @override
  Future<void> write(String token) async =>
      throw StateError('This fixture must not write credentials');
  @override
  Future<void> clear() async {}
}

String _title(String type) =>
    type == 'topic' ? '合成主题收藏 · id=7' : '合成回复收藏 · id=7';

class _BookmarkPages extends PageRepository {
  _BookmarkPages(super.client, this.firstType);

  static const firstPath = '/u/1/bookmarks';
  static const nextPath = '$firstPath?cursor=next';
  final String firstType;
  String get nextType => firstType == 'topic' ? 'post' : 'topic';
  final paths = <String>[];
  final nextPage = Completer<PagePayload>();

  Map<String, dynamic> _bookmark(String type) => {
    'id': 7,
    'type': type,
    'topicId': type == 'topic' ? 100 : 200,
    if (type == 'post') 'postId': 555,
    if (type == 'post') 'postNo': 3,
    'title': _title(type),
    'excerpt': type == 'topic' ? '主题 100（合成数据）' : '主题 200 · 第 3 楼（合成数据）',
    'url': type == 'topic' ? '/p/100' : '/p/200?postNo=3',
    'bookmarkedAt': '2026-01-01T00:00:00Z',
  };

  PagePayload response({required bool next}) {
    final json = userProfilePayloadJson();
    final props = json['props'] as Map<String, dynamic>;
    (props['user'] as Map<String, dynamic>)
      ..['username'] = 'synthetic-bookmarks'
      ..['nickname'] = '收藏验收（合成数据）'
      ..['bio'] = '本地合成收藏，不连接真实账号或服务。'
      ..['collectionCount'] = 18;
    props['activities'] = <Object>[];
    props['bookmarks'] = next
        ? [_bookmark(firstType), _bookmark(nextType), _bookmark(nextType)]
        : [
            // Keep the footer outside native viewport prefetch until a real
            // scroll. The equal-ID pair remains adjacent across the page edge.
            for (var i = 0; i < 16; i++)
              {
                'id': 1000 + i,
                'type': 'topic',
                'topicId': 1000 + i,
                'title': '合成已有收藏 ${i + 1}',
                'excerpt': '用于滚动到下一页的本地合成内容。',
                'url': '/p/${1000 + i}',
                'bookmarkedAt': '2026-01-01T00:00:00Z',
              },
            _bookmark(firstType),
          ];
    props['pagination'] = {
      'page': next ? 2 : 1,
      'nextPage': next ? 0 : 2,
      'hasNext': !next,
      'nextUrl': next ? '' : nextPath,
    };
    return parsePayload(json);
  }

  @override
  Future<PagePayload> fetch(String path, {CancelToken? cancelToken}) async {
    paths.add(path);
    return switch (path) {
      firstPath => response(next: false),
      nextPath => nextPage.future,
      _ => throw StateError('Unexpected fixture read: $path'),
    };
  }
}

Future<void> _until(WidgetTester tester, bool Function() ready) async {
  for (var attempt = 0; attempt < 100 && !ready(); attempt++) {
    await tester.pump(const Duration(milliseconds: 100));
  }
  expect(
    ready(),
    isTrue,
    reason: 'Native profile did not reach the expected state',
  );
}

/// Scroll the production list, keeping the complete target rows below its
/// pinned tabs and above the real device's bottom safe area for review.
Future<void> _showRows(WidgetTester tester, List<String> titles) async {
  final scroll = find.byType(CustomScrollView).first;
  final rows = [
    for (final title in titles)
      find.ancestor(of: find.text(title), matching: find.byType(GfContentRow)),
  ];
  for (var attempt = 0; attempt < 40; attempt++) {
    var delta = -200.0;
    if (rows.every((row) => row.evaluate().length == 1)) {
      final top = tester.getRect(find.byType(ProfileTabs)).bottom + 12;
      final view = tester.view;
      final bottom =
          (view.physicalSize.height - view.padding.bottom) /
              view.devicePixelRatio -
          12;
      final first = tester.getRect(rows.first);
      final last = tester.getRect(rows.last);
      if (first.top >= top && last.bottom <= bottom) {
        for (final row in rows) {
          expect(row.hitTestable(), findsOneWidget);
        }
        return;
      }
      delta = first.top < top ? top - first.top : bottom - last.bottom;
      delta = delta.clamp(-200.0, 200.0).toDouble();
      // A sub-slop adjustment would never become a native drag gesture.
      if (delta.abs() < 40) delta = delta.sign * 40;
    }
    await tester.drag(scroll, Offset(0, delta));
    // A pending page deliberately keeps its progress animation alive.
    await tester.pump(const Duration(milliseconds: 100));
  }
  fail('Could not show the complete bookmark rows in the native viewport');
}

/// Real iOS Flutter rendering with synthetic repository responses only. This
/// does not exercise a live backend, sign-in, or a physical device.
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();
  binding.framePolicy = LiveTestWidgetsFlutterBindingFramePolicy.fullyLive;
  for (final brightness in Brightness.values) {
    for (final firstType in ['topic', 'post']) {
      testWidgets('native bookmarks ${brightness.name}, $firstType first', (
        tester,
      ) async {
        expect(
          Platform.isIOS,
          isTrue,
          reason: 'Run on an iOS simulator/device',
        );
        final requests = <String>[];
        final dio = Dio()
          ..interceptors.add(
            InterceptorsWrapper(
              onRequest: (options, handler) {
                requests.add('${options.method} ${options.uri}');
                handler.reject(
                  DioException(
                    requestOptions: options,
                    error: 'Native bookmark fixtures forbid network requests',
                  ),
                );
              },
            ),
          );
        final tokens = _MemoryTokens();
        final client = GfApiClient(
          dio: dio,
          tokenStorage: tokens,
          baseUrl: 'https://bookmarks.invalid',
        );
        final pages = _BookmarkPages(client, firstType);
        final container = ProviderContainer(
          overrides: [
            dioProvider.overrideWithValue(dio),
            apiClientProvider.overrideWithValue(client),
            tokenStorageProvider.overrideWithValue(tokens),
            currentUserProvider.overrideWith(
              (_) async =>
                  const CurrentUser(id: 1, username: 'synthetic-bookmarks'),
            ),
            pageRepositoryProvider.overrideWithValue(pages),
          ],
        );
        try {
          await tester.pumpWidget(
            UncontrolledProviderScope(
              container: container,
              child: MaterialApp(
                theme: gfThemeData(brightness),
                locale: const Locale('zh'),
                localizationsDelegates: AppLocalizations.localizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                home: const ProfilePage(userId: 1, initialStream: 'bookmarks'),
              ),
            ),
          );
          await _until(
            tester,
            () => find.text('合成已有收藏 1').evaluate().isNotEmpty,
          );
          expect(pages.paths, [_BookmarkPages.firstPath]);
          final context = tester.element(find.byType(ProfilePage));
          expect(Theme.of(context).brightness, brightness);
          expect(
            MediaQuery.sizeOf(context),
            tester.view.physicalSize / tester.view.devicePixelRatio,
          );
          expect(
            MediaQuery.paddingOf(context),
            EdgeInsets.fromViewPadding(
              tester.view.padding,
              tester.view.devicePixelRatio,
            ),
          );
          debugPrint(
            'PROFILE_BOOKMARK_NATIVE ${brightness.name} $firstType-first '
            'size=${MediaQuery.sizeOf(context)} '
            'dpr=${tester.view.devicePixelRatio} '
            'padding=${MediaQuery.paddingOf(context)} synthetic=true',
          );

          await _showRows(tester, [_title(firstType)]);
          await _until(
            tester,
            () => pages.paths.contains(_BookmarkPages.nextPath),
          );
          expect(find.text(_title(firstType)), findsOneWidget);
          expect(find.text(_title(pages.nextType)), findsNothing);
          expect(
            tester.widget<GfListFooter>(find.byType(GfListFooter)).loading,
            isTrue,
          );
          // Repeated scrolling while the page is pending must not duplicate it.
          await tester.drag(
            find.byType(CustomScrollView).first,
            const Offset(0, -80),
          );
          await _showRows(tester, [_title(firstType)]);
          expect(pages.paths, [
            _BookmarkPages.firstPath,
            _BookmarkPages.nextPath,
          ]);
          expect(tester.takeException(), isNull);
          final name = 'profile-bookmarks-${brightness.name}-$firstType-first';
          if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
            await binding.takeScreenshot('$name-pending');
          }

          pages.nextPage.complete(pages.response(next: true));
          await tester.pumpAndSettle();
          await _showRows(tester, [_title(firstType), _title(pages.nextType)]);
          expect(find.text(_title('topic')), findsOneWidget);
          expect(find.text(_title('post')), findsOneWidget);
          expect(
            tester.getTopLeft(find.text(_title(firstType))).dy,
            lessThan(tester.getTopLeft(find.text(_title(pages.nextType))).dy),
          );
          expect(find.byType(GfListFooter), findsNothing);
          expect(pages.paths, [
            _BookmarkPages.firstPath,
            _BookmarkPages.nextPath,
          ]);
          expect(
            requests,
            isEmpty,
            reason: 'No real API reads or writes are allowed',
          );
          expect(tester.takeException(), isNull);
          await tester.pumpAndSettle();
          if (const bool.fromEnvironment('YOURTJ_TEST_SCREENSHOTS')) {
            await binding.takeScreenshot('$name-merged');
          }
        } finally {
          await tester.pumpWidget(const SizedBox.shrink());
          await tester.pumpAndSettle();
          container.dispose();
          dio.close(force: true);
        }
      });
    }
  }
}
