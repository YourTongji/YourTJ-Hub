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
import 'package:forum_app/src/asset_url.dart';
import 'package:forum_app/src/pages/courses/catalog_page.dart';
import 'package:forum_app/src/pages/courses/detail_page.dart';
import 'package:forum_app/src/pages/courses/review_form_sheet.dart';
import 'package:forum_app/src/pages/courses/course_common.dart';
import 'package:forum_app/src/pages/courses/review_reaction.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/rich_content/gf_html_content.dart';
import 'package:forum_app/src/widgets/share/share_image_preview.dart';
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
  void setServerReview(ReviewPayload review) =>
      _serverReviews[review.id] = review;

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
      helpfulCount: helpful
          ? (nextCount < 0 ? 0 : nextCount)
          : review.helpfulCount,
      dislikeCount: helpful
          ? review.dislikeCount
          : (nextCount < 0 ? 0 : nextCount),
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

/// 卡片功能区的单行动作（有用/无用/分享）；标签只保留计数。
Finder _reviewAction(int reviewId, String action) =>
    find.byKey(ValueKey<String>('review-$action-$reviewId'));

/// 动作 chip 内可见 pill（Container）实际绘制的背景色：命中区仍是外层
/// TextButton（44dp），可见高度 32dp，禁用态不再参与背景绘制。
Color? _chipBackground(WidgetTester tester, Finder chip) {
  final Container pill = tester.widget<Container>(
    find.descendant(of: chip, matching: find.byType(Container)).first,
  );
  return (pill.decoration as BoxDecoration?)?.color;
}

/// 动作 chip 内计数文本实际使用的颜色。
Color? _chipLabelColor(WidgetTester tester, Finder chip, String label) => tester
    .renderObject<RenderParagraph>(
      find.descendant(of: chip, matching: find.text(label)),
    )
    .text
    .style
    ?.color;

/// 卡片右上溢出菜单按钮。
Finder _reviewMenu(int reviewId) =>
    find.byKey(ValueKey<String>('review-menu-$reviewId'));

