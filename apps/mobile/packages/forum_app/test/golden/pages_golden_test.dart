import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:forum_app/src/pages/auth/login_page.dart';
import 'package:forum_app/src/pages/courses/catalog_page.dart';
import 'package:forum_app/src/pages/courses/detail_page.dart';
import 'package:forum_app/src/pages/home/home_page.dart';
import 'package:forum_app/src/pages/messages/messages_page.dart';
import 'package:forum_app/src/pages/notifications/notifications_page.dart';
import 'package:forum_app/src/pages/schedule/schedule_page.dart';
import 'package:forum_app/src/pages/search/search_page.dart';
import 'package:forum_app/src/pages/topic/topic_page.dart';
import 'package:forum_app/src/pages/wiki/wiki_home_page.dart';
import 'package:forum_app/src/pages/wiki/wiki_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:core/core.dart';

import '../golden_helper.dart';
import '../pages_smoke_test.dart'
    show
        FakePageRepository,
        FakeTopicRepository,
        MemoryTokenStorage,
        NoopOfflineCache;

/// 返回单条通知的 NotificationRepository(页面 golden 用)。
class OneNotificationRepository extends NotificationRepository {
  OneNotificationRepository(super.client);

  @override
  Future<NotificationListResponse> fetchNotifications({
    String filter = 'all',
    int cursor = 0,
    int limit = 20,
  }) async {
    final NotificationPayload n = NotificationPayload(
      id: 1,
      eventType: 'reply',
      isRead: false,
      // 固定过去日期(>7 天):timeAgo 回退绝对日期,不随 CI 运行时刻漂移。
      createdAt: '2025-01-15T10:00:00+08:00',
      title: '有人回复了你的话题',
      content: '回复内容预览',
      actor: const NotificationActorPayload(
        id: 2,
        username: 'bob',
        avatarUrl: '',
      ),
      payload: const NotificationInnerPayload(
        actorId: 2,
        topicId: 100,
        topicTitle: '移动端测试话题',
      ),
    );
    return NotificationListResponse(
      items: [n],
      nextCursor: 0,
      hasNext: false,
      unreadCount: 1,
    );
  }
}

/// Route A 页面 golden 用内存 client（fake 仓库构造占位，不发真实网络）。
GfApiClient buildGoldenClient() => GfApiClient(
  dio: Dio(),
  tokenStorage: MemoryTokenStorage(),
  baseUrl: 'http://fake.local',
);

/// 最小 layout JSON（页面级数据通道 payload 解析必需；与 page_fixtures 同形，
/// 自包含在 golden 文件内）。
Map<String, dynamic> goldenLayoutJson() => <String, dynamic>{
  'site': <String, dynamic>{
    'name': 'yourtj',
    'description': '同济大学校园论坛',
    'logo': '',
    'favicon': '',
    'brandType': 'text',
    'brandText': 'yourtj',
    'brandImage': '',
  },
  'viewer': <String, dynamic>{
    'id': 1,
    'username': 'viewer',
    'email': '',
    'avatarUrl': '',
    'isAuthenticated': false,
    'canAccessAdmin': false,
    'isModerator': false,
    'requiresEmailVerification': false,
  },
  'sidebar': <String, dynamic>{'categories': <Object>[], 'activeKey': ''},
  'footer': <String, dynamic>{'links': <Object>[], 'primary': <Object>[]},
  'unread': <String, dynamic>{'notifications': false, 'messages': false},
  'theme': <String, dynamic>{
    'enabled': false,
    'current': 'light',
    'themeColor': '#2563eb',
  },
};

/// 课程目录页 SSR 值域 props（院系/学期/校区）。
Map<String, dynamic> courseCatalogOptionsJson() => <String, dynamic>{
  'component': PageComponent.course,
  'props': <String, dynamic>{
    'departments': <String>['数学科学学院', '物理科学与工程学院'],
    'terms': <Map<String, String>>[
      <String, String>{'value': '2025-2026-1', 'label': '2025-2026 第一学期'},
      <String, String>{'value': '2025-2026-2', 'label': '2025-2026 第二学期'},
    ],
    'campuses': <String>['四平路校区'],
  },
  'meta': <String, dynamic>{'title': '课程目录'},
  'layout': goldenLayoutJson(),
  'url': '/courses',
  'version': '1.0',
};

