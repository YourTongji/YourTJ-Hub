import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/courses/my_reviews_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';

class _Tokens implements TokenStorage {
  @override
  Future<String?> read() async => 'session';
  @override
  Future<void> write(String value) async {}
  @override
  Future<void> clear() async {}
}

ReviewPayload _review(int id, String content, {bool hidden = false}) =>
    ReviewPayload(
      id: id,
      offeringId: id + 900,
      rating: 5,
      content: content,
      contentHtml: '<p>$content</p>',
      author: const ReviewAuthorPayload(kind: 'anonymous', label: '匿名同学'),
      viewer: ReviewViewerPayload(
        canEdit: !hidden,
        canDelete: true,
        isHelpful: false,
      ),
      helpfulCount: 0,
      createdAt: '2026-09-01T00:00:00Z',
      updatedAt: '2026-09-01T00:00:00Z',
    );
OwnCourseReviewItem _item(int id, String content, {bool hidden = false}) =>
    OwnCourseReviewItem(
      review: _review(id, content, hidden: hidden),
      courseId: 42 + id,
      courseName: '课程 $id',
      courseCode: 'CODE$id',
      hidden: hidden,
      canOpenCourse: !hidden,
    );

class _Repository extends CourseRepository {
  _Repository()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: _Tokens(),
          baseUrl: 'http://fake.local',
        ),
      );
  final deletes = <int>[];
  final cursors = <String>[];
  bool failDelete = false;
  bool showHidden = false;
  @override
  Future<OwnCourseReviewPage> ownReviews({
    String cursor = '',
    int pageSize = 20,
  }) async {
    cursors.add(cursor);
    return OwnCourseReviewPage(
      list: [
        cursor.isEmpty
            ? _item(2, '第一条匿名评价', hidden: showHidden)
            : _item(1, '更早的评价'),
      ],
      nextCursor: cursor.isEmpty ? '2' : '',
    );
  }

  @override
  Future<ReviewPayload> updateReview(
    int id,
    UpdateCourseReviewInput input,
  ) async => _review(id, input.content!);
  @override
  Future<bool> deleteReview(int id) async {
    deletes.add(id);
    if (failDelete) throw const NetworkException(fallbackMessage: 'offline');
    return true;
  }
}

/// 卡片右上溢出菜单（查看课程 / 编辑 / 删除）。
Finder _ownMenu(int reviewId) =>
    find.byKey(ValueKey<String>('own-review-menu-$reviewId'));

Future<void> _openOwnMenu(WidgetTester tester, int reviewId) async {
  await tester.tap(_ownMenu(reviewId));
  await tester.pumpAndSettle();
}

Future<void> _closeMenu(WidgetTester tester) async {
  await tester.tapAt(const Offset(4, 4));
  await tester.pumpAndSettle();
}

Future<void> _pump(WidgetTester tester, _Repository repo) async {
  final router = GoRouter(
    routes: [
      GoRoute(path: '/', builder: (_, _) => const MyCourseReviewsPage()),
      GoRoute(
        path: '/courses/:id',
        builder: (_, state) => Scaffold(body: Text(state.uri.toString())),
      ),
    ],
  );
  addTearDown(router.dispose);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [courseRepositoryProvider.overrideWithValue(repo)],
      child: MaterialApp.router(
        routerConfig: router,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        locale: const Locale('zh'),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets(
    'lists own reviews across courses and opens the exact offering/review',
    (tester) async {
      final repo = _Repository();
      await _pump(tester, repo);
      expect(find.text('第一条匿名评价'), findsOneWidget);
      // Short first page loads the next cursor automatically.
      expect(repo.cursors, ['', '2']);
      expect(find.text('更早的评价'), findsOneWidget);
      // 匿名课评走生成头像，动作收进右上溢出菜单。
      expect(
        tester
            .widgetList<GfBeamAvatar>(find.byType(GfBeamAvatar))
            .map((avatar) => avatar.seed),
        containsAll(<String>['匿名同学-2', '匿名同学-1']),
      );
      expect(find.text('查看详情'), findsNothing);
      await _openOwnMenu(tester, 2);
      await tester.tap(find.text('查看详情'));
      await tester.pumpAndSettle();
      expect(
        find.text('/courses/44?offeringId=902&reviewId=2'),
        findsOneWidget,
      );
    },
  );
  testWidgets('editing reuses the form and updates the management card', (
    tester,
  ) async {
    final repo = _Repository();
    await _pump(tester, repo);
    await _openOwnMenu(tester, 2);
    await tester.tap(find.text('编辑'));
    await tester.pumpAndSettle();
    final editor = tester.widget<QuillEditor>(find.byType(QuillEditor));
    editor.controller.replaceText(
      0,
      editor.controller.document.length - 1,
      '修改后的评价',
      const TextSelection.collapsed(offset: 6),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('保存'));
    await tester.pumpAndSettle();
    expect(find.text('修改后的评价'), findsOneWidget);
    await tester.pump(const Duration(seconds: 8));
  });
  testWidgets(
    'delete previews the target; cancel does not delete; failure keeps it',
    (tester) async {
      final repo = _Repository();
      await _pump(tester, repo);
      await _openOwnMenu(tester, 2);
      await tester.tap(find.text('删除'));
      await tester.pumpAndSettle();
      expect(find.byType(AlertDialog), findsOneWidget);
      expect(
        find.descendant(
          of: find.byType(AlertDialog),
          matching: find.text('第一条匿名评价'),
        ),
        findsOneWidget,
      );
      await tester.tap(find.text('取消'));
      await tester.pumpAndSettle();
      expect(repo.deletes, isEmpty);
      repo.failDelete = true;
      await _openOwnMenu(tester, 2);
      await tester.tap(find.text('删除'));
      await tester.pumpAndSettle();
      await tester.tap(
        find.descendant(
          of: find.byType(AlertDialog),
          matching: find.text('删除'),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.textContaining('网络连接失败'), findsOneWidget);
      expect(find.text('第一条匿名评价'), findsOneWidget);
      await tester.pump(const Duration(seconds: 8));
      repo.failDelete = false;
      await _openOwnMenu(tester, 2);
      await tester.tap(find.text('删除'));
      await tester.pumpAndSettle();
      await tester.tap(
        find.descendant(
          of: find.byType(AlertDialog),
          matching: find.text('删除'),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('第一条匿名评价'), findsNothing);
      await tester.pump(const Duration(seconds: 8));
    },
  );
  testWidgets(
    'hidden reviews remain deletable without an edit or public link',
    (tester) async {
      await _pump(tester, _Repository()..showHidden = true);
      expect(find.textContaining('已被隐藏'), findsOneWidget);
      // 被隐藏的是较新的那条（id=2）：只能删除，没有公开链接也编辑不了。
      await _openOwnMenu(tester, 2);
      expect(find.text('查看详情'), findsNothing);
      expect(find.text('编辑'), findsNothing);
      expect(find.text('删除'), findsOneWidget);
      await _closeMenu(tester);
      // 可见的旧评价（id=1）：查看课程 / 编辑 / 删除。
      await _openOwnMenu(tester, 1);
      expect(find.text('查看详情'), findsOneWidget);
      expect(find.text('编辑'), findsOneWidget);
      expect(find.text('删除'), findsOneWidget);
      await _closeMenu(tester);
      expect(find.text('编辑'), findsNothing);
      expect(find.text('删除'), findsNothing);
    },
  );
}
