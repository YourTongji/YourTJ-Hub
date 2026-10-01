import 'dart:async';
import 'dart:ui' show Tristate;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_test/flutter_test.dart';
// The app uses this transitive dependency; disable its batching timer in widget tests.
// ignore: depend_on_referenced_packages
import 'package:visibility_detector/visibility_detector.dart';

import 'package:core/core.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/courses/catalog_page.dart';
import 'package:forum_app/src/pages/courses/detail_page.dart';
import 'package:forum_app/src/pages/courses/review_form_sheet.dart';
import 'package:forum_app/src/pages/courses/course_common.dart';
import 'package:forum_app/src/pages/courses/review_reaction.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/providers.dart';
import 'package:ui_kit/ui_kit.dart';

import 'fixtures/page_fixtures.dart';

Finder _chipFill(Finder chip) {
  final Finder semantics = find
      .ancestor(of: chip, matching: find.byType(Semantics))
      .first;
  return find.descendant(of: semantics, matching: find.byType(Ink)).first;
}

/// 课程目录/详情/课评 UI 行为测试：自包含 FakeCourseRepository，
/// 按方法分发 canned payload（不触网），镜像仓库 wire 语义
/// （列表 hasNext 翻页、多值筛选、cursor、offeringId 聚焦、写评/有用/收藏）。

class _MemoryTokenStorage implements TokenStorage {
  _MemoryTokenStorage() : token = 'header.eyJVc2VySWQiOjEyM30.signature';

  String? token;

  @override
  Future<String?> read() async => token;

  @override
  Future<void> write(String value) async => token = value;

  @override
  Future<void> clear() async => token = null;
}

class CourseRepoCall {
  CourseRepoCall({
    this.page,
    this.keyword,
    this.offeringId,
    this.cursor,
    this.departments = const [],
    this.terms = const [],
    this.campuses = const [],
    this.instructors = const [],
    this.onlyWithReviews = false,
  });

  final int? page;
  final String? keyword;
  final List<String> departments;
  final List<String> terms;
  final List<String> campuses;
  final List<String> instructors;
  final bool onlyWithReviews;
  final int? offeringId;
  final String? cursor;
}

/// 自包含课程仓库 fake：构造参数即 canned 数据，调用记录全部入队供断言。
class FakeCourseRepository extends CourseRepository {
  FakeCourseRepository(
    super.client, {
    this.detailPayload,
    this.relatedPayload,
    this.reviewPayloads = const [],
    this.listPages = const {},
    this.summaryStatus = 'none',
    this.summaryPayload,
    this.failBookmark = false,
  }) {
    _serverReviews = {for (final review in reviewPayloads) review.id: review};
  }

  final CourseDetailPayload? detailPayload;
  final CourseRelatedResult? relatedPayload;
  final List<ReviewPayload> reviewPayloads;
  late final Map<int, ReviewPayload> _serverReviews;
  ReviewPayload serverReview(int id) => _serverReviews[id]!;
  void setServerReview(ReviewPayload review) => _serverReviews[review.id] = review;

  /// page → (courses, hasNext)。
  final Map<int, (List<CourseSummaryPayload>, bool)> listPages;
  final String summaryStatus;
  final CourseAiSummaryPayload? summaryPayload;
  bool failBookmark;
  Object? createError;

  final List<CourseRepoCall> listCalls = <CourseRepoCall>[];
  final List<CourseRepoCall> reviewCalls = <CourseRepoCall>[];
  final List<int> bookmarkCalls = <int>[];
  final List<CreateCourseReviewInput> createInputs =
      <CreateCourseReviewInput>[];
  final List<(int, bool)> helpfulCalls = <(int, bool)>[];
  final List<(String, int, bool)> reactionCalls = <(String, int, bool)>[];
  final List<(int, String, String)> reportCalls = <(int, String, String)>[];
  Object? failHelpful;
  Object? failDislike;
  bool loseHelpfulOnResponse = false;
  Object? reportError;
  Completer<void>? waitHelpful;
  Completer<void>? waitReport;

  @override
  Future<CourseListResultPayload> list({
    String? keyword,
    List<String> departments = const [],
    List<String> terms = const [],
    List<String> campuses = const [],
    List<String> instructors = const [],
    bool onlyWithReviews = false,
    String sortBy = '',
    int page = 1,
    int size = 20,
  }) async {
    listCalls.add(
      CourseRepoCall(
        page: page,
        keyword: keyword,
        departments: departments,
        terms: terms,
        campuses: campuses,
        instructors: instructors,
        onlyWithReviews: onlyWithReviews,
      ),
    );
    final (List<CourseSummaryPayload> list, bool hasNext) =
        listPages[page] ?? (const <CourseSummaryPayload>[], false);
    return CourseListResultPayload(
      list: list,
      page: page,
      size: size,
      total: list.length,
      hasNext: hasNext,
    );
  }

  @override
  Future<CourseDetailPayload> detail(int courseId) async {
    return detailPayload ??
        CourseDetailPayload(
          id: courseId,
          primaryCode: '100001',
          name: '高等数学(A)上',
          department: '数学科学学院',
          creditX10: 50,
        );
  }

  @override
  Future<CourseRelatedResult> related(int courseId) async {
    return relatedPayload ??
        const CourseRelatedResult(
          teacherOtherCourses: <RelatedCourseItem>[],
          sameCourseOtherTeachers: <RelatedCourseItem>[],
        );
  }

  @override
  Future<CourseAiSummaryResult> aiSummary(
    int courseId, {
    bool refresh = false,
    bool check = false,
  }) async {
    return CourseAiSummaryResult(
      status: summaryStatus,
      summary: summaryPayload,
    );
  }

  @override
  Future<bool> bookmark({
    required int courseId,
    required bool bookmarked,
  }) async {
    bookmarkCalls.add(courseId);
    if (failBookmark) {
      throw const ApiException(fallbackMessage: 'boom');
    }
    return true;
  }

  @override
  Future<ReviewListResult> reviews(
    int courseId, {
    int? offeringId,
    String? cursor,
    int pageSize = 20,
  }) async {
    reviewCalls.add(CourseRepoCall(offeringId: offeringId, cursor: cursor));
    if (cursor != null) {
      return const ReviewListResult(list: <ReviewPayload>[], total: 0);
    }
    return ReviewListResult(
      list: _serverReviews.values.toList(growable: false),
      total: _serverReviews.length,
    );
  }

  @override
  Future<ReviewPayload> createReview(CreateCourseReviewInput input) async {
    createInputs.add(input);
    if (createError != null) throw createError!;
    return ReviewPayload(
      id: 99,
      offeringId: input.offeringId,
      rating: input.rating,
      content: input.content,
      contentHtml: '<p>${input.content}</p>',
      author: const ReviewAuthorPayload(kind: 'anonymous', label: '匿名同学'),
      viewer: const ReviewViewerPayload(
        canEdit: false,
        canDelete: false,
        isHelpful: false,
      ),
      helpfulCount: 0,
      createdAt: '2026-09-06T12:00:00+08:00',
      updatedAt: '2026-09-06T12:00:00+08:00',
    );
  }

  @override
  Future<bool> markHelpful(int reviewId, {required bool on}) async {
    helpfulCalls.add((reviewId, on));
    reactionCalls.add(('helpful', reviewId, on));
    await waitHelpful?.future;
    if (failHelpful != null) throw failHelpful!;
    _setServerReaction(reviewId, CourseReviewReaction.helpful, on);
    if (on && loseHelpfulOnResponse) {
      loseHelpfulOnResponse = false;
      throw const ApiException(fallbackMessage: 'response lost');
    }
    return true;
  }