Future<void> _openReviewMenu(WidgetTester tester, int reviewId) async {
  await tester.tap(_reviewMenu(reviewId));
  await tester.pumpAndSettle();
}

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
  TokenStorage? tokenStorage,
}) {
  final ProviderContainer container = ProviderContainer(
    overrides: <Override>[
      courseRepositoryProvider.overrideWithValue(courseRepo),
      tokenStorageProvider.overrideWithValue(
        tokenStorage ?? _MemoryTokenStorage(),
      ),
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

    testWidgets('catalog row chips share one spec and double the group gap', (
      tester,
    ) async {
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        listPages: <int, (List<CourseSummaryPayload>, bool)>{
          1: (<CourseSummaryPayload>[_course(101, '课程 A1')], false),
        },
      );
      final ProviderContainer container = _container(
        courseRepo: course,
        pageRepo: FakePageRepository(_client()),
      );
      await tester.pumpWidget(_app(container, const CourseCatalogPage()));
      await tester.pumpAndSettle();

      final Finder codeChip = find
          .ancestor(of: find.text('1000101'), matching: find.byType(Container))
          .first;
      final Finder toggle = find
          .ancestor(of: find.text('+1'), matching: find.byType(Container))
          .first;
      BoxDecoration decorationOf(Finder chip) =>
          tester.widget<Container>(chip).decoration! as BoxDecoration;

      // 统一 chip 规范：课号 chip 与「学期+计数」chip 同高、同圆角（selector token）。
      expect(tester.getSize(codeChip).height, closeTo(24, .5));
      expect(tester.getSize(toggle).height, closeTo(24, .5));
      expect(
        decorationOf(toggle).borderRadius,
        decorationOf(codeChip).borderRadius,
      );
      expect(decorationOf(codeChip).borderRadius, BorderRadius.circular(8));
      // 首个学期与「+n」合并在同一个 chip 内，不再是两个松散小按钮。
      expect(
        find.descendant(of: toggle, matching: find.text('25春')),
        findsOneWidget,
      );

      await tester.tap(toggle);
      await tester.pumpAndSettle();
      final Finder expandedToggle = find
          .ancestor(of: find.text('收起'), matching: find.byType(Container))
          .first;
      final Finder secondTerm = find
          .ancestor(of: find.text('25秋'), matching: find.byType(Container))
          .first;
      // 组内 gap = 6、组间距 = 12（≥ 2×），展开后仍与 chip 同高。
      final double intraGap =
          tester.getRect(secondTerm).left -
          tester.getRect(expandedToggle).right;
      final double groupGap =
          tester.getRect(expandedToggle).left - tester.getRect(codeChip).right;
      expect(intraGap, closeTo(6, .5));
      expect(groupGap, closeTo(12, .5));
      expect(groupGap, greaterThanOrEqualTo(2 * intraGap - .5));
      expect(tester.getSize(secondTerm).height, closeTo(24, .5));
      expect(tester.takeException(), isNull);
    });

    testWidgets('catalog filter chips are 32dp pills inside 44dp targets', (
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

      for (final String label in <String>['院系', '学期', '校区', '教师', '只看有评价']) {
        final Finder chip = find
            .ancestor(of: find.text(label), matching: find.byType(InkWell))
            .first;
        // 命中区 ≥44dp，可见 pill 只有 32dp（不再是大圆丸）。
        expect(
          tester.getSize(chip).height,
          greaterThanOrEqualTo(44),
          reason: label,
        );
        final Finder ink = find
            .descendant(of: chip, matching: find.byType(Ink))
            .first;
        expect(tester.getSize(ink).height, closeTo(32, .5), reason: label);
        expect(
          (tester.widget<Ink>(ink).decoration! as BoxDecoration).borderRadius,
          BorderRadius.circular(8),
          reason: label,
        );
      }

      // picker 带下拉 affordance；toggle 未选中时没有勾选图标。
      Finder symbolIn(String label, String name) => find.descendant(
        of: find
            .ancestor(of: find.text(label), matching: find.byType(InkWell))
            .first,
        matching: find.byWidgetPredicate(
          (Widget widget) => widget is GfSymbol && widget.name == name,
        ),
      );
      expect(symbolIn('院系', 'chevron-down'), findsOneWidget);
      expect(symbolIn('学期', 'chevron-down'), findsOneWidget);
      expect(symbolIn('只看有评价', 'chevron-down'), findsNothing);
      expect(symbolIn('只看有评价', 'check'), findsNothing);
    });

    testWidgets('catalog card keeps symmetric top and bottom gaps', (
      tester,
    ) async {
      for (final double scale in <double>[1, 1.3, 2]) {
        tester.view.physicalSize = const Size(320, 1200);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.reset);
        final course = FakeCourseRepository(
          _client(),
          listPages: {
            1: ([_course(1, '普通化学实验A2')], false),
          },
        );
        await tester.pumpWidget(
          _app(
            _container(
              courseRepo: course,
              pageRepo: FakePageRepository(_client()),
            ),
            const CourseCatalogPage(),
            textScale: scale,
          ),
        );
        await tester.pumpAndSettle();

        final Finder title = find.text('普通化学实验A2');
        final Finder card = find
            .ancestor(of: title, matching: find.byType(InkWell))
            .first;
        final Finder chip = find
            .ancestor(of: find.text('+1'), matching: find.byType(Container))
            .first;
        final Rect cardRect = tester.getRect(card);
        final Rect titleRect = tester.getRect(title);
        final Rect chipRect = tester.getRect(chip);

        // 44dp 命中区仍在（学期 chip）。
        expect(
          tester
              .getSize(
                find
                    .ancestor(
                      of: find.text('+1'),
                      matching: find.byType(InkWell),
                    )
                    .first,
              )
              .height,
          greaterThanOrEqualTo(44),
        );
        // 可见上下留白对称：顶部布局 8 + CJK 字体行盒自带的 ~4dp 上内边距
        // ≈ 底部可见 12（命中区的隐形半高已扣除，不再把卡片重心拉低）。
        expect(
          titleRect.top - cardRect.top,
          closeTo(8, 1),
          reason: 'textScale $scale',
        );
        expect(
          cardRect.bottom - chipRect.bottom,
          closeTo(12, 1.5),
          reason: 'textScale $scale',
        );
        expect(tester.takeException(), isNull);

        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pumpAndSettle();
      }
    });

    testWidgets('single and multi term cards share one even rhythm', (
      tester,
    ) async {
      final course = FakeCourseRepository(
        _client(),
        listPages: {
          1: (
            [
              _course(
                1,
                '单学期课程',
              ).copyWith(recentTerms: const <String>['2025-2026-1']),
              _course(2, '多学期课程'),
            ],
            false,
          ),
        },
      );
      await tester.pumpWidget(
        _app(
          _container(
            courseRepo: course,
            pageRepo: FakePageRepository(_client()),
          ),
          const CourseCatalogPage(),
        ),
      );
      await tester.pumpAndSettle();

      Rect cardOf(String name) => tester.getRect(
        find
            .ancestor(of: find.text(name), matching: find.byType(InkWell))
            .first,
      );
      // 首张（单学期、无 44dp toggle）与其他卡片高度一致：下边距不再不同。
      expect(cardOf('单学期课程').height, closeTo(cardOf('多学期课程').height, .5));
      // 布局间距：标题→教师 6、教师→可见 chip 10。CJK 字体行盒自带上下内边距
      // （测试字体没有），真机上两段字形间距都≈13dp。
      final Rect title = tester.getRect(find.text('多学期课程'));
      final Rect meta = tester.getRect(find.text('张三 · 数学科学学院').last);
      final Rect chip = tester.getRect(
        find
            .ancestor(of: find.text('10002'), matching: find.byType(Container))
            .first,
      );
      expect(meta.top - title.bottom, closeTo(6, .5));
      expect(chip.top - meta.bottom, closeTo(10, 1));
      expect(tester.takeException(), isNull);
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

    testWidgets(
      'review action bar stays on one row with 44 pixel targets at 200% text',
      (tester) async {
        // 与「German large text」布局测试同样只渲染课评卡（开课班级区块
        // 有自己的窄屏布局问题，不属于本任务范围）。
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
            textScale: 2,
          ),
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);

        final helpful = _reviewAction(4, 'helpful');
        final dislike = _reviewAction(4, 'dislike');
        final share = _reviewAction(4, 'share');
        expect(helpful, findsOneWidget);
        expect(dislike, findsOneWidget);
        expect(share, findsOneWidget);
        // 同一行：三个动作的水平中线一致（折行会出现第二行，中线不同）。
        final centers = <double>{
          for (final chip in <Finder>[helpful, dislike, share])
            tester.getRect(chip).center.dy,
        };
        expect(centers, hasLength(1));
        for (final chip in <Finder>[helpful, dislike, share]) {
          expect(tester.getSize(chip).width, greaterThanOrEqualTo(44));
          expect(tester.getSize(chip).height, greaterThanOrEqualTo(44));
        }
        // 单行 = 横向滚动，而不是 Wrap。
        expect(
          find.ancestor(of: helpful, matching: find.byType(Wrap)),
          findsNothing,
        );
        expect(
          find.ancestor(
            of: helpful,
            matching: find.byWidgetPredicate(
              (widget) =>
                  widget is SingleChildScrollView &&
                  widget.scrollDirection == Axis.horizontal,
            ),
          ),
          findsWidgets,
        );
        // 标签只剩计数，可读名由 Semantics 提供。
        expect(
          find.descendant(of: helpful, matching: find.text('2')),
          findsOneWidget,
        );
        expect(tester.getSemantics(helpful).label, contains('有用 2'));
        expect(tester.getSemantics(dislike).label, contains('无用 0'));

        // 自己的评价：编辑/删除只出现在右上溢出菜单里。
        expect(find.text('编辑'), findsNothing);
        expect(find.text('删除'), findsNothing);
        await _openReviewMenu(tester, 4);
        expect(find.text('编辑'), findsOneWidget);
        expect(find.text('删除'), findsOneWidget);
        expect(find.text('举报内容'), findsNothing);
        await tester.tap(find.text('编辑'));
        await tester.pumpAndSettle();
        expect(find.byType(CourseReviewFormSheet), findsOneWidget);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pump(const Duration(milliseconds: 500));
      },
    );

    testWidgets('report entry follows the signed-in state', (tester) async {
      final other = _reviewPayloads().first; // id=1，他人评价（canEdit=false）
      final course = FakeCourseRepository(_client(), reviewPayloads: [other]);
      await pumpDetail(tester, course, size: const Size(500, 1200));
      await _openReviewMenu(tester, other.id);
      expect(find.text('举报内容'), findsOneWidget);
      expect(find.text('编辑'), findsNothing);
      await tester.tapAt(const Offset(4, 4));
      await tester.pumpAndSettle();

      // 未登录：不显示举报入口。
      final guest = FakeCourseRepository(_client(), reviewPayloads: [other]);
      await tester.pumpWidget(
        _app(
          _container(
            courseRepo: guest,
            tokenStorage: _MemoryTokenStorage()..token = null,
          ),
          const CourseDetailPage(courseId: 42),
        ),
      );
      await tester.pumpAndSettle();
      // 未登录且无可编辑/可删除项：连溢出菜单都不渲染（没有举报入口）。
      expect(_reviewMenu(other.id), findsNothing);
      expect(find.text('举报内容'), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
    });

    testWidgets('review action bar taps still toggle helpful', (tester) async {
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
      await tester.tap(_reviewAction(4, 'helpful'));
      await tester.pumpAndSettle();
      expect(course.helpfulCalls, [(4, true)]);
      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump(const Duration(milliseconds: 500));
    });

    testWidgets('review action chips expose one labelled clickable node', (
      tester,
    ) async {
      // fixture 的 review 4 默认未点过「有用」；选中态需要单独构造，
      // 否则 toggled=false 是正确的（旧断言把未选中当成了选中）。
      final ReviewPayload selected = _reviewPayloads().last.copyWith(
        viewer: _reviewPayloads().last.viewer.copyWith(isHelpful: true),
        helpfulCount: 3,
      );
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: [selected],
      );
      await pumpDetail(tester, course, size: const Size(390, 1200));

      // 读屏/uiautomator 必须在一个节点上同时拿到可读名、按钮、点击动作与
      // 选中态（checked）；之前的父子两棵语义树只见得到「有 label 但不可点」。
      final SemanticsData data = tester
          .getSemantics(_reviewAction(4, 'helpful'))
          .getSemanticsData();
      expect(data.label, contains('有用 3'));
      expect(data.hasAction(SemanticsAction.tap), isTrue);
      expect(data.flagsCollection.isButton, isTrue);
      expect(data.flagsCollection.isToggled, Tristate.isTrue);
      // 未选中一侧暴露 checked=false，而不是缺省/未知。
      final SemanticsData dislike = tester
          .getSemantics(_reviewAction(4, 'dislike'))
          .getSemanticsData();
      expect(dislike.flagsCollection.isToggled, Tristate.isFalse);
      expect(tester.takeException(), isNull);
    });

    testWidgets('anonymous reviews get the shared beam avatar', (tester) async {
      final course = FakeCourseRepository(
        _client(),
        reviewPayloads: [_reviewPayloads()[1]],
      );
      await pumpDetail(tester, course, size: const Size(390, 1200));

      expect(find.byType(GfNetworkImage), findsNothing);
      final beams = tester
          .widgetList<GfBeamAvatar>(find.byType(GfBeamAvatar))
          .toList();
      // seed 与 Web `reviewAvatarSrc` 完全一致：<label>-<reviewId>
      expect(beams.map((avatar) => avatar.seed), contains('历史匿名评价-2'));
    });

    testWidgets('member reviews render the server avatar with a 40px slot', (
      tester,
    ) async {
      final member = _reviewPayloads().last.copyWith(
        author: const ReviewAuthorPayload(
          kind: 'member',
          label: 'alice',
          avatarUrl: 'https://cdn.example.com/avatars/alice.png',
        ),
      );
      final course = FakeCourseRepository(_client(), reviewPayloads: [member]);
      await pumpDetail(tester, course, size: const Size(390, 1200));

      final network = find.byWidgetPredicate(
        (widget) =>
            widget is GfNetworkImage &&
            widget.semanticLabel == 'alice' &&
            widget.url == 'https://cdn.example.com/avatars/alice.png',
      );
      expect(network, findsOneWidget);
      expect(tester.getSize(network), const Size(40, 40));
      // 头像槽位是圆形裁切，不会出现方形空洞。
      expect(
        find.ancestor(of: network, matching: find.byType(ClipOval)),
        findsWidgets,
      );
    });

    testWidgets('failed member avatar falls back to the beam avatar', (
      tester,
    ) async {
      // 测试环境的 HTTP client 对所有请求返回 400，等价于头像 404：
      // 失败时必须立刻回落生成头像，不能留下圆形空洞。
      final member = _reviewPayloads().last.copyWith(
        author: const ReviewAuthorPayload(
          kind: 'member',
          label: 'alice',
          avatarUrl: 'https://unreachable.invalid/alice.png',
        ),
      );
      final course = FakeCourseRepository(_client(), reviewPayloads: [member]);
      await pumpDetail(tester, course, size: const Size(390, 1200));
      await tester.pumpAndSettle();

      final beams = tester
          .widgetList<GfBeamAvatar>(find.byType(GfBeamAvatar))
          .toList();
      expect(beams.map((avatar) => avatar.seed), contains('alice-4'));
      expect(
        tester.getSize(find.byType(GfBeamAvatar).first),
        const Size(40, 40),
      );
      expect(tester.takeException(), isNull);
    });

    testWidgets('member avatar resolves relative server paths', (tester) async {
      final member = _reviewPayloads().last.copyWith(
        author: const ReviewAuthorPayload(
          kind: 'member',
          label: 'alice',
          avatarUrl: '/static/pic/9.webp',
        ),
      );
      final course = FakeCourseRepository(_client(), reviewPayloads: [member]);
      await pumpDetail(tester, course, size: const Size(390, 1200));

      final Finder network = find.byWidgetPredicate(
        (widget) => widget is GfNetworkImage && widget.semanticLabel == 'alice',
      );
      expect(network, findsOneWidget);
      // 相对路径必须解析成绝对 URL，否则 GfNetworkImage 会加载失败并回落 beam。
      expect(
        tester.widget<GfNetworkImage>(network).url,
        resolveApiAssetUrl('/static/pic/9.webp'),
      );
      expect(tester.takeException(), isNull);
    });

    testWidgets('long offering metadata stays on one un-truncated meta line', (
      tester,
    ) async {
      final CourseDetailPayload detail = _detailPayload();
      final List<CourseOfferingPayload> offerings = detail.offerings!;
      final CourseDetailPayload longDetail = detail.copyWith(
        offerings: <CourseOfferingPayload>[
          offerings.first.copyWith(
            className: '17班',
            classCode: '54009917',
            campus: '四平路校区',
            faculty: '马克思主义学院',
            instructors: const <String>['王小莉'],
          ),
          ...offerings.skip(1),
        ],
      );
      final review = _reviewPayloads().first.copyWith(
        offeringId: offerings.first.id,
      );
      final course = FakeCourseRepository(
        _client(),
        detailPayload: longDetail,
        reviewPayloads: <ReviewPayload>[review],
      );

      tester.view.physicalSize = const Size(320, 2400);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(
        _app(
          _container(courseRepo: course),
          const CourseDetailPage(courseId: 42),
          textScale: 2,
        ),
      );
      await tester.pumpAndSettle();

      // 单行元信息：学期 · 班次 · 教师 · 短日期（班号/校区/院系不在卡片重复）。
      final Finder meta = find.textContaining('25秋 · 17班 · 王小莉');
      expect(meta, findsOneWidget);
      // 大字号下可以换行，但不设 maxLines、不截断关键信息。
      expect(tester.widget<Text>(meta).maxLines, isNull);
      expect(
        tester.renderObject<RenderParagraph>(meta).didExceedMaxLines,
        isFalse,
      );
      // 次要行已收敛：卡片里没有班号/校区/院系这一行。
      expect(find.text('17班 · 54009917 · 四平路校区 · 马克思主义学院'), findsNothing);
      expect(tester.takeException(), isNull);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    });

    testWidgets('review body renders relative images and wires links', (
      tester,
    ) async {
      final ReviewPayload review = _reviewPayloads().last.copyWith(
        contentHtml:
            '<p>题图如下</p><img src="/file/img/answer.png" alt="题图">'
            '<p><a href="https://example.com/ref">参考资料</a></p>',
      );
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: <ReviewPayload>[review],
      );
      await pumpDetail(tester, course, size: const Size(390, 1600));

      // 相对 src 必须解析为绝对地址后才交给 GfNetworkImage。
      final Finder image = find.byWidgetPredicate(
        (widget) =>
            widget is GfNetworkImage &&
            widget.url == resolveApiAssetUrl('/file/img/answer.png'),
      );
      expect(image, findsOneWidget);
      expect(find.text('参考资料', findRichText: true), findsOneWidget);

      final GfHtmlContent content = tester.widget<GfHtmlContent>(
        find.byType(GfHtmlContent).first,
      );
      expect(content.customWidgetBuilder, isNotNull);
      expect(content.onTapUrl, isNotNull);
      expect(content.baseUrl, isNotNull);

      // 点击图片进入共享图片查看器（wiki/Markdown 阅读同一约定）。
      await tester.tap(image);
      await tester.pumpAndSettle();
      expect(find.byType(GfImageViewer), findsOneWidget);
      expect(tester.takeException(), isNull);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    });

    testWidgets('bottom dock reserves its own space instead of overlaying', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(390, 844);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: <ReviewPayload>[_reviewPayloads().last],
      );
      await tester.pumpWidget(
        _app(
          _container(courseRepo: course),
          const CourseDetailPage(courseId: 42),
        ),
      );
      await tester.pumpAndSettle();

      final Finder dock = find
          .ancestor(of: find.text('写课评'), matching: find.byType(ColoredBox))
          .first;
      final Finder scroll = find.byType(CustomScrollView).first;
      // 滚动视口底部不得进入 dock 区域：课评功能区永远不会被 dock 盖住，
      // 也不会把「分享」点击转给「写课评」。
      expect(
        tester.getRect(scroll).bottom,
        lessThanOrEqualTo(tester.getRect(dock).top + 0.5),
      );
      expect(tester.takeException(), isNull);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    });

    testWidgets('review share preview renders its action icons', (
      tester,
    ) async {
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: <ReviewPayload>[_reviewPayloads().last],
      );
      await pumpDetail(tester, course, size: const Size(800, 1800));

      final Finder share = _reviewAction(4, 'share');
      await tester.ensureVisible(share);
      await tester.pumpAndSettle();
      await tester.tap(share);
      await tester.pumpAndSettle();

      expect(find.byType(ShareImageCard), findsOneWidget);
      expect(tester.takeException(), isNull);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pumpAndSettle();
    });

    testWidgets('review action bar keeps one row at 375 and 430 px', (
      tester,
    ) async {
      for (final width in <double>[375, 430]) {
        tester.view.physicalSize = Size(width, 2000);
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
          ),
        );
        await tester.pumpAndSettle();

        final centers = <double>{
          for (final chip in <Finder>[
            _reviewAction(4, 'helpful'),
            _reviewAction(4, 'dislike'),
            _reviewAction(4, 'share'),
          ])
            tester.getRect(chip).center.dy,
        };
        expect(
          centers,
          hasLength(1),
          reason: 'width $width must stay on one row',
        );
        expect(
          tester.getRect(_reviewAction(4, 'share')).right,
          lessThanOrEqualTo(width),
        );
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox.shrink());
        await tester.pumpAndSettle();
      }
    });

    testWidgets('review action pills stay flat inside a 44 pixel hit target', (
      tester,
    ) async {
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: [_reviewPayloads().last],
      );
      await pumpDetail(tester, course, size: const Size(800, 1800));

      final Finder helpful = _reviewAction(4, 'helpful');
      final Finder pill = find
          .descendant(of: helpful, matching: find.byType(Container))
          .first;
      // 可见 pill 32dp（扁长），命中区仍由 44dp 的 TextButton 承担。
      expect(tester.getSize(pill).height, 32);
      expect(tester.getSize(helpful).height, greaterThanOrEqualTo(44));
      expect(tester.takeException(), isNull);
    });

    testWidgets('reaction switch clears the opposite before the target', (
      tester,
    ) async {
      final ReviewPayload disliked = _reviewPayloads().first.copyWith(
        viewer: _reviewPayloads().first.viewer.copyWith(isDisliked: true),
        dislikeCount: 1,
        helpfulCount: 9,
      );
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: [disliked],
      );
      await pumpDetail(tester, course, size: const Size(800, 1800));

      final Finder helpful = _reviewAction(disliked.id, 'helpful');
      final Finder dislike = _reviewAction(disliked.id, 'dislike');
      final double chipWidth = tester.getSize(helpful).width;

      await tester.tap(helpful);
      await tester.pumpAndSettle();

      // 兼容未部署 #991 的旧服务端：切换时先显式删另一侧，再写目标状态
      // （新服务端两个操作幂等），成功后不重读整表；本地立即互斥。
      expect(course.reactionCalls, [
        ('dislike', disliked.id, false),
        ('helpful', disliked.id, true),
      ]);
      expect(course.reviewCalls, hasLength(1));
      // 本地立即互斥：有用 9→10，无用 1→0。
      expect(
        find.descendant(of: helpful, matching: find.text('10')),
        findsOneWidget,
      );
      expect(
        find.descendant(of: dislike, matching: find.text('0')),
        findsOneWidget,
      );
      // 两位数计数与一位数同宽：功能区不会因进位而横向抽动。
      expect(tester.getSize(helpful).width, chipWidth);
      expect(
        find.byWidgetPredicate(
          (widget) => widget is Semantics && widget.properties.toggled == true,
        ),
        findsOneWidget,
      );
    });

    testWidgets(
      'reaction chip keeps enabled colors while the write is pending',
      (tester) async {
        final ReviewPayload review = _reviewPayloads().last; // id=4，自己的评价
        final course = FakeCourseRepository(
          _client(),
          detailPayload: _detailPayload(),
          reviewPayloads: [review],
        )..waitHelpful = Completer<void>();
        await pumpDetail(tester, course, size: const Size(800, 1800));

        final Finder helpful = _reviewAction(review.id, 'helpful');
        await tester.tap(helpful);
        await tester.pumpAndSettle();

        // 乐观更新已生效（2→3），写入仍 pending。
        expect(course.reactionCalls, [('helpful', review.id, true)]);
        expect(
          find.descendant(of: helpful, matching: find.text('3')),
          findsOneWidget,
        );

        final TextButton pending = tester.widget<TextButton>(
          find.descendant(of: helpful, matching: find.byType(TextButton)),
        );
        final Color? activeForeground = pending.style!.foregroundColor!.resolve(
          const <WidgetState>{},
        );
        final GfColors colors = GfTheme.colorsOf(tester.element(helpful));
        final Color activeBackground = colors.warning.withValues(alpha: 0.1);
        // 请求期间不能进入 Material disabled 态：那会把背景换成透明、
        // 前景降到 38%，每次点赞都闪一下。
        expect(pending.enabled, isTrue);
        expect(_chipBackground(tester, helpful), activeBackground);
        expect(_chipLabelColor(tester, helpful, '3'), activeForeground);

        // pending 期间再加一帧，颜色不变。
        await tester.pump();
        expect(_chipBackground(tester, helpful), activeBackground);
        expect(_chipLabelColor(tester, helpful, '3'), activeForeground);

        course.waitHelpful!.complete();
        await tester.pumpAndSettle();
        expect(course.reactionCalls, [('helpful', review.id, true)]);
        expect(_chipBackground(tester, helpful), activeBackground);
      },
    );

    testWidgets('in-flight review reload keeps the optimistic reaction state', (
      tester,
    ) async {
      final ReviewPayload review = _reviewPayloads().first; // id=1
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        reviewPayloads: [review],
      )..waitHelpful = Completer<void>();
      await pumpDetail(tester, course, size: const Size(800, 2400));

      final Finder helpful = _reviewAction(review.id, 'helpful');
      await tester.tap(helpful);
      await tester.pumpAndSettle();
      expect(
        find.descendant(of: helpful, matching: find.text('1')),
        findsOneWidget,
      );

      // 写入仍 pending 时触发一次完整列表重载：服务端尚未提交，这次响应
      // 是旧快照（有用 0），不能覆盖本地乐观态。
      final Finder offering = find.text('01班 · 10000101');
      await tester.scrollUntilVisible(
        offering,
        400,
        scrollable: find.byType(Scrollable).first,
      );
      await tester.pumpAndSettle();
      await tester.tap(offering);
      await tester.pumpAndSettle();
      expect(course.reviewCalls.length, greaterThanOrEqualTo(2));

      expect(
        find.descendant(
          of: _reviewAction(review.id, 'helpful'),
          matching: find.text('1'),
        ),
        findsOneWidget,
      );
      expect(
        find.descendant(
          of: _reviewAction(review.id, 'dislike'),
          matching: find.text('0'),
        ),
        findsOneWidget,
      );

      course.waitHelpful!.complete();
      await tester.pumpAndSettle();
      expect(course.reactionCalls, [('helpful', review.id, true)]);
      expect(
        find.descendant(
          of: _reviewAction(review.id, 'helpful'),
          matching: find.text('1'),
        ),
        findsOneWidget,
      );
    });

    testWidgets('failed reaction restores both sides without reloading', (
      tester,
    ) async {
      final ReviewPayload disliked = _reviewPayloads().first.copyWith(
        viewer: _reviewPayloads().first.viewer.copyWith(isDisliked: true),
        dislikeCount: 1,
      );
      final course = FakeCourseRepository(_client(), reviewPayloads: [disliked])
        ..failHelpful = const ApiException(fallbackMessage: 'write failed');
      await pumpDetail(tester, course, size: const Size(800, 1800));

      await tester.tap(_reviewAction(disliked.id, 'helpful'));
      await tester.pumpAndSettle();

      // 第一步（清相反侧）成功、第二步（写目标）失败：本地按原值回滚。
      expect(course.reactionCalls, [
        ('dislike', disliked.id, false),
        ('helpful', disliked.id, true),
      ]);
      expect(course.reviewCalls, hasLength(1));
      // 旧服务端上清相反侧已落库；下一次列表加载会与服务端对齐。
      expect(course.serverReview(disliked.id).viewer.isDisliked, isFalse);
      expect(
        find.descendant(
          of: _reviewAction(disliked.id, 'helpful'),
          matching: find.text('0'),
        ),
        findsOneWidget,
      );
      expect(
        find.descendant(
          of: _reviewAction(disliked.id, 'dislike'),
          matching: find.text('1'),
        ),
        findsOneWidget,
      );
    });

    testWidgets('failed opposite cleanup stops before the target write', (
      tester,
    ) async {
      final ReviewPayload disliked = _reviewPayloads().first.copyWith(
        viewer: _reviewPayloads().first.viewer.copyWith(isDisliked: true),
        dislikeCount: 1,
      );
      final course = FakeCourseRepository(_client(), reviewPayloads: [disliked])
        ..failDislike = const ApiException(fallbackMessage: 'write failed');
      await pumpDetail(tester, course, size: const Size(800, 1800));

      await tester.tap(_reviewAction(disliked.id, 'helpful'));
      await tester.pumpAndSettle();

      // 第一步失败：不继续发目标请求，无整表重读，UI 恢复原状态。
      expect(course.reactionCalls, [('dislike', disliked.id, false)]);
      expect(course.reviewCalls, hasLength(1));
      expect(
        find.descendant(
          of: _reviewAction(disliked.id, 'helpful'),
          matching: find.text('0'),
        ),
        findsOneWidget,
      );
      expect(
        find.descendant(
          of: _reviewAction(disliked.id, 'dislike'),
          matching: find.text('1'),
        ),
        findsOneWidget,
      );
    });

    testWidgets(
      'lost reaction response rolls back locally without re-reading',
      (tester) async {
        final ReviewPayload disliked = _reviewPayloads().first.copyWith(
          viewer: _reviewPayloads().first.viewer.copyWith(isDisliked: true),
          dislikeCount: 1,
        );
        final course = FakeCourseRepository(
          _client(),
          reviewPayloads: [disliked],
        )..loseHelpfulOnResponse = true;
        await pumpDetail(tester, course, size: const Size(800, 1800));

        await tester.tap(_reviewAction(disliked.id, 'helpful'));
        await tester.pumpAndSettle();

        expect(course.reactionCalls, [
          ('dislike', disliked.id, false),
          ('helpful', disliked.id, true),
        ]);
        expect(course.reviewCalls, hasLength(1));
        // 响应丢失时按本地原值回滚；服务端可能已提交，以服务端为准的收敛
        // 交给下一次列表加载，不再在失败路径触发整表重读。
        expect(
          find.descendant(
            of: _reviewAction(disliked.id, 'helpful'),
            matching: find.text('0'),
          ),
          findsOneWidget,
        );
        expect(
          find.descendant(
            of: _reviewAction(disliked.id, 'dislike'),
            matching: find.text('1'),
          ),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'report requires an explicit reason and preserves a failed draft',
      (tester) async {
        final course =
            FakeCourseRepository(
                _client(),
                reviewPayloads: [_reviewPayloads().first],
              )
              ..reportError = const ApiException(
                fallbackMessage: 'report failed',
                messageCode: 'review.report.failed',
              );
        await pumpDetail(tester, course, size: const Size(500, 1200));

        await _openReviewMenu(tester, 1);
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
      await _openReviewMenu(tester, 1);
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
      await _pumpStandaloneReviewForm(tester, emptyRepo, offerings: offerings);
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

    // 「快速评价」含空列表项（`-`），转换后根节点里会出现 Block 而非 Line。
    testWidgets('list template applies without assuming every node is a line', (
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

      await tester.tap(find.byKey(const Key('course-review-templates')));
      await tester.pumpAndSettle();
      await tester.tap(find.text(l10n.courseReviewTemplateQuickName));
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
      final controller = tester
          .widget<QuillEditor>(find.byType(QuillEditor))
          .controller;
      final document = controller.document;
      expect(
        document.toPlainText(),
        allOf(contains('总体评价'), contains('优点'), contains('缺点'), contains('建议')),
      );
      // 每个 `-` 落成一条空的无序列表行，供用户接着填写。
      final bullets = document.root.children
          .whereType<Block>()
          .expand((block) => block.children)
          .whereType<Line>()
          .where((line) => line.style.attributes['list'] == Attribute.ul)
          .toList();
      expect(bullets, hasLength(2));
      // 光标落在首行标签「总体评价：」之后，续写的文字不继承粗体。
      expect(controller.selection.baseOffset, '总体评价：'.length);
      controller.replaceText(
        controller.selection.baseOffset,
        0,
        '好课',
        const TextSelection.collapsed(offset: 7),
      );
      await tester.pump();
      expect(document.toPlainText(), startsWith('总体评价：好课\n'));
      final firstOps = document.toDelta().toList();
      expect(firstOps[0].data, '总体评价：');
      expect(firstOps[0].attributes, {'bold': true});
      expect(firstOps[1].data, startsWith('好课'));
      expect(firstOps[1].attributes?['bold'], isNull);
      await tester.pumpWidget(const SizedBox.shrink());
    });

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

    testWidgets('review template inserts in place and stays undoable', (
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

      final editor = tester.widget<QuillEditor>(find.byType(QuillEditor));
      final controller = editor.controller;
      controller.replaceText(
        0,
        controller.document.length - 1,
        '原文',
        const TextSelection.collapsed(offset: 2),
      );
      await tester.pumpAndSettle();
      // Quill 用真实时间（400ms）合并连续改动；让合并窗口过去，
      // 插入模板才会记为独立的一步撤销。
      await tester.runAsync(
        () => Future<void>.delayed(const Duration(milliseconds: 450)),
      );

      await tester.tap(find.byKey(const Key('course-review-templates')));
      await tester.pumpAndSettle();
      await tester.tap(find.text(l10n.courseReviewTemplateComprehensiveName));
      await tester.pumpAndSettle();
      // 非空正文仍需二次确认。
      expect(find.text(l10n.courseReviewTemplateReplaceTitle), findsOneWidget);
      await tester.tap(find.text(l10n.courseReviewTemplateApply));
      await tester.pumpAndSettle();

      // 不换 controller（实例、撤销栈、焦点都保留）。
      final after = tester.widget<QuillEditor>(find.byType(QuillEditor));
      expect(identical(controller, after.controller), isTrue);
      expect(after.focusNode.hasFocus, isTrue);
      expect(after.controller.document.toPlainText(), contains('课程内容'));
      expect(after.controller.document.toPlainText(), isNot(contains('原文')));
      // 全是标题的模板：光标落在首行标题「课程内容」末尾。
      expect(after.controller.selection.baseOffset, 4);

      // 撤销回到插入前的内容。
      after.controller.undo();
      await tester.pumpAndSettle();
      expect(after.controller.document.toPlainText(), contains('原文'));

      // `_content` 由文档监听回填：提交时带上模板正文。
      await tester.tap(find.byKey(const ValueKey('review-rating-5')));
      await tester.pump();
      await tester.tap(find.text(l10n.reviewSubmit));
      await tester.pumpAndSettle();
      expect(course.createInputs, hasLength(1));
      expect(course.createInputs.single.content, contains('原文'));

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
        await _openReviewMenu(tester, 4);
        await tester.tap(find.text('编辑'));
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

    testWidgets('review action bar stays on one row with German large text', (
      tester,
    ) async {
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
      // 动作只在溢出菜单里（功能区不再折行出现编辑/删除）。
      expect(find.text(l.commonEdit), findsNothing);
      await _openReviewMenu(tester, 4);
      expect(find.text(l.commonEdit), findsOneWidget);
      await tester.tapAt(const Offset(4, 4));
      await tester.pumpAndSettle();
      final helpful = _reviewAction(4, 'helpful');
      final dislike = _reviewAction(4, 'dislike');
      final share = _reviewAction(4, 'share');
      final centers = <double>{
        for (final chip in <Finder>[helpful, dislike, share])
          tester.getRect(chip).center.dy,
      };
      expect(centers, hasLength(1));
      expect(tester.getRect(helpful).right, lessThanOrEqualTo(320));
      expect(tester.takeException(), isNull);
    });

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

      await _openReviewMenu(tester, 1);
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

    testWidgets('rating summary shows the score ring and distribution rows', (
      tester,
    ) async {
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
      );
      await pumpDetail(tester, course, size: const Size(390, 1400));

      // 均分进度环 + 中心分数（Web RatingSummaryCard 对齐），5★→1★ 五行分布。
      expect(find.byKey(const Key('course-rating-ring')), findsOneWidget);
      expect(
        find.byKey(const ValueKey<String>('course-rating-score')),
        findsOneWidget,
      );
      for (int star = 5; star >= 1; star--) {
        expect(
          find.byKey(ValueKey<String>('rating-distribution-$star')),
          findsOneWidget,
        );
      }
      expect(tester.takeException(), isNull);
    });

    testWidgets('rating ring keeps its centre inside the ring at 200% text', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 2400);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      // 不带开课列表：只测评分环本身，避免混入其他窄屏布局问题。
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        detailPayload: const CourseDetailPayload(
          id: 42,
          primaryCode: '100001',
          name: '高等数学(A)上',
          department: '数学科学学院',
          creditX10: 50,
          ratingAvg: 4.9,
          reviewCount: 12,
          ratingDistribution: <int>[0, 0, 1, 1, 3],
        ),
      );
      await tester.pumpWidget(
        _app(
          _container(courseRepo: course),
          const CourseDetailPage(courseId: 42),
          textScale: 2,
        ),
      );
      await tester.pumpAndSettle();

      // 320dp + 200%：环心文字等比缩进 104dp 环内，不溢出也不压住弧线。
      expect(tester.takeException(), isNull);
      final Finder ring = find.byKey(const Key('course-rating-ring'));
      final Finder score = find.byKey(
        const ValueKey<String>('course-rating-score'),
      );
      expect(ring, findsOneWidget);
      expect(score, findsOneWidget);
      final Rect ringRect = tester.getRect(ring);
      final Rect scoreRect = tester.getRect(score);
      expect(ringRect.inflate(0.5).contains(scoreRect.topLeft), isTrue);
      expect(ringRect.inflate(0.5).contains(scoreRect.bottomRight), isTrue);
      // 分数仍比 100% 字号时大（不是把可读性缩回去）。
      expect(scoreRect.height, greaterThan(24));
    });

    testWidgets('rating distribution bars brighten toward five stars', (
      tester,
    ) async {
      final FakeCourseRepository course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
      );
      await pumpDetail(tester, course, size: const Size(390, 1400));

      Color fill(int star) {
        final Iterable<Container> containers = tester.widgetList<Container>(
          find.descendant(
            of: find.byKey(ValueKey<String>('rating-distribution-$star')),
            matching: find.byType(Container),
          ),
        );
        // 行内最后一个 Container 是填充条（前一个是轨道）。
        return (containers.last.decoration! as BoxDecoration).color!;
      }

      // Web ROW_OPACITY = [0.95, 0.72, 0.5, 0.34, 0.24]（5★ → 1★）：
      // 高分满亮、低分窄暗；配色与环的 warning 起点呼应。
      expect(fill(5).a, closeTo(0.95, .01));
      expect(fill(4).a, closeTo(0.72, .01));
      expect(fill(1).a, closeTo(0.24, .01));
      expect(fill(5).a, greaterThan(fill(4).a));
      expect(fill(4).a, greaterThan(fill(1).a));
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

    testWidgets('rating ring keeps the score centred at phone width', (
      tester,
    ) async {
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload().copyWith(ratingAvg: 4.9),
      );
      await pumpDetail(tester, course);
      tester.view.physicalSize = const Size(390, 1200);
      await tester.pumpAndSettle();
      // 环心：分数与 / 5.0 各自一行、水平同轴居中，不溢出。
      final score = find.byKey(const ValueKey('course-rating-score'));
      expect(score, findsOneWidget);
      expect(tester.widget<Text>(score).data, '4.9');
      final Finder outOf = find.text('/ 5.0');
      expect(outOf, findsOneWidget);
      expect(
        tester.getCenter(score).dx,
        closeTo(tester.getCenter(outOf).dx, 0.5),
      );
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

    testWidgets('anonymous switch still toggles in the single-line meta row', (
      tester,
    ) async {
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
      );
      await pumpDetail(tester, course);
      await tester.tap(find.text('写课评'));
      await tester.pumpAndSettle();
      expect(tester.widget<Switch>(find.byType(Switch)).value, isTrue);

      // 单行元信息在手机宽度下横向滚动：先滑到匿名开关再切换。
      await tester.dragUntilVisible(
        find.byType(Switch),
        find.byKey(const Key('course-review-offering')),
        const Offset(-120, 0),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.byType(Switch));
      await tester.pumpAndSettle();
      expect(tester.widget<Switch>(find.byType(Switch)).value, isFalse);

      await tester.tap(find.byKey(const ValueKey('review-rating-5')));
      _replaceReviewEditorText(tester, '公开评价');
      await tester.pump();
      await tester.tap(find.text('发布评价'));
      await tester.pumpAndSettle();
      expect(course.createInputs.single.isAnonymous, isFalse);
      await tester.pump(const Duration(seconds: 5));
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

    testWidgets(
      'cached AI summary starts collapsed with refresh in the header',
      (tester) async {
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
        final Finder refresh = find.byKey(
          const ValueKey<String>('ai-summary-refresh'),
        );
        expect(find.text('#重点清晰'), findsNothing);
        // 刷新常驻头部行（与标题垂直居中同一行），不再单独占一行。
        expect(refresh, findsOneWidget);
        expect(
          (tester.getCenter(refresh).dy -
                  tester.getCenter(find.text(l.courseDetailAiSummary)).dy)
              .abs(),
          lessThan(2),
        );
        expect(tester.getSize(refresh).height, greaterThanOrEqualTo(44));
        // 刷新与展开 chevron 并排贴右，中间没有额外空隙。
        final Finder toggle = find.byKey(
          const ValueKey<String>('ai-summary-toggle'),
        );
        expect(
          tester.getRect(toggle).left,
          closeTo(tester.getRect(refresh).right, .5),
        );
        expect(
          tester.getRect(toggle).right,
          closeTo(tester.getSize(find.byType(CourseDetailPage)).width - 5, .5),
        );
        await tester.ensureVisible(find.text(l.courseDetailAiSummary));
        await tester.tap(find.text(l.courseDetailAiSummary));
        await tester.pumpAndSettle();
        expect(find.text('#重点清晰'), findsOneWidget);
        // 展开后刷新仍在头部行，没有孤立刷新行。
        expect(
          (tester.getCenter(refresh).dy -
                  tester.getCenter(find.text(l.courseDetailAiSummary)).dy)
              .abs(),
          lessThan(2),
        );
      },
    );

    testWidgets('AI summary keeps verdict inline and scrolls tags and quotes', (
      tester,
    ) async {
      final course = FakeCourseRepository(
        _client(),
        detailPayload: _detailPayload(),
        summaryStatus: 'cached',
        summaryPayload: const CourseAiSummaryPayload(
          consensus: 'recommend',
          keywords: <String>['重点清晰', '作业偏多', '给分友好', '老师负责', '内容扎实', '节奏偏快'],
          pros: <String>['材料完整'],
          cons: <String>['作业偏多'],
          representativeReviews: <CourseAiSummaryRepresentativeReview>[
            CourseAiSummaryRepresentativeReview(
              excerpt: '老师讲得很清楚，课件也整理得很好，期末复习压力不大。',
              sentiment: 'positive',
            ),
            CourseAiSummaryRepresentativeReview(
              excerpt: '作业多。',
              sentiment: 'negative',
            ),
          ],
        ),
      );
      await pumpDetail(tester, course, size: const Size(390, 1600));
      final l = AppLocalizations.of(
        tester.element(find.byType(CourseDetailPage)),
      );
      final CourseCopy copy = CourseCopy(l);
      // 结论 pill 与标题同行（折叠态也可见），不再单独占一行。
      final Finder verdict = find.text(copy.summaryConsensus('recommend'));
      expect(verdict, findsOneWidget);
      expect(
        (tester.getCenter(verdict).dy -
                tester.getCenter(find.text(l.courseDetailAiSummary)).dy)
            .abs(),
        lessThan(2),
      );
      await tester.ensureVisible(find.text(l.courseDetailAiSummary));
      await tester.tap(find.text(l.courseDetailAiSummary));
      await tester.pumpAndSettle();

      // 关键词单行横滑：同一基线，不换行，也没有孤立的「关键词：」标签。
      expect(find.text('${copy.summaryKeywords}：'), findsNothing);
      final double firstTag = tester
          .getCenter(find.textContaining('重点清晰', findRichText: true))
          .dy;
      final double lastTag = tester
          .getCenter(find.textContaining('节奏偏快', findRichText: true))
          .dy;
      expect(lastTag, closeTo(firstTag, 0.5));
      expect(
        find.ancestor(
          of: find.textContaining('节奏偏快', findRichText: true),
          matching: find.byWidgetPredicate(
            (w) =>
                w is SingleChildScrollView &&
                w.scrollDirection == Axis.horizontal,
          ),
        ),
        findsOneWidget,
      );
      // 优缺点合为一列：+ 在前、− 在后，左缘对齐。
      final Offset pro = tester.getTopLeft(
        find.byKey(const ValueKey<String>('summary-pro')),
      );
      final Offset con = tester.getTopLeft(
        find.byKey(const ValueKey<String>('summary-con')),
      );
      expect(con.dy, greaterThan(pro.dy));
      expect(con.dx, closeTo(pro.dx, 0.5));
      // 多条代表性评价等高并排（不会一高一矮把卡片重心拉偏）。
      final Size good = tester.getSize(
        find
            .ancestor(
              of: find.text(copy.summarySentiment('positive')),
              matching: find.byType(Container),
            )
            .first,
      );
      final Size bad = tester.getSize(
        find
            .ancestor(
              of: find.text(copy.summarySentiment('negative')),
              matching: find.byType(Container),
            )
            .first,
      );
      expect(bad.height, closeTo(good.height, 0.5));
      expect(bad.width, closeTo(good.width, 0.5));
      expect(tester.takeException(), isNull);
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
      final Finder firstHelpfulChip = _reviewAction(4, 'helpful');
      expect(
        find.descendant(of: firstHelpfulChip, matching: find.text('2')),
        findsOneWidget,
      );
      await tester.tap(firstHelpfulChip);
      await tester.pumpAndSettle();

      expect(course.helpfulCalls, <(int, bool)>[(4, true)]);
      expect(
        find.descendant(of: firstHelpfulChip, matching: find.text('3')),
        findsOneWidget,
      );
    });
  });
}