/// wiki 详情页 PagePayload JSON（`props.page` 为 WikiPageDetail 形状）。
Map<String, dynamic> wikiDetailPayloadJson() => <String, dynamic>{
  'component': 'wiki.detail',
  'props': <String, dynamic>{
    'page': <String, dynamic>{
      'id': 11,
      'topicId': 0,
      'namespace': 'guide',
      'path': 'guide/start',
      'title': '快速开始',
      'content':
          '<h2 id="intro">介绍</h2><p>欢迎阅读 wiki 正文，这里是第一段。</p>'
          '<h3 id="details">细节说明</h3><p>更详细的内容说明。</p>',
      'toc': <Map<String, dynamic>>[
        <String, dynamic>{'level': 2, 'id': 'intro', 'text': '介绍'},
        <String, dynamic>{'level': 3, 'id': 'details', 'text': '细节说明'},
      ],
      'updatedAt': '2026-08-10T15:00:00+08:00',
      'likeCount': 5,
      'viewCount': 42,
      'postCount': 0,
      'liked': false,
      'bookmarked': false,
      'watched': false,
      'canEdit': true,
      'publishedRevisionNo': 0,
      'editUrl': 'https://github.com/yourtj/wiki/edit/main/guide/start.md',
      'historyUrl': '',
    },
    'contributors': <Object>[],
    'hotTopics': <Object>[],
  },
  'meta': <String, dynamic>{'title': '快速开始'},
  'layout': goldenLayoutJson(),
  'url': '/wiki/guide/start',
  'version': '1.0',
};

/// wiki 首页数据：2 命名空间 + 2 最近更新。
Map<String, dynamic> wikiHomeDataJson() => <String, dynamic>{
  'namespaces': <Map<String, dynamic>>[
    <String, dynamic>{
      'name': '指南',
      'description': '社区使用指南',
      'sortOrder': 10,
      'pageCount': 3,
      'updatedAt': '2026-08-10T15:00:00+08:00',
      'firstPagePath': 'guide/getting-started',
    },
    <String, dynamic>{
      'name': '校园生活',
      'description': '同济校园生活攻略',
      'sortOrder': 20,
      'pageCount': 12,
      'updatedAt': '2026-08-09T12:00:00+08:00',
      'firstPagePath': 'campus/food',
    },
  ],
  'recent': <Map<String, dynamic>>[
    <String, dynamic>{
      'pageId': 1001,
      'path': 'guide/content',
      'title': '内容规范',
      'updatedAt': '2026-08-10T14:00:00+08:00',
    },
    <String, dynamic>{
      'pageId': 1002,
      'path': 'campus/food',
      'title': '食堂推荐',
      'updatedAt': '2026-08-08T09:00:00+08:00',
    },
  ],
};

/// 课程目录 fake：list 按页返回（golden 只用第 1 页）。
class FakeCourseRepository extends CourseRepository {
  FakeCourseRepository(
    super.client, {
    this.detailPayload,
    this.relatedPayload,
    this.reviewPayloads = const <ReviewPayload>[],
    this.listPages = const <int, (List<CourseSummaryPayload>, bool)>{},
    this.aiResult,
  });

  final CourseDetailPayload? detailPayload;
  final CourseRelatedResult? relatedPayload;
  final List<ReviewPayload> reviewPayloads;
  final Map<int, (List<CourseSummaryPayload>, bool)> listPages;
  final CourseAiSummaryResult? aiResult;

