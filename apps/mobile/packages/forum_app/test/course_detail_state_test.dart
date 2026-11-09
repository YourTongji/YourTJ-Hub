import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/courses/detail_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/app_refresh_indicator.dart';
import 'package:ui_kit/ui_kit.dart';

import 'courses_page_test.dart' show FakeCourseRepository;
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _ReviewRequest {
  _ReviewRequest(this.offeringId, this.cursor);
  final int? offeringId;
  final String? cursor;
  final result = Completer<ReviewListResult>();
}

class _Courses extends FakeCourseRepository {
  _Courses(super.client);

  bool? serverBookmarked = false;
  Completer<CourseDetailPayload>? pendingDetail;
  Completer<void>? pendingBookmark;
  final details = <int>[];
  final relatedCalls = <int>[];
  final requests = <_ReviewRequest>[];
  final bookmarks = <(int, bool)>[];

  @override
  Future<CourseDetailPayload> detail(int courseId) async {
    details.add(courseId);
    final pending = pendingDetail;
    if (pending != null) return pending.future;
    return _detail(serverBookmarked).copyWith(id: courseId);
  }

  @override
  Future<CourseRelatedResult> related(int courseId) {
    relatedCalls.add(courseId);
    return super.related(courseId);
  }

  @override
  Future<ReviewListResult> reviews(
    int courseId, {
    int? offeringId,
    String? cursor,
    int pageSize = 20,
  }) {
    final request = _ReviewRequest(offeringId, cursor);
    requests.add(request);
    return request.result.future;
  }

  @override
  Future<bool> bookmark({
    required int courseId,
    required bool bookmarked,
  }) async {
    bookmarks.add((courseId, bookmarked));
    await pendingBookmark?.future;
    serverBookmarked = bookmarked;
    return true;
  }
}

CourseDetailPayload _detail(bool? bookmarked) => CourseDetailPayload.fromJson({
  'id': 42,
  'primaryCode': 'TEST42',
  'name': '合成课程',
  'department': '测试学院',
  'creditX10': 20,
  'bookmarked': ?bookmarked,
  'offerings': [
    {
      'id': 901,
      'termCode': '2026-1',
      'termName': '第一学期',
      'className': '01班',
      'instructors': ['测试教师'],
    },
    {
      'id': 902,
      'termCode': '2026-2',
      'termName': '第二学期',
      'className': '02班',
      'instructors': ['测试教师'],
    },
  ],
});

ReviewListResult _reviews(int base, {String? cursor, int offeringId = 901}) =>
    ReviewListResult(
      list: [
        for (var i = 0; i < 10; i++)
          ReviewPayload(
            id: base + i,
            offeringId: offeringId,
            rating: 4,
            helpfulCount: 0,
            createdAt: '2026-09-01T08:00:00+08:00',
            updatedAt: '2026-09-01T08:00:00+08:00',
            content: '合成评价 ${base + i}',
            contentHtml: '<p>合成评价 ${base + i}</p>',
            author: const ReviewAuthorPayload(kind: 'legacy', label: '测试读者'),
            viewer: const ReviewViewerPayload(
              canEdit: false,
              canDelete: false,
              isHelpful: false,
            ),
          ),
      ],
      total: cursor == null ? 10 : 20,
      nextCursor: cursor,
    );