  @override
  Future<bool> markDislike(int reviewId, {required bool on}) async {
    reactionCalls.add(('dislike', reviewId, on));
    if (failDislike != null) throw failDislike!;
    _setServerReaction(reviewId, CourseReviewReaction.dislike, on);
    return true;
  }

  void _setServerReaction(
    int reviewId,
    CourseReviewReaction reaction,
    bool on,
  ) {
    final review = _serverReviews[reviewId];
    if (review == null) return;
    final helpful = reaction == CourseReviewReaction.helpful;
    final wasOn = helpful ? review.viewer.isHelpful : review.viewer.isDisliked;
    final count = helpful ? review.helpfulCount : review.dislikeCount;
    final nextCount = count + (on ? (wasOn ? 0 : 1) : (wasOn ? -1 : 0));
    _serverReviews[reviewId] = review.copyWith(
      viewer: review.viewer.copyWith(
        isHelpful: helpful ? on : review.viewer.isHelpful,
        isDisliked: helpful ? review.viewer.isDisliked : on,
      ),
      helpfulCount: helpful ? (nextCount < 0 ? 0 : nextCount) : review.helpfulCount,
      dislikeCount: helpful ? review.dislikeCount : (nextCount < 0 ? 0 : nextCount),
    );
  }

  @override
  Future<bool> reportReview({
    required int reviewId,
    required String reason,
    String note = '',
  }) async {
    reportCalls.add((reviewId, reason, note));
    await waitReport?.future;
    if (reportError != null) throw reportError!;
    return true;
  }
}

class FakePageRepository extends PageRepository {
  FakePageRepository(super.client);

  @override
  Future<PagePayload> fetch(String path, {Object? cancelToken}) async {
    if (path == '/courses') {
      return parsePayload(<String, dynamic>{
        'component': PageComponent.course,
        'props': <String, dynamic>{
          'departments': <String>['数学科学学院', '物理科学与工程学院'],
          'terms': <Map<String, String>>[
            <String, String>{'value': '2025-2026-1', 'label': '2025-2026 第一学期'},
          ],
          'campuses': <String>['四平路校区'],
        },
        'meta': <String, String>{'title': '课程目录'},
        'layout': minimalLayoutJson(),
        'url': '/courses',
        'version': '1.0',
      });
    }
    throw UnimplementedError('unexpected page path: $path');
  }
}

CourseSummaryPayload _course(int id, String name, {int creditX10 = 25}) {
  return CourseSummaryPayload(
    id: id,
    primaryCode: '1000${id % 1000}',
    name: name,
    department: '数学科学学院',
    creditX10: creditX10,
    teacherName: '张三',
    instructors: const <String>['张三'],
    recentTerms: const <String>['2025-2026-1', '2025-2026-2'],
    ratingAvg: 4.5,
    reviewCount: 5,
  );
}

List<CourseSummaryPayload> _catalogPageOne() {
  return <CourseSummaryPayload>[
    for (int i = 1; i <= 20; i++) _course(100 + i, '课程 A$i'),
  ];
}

List<CourseSummaryPayload> _catalogPageTwo() {
  return <CourseSummaryPayload>[
    _course(201, '课程 B1'),
    _course(202, '课程 B2'),
    _course(203, '课程 B3'),
  ];
}

CourseDetailPayload _detailPayload() {
  return CourseDetailPayload(
    id: 42,
    primaryCode: '100001',
    name: '高等数学(A)上',
    department: '数学科学学院',
    creditX10: 50,
    teacherName: '张三',
    aliases: const <String>['高数'],
    legacyNames: const <String>['高等数学上(旧)'],
    reviewScope: 'teacher',
    offerings: const <CourseOfferingPayload>[
      CourseOfferingPayload(
        id: 901,
        termCode: '2025-2026-1',
        termName: '2025-2026 第一学期',
        campus: '四平路校区',
        faculty: '数学科学学院',
        classCode: '10000101',
        className: '01班',
        instructors: <String>['张三'],
        ratingAvg: 4.5,
        reviewCount: 3,
      ),
      CourseOfferingPayload(
        id: 902,
        termCode: '2025-2026-2',
        termName: '2025-2026 第二学期',
        campus: '四平路校区',
        className: '02班',
        instructors: <String>['张三'],
        ratingAvg: 4.0,
        reviewCount: 2,
      ),
    ],
    ratingAvg: 4.5,
    reviewCount: 5,
    ratingDistribution: const <int>[0, 0, 1, 1, 3],
  );
}

CourseRelatedResult _relatedPayload() {
  return CourseRelatedResult(
    teacherOtherCourses: const <RelatedCourseItem>[
      RelatedCourseItem(
        id: 43,
        primaryCode: '100002',
        name: '线性代数',
        department: '数学科学学院',
        teacherName: '张三',
        ratingAvg: 4.0,
        ratingCount: 2,
        reviewCount: 2,
      ),
    ],
    sameCourseOtherTeachers: const <RelatedCourseItem>[
      RelatedCourseItem(
        id: 47,
        primaryCode: '100001',
        name: '高等数学(A)上(李四)',
        department: '数学科学学院',
        teacherName: '李四',
        ratingAvg: 4.0,
        ratingCount: 1,
        reviewCount: 1,
      ),
    ],
    lineage: const <RelationItem>[
      RelationItem(
        relationId: 11,
        fromCourseId: 41,
        fromName: '高等数学(A)上',
        toCourseId: 42,
        toName: '高等数学(A)上',
        relationType: 'EQUIVALENT',
        status: 'merged',
        direction: 'to',
      ),
    ],
  );
}

/// Review bodies are rendered from server HTML, so assertions look at rich
/// text instead of a plain [Text] widget.
Finder _reviewHtml(String text) =>
    find.textContaining(text, findRichText: true);

List<ReviewPayload> _reviewPayloads() {
  return <ReviewPayload>[
    const ReviewPayload(
      id: 1,
      offeringId: 901,
      rating: 5,
      content: '好课',
      contentHtml: '<p>好课</p>',
      author: ReviewAuthorPayload(kind: 'member', label: 'bob'),
      viewer: ReviewViewerPayload(
        canEdit: false,
        canDelete: false,
        isHelpful: false,
      ),
      helpfulCount: 0,
      createdAt: '2026-09-01T08:00:00+08:00',
      updatedAt: '2026-09-01T08:00:00+08:00',
    ),
    const ReviewPayload(
      id: 2,
      offeringId: 901,
      rating: null,
      content: '历史评价',
      contentHtml: '<p>历史评价</p>',
      author: ReviewAuthorPayload(kind: 'legacy', label: '历史匿名评价'),
      viewer: ReviewViewerPayload(
        canEdit: false,
        canDelete: false,
        isHelpful: false,
      ),
      helpfulCount: 3,
      createdAt: '2026-08-01T10:00:00+08:00',
      updatedAt: '2026-08-01T10:00:00+08:00',
    ),
    const ReviewPayload(
      id: 4,
      offeringId: 902,
      rating: 4,
      content: '很不错',
      contentHtml: '<p>很不错</p>',
      author: ReviewAuthorPayload(kind: 'member', label: 'alice'),
      viewer: ReviewViewerPayload(
        canEdit: true,
        canDelete: true,
        isHelpful: false,
      ),
      helpfulCount: 2,
      createdAt: '2026-08-02T10:00:00+08:00',
      updatedAt: '2026-08-02T10:00:00+08:00',
    ),
  ];
}