  @override
  Future<CourseListResultPayload> list({
    String? keyword,
    List<String> departments = const <String>[],
    List<String> terms = const <String>[],
    List<String> campuses = const <String>[],
    List<String> instructors = const <String>[],
    bool onlyWithReviews = false,
    String sortBy = '',
    int page = 1,
    int size = 20,
  }) async {
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
  Future<CourseDetailPayload> detail(int courseId) async =>
      detailPayload ??
      CourseDetailPayload(
        id: courseId,
        primaryCode: '100001',
        name: '高等数学(A)上',
        department: '数学科学学院',
        creditX10: 50,
      );

  @override
  Future<CourseRelatedResult> related(int courseId) async =>
      relatedPayload ??
      const CourseRelatedResult(
        teacherOtherCourses: <RelatedCourseItem>[],
        sameCourseOtherTeachers: <RelatedCourseItem>[],
      );

  @override
  Future<CourseAiSummaryResult> aiSummary(
    int courseId, {
    bool refresh = false,
    bool check = false,
  }) async => aiResult ?? const CourseAiSummaryResult(status: 'none');

  @override
  Future<ReviewListResult> reviews(
    int courseId, {
    int? offeringId,
    String? cursor,
    int pageSize = 20,
  }) async {
    if (cursor != null) {
      return const ReviewListResult(list: <ReviewPayload>[], total: 0);
    }
    return ReviewListResult(list: reviewPayloads, total: reviewPayloads.length);
  }

  @override
  Future<bool> bookmark({
    required int courseId,
    required bool bookmarked,
  }) async => true;
}

/// 课程目录页页面通道 fake：只回 `/courses` 值域 props。
class CatalogPageRepository extends PageRepository {
  CatalogPageRepository(super.client);

  @override
  Future<PagePayload> fetch(String path) async {
    if (path == '/courses') {
      return PagePayload.fromJson(courseCatalogOptionsJson());
    }
    throw UnimplementedError('unexpected page path: $path');
  }
}

/// wiki 页面通道 fake：只回 `/wiki/guide/start` 详情 payload。
class WikiDetailPageRepository extends PageRepository {
  WikiDetailPageRepository(super.client);

  @override
  Future<PagePayload> fetch(String path) async {
    if (path == '/wiki/guide/start') {
      return PagePayload.fromJson(wikiDetailPayloadJson());
    }
    throw UnimplementedError('unexpected page path: $path');
  }
}

/// wiki 首页 fake：固定 home 数据（tree 不在首页使用，一并喂空树）。
class FakeWikiRepository extends WikiRepository {
  FakeWikiRepository(super.client, {required this.homeData});

  final WikiHomeData homeData;

  @override
  Future<WikiHomeData> home() async => homeData;

  @override
  Future<WikiTreeResult> tree() async =>
      const WikiTreeResult(namespaces: <WikiTreeNamespace>[]);
}

/// 排课器 fake：固定 130 学期（11 节新制）+ 11 节作息 + 无 P11 更新。
class FakePkRepository extends PkRepository {
  FakePkRepository(super.client);

  @override
  Future<List<PkCourseByMajorItem>> coursesByMajor({
    required int grade,
    required String code,
    required int calendarId,
  }) async => [
    PkCourseByMajorItem.fromJson({
      'courseCode': '122004',
      'courseName': '高等数学(A)上',
      'faculty': '数学科学学院',
      'credit': 4,
      'grade': grade,
      'courseNature': ['必修'],
      'courses': [],
    }),
    PkCourseByMajorItem.fromJson({
      'courseCode': '122006',
      'courseName': '大学物理(B)',
      'faculty': '物理科学与工程学院',
      'credit': 3,
      'grade': grade,
      'courseNature': ['必修'],
      'courses': [],
    }),
  ];

  static const List<PkCalendarItem> calendarsFixture = <PkCalendarItem>[
    PkCalendarItem(
      calendarId: 130,
      calendarName: '2025-2026学年第1学期',
      startDate: '2026-08-31',
    ),
  ];

  /// 当前 11 节作息：1-8 白天 08:00..17:05，9 晚间 18:30 起（无 12 节）。
  static const SectionTimesPayload sectionTimesFixture = SectionTimesPayload(
    sectionTimes: <SectionTimeSetting>[
      SectionTimeSetting(section: 1, start: '08:00', end: '08:45'),
      SectionTimeSetting(section: 2, start: '08:50', end: '09:35'),
      SectionTimeSetting(section: 3, start: '10:00', end: '10:45'),
      SectionTimeSetting(section: 4, start: '10:50', end: '11:35'),
      SectionTimeSetting(section: 5, start: '13:30', end: '14:15'),
      SectionTimeSetting(section: 6, start: '14:20', end: '15:05'),
      SectionTimeSetting(section: 7, start: '15:30', end: '16:15'),
      SectionTimeSetting(section: 8, start: '16:20', end: '17:05'),
      SectionTimeSetting(section: 9, start: '18:30', end: '19:15'),
      SectionTimeSetting(section: 10, start: '19:20', end: '20:05'),
      SectionTimeSetting(section: 11, start: '20:10', end: '20:55'),
    ],
    maxRowsDefault: 11,
  );