void main() {
  late _Courses courses;
  late ProviderContainer container;
  late ValueNotifier<int> courseId;

  Future<void> pumpDetail(
    WidgetTester tester, {
    bool bookmarked = false,
  }) async {
    tester.view.physicalSize = const Size(1100, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final storage = MemoryTokenStorage();
    await storage.write('header.eyJVc2VySWQiOjEyM30.signature');
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    courses = _Courses(client)..serverBookmarked = bookmarked;
    courseId = ValueNotifier(42);
    addTearDown(courseId.dispose);
    container = ProviderContainer(
      overrides: [
        apiClientProvider.overrideWithValue(client),
        tokenStorageProvider.overrideWithValue(storage),
        courseRepositoryProvider.overrideWithValue(courses),
      ],
    );
    addTearDown(() async {
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(seconds: 5));
      container.dispose();
    });
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: const Locale('zh'),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: ValueListenableBuilder<int>(
            valueListenable: courseId,
            builder: (_, id, _) => CourseDetailPage(courseId: id),
          ),
        ),
      ),
    );
    await tester.pump();
    await tester.pump();
    expect(courses.requests, hasLength(1));
    courses.requests.single.result.complete(_reviews(100, cursor: 'all-next'));
    await tester.pumpAndSettle();
  }

  Future<void> scrollToEnd(WidgetTester tester) async {
    final controller = tester
        .widget<CustomScrollView>(find.byType(CustomScrollView))
        .controller!;
    controller.jumpTo(0);
    controller.jumpTo(controller.position.maxScrollExtent);
    await tester.pump();
  }

  Future<void> selectOffering(WidgetTester tester) async {
    final offering = find.descendant(
      of: find.byKey(const Key('course-detail-sidebar')),
      matching: find.text('02班'),
    );
    await tester.ensureVisible(offering);
    await tester.pump();
    await tester.tap(offering);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
  }

  for (final oldFails in [false, true]) {
    testWidgets(
      'offering change releases old pagination lock (old failure: $oldFails)',
      (tester) async {
        await pumpDetail(tester);
        await scrollToEnd(tester);
        final oldPage = courses.requests.last;
        expect(oldPage.cursor, 'all-next');

        await selectOffering(tester);
        final replacement = courses.requests.last;
        expect(replacement.offeringId, 902);
        expect(replacement.cursor, isNull);
        replacement.result.complete(
          _reviews(200, cursor: 'offering-next', offeringId: 902),
        );
        await tester.pump();
        await scrollToEnd(tester);
        final newPage = courses.requests.last;
        expect(newPage.cursor, 'offering-next');
        expect(newPage.offeringId, 902);
        final requestCount = courses.requests.length;

        if (oldFails) {
          oldPage.result.completeError(
            const ApiException(fallbackMessage: 'old failure'),
          );
        } else {
          oldPage.result.complete(_reviews(900, cursor: 'stale-next'));
        }
        await tester.pump();
        await scrollToEnd(tester);
        // The stale finally must not unlock the newer in-flight page.
        expect(courses.requests, hasLength(requestCount));

        newPage.result.complete(_reviews(300, offeringId: 902));
        await tester.pumpAndSettle();
        await scrollToEnd(tester);
        await tester.pumpAndSettle();
        expect(
          find.textContaining('合成评价 309', findRichText: true),
          findsOneWidget,
        );
        expect(
          find.textContaining('合成评价 90', findRichText: true),
          findsNothing,
        );
        expect(tester.takeException(), isNull);
      },
    );
  }

  for (final oldFails in [false, true]) {
    for (final oldFinishesFirst in [true, false]) {
      testWidgets('epoch change isolates old and new pagination '
          '(old failure: $oldFails, old finishes first: $oldFinishesFirst)', (
        tester,
      ) async {
        await pumpDetail(tester);
        await scrollToEnd(tester);
        final oldPage = courses.requests.last;
        expect(oldPage.cursor, 'all-next');

        container.read(offlineCacheEpochProvider.notifier).invalidate();
        await tester.pump();
        await tester.pump();
        expect(courses.details, [42, 42]);
        final replacement = courses.requests.last;
        expect(replacement, isNot(same(oldPage)));
        expect(replacement.cursor, isNull);
        replacement.result.complete(_reviews(200, cursor: 'session-next'));
        await tester.pump();
        await scrollToEnd(tester);
        final newPage = courses.requests.last;
        expect(newPage.cursor, 'session-next');
        final requestCount = courses.requests.length;

        void completeOldPage() {
          if (oldFails) {
            oldPage.result.completeError(
              const ApiException(fallbackMessage: 'old session page failure'),
            );
          } else {
            oldPage.result.complete(
              _reviews(900, cursor: 'stale-session-next'),
            );
          }
        }

        if (oldFinishesFirst) {
          completeOldPage();
          await tester.pump();
          await scrollToEnd(tester);
          // A disposed epoch's finally cannot release the new epoch's lock.
          expect(courses.requests, hasLength(requestCount));
          // Check while stale rows would still be visible at the list's end.
          expect(
            find.textContaining('合成评价 90', findRichText: true),
            findsNothing,
          );
          expect(tester.takeException(), isNull);
        }

        newPage.result.complete(_reviews(300, cursor: 'session-following'));
        await tester.pump();
        if (!oldFinishesFirst) {
          // A late old result cannot replace the new page's following cursor.
          completeOldPage();
          await tester.pump();
        }
        await scrollToEnd(tester);
        final following = courses.requests.last;
        expect(following.cursor, 'session-following');
        expect(courses.requests, hasLength(requestCount + 1));
        expect(
          find.textContaining('合成评价 90', findRichText: true),
          findsNothing,
        );
        following.result.complete(_reviews(400));
        await tester.pumpAndSettle();
        await scrollToEnd(tester);
        await tester.pumpAndSettle();
        expect(
          find.textContaining('合成评价 409', findRichText: true),
          findsOneWidget,
        );
        expect(
          find.textContaining('合成评价 90', findRichText: true),
          findsNothing,
        );
        expect(tester.takeException(), isNull);
      });
    }
  }

  testWidgets(
    'clearing focus releases pending pagination and current failures can retry',
    (tester) async {
      await pumpDetail(tester);
      await selectOffering(tester);
      courses.requests.last.result.complete(
        _reviews(200, cursor: 'focused-next', offeringId: 902),
      );
      await tester.pumpAndSettle();
      await scrollToEnd(tester);
      final oldPage = courses.requests.last;
      expect(oldPage.cursor, 'focused-next');

      final controller = tester
          .widget<CustomScrollView>(find.byType(CustomScrollView))
          .controller!;
      controller.jumpTo(0);
      await tester.pump();
      await tester.tap(find.text('查看全部评价'));
      await tester.pump();
      final all = courses.requests.last;
      expect(all.offeringId, isNull);
      expect(all.cursor, isNull);
      all.result.complete(_reviews(300, cursor: 'all-new-next'));
      await tester.pump();
      oldPage.result.complete(_reviews(900));
      await tester.pump();
      await scrollToEnd(tester);
      final failed = courses.requests.last;
      expect(failed.cursor, 'all-new-next');
      failed.result.completeError(
        const ApiException(fallbackMessage: 'retry page'),
      );
      await tester.pump();
      final count = courses.requests.length;
      await scrollToEnd(tester);
      expect(courses.requests, hasLength(count + 1));
      expect(courses.requests.last.cursor, 'all-new-next');
      courses.requests.last.result.complete(_reviews(400));
      await tester.pumpAndSettle();
    },
  );

  testWidgets(
    'refresh releases a pending review page without applying its rows',
    (tester) async {
      await pumpDetail(tester);
      await scrollToEnd(tester);
      final oldPage = courses.requests.last;
      final refresh = tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.pump();
      await refresh;
      final refreshed = courses.requests.last;
      expect(refreshed.cursor, isNull);
      refreshed.result.complete(_reviews(300, cursor: 'refreshed-next'));
      await tester.pump();
      oldPage.result.complete(_reviews(900));
      await tester.pump();
      await scrollToEnd(tester);
      expect(courses.requests.last.cursor, 'refreshed-next');
      courses.requests.last.result.complete(_reviews(400));
      await tester.pumpAndSettle();
    },
  );

  testWidgets(
    'legacy missing bookmark state remains unknown and offers a read retry',
    (tester) async {
      await pumpDetail(tester);
      courses.serverBookmarked = null;
      final refresh = tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.pump();
      await refresh;
      courses.requests.last.result.complete(_reviews(300));
      await tester.pumpAndSettle();
      expect(find.byTooltip('收藏'), findsNothing);
      expect(find.byTooltip('已收藏'), findsNothing);
      expect(find.byTooltip('重试'), findsOneWidget);
      final count = courses.details.length;
      courses.serverBookmarked = true;
      await tester.tap(find.byTooltip('重试'));
      await tester.pump();
      await tester.pump();
      courses.requests.last.result.complete(_reviews(400));
      await tester.pumpAndSettle();
      expect(courses.details, hasLength(count + 1));
      expect(courses.bookmarks, isEmpty);
      expect(find.byTooltip('已收藏'), findsOneWidget);
    },
  );

  testWidgets(
    'failed offering replacement cannot reuse the old rows or cursor',
    (tester) async {
      await pumpDetail(tester);
      await scrollToEnd(tester);
      final oldPage = courses.requests.last;
      await selectOffering(tester);
      courses.requests.last.result.completeError(
        const ApiException(fallbackMessage: 'replacement failed'),
      );
      oldPage.result.complete(_reviews(900));
      await tester.pump();
      final count = courses.requests.length;
      await scrollToEnd(tester);
      expect(courses.requests, hasLength(count));
      expect(find.textContaining('合成评价 10', findRichText: true), findsNothing);
      expect(find.textContaining('合成评价 90', findRichText: true), findsNothing);
      await tester.tap(find.text('重试'));
      await tester.pump();
      expect(courses.requests.last.offeringId, 902);
      expect(courses.requests.last.cursor, isNull);
      courses.requests.last.result.complete(
        _reviews(200, cursor: 'offering-next', offeringId: 902),
      );
      await tester.pump();
      await scrollToEnd(tester);
      expect(courses.requests.last.cursor, 'offering-next');
      courses.requests.last.result.complete(_reviews(300, offeringId: 902));
      await tester.pumpAndSettle();
    },
  );

  for (final boundary in ['session', 'course', 'site']) {
    testWidgets('pending refresh stops after a $boundary boundary', (
      tester,
    ) async {
      await pumpDetail(tester, bookmarked: true);
      final oldRead = Completer<CourseDetailPayload>();
      courses.pendingDetail = oldRead;
      final refresh = tester
          .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
          .onRefresh();
      await tester.pump();
      courses.pendingDetail = null;
      courses.serverBookmarked = false;
      if (boundary == 'session') {
        container.read(offlineCacheEpochProvider.notifier).invalidate();
      } else if (boundary == 'course') {
        courseId.value = 43;
      } else {
        final storage = container.read(tokenStorageProvider);
        container.updateOverrides([
          apiClientProvider.overrideWithValue(
            GfApiClient(
              dio: Dio(),
              tokenStorage: storage,
              baseUrl: 'http://other.fake.local',
            ),
          ),
          tokenStorageProvider.overrideWithValue(storage),
          courseRepositoryProvider.overrideWithValue(courses),
        ]);
      }
      await tester.pump();
      await tester.pump();
      // Initial load, the held refresh, then one fresh boundary load.
      expect(courses.details, [42, 42, boundary == 'course' ? 43 : 42]);
      final relatedCount = courses.relatedCalls.length;
      courses.requests.last.result.complete(_reviews(400));
      oldRead.complete(_detail(true));
      await tester.pump();
      await refresh;
      await tester.pumpAndSettle();
      expect(courses.relatedCalls, hasLength(relatedCount));
      expect(find.byTooltip('收藏'), findsOneWidget);
      expect(find.byTooltip('已收藏'), findsNothing);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets('saved course restores bookmark and first click removes it', (
    tester,
  ) async {
    await pumpDetail(tester, bookmarked: true);
    expect(find.byTooltip('已收藏'), findsOneWidget);
    await tester.tap(find.byTooltip('已收藏'));
    await tester.pumpAndSettle();
    expect(courses.bookmarks, [(42, false)]);
    expect(find.byTooltip('收藏'), findsOneWidget);
  });

  testWidgets(
    'bookmark write is single flight and a failure restores its prior state',
    (tester) async {
      await pumpDetail(tester, bookmarked: true);
      final write = Completer<void>();
      courses.pendingBookmark = write;
      await tester.tap(find.byTooltip('已收藏'));
      await tester.pump();
      final button = tester
          .widgetList<IconButton>(find.byType(IconButton))
          .singleWhere((button) => button.tooltip == '收藏');
      expect(button.onPressed, isNull);
      expect(courses.bookmarks, [(42, false)]);
      write.completeError(const ApiException(fallbackMessage: 'write failed'));
      await tester.pump();
      await tester.pump(const Duration(seconds: 5));
      expect(find.byTooltip('已收藏'), findsOneWidget);
    },
  );

  testWidgets(
    'new session discards pending bookmark failure and reloads authority',
    (tester) async {
      await pumpDetail(tester, bookmarked: true);
      final write = Completer<void>();
      courses.pendingBookmark = write;
      await tester.tap(find.byTooltip('已收藏'));
      await tester.pump();
      courses.serverBookmarked = false;
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      await tester.pump();
      await tester.pump();
      expect(courses.details, hasLength(2));
      courses.requests.last.result.complete(_reviews(400));
      write.completeError(
        const ApiException(fallbackMessage: 'old session failure'),
      );
      await tester.pumpAndSettle();
      expect(find.byTooltip('收藏'), findsOneWidget);
      expect(find.byTooltip('已收藏'), findsNothing);
      expect(courses.bookmarks, [(42, false)]);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('refresh cannot overwrite a newer bookmark write', (
    tester,
  ) async {
    await pumpDetail(tester, bookmarked: true);
    final write = Completer<void>();
    courses.pendingBookmark = write;
    await tester.tap(find.byTooltip('已收藏'));
    await tester.pump();
    final read = Completer<CourseDetailPayload>();
    courses.pendingDetail = read;
    final refresh = tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pump();
    write.complete();
    await tester.pump();
    read.complete(_detail(true));
    await tester.pump();
    await refresh;
    courses.requests.last.result.complete(_reviews(500));
    await tester.pumpAndSettle();
    expect(find.byTooltip('收藏'), findsOneWidget);
    expect(find.byTooltip('已收藏'), findsNothing);
  });
}
