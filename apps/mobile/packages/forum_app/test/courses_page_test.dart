import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:core/core.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/courses/catalog_page.dart';
import 'package:forum_app/src/pages/courses/detail_page.dart';
import 'package:forum_app/src/pages/courses/course_common.dart';
import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/providers.dart';

import 'fixtures/page_fixtures.dart';

/// 课程目录/详情/课评 UI 行为测试：自包含 FakeCourseRepository，
/// 按方法分发 canned payload（不触网），镜像仓库 wire 语义
/// （列表 hasNext 翻页、多值筛选、cursor、offeringId 聚焦、写评/有用/收藏）。

class _MemoryTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
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
    this.failBookmark = false,
  });

  final CourseDetailPayload? detailPayload;
  final CourseRelatedResult? relatedPayload;
  final List<ReviewPayload> reviewPayloads;

  /// page → (courses, hasNext)。
  final Map<int, (List<CourseSummaryPayload>, bool)> listPages;
  final String summaryStatus;
  bool failBookmark;
  Object? createError;

  final List<CourseRepoCall> listCalls = <CourseRepoCall>[];
  final List<CourseRepoCall> reviewCalls = <CourseRepoCall>[];
  final List<int> bookmarkCalls = <int>[];
  final List<CreateCourseReviewInput> createInputs =
      <CreateCourseReviewInput>[];
  final List<(int, bool)> helpfulCalls = <(int, bool)>[];

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
    return CourseAiSummaryResult(status: summaryStatus);
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
    return ReviewListResult(list: reviewPayloads, total: reviewPayloads.length);
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
      contentHtml: '',
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
    return true;
  }
}

class FakePageRepository extends PageRepository {
  FakePageRepository(super.client);

  @override
  Future<PagePayload> fetch(String path) async {
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

List<ReviewPayload> _reviewPayloads() {
  return <ReviewPayload>[
    const ReviewPayload(
      id: 1,
      offeringId: 901,
      rating: 5,
      content: '好课',
      contentHtml: '',
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
      contentHtml: '',
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
      contentHtml: '',
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

Widget _app(ProviderContainer container, Widget home) {
  return UncontrolledProviderScope(
    container: container,
    child: MaterialApp(
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      locale: const Locale('zh'),
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

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

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
      await tester.tap(find.byIcon(Icons.add));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).last, '李四');
      await tester.tap(find.byIcon(Icons.add));
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
      final list = find.byType(ListView).first;
      await tester.drag(list, const Offset(0, -6000));
      await tester.pumpAndSettle();
      // Lazy rows revise the scroll extent after layout, especially at larger
      // font sizes. Settle at the real end before checking dock clearance.
      final position = tester.widget<ListView>(list).controller!.position;
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
      expect(find.text('01班'), findsOneWidget);
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
      expect(find.text('很不错').hitTestable(), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

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
        find.text('很不错'),
        180,
        scrollable: find.byType(Scrollable).first,
      );
      expect(
        tester.getTopLeft(find.text('很不错')).dy,
        lessThan(tester.getTopLeft(find.text('好课')).dy),
      );
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
      await tester.tap(
        find
            .descendant(
              of: find.byType(BottomSheet),
              matching: find.byIcon(Icons.star_border),
            )
            .last,
      );
      await tester.enterText(find.byType(TextField).last, '保留这段评价');
      await tester.tap(find.text('发布评价'));
      await tester.pumpAndSettle();
      expect(find.text('你已评价过该开课实例。'), findsOneWidget);
      expect(tester.getTopLeft(find.text('你已评价过该开课实例。')).dy, lessThan(160));
      expect(find.text('保留这段评价'), findsOneWidget);
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

      // 选 5 星 + 输入内容（textarea 是页面唯一输入框）。
      final Finder sheetStars = find.descendant(
        of: find.byType(BottomSheet),
        matching: find.byIcon(Icons.star_border),
      );
      await tester.tap(sheetStars.last);
      await tester.pump();
      await tester.enterText(find.byType(TextField).last, '老师讲得清楚');
      await tester.tap(find.text('发布评价'));
      await tester.pumpAndSettle();

      expect(course.createInputs, hasLength(1));
      expect(course.createInputs.single.offeringId, 901);
      expect(course.createInputs.single.rating, 5);
      expect(course.createInputs.single.isAnonymous, isTrue);
      expect(course.createInputs.single.content, '老师讲得清楚');
      expect(find.text('老师讲得清楚'), findsOneWidget);

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
        of: find.text('有用').first,
        matching: find.byType(InkWell),
      );
      await tester.tap(find.text('有用').first);
      await tester.pumpAndSettle();

      expect(course.helpfulCalls, <(int, bool)>[(4, true)]);
      expect(
        find.descendant(of: firstHelpfulChip, matching: find.text('3')),
        findsOneWidget,
      );
    });
  });
}