  @override
  Future<List<PkCalendarItem>> calendars() async => calendarsFixture;

  @override
  Future<SectionTimesPayload?> sectionTimes() async => sectionTimesFixture;

  @override
  Future<String?> latestUpdate() async => null;
}

/// 排课器持久化种子：`pk.plans` = 方案 1 含两门已选（status=2）课程，
/// 周 1-8、节次均在 11 节新制内；连同 activePlanId/majorSelected 一起写入，
/// 让 SchedulePage 直接以「已选好学期+已排课」状态渲染网格课程卡。
String schedulePlansSeedJson() {
  Map<String, dynamic> arrangement(
    int day,
    List<int> sections,
    String room,
    String teacherAndCode,
  ) => <String, dynamic>{
    'arrangementText': '1-8周 周$day ${sections.first}-${sections.last}节 $room',
    'occupyDay': day,
    'occupyTime': sections,
    'occupyWeek': <int>[1, 2, 3, 4, 5, 6, 7, 8],
    'occupyRoom': room,
    'teacherAndCode': teacherAndCode,
  };

  Map<String, dynamic> course({
    required String baseCode,
    required String className,
    required String courseName,
    required double credit,
    required String courseType,
    required String teacherName,
    required String teacherCode,
    required int day,
    required List<int> sections,
    required String room,
  }) => <String, dynamic>{
    'courseCode': baseCode,
    'courseName': courseName,
    'courseNameReserved': courseName,
    'credit': credit,
    'courseType': courseType,
    'teacher': <Map<String, String>>[
      <String, String>{'teacherName': teacherName, 'teacherCode': teacherCode},
    ],
    'status': 2,
    'courseDetail': <Map<String, dynamic>>[
      <String, dynamic>{
        'code': className,
        'status': 2,
        'campus': '四平',
        'teachingLanguage': '',
        'teachers': <Map<String, String>>[
          <String, String>{
            'teacherName': teacherName,
            'teacherCode': teacherCode,
          },
        ],
        'arrangementInfo': <Map<String, dynamic>>[
          arrangement(day, sections, room, '$teacherName($teacherCode)'),
        ],
      },
    ],
  };

  return jsonEncode(<dynamic>[
    <String, dynamic>{
      'id': 'plan-a',
      'name': '方案 1',
      'createdAt': 1725000000000,
      'stagedCourses': <Map<String, dynamic>>[
        course(
          baseCode: '122004',
          className: '122004.01',
          courseName: '高等数学(A)上',
          credit: 4,
          courseType: '必',
          teacherName: '张老师',
          teacherCode: 'T001',
          day: 1,
          sections: <int>[3, 4],
          room: 'A101',
        ),
        course(
          baseCode: '122006',
          className: '122006.01',
          courseName: '大学物理(B)',
          credit: 3,
          courseType: '选',
          teacherName: '李老师',
          teacherCode: 'T002',
          day: 2,
          sections: <int>[1, 2],
          room: 'B202',
        ),
      ],
      'selectedCourses': <String>['122004.01', '122006.01'],
      'customEvents': <Object>[],
    },
  ]);
}

/// 课程目录 golden 行：带评分（第 1 行）、无评分但有评价数（第 2 行）、
/// 无教师无评分（第 3 行），覆盖目录行三种视觉形态。
List<CourseSummaryPayload> courseCatalogRowsFixture() =>
    const <CourseSummaryPayload>[
      CourseSummaryPayload(
        id: 101,
        primaryCode: '100001',
        name: '高等数学(A)上',
        department: '数学科学学院',
        creditX10: 50,
        teacherName: '张三',
        instructors: <String>['张三'],
        recentTerms: <String>['2025-2026-1', '2025-2026-2'],
        ratingAvg: 4.5,
        reviewCount: 5,
      ),
      CourseSummaryPayload(
        id: 102,
        primaryCode: '100002',
        name: '线性代数',
        department: '数学科学学院',
        creditX10: 30,
        teacherName: '李四',
        instructors: <String>['李四'],
        recentTerms: <String>['2025-2026-1'],
        reviewCount: 2,
      ),
      CourseSummaryPayload(
        id: 103,
        primaryCode: '100003',
        name: '数据结构',
        department: '计算机科学与技术学院',
        creditX10: 40,
        recentTerms: <String>['2025-2026-2'],
      ),
    ];

/// 课程详情 golden fixture：评分分布 + 两个教学班 + 原名标注。
CourseDetailPayload courseDetailFixture() => const CourseDetailPayload(
  id: 42,
  primaryCode: '100001',
  name: '高等数学(A)上',
  department: '数学科学学院',
  creditX10: 50,
  teacherName: '张三',
  aliases: <String>['高数'],
  legacyNames: <String>['旧课名'],
  reviewScope: 'teacher',
  offerings: <CourseOfferingPayload>[
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
      ratingAvg: 3.5,
      reviewCount: 2,
    ),
  ],
  ratingAvg: 3.9,
  reviewCount: 29,
  ratingDistribution: <int>[12, 8, 5, 3, 1],
);