Widget _app(
  ProviderContainer container,
  Widget home, {
  Locale locale = const Locale('zh'),
  double textScale = 1,
}) {
  return UncontrolledProviderScope(
    container: container,
    child: MaterialApp(
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      locale: locale,
      builder: (context, child) => MediaQuery(
        data: MediaQuery.of(
          context,
        ).copyWith(textScaler: TextScaler.linear(textScale)),
        child: child!,
      ),
      home: home,
    ),
  );
}

ProviderContainer _container({
  required FakeCourseRepository courseRepo,
  PageRepository? pageRepo,
}) {
  final ProviderContainer container = ProviderContainer(
    overrides: <Override>[
      courseRepositoryProvider.overrideWithValue(courseRepo),
      tokenStorageProvider.overrideWithValue(_MemoryTokenStorage()),
      if (pageRepo != null) pageRepositoryProvider.overrideWithValue(pageRepo),
    ],
  );
  addTearDown(container.dispose);
  return container;
}

GfApiClient _client() => GfApiClient(
  dio: Dio(),
  tokenStorage: _MemoryTokenStorage(),
  baseUrl: 'http://fake.local',
);

void _replaceReviewEditorText(WidgetTester tester, String text) {
  final controller = tester
      .widget<QuillEditor>(find.byType(QuillEditor))
      .controller;
  controller.replaceText(
    0,
    controller.document.length - 1,
    text,
    TextSelection.collapsed(offset: text.length),
  );
}