CourseRelatedResult courseRelatedFixture() => const CourseRelatedResult(
  teacherOtherCourses: <RelatedCourseItem>[
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
  sameCourseOtherTeachers: <RelatedCourseItem>[
    RelatedCourseItem(
      id: 47,
      primaryCode: '100001',
      name: '高等数学(A)上',
      department: '数学科学学院',
      teacherName: '王五',
      ratingAvg: 4.0,
      ratingCount: 1,
      reviewCount: 1,
    ),
  ],
  lineage: <RelationItem>[
    RelationItem(
      relationId: 11,
      fromCourseId: 41,
      fromName: '高等数学(上)',
      toCourseId: 42,
      toName: '高等数学(A)上',
      relationType: 'EQUIVALENT',
      status: 'merged',
      direction: 'to',
    ),
  ],
);

/// 课评 golden rows：一条带星评分，一条历史无评分（rating=null）。
List<ReviewPayload> courseReviewsFixture() => const <ReviewPayload>[
  ReviewPayload(
    id: 1,
    offeringId: 901,
    rating: 5,
    content: '老师讲课非常清晰，例题丰富，板书工整。',
    contentHtml: '',
    author: ReviewAuthorPayload(kind: 'member', label: 'bob'),
    viewer: ReviewViewerPayload(
      canEdit: false,
      canDelete: false,
      isHelpful: false,
    ),
    helpfulCount: 2,
    createdAt: '2026-09-01T08:00:00+08:00',
    updatedAt: '2026-09-01T08:00:00+08:00',
  ),
  ReviewPayload(
    id: 2,
    offeringId: 901,
    rating: null,
    content: '早年修过这门课，内容很扎实。',
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
];

/// AI 课评摘要 golden fixture（status=cached，直接展开内容）。
CourseAiSummaryResult courseAiSummaryFixture() => const CourseAiSummaryResult(
  status: 'cached',
  summary: CourseAiSummaryPayload(
    consensus: 'recommend',
    keywords: <String>['板书清晰', '例题丰富', '节奏较快'],
    pros: <String>['讲解细致', '作业量适中'],
    cons: <String>['点名较多'],
    representativeReviews: <CourseAiSummaryRepresentativeReview>[
      CourseAiSummaryRepresentativeReview(
        excerpt: '老师讲得很好，跟着例题走收获很大。',
        sentiment: 'positive',
      ),
    ],
  ),
  generatedAt: '2026-09-05T10:00:00+08:00',
);

/// Page-level golden baselines (390x844 mobile surface, Roboto, zh locale).
///
/// Tagged `golden` and excluded from `melos run test` (the CI gate) via
/// `--exclude-tags=golden`. Regenerate intentionally after a visual change:
/// `flutter test --update-goldens test/golden/pages_golden_test.dart`
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  // Golden baselines are rendered by flutter_tester on Linux CI (see
  // golden_helper.dart). flutter_tester rasterizes text differently per
  // host OS (hinting/AA), so the same PNG cannot match on macOS; run
  // goldens on Linux and skip elsewhere.
  final bool skipGoldens = !Platform.isLinux;

  setUp(() {
    SharedPreferences.setMockInitialValues(<String, Object>{});
  });

  Future<ProviderContainer> makeContainer({
    NotificationRepository? notifRepo,
  }) async {
    final storage = MemoryTokenStorage();
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: storage,
      baseUrl: 'http://fake.local',
    );
    final authDio = Dio()
      ..interceptors.add(
        InterceptorsWrapper(
          onRequest: (request, handler) {
            expect(request.path, '/login');
            handler.resolve(
              Response(
                requestOptions: request,
                data: {
                  'component': 'auth.login',
                  'props': {
                    'initialMode': 'login',
                    'redirectUrl': '/',
                    'githubUrl': '/api/auth/github',
                    'googleReady': true,
                  },
                  'layout': goldenLayoutJson(),
                  'meta': {'title': 'Login'},
                  'url': '/login',
                  'version': '1',
                },
              ),
            );
          },
        ),
      );
    addTearDown(authDio.close);
    final container = ProviderContainer(
      overrides: [
        authDioProvider.overrideWithValue(authDio),
        tokenStorageProvider.overrideWithValue(MemoryTokenStorage()),
        pageRepositoryProvider.overrideWithValue(FakePageRepository(client)),
        topicRepositoryProvider.overrideWithValue(FakeTopicRepository(client)),
        notificationRepositoryProvider.overrideWithValue(
          notifRepo ?? OneNotificationRepository(client),
        ),
        offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
        offlineChatCacheProvider.overrideWithValue(NoopOfflineCache()),
      ],
    );
    addTearDown(container.dispose);
    return container;
  }

  Future<void> settleBrandLogo(WidgetTester tester) async {
    final BuildContext context = tester.element(find.byType(Scaffold).first);
    await tester.runAsync(() async {
      await precacheImage(
        const AssetImage('assets/images/brand-default.png'),
        context,
      );
    });
    await tester.pump();
  }

  Future<ProviderContainer> makeRouteAContainer({
    GfApiClient? client,
    PageRepository? pageRepo,
    CourseRepository? courseRepo,
    WikiRepository? wikiRepo,
    PkRepository? pkRepo,
    ScheduleStoreNotifier? notifier,
  }) async {
    final GfApiClient api = client ?? buildGoldenClient();
    final ProviderContainer container = ProviderContainer(
      overrides: <Override>[
        tokenStorageProvider.overrideWithValue(MemoryTokenStorage()),
        pageRepositoryProvider.overrideWithValue(
          pageRepo ?? FakePageRepository(api),
        ),
        courseRepositoryProvider.overrideWithValue(
          courseRepo ?? FakeCourseRepository(api),
        ),
        wikiRepositoryProvider.overrideWithValue(
          wikiRepo ??
              FakeWikiRepository(
                api,
                homeData: WikiHomeData.fromJson(wikiHomeDataJson()),
              ),
        ),
        pkRepositoryProvider.overrideWithValue(pkRepo ?? FakePkRepository(api)),
        if (notifier != null)
          scheduleStoreProvider.overrideWith((ref) => notifier),
        offlineTopicCacheProvider.overrideWithValue(NoopOfflineCache()),
        offlineChatCacheProvider.overrideWithValue(NoopOfflineCache()),
      ],
    );
    addTearDown(container.dispose);
    return container;
  }

  testWidgets('home page golden', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(container: container, child: const HomePage()),
    );
    await settleBrandLogo(tester);
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/home_page.png'),
    );
  });

  testWidgets('home page list golden', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues(<String, Object>{
      'goose:home-feed-mode': 'list',
    });
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(container: container, child: const HomePage()),
    );
    await settleBrandLogo(tester);
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/home_page_list.png'),
    );
  });

  testWidgets('topic page golden', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const TopicPage(topicId: 100),
      ),
    );
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/topic_page.png'),
    );
    // markdown_widget 的 VisibilityDetector 会创建 500ms 延迟 Timer,
    // 需推进时钟让其过期,避免 "Timer is still pending"。
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 600));
  });

  testWidgets('notifications page golden', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const NotificationsPage(),
      ),
    );
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/notifications_page.png'),
    );
  });

  testWidgets('messages page golden', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const MessagesPage(),
      ),
    );
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/messages_page.png'),
    );
  });

  testWidgets(
    'messages new chat sheet golden',
    skip: skipGoldens,
    tags: 'golden',
    (tester) async {
      final container = await makeContainer();
      await pumpPageGolden(
        tester,
        UncontrolledProviderScope(
          container: container,
          child: const MessagesPage(),
        ),
      );
      await tester.tap(find.byTooltip('新私信'));
      await tester.pumpAndSettle();
      await expectLater(
        find.byType(Overlay).first,
        matchesGoldenFile('golden/pages/messages_new_chat.png'),
      );
    },
  );

  testWidgets(
    'search page grouped results golden',
    skip: skipGoldens,
    tags: 'golden',
    (tester) async {
      final container = await makeContainer();
      await pumpPageGolden(
        tester,
        UncontrolledProviderScope(
          container: container,
          child: const SearchPage(),
        ),
      );
      await tester.enterText(find.byType(TextField), '同济');
      await tester.testTextInput.receiveAction(TextInputAction.search);
      await tester.pumpAndSettle();
      await expectLater(
        find.byType(Scaffold).first,
        matchesGoldenFile('golden/pages/search_page.png'),
      );
    },
  );

  testWidgets('login page golden', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
    final container = await makeContainer();
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(container: container, child: const LoginPage()),
    );
    await settleBrandLogo(tester);
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/login_page.png'),
    );
  });
  // ---- Route A 页面（课表 / 课程目录 / 课程详情 / Wiki 首页 / Wiki 详情）----

  Future<void> pumpSeededSchedule(
    WidgetTester tester, {
    required Brightness brightness,
    required String name,
  }) async {
    final ScheduleStoreNotifier notifier = ScheduleStoreNotifier();
    await notifier.ready;
    // 同步清空持久化队列（页面加载仍走 mock prefs 读取，覆盖内部状态）。
    // 注意：不要在这里调 setMajorSelection —— 该方法带「换学期/换专业清空
    // 全方案」语义，会把 prefs 种子的已排课程清空（网格变空、统计归零）；
    // majorSelected 已随 SharedPreferences 种子写入，notifier.ready 直接加载。
    await notifier.flush;
    final ProviderContainer container = await makeRouteAContainer(
      notifier: notifier,
    );
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const SchedulePage(),
      ),
      brightness: brightness,
    );
    await tester.pumpAndSettle();
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/schedule_page_$name.png'),
    );
    // 容器 dispose 会一并释放被 override 的 notifier，勿重复 dispose。
  }

  testWidgets('schedule page golden light', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues(<String, Object>{
      ScheduleStorageKeys.plans: schedulePlansSeedJson(),
      ScheduleStorageKeys.activePlanId: '"plan-a"',
      ScheduleStorageKeys.majorSelected: jsonEncode(<String, Object>{
        'calendarId': 130,
        'grade': 2025,
        'major': 'm1',
        'majorName': '软件工程',
      }),
    });
    await pumpSeededSchedule(
      tester,
      brightness: Brightness.light,
      name: 'light',
    );
  });

  testWidgets('schedule page golden dark', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues(<String, Object>{
      ScheduleStorageKeys.plans: schedulePlansSeedJson(),
      ScheduleStorageKeys.activePlanId: '"plan-a"',
      ScheduleStorageKeys.majorSelected: jsonEncode(<String, Object>{
        'calendarId': 130,
        'grade': 2025,
        'major': 'm1',
        'majorName': '软件工程',
      }),
    });
    await pumpSeededSchedule(tester, brightness: Brightness.dark, name: 'dark');
  });

  Future<void> pumpCourseCatalog(
    WidgetTester tester, {
    required Brightness brightness,
    required String name,
  }) async {
    final ProviderContainer container = await makeRouteAContainer(
      courseRepo: FakeCourseRepository(
        buildGoldenClient(),
        listPages: <int, (List<CourseSummaryPayload>, bool)>{
          1: (courseCatalogRowsFixture(), false),
        },
      ),
      pageRepo: CatalogPageRepository(buildGoldenClient()),
    );
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const CourseCatalogPage(),
      ),
      brightness: brightness,
    );
    await tester.pumpAndSettle();
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/course_catalog_page_$name.png'),
    );
  }

  testWidgets(
    'course catalog page golden light',
    skip: skipGoldens,
    tags: 'golden',
    (tester) async {
      await pumpCourseCatalog(
        tester,
        brightness: Brightness.light,
        name: 'light',
      );
    },
  );

  testWidgets(
    'course catalog page golden dark',
    skip: skipGoldens,
    tags: 'golden',
    (tester) async {
      await pumpCourseCatalog(
        tester,
        brightness: Brightness.dark,
        name: 'dark',
      );
    },
  );

  Future<void> pumpCourseDetail(
    WidgetTester tester, {
    required Brightness brightness,
    required String name,
  }) async {
    final GfApiClient client = buildGoldenClient();
    final ProviderContainer container = await makeRouteAContainer(
      courseRepo: FakeCourseRepository(
        client,
        detailPayload: courseDetailFixture(),
        relatedPayload: courseRelatedFixture(),
        reviewPayloads: courseReviewsFixture(),
        aiResult: courseAiSummaryFixture(),
      ),
    );
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const CourseDetailPage(courseId: 42),
      ),
      brightness: brightness,
    );
    await tester.pumpAndSettle();
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/course_detail_page_$name.png'),
    );
  }

  testWidgets(
    'course detail page golden light',
    skip: skipGoldens,
    tags: 'golden',
    (tester) async {
      await pumpCourseDetail(
        tester,
        brightness: Brightness.light,
        name: 'light',
      );
    },
  );

  testWidgets(
    'course detail page golden dark',
    skip: skipGoldens,
    tags: 'golden',
    (tester) async {
      await pumpCourseDetail(tester, brightness: Brightness.dark, name: 'dark');
    },
  );

  Future<void> pumpWikiHome(
    WidgetTester tester, {
    required Brightness brightness,
    required String name,
  }) async {
    final ProviderContainer container = await makeRouteAContainer(
      wikiRepo: FakeWikiRepository(
        buildGoldenClient(),
        homeData: WikiHomeData.fromJson(wikiHomeDataJson()),
      ),
    );
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const WikiHomePage(),
      ),
      brightness: brightness,
    );
    await tester.pumpAndSettle();
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/wiki_home_page_$name.png'),
    );
  }

  testWidgets(
    'wiki home page golden light',
    skip: skipGoldens,
    tags: 'golden',
    (tester) async {
      await pumpWikiHome(tester, brightness: Brightness.light, name: 'light');
    },
  );

  testWidgets('wiki home page golden dark', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
    await pumpWikiHome(tester, brightness: Brightness.dark, name: 'dark');
  });

  Future<void> pumpWikiDetail(
    WidgetTester tester, {
    required Brightness brightness,
    required String name,
  }) async {
    final ProviderContainer container = await makeRouteAContainer(
      pageRepo: WikiDetailPageRepository(buildGoldenClient()),
    );
    await pumpPageGolden(
      tester,
      UncontrolledProviderScope(
        container: container,
        child: const WikiPage(wikiPath: 'guide/start'),
      ),
      brightness: brightness,
    );
    await tester.pumpAndSettle();
    await expectLater(
      find.byType(Scaffold).first,
      matchesGoldenFile('golden/pages/wiki_page_$name.png'),
    );
  }

  testWidgets('wiki page golden light', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
    await pumpWikiDetail(tester, brightness: Brightness.light, name: 'light');
  });

  testWidgets('wiki page golden dark', skip: skipGoldens, tags: 'golden', (
    tester,
  ) async {
    await pumpWikiDetail(tester, brightness: Brightness.dark, name: 'dark');
  });
}