Future<void> _pumpStandaloneReviewForm(
  WidgetTester tester,
  FakeCourseRepository repository, {
  List<CourseOfferingPayload> offerings = const <CourseOfferingPayload>[],
}) async {
  await tester.pumpWidget(
    _app(
      _container(courseRepo: repository),
      Scaffold(
        body: Builder(
          builder: (context) => TextButton(
            onPressed: () => showGfBottomSheet<void>(
              context,
              height: 600,
              keyboardAware: true,
              barrierDismissible: false,
              enableDrag: false,
              builder: (_) => CourseReviewFormSheet(
                pageContext: context,
                repository: repository,
                offerings: offerings,
                initialOfferingId: offerings.firstOrNull?.id,
              ),
            ),
            child: const Text('Open review form'),
          ),
        ),
      ),
    ),
  );
  await tester.tap(find.text('Open review form'));
  await tester.pumpAndSettle();
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  VisibilityDetectorController.instance.updateInterval = Duration.zero;

  testWidgets('course catalog starts with the forwarded global query', (
    tester,
  ) async {
    final course = FakeCourseRepository(_client());
    final container = _container(
      courseRepo: course,
      pageRepo: FakePageRepository(_client()),
    );
    await tester.pumpWidget(
      _app(container, const CourseCatalogPage(initialQuery: '高等数学')),
    );
    await tester.pumpAndSettle();
    expect(course.listCalls.single.keyword, '高等数学');
    expect(
      tester.widget<TextField>(find.byType(TextField)).controller!.text,
      '高等数学',
    );
  });

  group('课程目录', () {
    testWidgets('渲染列表行，滚动到底触发 hasNext 翻页', (tester) async {
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        listPages: <int, (List<CourseSummaryPayload>, bool)>{
          1: (_catalogPageOne(), true),
          2: (_catalogPageTwo(), false),
        },
      );
      final ProviderContainer container = _container(
        courseRepo: course,
        pageRepo: FakePageRepository(_client()),
      );
      await tester.pumpWidget(_app(container, const CourseCatalogPage()));
      await tester.pumpAndSettle();

      expect(find.text('课程 A1'), findsOneWidget);
      expect(
        tester
            .getSize(
              find
                  .ancestor(
                    of: find.text('+1').first,
                    matching: find.byType(InkWell),
                  )
                  .first,
            )
            .width,
        lessThan(100),
      );
      // ListView 懒加载只构建可视行，A20 未必在首帧内；靠滚动断言翻页。
      expect(find.text('4.5'), findsWidgets);

      // 滚动到底部触发加载更多（hasNext=true → page 2）。
      await tester.drag(
        find.byKey(const Key('course-catalog-list')),
        const Offset(0, -3000),
      );
      await tester.pumpAndSettle();

      expect(find.text('课程 B1'), findsOneWidget);
      expect(course.listCalls.map((CourseRepoCall c) => c.page), contains(2));
      expect(
        course.listCalls.map((CourseRepoCall c) => c.page),
        isNot(contains(3)),
      );
    });

    testWidgets('教师筛选把多值传给仓库（并集语义）', (tester) async {
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        listPages: <int, (List<CourseSummaryPayload>, bool)>{
          1: (_catalogPageOne(), false),
        },
      );
      final ProviderContainer container = _container(
        courseRepo: course,
        pageRepo: FakePageRepository(_client()),
      );
      await tester.pumpWidget(_app(container, const CourseCatalogPage()));
      await tester.pumpAndSettle();

      // 打开「教师」筛选（自由输入多值，底部 sheet 逐条添加）。
      await tester.tap(find.text('教师'));
      await tester.pumpAndSettle();

      await tester.enterText(find.byType(TextField).last, '张三');
      await tester.tap(find.byTooltip('添加'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).last, '李四');
      await tester.tap(find.byTooltip('添加'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('完成'));
      await tester.pumpAndSettle();

      final CourseRepoCall last = course.listCalls.last;
      expect(last.instructors, <String>['张三', '李四']);
      // 只看有评价开关也反映到查询参数。
      await tester.tap(find.text('只看有评价'));
      await tester.pumpAndSettle();
      expect(course.listCalls.last.onlyWithReviews, isTrue);
    });

    testWidgets(
      'selected filter chip updates immediately and paints its state',
      (tester) async {
        final FakeCourseRepository course = FakeCourseRepository(
          _client(),
          listPages: <int, (List<CourseSummaryPayload>, bool)>{
            1: (_catalogPageOne(), false),
          },
        );
        final ProviderContainer container = _container(
          courseRepo: course,
          pageRepo: FakePageRepository(_client()),
        );
        await tester.pumpWidget(_app(container, const CourseCatalogPage()));
        await tester.pumpAndSettle();
        final SemanticsHandle semantics = tester.ensureSemantics();
        try {
          final Finder chip = find
              .ancestor(of: find.text('只看有评价'), matching: find.byType(InkWell))
              .first;
          final Finder fill = _chipFill(chip);
          expect(tester.getSize(chip).height, greaterThanOrEqualTo(44));
          expect(tester.getSize(chip).width, greaterThanOrEqualTo(44));
          final Color? before =
              (tester.widget<Ink>(fill).decoration! as BoxDecoration).color;
          final Color expected = GfTheme.colorsOf(
            tester.element(chip),
          ).primary.withValues(alpha: 0.1);
          final SemanticsData beforeSemantics = tester
              .getSemantics(chip)
              .getSemanticsData();
          expect(beforeSemantics.flagsCollection.isButton, isTrue);
          expect(beforeSemantics.flagsCollection.isEnabled, Tristate.isTrue);
          expect(beforeSemantics.flagsCollection.isSelected, Tristate.isFalse);

          final Rect chipRect = tester.getRect(chip);
          await tester.tapAt(Offset(chipRect.left + 2, chipRect.center.dy));
          await tester.pump();
          expect(course.listCalls.last.onlyWithReviews, isTrue);
          await tester.pump(const Duration(milliseconds: 80));
          final Color? middle =
              (tester.widget<Ink>(fill).decoration! as BoxDecoration).color;
          expect(middle, isNot(before));
          expect(middle, isNot(expected));
          await tester.pumpAndSettle();
          final Color? after =
              (tester.widget<Ink>(fill).decoration! as BoxDecoration).color;
          expect(after, expected);
          final SemanticsData afterSemantics = tester
              .getSemantics(chip)
              .getSemanticsData();
          expect(afterSemantics.flagsCollection.isSelected, Tristate.isTrue);
        } finally {
          semantics.dispose();
        }
      },
    );

    testWidgets('selected filter chip honors reduced motion', (tester) async {
      tester.platformDispatcher.accessibilityFeaturesTestValue =
          const FakeAccessibilityFeatures(disableAnimations: true);
      addTearDown(
        tester.platformDispatcher.clearAccessibilityFeaturesTestValue,
      );
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        listPages: <int, (List<CourseSummaryPayload>, bool)>{
          1: (_catalogPageOne(), false),
        },
      );
      final ProviderContainer container = _container(
        courseRepo: course,
        pageRepo: FakePageRepository(_client()),
      );
      await tester.pumpWidget(_app(container, const CourseCatalogPage()));
      await tester.pumpAndSettle();

      final Finder chip = find
          .ancestor(of: find.text('只看有评价'), matching: find.byType(InkWell))
          .first;
      final Finder fill = _chipFill(chip);
      final Color? before =
          (tester.widget<Ink>(fill).decoration! as BoxDecoration).color;
      await tester.tap(chip);
      await tester.pump();
      expect(course.listCalls.last.onlyWithReviews, isTrue);
      final Color? after =
          (tester.widget<Ink>(fill).decoration! as BoxDecoration).color;
      expect(after, isNot(before));
    });

    testWidgets('filter selection settles when reduced motion turns on', (
      tester,
    ) async {
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        listPages: <int, (List<CourseSummaryPayload>, bool)>{
          1: (_catalogPageOne(), false),
        },
      );
      final ProviderContainer container = _container(
        courseRepo: course,
        pageRepo: FakePageRepository(_client()),
      );
      await tester.pumpWidget(_app(container, const CourseCatalogPage()));
      await tester.pumpAndSettle();

      final Finder chip = find
          .ancestor(of: find.text('只看有评价'), matching: find.byType(InkWell))
          .first;
      final Finder fill = _chipFill(chip);
      final Element chipElement = tester.element(chip);
      final Color expected = GfTheme.colorsOf(
        tester.element(chip),
      ).primary.withValues(alpha: 0.1);
      final Color before =
          (tester.widget<Ink>(fill).decoration! as BoxDecoration).color!;
      await tester.tap(chip);
      await tester.pump();
      expect(course.listCalls.last.onlyWithReviews, isTrue);
      await tester.pump(const Duration(milliseconds: 40));
      final Color middle =
          (tester.widget<Ink>(fill).decoration! as BoxDecoration).color!;
      expect(middle, isNot(before));
      expect(middle, isNot(expected));
      tester.platformDispatcher.accessibilityFeaturesTestValue =
          const FakeAccessibilityFeatures(disableAnimations: true);
      addTearDown(
        tester.platformDispatcher.clearAccessibilityFeaturesTestValue,
      );
      await tester.pump();
      expect(
        (tester.widget<Ink>(fill).decoration! as BoxDecoration).color,
        expected,
      );
      expect(identical(chipElement, tester.element(chip)), isTrue);
      expect(
        tester
            .getSemantics(chip)
            .getSemanticsData()
            .hasAction(SemanticsAction.tap),
        isTrue,
      );
    });

    testWidgets('term controls grow with text and keep a 44 pixel target', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 1200);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final course = FakeCourseRepository(
        _client(),
        listPages: {
          1: ([_course(1, '课程')], false),
        },
      );
      final container = _container(
        courseRepo: course,
        pageRepo: FakePageRepository(_client()),
      );
      await tester.pumpWidget(
        _app(container, const CourseCatalogPage(), textScale: 2),
      );
      await tester.pumpAndSettle();
      final expand = find
          .ancestor(of: find.text('+1'), matching: find.byType(InkWell))
          .first;
      expect(tester.getSize(expand).height, greaterThanOrEqualTo(44));
      expect(tester.getSize(expand).width, greaterThanOrEqualTo(44));
      await tester.ensureVisible(expand);
      await tester.tap(expand);
      await tester.pumpAndSettle();
      expect(find.text('+1'), findsNothing);
      expect(find.text('25春'), findsOneWidget);
      expect(find.text('25秋'), findsOneWidget);
      expect(find.text('收起'), findsOneWidget);
      expect(tester.takeException(), isNull);

      final Finder collapse = find
          .ancestor(of: find.text('收起'), matching: find.byType(InkWell))
          .first;
      await tester.tap(collapse);
      await tester.pumpAndSettle();
      expect(find.text('收起'), findsNothing);
      expect(find.text('+1'), findsOneWidget);
    });

    testWidgets('course identity leads metadata on a 320px phone at 2x', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 1200);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      const name = '跨学科计算方法与城市系统设计专题研讨课程';
      final course = FakeCourseRepository(
        _client(),
        listPages: {
          1: ([_course(1, name)], false),
        },
      );
      await tester.pumpWidget(
        _app(
          _container(
            courseRepo: course,
            pageRepo: FakePageRepository(_client()),
          ),
          const CourseCatalogPage(),
          textScale: 2,
        ),
      );
      await tester.pumpAndSettle();

      final nameRect = tester.getRect(find.text(name));
      final teacherRect = tester.getRect(find.text('张三 · 数学科学学院'));
      final codeRect = tester.getRect(find.text('10001'));
      expect(nameRect.top, lessThan(teacherRect.top));
      expect(teacherRect.top, lessThan(codeRect.top));
      expect(nameRect.right, lessThanOrEqualTo(320));
      expect(
        tester.renderObject<RenderParagraph>(find.text(name)).didExceedMaxLines,
        isFalse,
      );
      expect(
        tester
            .renderObject<RenderParagraph>(find.text('张三 · 数学科学学院'))
            .didExceedMaxLines,
        isFalse,
      );
      expect(tester.takeException(), isNull);
    });

    testWidgets('teacher removal has a named 44 pixel control', (tester) async {
      final course = FakeCourseRepository(_client());
      final container = _container(
        courseRepo: course,
        pageRepo: FakePageRepository(_client()),
      );
      await tester.pumpWidget(_app(container, const CourseCatalogPage()));
      await tester.pumpAndSettle();
      await tester.tap(find.text('教师'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).last, '张三');
      await tester.tap(find.byTooltip('添加'));
      await tester.pumpAndSettle();
      final remove = find.byTooltip('删除 张三');
      expect(remove, findsOneWidget);
      expect(tester.getSize(remove).height, greaterThanOrEqualTo(44));
      expect(tester.getSize(remove).width, greaterThanOrEqualTo(44));
      await tester.tap(remove);
      await tester.pumpAndSettle();
      expect(find.text('张三'), findsNothing);
      await tester.tap(find.text('完成'));
      await tester.pumpAndSettle();
      expect(course.listCalls.last.instructors, isEmpty);
    });
  });

  group('课程详情', () {
    Future<void> pumpDetail(
      WidgetTester tester,
      FakeCourseRepository course, {
      int? focusOfferingId,
      int? focusReviewId,
      Size size = const Size(800, 3000),
    }) async {
      // 加高画布让整页一次构建，避免 ListView 懒加载影响断言。
      tester.view.physicalSize = size;
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      addTearDown(() async {
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump(const Duration(milliseconds: 500));
      });
      final ProviderContainer container = _container(courseRepo: course);
      await tester.pumpWidget(
        _app(
          container,
          CourseDetailPage(
            courseId: 42,
            focusOfferingId: focusOfferingId,
            focusReviewId: focusReviewId,
          ),
        ),
      );
      await tester.pumpAndSettle();
    }

    testWidgets('review actions have one label and 44 pixel touch targets', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(390, 2000);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: [_reviewPayloads().last],
      );
      await tester.pumpWidget(
        _app(
          _container(courseRepo: course),
          const CourseDetailPage(courseId: 42),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('编辑'), findsOneWidget);
      expect(find.text('删除'), findsOneWidget);
      for (final label in ['编辑', '删除', '2 有用']) {
        final action = find
            .ancestor(of: find.text(label), matching: find.byType(TextButton))
            .first;
        expect(tester.getSize(action).height, greaterThanOrEqualTo(44));
      }
      await tester.tap(find.text('2 有用'));
      await tester.pumpAndSettle();
      expect(course.helpfulCalls, [(4, true)]);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 500));
    });

    testWidgets('switching review reactions deletes the opposite first', (
      tester,
    ) async {
      final ReviewPayload disliked = _reviewPayloads().first.copyWith(
        viewer: _reviewPayloads().first.viewer.copyWith(isDisliked: true),
        dislikeCount: 1,
      );
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: [disliked],
      );
      await pumpDetail(tester, course, size: const Size(800, 1800));

      await tester.tap(find.text('0 有用'));
      await tester.pumpAndSettle();

      expect(course.reactionCalls, [
        ('dislike', disliked.id, false),
        ('helpful', disliked.id, true),
      ]);
      expect(find.text('1 有用'), findsOneWidget);
      expect(find.text('0 无用'), findsOneWidget);
      expect(
        find.byWidgetPredicate(
          (widget) =>
              widget is Semantics && widget.properties.toggled == true,
        ),
        findsOneWidget,
      );
    });

    testWidgets(
      'failed reaction switch restores server and keeps local state',
      (tester) async {
        final ReviewPayload disliked = _reviewPayloads().first.copyWith(
          viewer: _reviewPayloads().first.viewer.copyWith(isDisliked: true),
          dislikeCount: 1,
        );
        final course = FakeCourseRepository(
          _client(),
          reviewPayloads: [disliked],
        )..failHelpful = const ApiException(fallbackMessage: 'write failed');
        await pumpDetail(tester, course, size: const Size(800, 1800));

        await tester.tap(find.text('0 有用'));
        await tester.pumpAndSettle();

        expect(course.reactionCalls, [
          ('dislike', disliked.id, false),
          ('helpful', disliked.id, true),
          ('helpful', disliked.id, false),
          ('dislike', disliked.id, true),
        ]);
        expect(course.reviewCalls, hasLength(2));
        expect(course.serverReview(disliked.id).viewer.isDisliked, isTrue);
        expect(find.text('0 有用'), findsOneWidget);
        expect(find.text('1 无用'), findsOneWidget);
      },
    );

    testWidgets('lost reaction response is cleaned up and reconciled', (
      tester,
    ) async {
      final ReviewPayload disliked = _reviewPayloads().first.copyWith(
        viewer: _reviewPayloads().first.viewer.copyWith(isDisliked: true),
        dislikeCount: 1,
      );
      final course = FakeCourseRepository(
        _client(),
        reviewPayloads: [disliked],
      )..loseHelpfulOnResponse = true;
      await pumpDetail(tester, course, size: const Size(800, 1800));

      await tester.tap(find.text('0 有用'));
      await tester.pumpAndSettle();

      expect(course.reactionCalls, [
        ('dislike', disliked.id, false),
        ('helpful', disliked.id, true),
        ('helpful', disliked.id, false),
        ('dislike', disliked.id, true),
      ]);
      expect(course.reviewCalls, hasLength(2));
      expect(course.serverReview(disliked.id).viewer.isDisliked, isTrue);
      expect(course.serverReview(disliked.id).viewer.isHelpful, isFalse);
      expect(find.text('0 有用'), findsOneWidget);
      expect(find.text('1 无用'), findsOneWidget);
    });

    testWidgets('reaction refresh uses server counts changed during write', (
      tester,
    ) async {
      final initial = _reviewPayloads().first;
      final course = FakeCourseRepository(
        _client(),
        reviewPayloads: [initial],
      )..waitHelpful = Completer<void>();
      await pumpDetail(tester, course, size: const Size(800, 1800));

      await tester.tap(find.text('0 有用'));
      await tester.pump();
      course.setServerReview(initial.copyWith(helpfulCount: 10));
      course.waitHelpful!.complete();
      await tester.pumpAndSettle();

      expect(course.reviewCalls, hasLength(2));
      expect(find.text('11 有用'), findsOneWidget);
    });

    testWidgets(
      'report requires an explicit reason and preserves a failed draft',
      (tester) async {
        final course = FakeCourseRepository(
          _client(),
          reviewPayloads: [_reviewPayloads().first],
        )..reportError = const ApiException(
          fallbackMessage: 'report failed',
          messageCode: 'review.report.failed',
        );
        await pumpDetail(tester, course, size: const Size(500, 1200));

        await tester.tap(find.text('举报内容'));
        await tester.pumpAndSettle();
        final l10n = AppLocalizations.of(
          tester.element(find.byType(DropdownButtonFormField<String>)),
        );
        final submit = find.widgetWithText(
          FilledButton,
          l10n.topicReportSubmit,
        );
        expect(tester.widget<FilledButton>(submit).onPressed, isNull);
        await tester.tap(find.byType(DropdownButtonFormField<String>));
        await tester.pumpAndSettle();
        await tester.tap(find.text('垃圾信息').last);
        await tester.pumpAndSettle();
        await tester.enterText(find.byType(TextField).last, '  same link  ');
        await tester.ensureVisible(submit);
        await tester.tap(submit);
        await tester.pumpAndSettle();

        expect(course.reportCalls, [(1, 'spam', 'same link')]);
        expect(find.text('举报评价失败，请稍后重试。'), findsOneWidget);
        expect(
          tester
              .widget<TextField>(find.byType(TextField).last)
              .controller
              ?.text,
          '  same link  ',
        );
      },
    );

    testWidgets('report note limit counts Unicode scalar values', (
      tester,
    ) async {
      final course = FakeCourseRepository(
        _client(),
        reviewPayloads: [_reviewPayloads().first],
      );
      await pumpDetail(tester, course, size: const Size(500, 1200));
      await tester.tap(find.text('举报内容'));
      await tester.pumpAndSettle();
      await tester.tap(find.byType(DropdownButtonFormField<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.text('垃圾信息').last);
      await tester.pumpAndSettle();
      final input = find.byType(TextField).last;
      await tester.enterText(input, '🙂' * 301);
      await tester.pump();
      final note = tester.widget<TextField>(input).controller!.text;
      expect(note.runes.length, 300);
      expect(note, '🙂' * 300);
    });

    testWidgets('review form validates each required field before writing', (
      tester,
    ) async {
      final offerings = _detailPayload().offerings!;
      final noRatingRepo = FakeCourseRepository(_client());
      await _pumpStandaloneReviewForm(
        tester,
        noRatingRepo,
        offerings: offerings,
      );
      final form = find.byType(CourseReviewFormSheet);
      final l10n = AppLocalizations.of(tester.element(form));
      final copy = CourseCopy(l10n);

      _replaceReviewEditorText(tester, 'A useful review');
      await tester.tap(find.text(l10n.reviewSubmit));
      await tester.pumpAndSettle();
      expect(find.text(copy.ratingRequired), findsOneWidget);
      expect(noRatingRepo.createInputs, isEmpty);
      await tester.pumpWidget(const SizedBox.shrink());

      final emptyRepo = FakeCourseRepository(_client());
      await _pumpStandaloneReviewForm(
        tester,
        emptyRepo,
        offerings: offerings,
      );
      await tester.tap(find.byKey(const ValueKey('review-rating-5')));
      await tester.tap(find.text(l10n.reviewSubmit));
      await tester.pumpAndSettle();
      expect(find.text(copy.contentRequired), findsOneWidget);
      expect(emptyRepo.createInputs, isEmpty);
      await tester.pumpWidget(const SizedBox.shrink());

      final tooLongRepo = FakeCourseRepository(_client());
      await _pumpStandaloneReviewForm(
        tester,
        tooLongRepo,
        offerings: offerings,
      );
      await tester.tap(find.byKey(const ValueKey('review-rating-5')));
      _replaceReviewEditorText(tester, 'x' * 2001);
      await tester.pump();
      await tester.tap(find.text(l10n.reviewSubmit));
      await tester.pumpAndSettle();
      expect(find.text(l10n.courseReviewContentLimitError), findsOneWidget);
      expect(tooLongRepo.createInputs, isEmpty);
      await tester.pumpWidget(const SizedBox.shrink());

      final noOfferingRepo = FakeCourseRepository(_client());
      await _pumpStandaloneReviewForm(tester, noOfferingRepo);
      await tester.tap(find.byKey(const ValueKey('review-rating-5')));
      _replaceReviewEditorText(tester, 'Valid body');
      await tester.pump();
      await tester.tap(find.text(l10n.reviewSubmit));
      await tester.pumpAndSettle();
      expect(find.text(copy.operationFailed), findsOneWidget);
      expect(noOfferingRepo.createInputs, isEmpty);
    });

    testWidgets('review length follows Unicode scalar count', (tester) async {
      final repo = FakeCourseRepository(_client());
      await _pumpStandaloneReviewForm(
        tester,
        repo,
        offerings: _detailPayload().offerings!,
      );
      final l10n = AppLocalizations.of(
        tester.element(find.byType(CourseReviewFormSheet)),
      );
      await tester.tap(find.byKey(const ValueKey('review-rating-5')));
      final content = '🙂' * 1100;
      _replaceReviewEditorText(tester, content);
      await tester.pump();
      await tester.tap(find.text(l10n.reviewSubmit));
      await tester.pumpAndSettle();

      expect(repo.createInputs.single.content, content);
    });

    testWidgets(
      'empty review applies a template without a replacement prompt',
      (tester) async {
        final course = FakeCourseRepository(
          _client(),
          detailPayload: _detailPayload(),
        );
        await pumpDetail(tester, course);
        await tester.tap(find.text('写课评'));
        await tester.pumpAndSettle();
        final l10n = AppLocalizations.of(
          tester.element(find.byType(CourseReviewFormSheet)),
        );

        await tester.tap(find.byKey(const Key('course-review-templates')));
        await tester.pumpAndSettle();
        await tester.tap(find.text(l10n.courseReviewTemplateComprehensiveName));
        await tester.pumpAndSettle();

        expect(find.text(l10n.courseReviewTemplateReplaceTitle), findsNothing);
        final editor = tester.widget<QuillEditor>(find.byType(QuillEditor));
        expect(editor.controller.document.toPlainText(), contains('课程内容'));
        await tester.pumpWidget(const SizedBox.shrink());
      },
    );

    testWidgets('populated review template asks before replacing content', (
      tester,
    ) async {
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
      );
      await pumpDetail(tester, course);
      await tester.tap(find.text('写课评'));
      await tester.pumpAndSettle();
      final l10n = AppLocalizations.of(
        tester.element(find.byType(CourseReviewFormSheet)),
      );
      _replaceReviewEditorText(tester, 'Existing review');
      await tester.pump();

      Future<void> chooseComprehensive() async {
        await tester.tap(find.byKey(const Key('course-review-templates')));
        await tester.pumpAndSettle();
        await tester.tap(find.text(l10n.courseReviewTemplateComprehensiveName));
        await tester.pumpAndSettle();
      }

      await chooseComprehensive();
      expect(find.text(l10n.courseReviewTemplateReplaceTitle), findsOneWidget);
      await tester.tap(find.text(l10n.courseReviewTemplateKeepEditing));
      await tester.pumpAndSettle();
      expect(
        tester
            .widget<QuillEditor>(find.byType(QuillEditor))
            .controller
            .document
            .toPlainText(),
        contains('Existing review'),
      );

      await chooseComprehensive();
      await tester.tap(find.text(l10n.courseReviewTemplateApply));
      await tester.pumpAndSettle();
      final content = tester
          .widget<QuillEditor>(find.byType(QuillEditor))
          .controller
          .document
          .toPlainText();
      expect(content, contains('课程内容'));
      expect(content, isNot(contains('Existing review')));
      await tester.pumpWidget(const SizedBox.shrink());
    });

    testWidgets(
      'editing review keeps its controller through keyboard resize and selection is clean',
      (tester) async {
        final course = FakeCourseRepository(
          _client(),
          detailPayload: _detailPayload(),
          reviewPayloads: _reviewPayloads(),
        );
        await pumpDetail(tester, course);
        await tester.tap(find.text('编辑').first);
        await tester.pumpAndSettle();
        final l10n = AppLocalizations.of(
          tester.element(find.byType(CourseReviewFormSheet)),
        );
        expect(find.text(l10n.courseReviewTemplateQuickName), findsNothing);
        final before = tester
            .widget<QuillEditor>(find.byType(QuillEditor))
            .controller;
        before.updateSelection(
          const TextSelection.collapsed(offset: 2),
          ChangeSource.local,
        );
        tester.view.viewInsets = const FakeViewPadding(bottom: 280);
        await tester.pumpAndSettle();
        final after = tester
            .widget<QuillEditor>(find.byType(QuillEditor))
            .controller;
        expect(identical(before, after), isTrue);
        expect(after.document.toPlainText(), contains('很不错'));
        await tester.tap(find.text(l10n.commonCancel));
        await tester.pumpAndSettle();
        expect(find.text(l10n.settingsUnsavedTitle), findsNothing);
        expect(find.byType(CourseReviewFormSheet), findsNothing);
        await tester.pumpWidget(const SizedBox.shrink());
      },
    );

    testWidgets(
      'review editor and actions fit a 320 pixel viewport at large text',
      (tester) async {
        tester.view.physicalSize = const Size(320, 844);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.reset);
        final course = FakeCourseRepository(
          _client(),
          detailPayload: _detailPayload(),
        );
        await tester.pumpWidget(
          _app(
            _container(courseRepo: course),
            const CourseDetailPage(courseId: 42),
            textScale: 2,
          ),
        );
        await tester.pumpAndSettle();
        await tester.tap(find.text('写课评'));
        await tester.pumpAndSettle();
        final l10n = AppLocalizations.of(
          tester.element(find.byType(CourseReviewFormSheet)),
        );
        tester.view.viewInsets = const FakeViewPadding(bottom: 260);
        await tester.pumpAndSettle();
        for (final label in [l10n.commonCancel, l10n.reviewSubmit]) {
          final action = find.text(label).last;
          await tester.ensureVisible(action);
          await tester.pumpAndSettle();
          expect(tester.getRect(action).bottom, lessThanOrEqualTo(844));
        }
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
      },
    );

    testWidgets(
      'review actions wrap on a narrow phone with German large text',
      (tester) async {
        tester.view.physicalSize = const Size(320, 3000);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.reset);
        final course = FakeCourseRepository(
          _client(),
          reviewPayloads: [_reviewPayloads().last],
        );
        await tester.pumpWidget(
          _app(
            _container(courseRepo: course),
            const CourseDetailPage(courseId: 42),
            locale: const Locale('de'),
            textScale: 2,
          ),
        );
        await tester.pumpAndSettle();
        final l = AppLocalizations.of(
          tester.element(find.byType(CourseDetailPage)),
        );
        expect(find.text(l.commonEdit), findsOneWidget);
        final actions = find
            .ancestor(of: find.text(l.commonEdit), matching: find.byType(Wrap))
            .first;
        expect(actions, findsOneWidget);
        expect(tester.getRect(actions).right, lessThanOrEqualTo(320));
        expect(tester.takeException(), isNull);
      },
    );

    testWidgets('bottom safe area does not cover the last related course', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      tester.view.padding = FakeViewPadding(bottom: 34);
      addTearDown(tester.view.reset);
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        relatedPayload: _relatedPayload(),
        reviewPayloads: _reviewPayloads(),
      );
      await tester.pumpWidget(
        _app(
          _container(courseRepo: course),
          const CourseDetailPage(courseId: 42),
        ),
      );
      await tester.pumpAndSettle();
      final list = find.byType(CustomScrollView).first;
      await tester.drag(list, const Offset(0, -6000));
      await tester.pumpAndSettle();
      // Lazy rows revise the scroll extent after layout, especially at larger
      // font sizes. Settle at the real end before checking dock clearance.
      final position = tester
          .widget<CustomScrollView>(list)
          .controller!
          .position;
      for (
        var attempt = 0;
        attempt < 4 && position.extentAfter > 0;
        attempt++
      ) {
        position.jumpTo(position.maxScrollExtent);
        await tester.pumpAndSettle();
      }
      expect(position.extentAfter, 0);
      final lastRow = find.text('等价');
      final dock = find
          .ancestor(of: find.text('写课评'), matching: find.byType(ColoredBox))
          .first;
      expect(
        tester.getBottomLeft(lastRow).dy,
        lessThanOrEqualTo(tester.getTopLeft(dock).dy),
      );
    });

    testWidgets('report sheet disables duplicate submissions', (tester) async {
      final course = FakeCourseRepository(
        _client(),
        reviewPayloads: [_reviewPayloads().first],
      )..waitReport = Completer<void>();
      await pumpDetail(tester, course, size: const Size(500, 1200));

      await tester.tap(find.text('举报内容'));
      await tester.pumpAndSettle();
      final l10n = AppLocalizations.of(
        tester.element(find.byType(DropdownButtonFormField<String>)),
      );
      await tester.tap(find.byType(DropdownButtonFormField<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.text('垃圾信息').last);
      await tester.pumpAndSettle();
      final submit = find.widgetWithText(FilledButton, l10n.topicReportSubmit);
      await tester.tap(submit);
      await tester.pump();
      expect(tester.widget<FilledButton>(submit).onPressed, isNull);
      await tester.tap(submit);
      await tester.pump();
      expect(course.reportCalls, [(1, 'spam', '')]);

      course.waitReport!.complete();
      await tester.pumpAndSettle();
      expect(find.byType(DropdownButtonFormField<String>), findsNothing);
    });

    testWidgets('展示评分分布、开课班级、相关课程与沿革', (tester) async {
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        relatedPayload: _relatedPayload(),
        reviewPayloads: _reviewPayloads(),
      );
      await pumpDetail(tester, course);

      // 头部：名称/课号/别名/原名标注。
      expect(find.text('高等数学(A)上'), findsWidgets);
      expect(find.text('100001'), findsOneWidget);
      expect(find.textContaining('别名'), findsOneWidget);
      expect(find.textContaining('原名'), findsOneWidget);

      // 评分：均分 + 分布条标题。
      expect(find.text('课程评分'), findsOneWidget);
      expect(find.text('4.5'), findsWidgets);

      // 开课班级：学期分组 + 班级行。
      expect(find.text('开课班级'), findsOneWidget);
      expect(find.text('2025-2026 第一学期'), findsOneWidget);
      expect(find.text('01班 · 10000101'), findsOneWidget);
      expect(find.text('02班'), findsOneWidget);

      // 相关课程 + 沿革 chips。
      expect(find.text('线性代数'), findsOneWidget);
      expect(find.text('课程沿革'), findsOneWidget);
      expect(find.text('等价'), findsOneWidget);
    });

    testWidgets('点击教学班聚焦课评（fake 收到 offeringId）', (tester) async {
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: _reviewPayloads(),
      );
      await pumpDetail(tester, course);

      expect(course.reviewCalls.first.offeringId, isNull);

      await tester.tap(find.text('02班'));
      await tester.pumpAndSettle();

      expect(course.reviewCalls.last.offeringId, 902);
      expect(find.textContaining('只看该教学班的评价'), findsOneWidget);
    });

    testWidgets('收藏乐观切换，失败回滚', (tester) async {
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
      );
      await pumpDetail(tester, course);

      await tester.tap(find.byTooltip('收藏'));
      await tester.pumpAndSettle();
      expect(find.byTooltip('已收藏'), findsOneWidget);

      course.failBookmark = true;
      // 再点已收藏态：乐观取消 → 失败 → 回滚恢复已收藏。
      await tester.tap(find.byTooltip('已收藏'));
      await tester.pumpAndSettle();
      expect(find.byTooltip('已收藏'), findsOneWidget);

      // 消化错误 toast 计时器。
      await tester.pump(const Duration(seconds: 5));
    });

    testWidgets('rating stays on one line at phone width', (tester) async {
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload().copyWith(ratingAvg: 4.9),
      );
      await pumpDetail(tester, course);
      tester.view.physicalSize = const Size(390, 1200);
      await tester.pumpAndSettle();
      final score = find.byKey(const ValueKey('course-rating-score'));
      expect(score, findsOneWidget);
      expect(tester.widget<Text>(score).textSpan!.toPlainText(), '4.9 / 5.0');
      expect(tester.getSize(score).height, lessThan(60));
      expect(tester.takeException(), isNull);
    });

    test('review transport failures have an actionable localized reason', () {
      expect(
        courseReviewError(
          AppLocalizationsZh(),
          const NetworkException(fallbackMessage: 'SocketException'),
        ),
        contains('检查网络'),
      );
      expect(
        courseReviewError(
          AppLocalizationsZh(),
          const ApiException(fallbackMessage: 'internal'),
        ),
        contains('没有提供具体原因'),
      );
    });

    testWidgets('management deep link reveals the review at phone height', (
      tester,
    ) async {
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: [_reviewPayloads().last],
      );
      await pumpDetail(
        tester,
        course,
        focusOfferingId: 902,
        focusReviewId: 4,
        size: const Size(390, 844),
      );
      expect(course.reviewCalls.first.offeringId, 902);
      expect(_reviewHtml('很不错').hitTestable(), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

    testWidgets(
      'reduced motion reveals a lazy review without zero-duration scrolling',
      (tester) async {
        tester.platformDispatcher.accessibilityFeaturesTestValue =
            const FakeAccessibilityFeatures(disableAnimations: true);
        addTearDown(
          tester.platformDispatcher.clearAccessibilityFeaturesTestValue,
        );
        final course = FakeCourseRepository(
          _client(),
          detailPayload: _detailPayload(),
          reviewPayloads: _reviewPayloads(),
        );
        await pumpDetail(
          tester,
          course,
          focusOfferingId: 902,
          focusReviewId: 2,
          size: const Size(320, 480),
        );
        expect(course.reviewCalls.first.offeringId, 902);
        expect(_reviewHtml('历史评价').hitTestable(), findsOneWidget);
        expect(tester.takeException(), isNull);
      },
    );

    testWidgets('own anonymous review precedes other reviews', (tester) async {
      final reviews = _reviewPayloads();
      reviews[2] = reviews[2].copyWith(
        author: const ReviewAuthorPayload(kind: 'anonymous', label: '匿名同学'),
      );
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: reviews,
      );
      await pumpDetail(tester, course);
      await tester.scrollUntilVisible(
        _reviewHtml('很不错'),
        180,
        scrollable: find.byType(Scrollable).first,
      );
      expect(
        tester.getTopLeft(_reviewHtml('很不错')).dy,
        lessThan(tester.getTopLeft(_reviewHtml('好课')).dy),
      );
    });

    testWidgets('review cancellation confirms unsaved content', (tester) async {
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
      );
      await pumpDetail(tester, course);
      await tester.tap(find.text('写课评'));
      await tester.pumpAndSettle();
      _replaceReviewEditorText(tester, '未保存的课评');
      await tester.pump();
      await tester.tap(find.text('取消'));
      await tester.pumpAndSettle();
      final l = AppLocalizations.of(
        tester.element(find.byType(CourseDetailPage)),
      );
      expect(find.text(l.settingsUnsavedTitle), findsOneWidget);
      await tester.tap(find.text(l.settingsKeepEditing));
      await tester.pumpAndSettle();
      expect(
        tester
            .widget<QuillEditor>(find.byType(QuillEditor))
            .controller
            .document
            .toPlainText(),
        contains('未保存的课评'),
      );
      await tester.tap(find.text('取消'));
      await tester.pumpAndSettle();
      await tester.tap(find.text(l.settingsDiscardChanges));
      await tester.pumpAndSettle();
      expect(find.byType(CourseReviewFormSheet), findsNothing);
    });

    testWidgets('review rating has a 48 pixel named touch target', (
      tester,
    ) async {
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
      );
      await pumpDetail(tester, course);
      await tester.tap(find.text('写课评'));
      await tester.pumpAndSettle();
      final target = find.byKey(const ValueKey('review-rating-5'));
      expect(target, findsOneWidget);
      expect(tester.getSize(target), const Size(48, 48));
      expect(tester.getSemantics(target).label, contains('5'));
    });

    testWidgets('cached AI summary starts collapsed without a refresh row', (
      tester,
    ) async {
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        summaryStatus: 'cached',
        summaryPayload: const CourseAiSummaryPayload(
          consensus: 'recommend',
          keywords: ['重点清晰'],
          pros: ['材料完整'],
          cons: [],
          representativeReviews: [],
        ),
      );
      await pumpDetail(tester, course);
      final l = AppLocalizations.of(
        tester.element(find.byType(CourseDetailPage)),
      );
      expect(find.text('重点清晰'), findsNothing);
      expect(find.byTooltip(CourseCopy(l).summaryRefresh), findsNothing);
      await tester.ensureVisible(find.text(l.courseDetailAiSummary));
      await tester.tap(find.text(l.courseDetailAiSummary));
      await tester.pumpAndSettle();
      expect(find.text('重点清晰'), findsOneWidget);
      expect(find.byTooltip(CourseCopy(l).summaryRefresh), findsOneWidget);
    });

    testWidgets('review failure explains server reason above the open sheet', (
      tester,
    ) async {
      final course =
          FakeCourseRepository(_client(), detailPayload: _detailPayload())
            ..createError = const ApiException(
              messageCode: 'review.duplicate',
              fallbackMessage: 'Duplicate',
            );
      await pumpDetail(tester, course);
      await tester.tap(find.text('写课评'));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const ValueKey('review-rating-5')));
      _replaceReviewEditorText(tester, '保留这段评价');
      await tester.pump();
      await tester.tap(find.text('发布评价'));
      await tester.pumpAndSettle();
      expect(find.text('你已评价过该开课实例。'), findsOneWidget);
      expect(tester.getTopLeft(find.text('你已评价过该开课实例。')).dy, lessThan(160));
      expect(
        tester
            .widget<QuillEditor>(find.byType(QuillEditor))
            .controller
            .document
            .toPlainText(),
        contains('保留这段评价'),
      );
      await tester.pump(const Duration(seconds: 8));
    });

    testWidgets('写课评成功前置插入列表', (tester) async {
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: _reviewPayloads(),
      );
      await pumpDetail(tester, course);

      await tester.tap(find.text('写课评'));
      await tester.pumpAndSettle();

      // 选星并输入 Markdown 正文。
      await tester.tap(find.byKey(const ValueKey('review-rating-5')));
      await tester.pump();
      _replaceReviewEditorText(tester, '老师讲得清楚');
      await tester.pump();
      await tester.tap(find.text('发布评价'));
      await tester.pumpAndSettle();

      expect(course.createInputs, hasLength(1));
      expect(course.createInputs.single.offeringId, 901);
      expect(course.createInputs.single.rating, 5);
      expect(course.createInputs.single.isAnonymous, isTrue);
      expect(course.createInputs.single.content, '老师讲得清楚');
      expect(_reviewHtml('老师讲得清楚'), findsOneWidget);

      await tester.pump(const Duration(seconds: 5));
    });

    testWidgets('有用标记乐观切换', (tester) async {
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: _reviewPayloads(),
      );
      await pumpDetail(tester, course);

      // 本人评价现在为首行（id=4，helpfulCount=2）。
      final Finder firstHelpfulChip = find.ancestor(
        of: find
            .byWidgetPredicate(
              (widget) => widget is GfSymbol && widget.name == 'thumbs-up',
            )
            .first,
        matching: find.byType(TextButton),
      );
      expect(
        find.descendant(of: firstHelpfulChip, matching: find.text('2 有用')),
        findsOneWidget,
      );
      await tester.tap(firstHelpfulChip);
      await tester.pumpAndSettle();

      expect(course.helpfulCalls, <(int, bool)>[(4, true)]);
      expect(
        find.descendant(of: firstHelpfulChip, matching: find.text('3 有用')),
        findsOneWidget,
      );
    });
  });
}
