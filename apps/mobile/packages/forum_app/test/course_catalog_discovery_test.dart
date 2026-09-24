import 'dart:async';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/courses/catalog_page.dart';
import 'package:forum_app/src/providers.dart';
import 'package:forum_app/src/widgets/app_refresh_indicator.dart';
import 'package:ui_kit/ui_kit.dart';

import 'courses_page_test.dart' show FakePageRepository;
import 'pages_smoke_test.dart' show MemoryTokenStorage;

class _Request {
  _Request(
    this.keyword,
    this.page,
    this.departments,
    this.terms,
    this.campuses,
    this.instructors,
    this.onlyWithReviews,
  );
  final String? keyword;
  final int page;
  final List<String> departments, terms, campuses, instructors;
  final bool onlyWithReviews;
  final result = Completer<CourseListResultPayload>();
}

class _Courses extends CourseRepository {
  _Courses(super.client);
  final requests = <_Request>[];
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
  }) {
    final request = _Request(
      keyword,
      page,
      departments,
      terms,
      campuses,
      instructors,
      onlyWithReviews,
    );
    requests.add(request);
    return request.result.future;
  }
}

class _Pages extends FakePageRepository {
  _Pages(super.client);
  bool fail = false;
  int calls = 0;
  @override
  Future<PagePayload> fetch(String path) async {
    calls++;
    if (fail) throw const NetworkException(fallbackMessage: 'offline');
    return super.fetch(path);
  }
}

CourseListResultPayload _result(
  String name, {
  int id = 1,
  int page = 1,
  bool more = false,
}) => CourseListResultPayload(
  list: [
    CourseSummaryPayload(
      id: id,
      name: name,
      primaryCode: '001$id',
      department: '数学科学学院',
      creditX10: 30,
      teacherName: '张三',
    ),
  ],
  page: page,
  size: 20,
  total: more ? 30 : 1,
  hasNext: more,
);

void main() {
  late _Courses courses;
  late ProviderContainer container;
  late _Pages pages;
  Future<void> pumpCatalog(
    WidgetTester tester, {
    bool optionsFail = false,
    Locale locale = const Locale('zh'),
    double textScale = 1,
  }) async {
    final client = GfApiClient(
      dio: Dio(),
      tokenStorage: MemoryTokenStorage(),
      baseUrl: 'http://fake.local',
    );
    courses = _Courses(client);
    pages = _Pages(client)..fail = optionsFail;
    container = ProviderContainer(
      overrides: [
        courseRepositoryProvider.overrideWithValue(courses),
        pageRepositoryProvider.overrideWithValue(pages),
      ],
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: gfThemeData(Brightness.light),
          locale: locale,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const CourseCatalogPage(),
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(textScaler: TextScaler.linear(textScale)),
            child: child!,
          ),
        ),
      ),
    );
    await tester.pump();
  }

  testWidgets('late initial response cannot replace a newer keyword result', (
    tester,
  ) async {
    await pumpCatalog(tester);
    await tester.enterText(find.byType(TextField).first, '物理');
    await tester.pump(const Duration(milliseconds: 310));
    expect(courses.requests, hasLength(2));
    courses.requests[1].result.complete(_result('新搜索物理'));
    await tester.pumpAndSettle();
    courses.requests[0].result.complete(_result('旧结果数学'));
    await tester.pumpAndSettle();
    expect(find.text('新搜索物理'), findsOneWidget);
    expect(find.text('旧结果数学'), findsNothing);
    await tester.pumpWidget(const SizedBox.shrink());
  });
  Future<void> deliverFirst(WidgetTester tester, {bool more = false}) async {
    courses.requests.first.result.complete(_result('保留的课程', more: more));
    await tester.pump();
    await tester.pump();
  }

  testWidgets(
    'typing invalidates the old request during the debounce interval',
    (tester) async {
      await pumpCatalog(tester);
      await tester.enterText(find.byType(TextField).first, '新查询');
      courses.requests.first.result.complete(_result('不应回来的旧结果'));
      await tester.pump(const Duration(milliseconds: 100));
      expect(find.text('不应回来的旧结果'), findsNothing);
      expect(courses.requests, hasLength(1));
      await tester.pump(const Duration(milliseconds: 210));
      courses.requests.last.result.complete(_result('正确的新结果'));
      await tester.pumpAndSettle();
      expect(find.text('正确的新结果'), findsOneWidget);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets(
    'short results paginate automatically and stale pages cannot append to a new search',
    (tester) async {
      await pumpCatalog(tester);
      await deliverFirst(tester, more: true);
      expect(courses.requests, hasLength(2));
      expect(courses.requests.last.page, 2);
      await tester.enterText(find.byType(TextField).first, '物理');
      await tester.pump(const Duration(milliseconds: 310));
      expect(courses.requests.last.keyword, '物理');
      courses.requests.last.result.complete(_result('新的物理课程', id: 3));
      await tester.pumpAndSettle();
      courses.requests[1].result.complete(_result('迟到的旧课程', id: 2, page: 2));
      await tester.pumpAndSettle();
      expect(find.text('新的物理课程'), findsOneWidget);
      expect(find.text('迟到的旧课程'), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets(
    'pagination failure retains rows and retries the same page only on demand',
    (tester) async {
      await pumpCatalog(tester);
      await deliverFirst(tester, more: true);
      courses.requests.last.result.completeError(
        const NetworkException(fallbackMessage: 'offline'),
      );
      await tester.pumpAndSettle();
      expect(find.text('保留的课程'), findsOneWidget);
      expect(find.text('重试'), findsOneWidget);
      await tester.pump(const Duration(seconds: 1));
      expect(courses.requests, hasLength(2));
      await tester.tap(find.text('重试'));
      await tester.pump();
      expect(courses.requests.last.page, 2);
      courses.requests.last.result.complete(_result('重试追加课程', id: 2, page: 2));
      await tester.pumpAndSettle();
      expect(find.text('保留的课程'), findsOneWidget);
      expect(find.text('重试追加课程'), findsOneWidget);
      expect(find.text('重试'), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets('duplicate pages do not create an automatic request loop', (
    tester,
  ) async {
    await pumpCatalog(tester);
    await deliverFirst(tester, more: true);
    courses.requests.last.result.complete(
      _result('保留的课程', page: 2, more: true),
    );
    await tester.pumpAndSettle();
    expect(find.text('保留的课程'), findsOneWidget);
    expect(find.text('暂未获取到更多课程，请重试本页。'), findsOneWidget);
    await tester.pump(const Duration(seconds: 1));
    expect(courses.requests, hasLength(2));
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets('refresh failure retains the current results and exposes retry', (
    tester,
  ) async {
    await pumpCatalog(tester);
    await deliverFirst(tester);
    final refresh = tester
        .widget<AppRefreshIndicator>(find.byType(AppRefreshIndicator))
        .onRefresh();
    await tester.pump();
    expect(find.text('保留的课程'), findsOneWidget);
    courses.requests.last.result.completeError(
      const NetworkException(fallbackMessage: 'offline'),
    );
    await refresh;
    await tester.pumpAndSettle();
    expect(find.text('保留的课程'), findsOneWidget);
    expect(find.text('重试'), findsOneWidget);
    await tester.tap(find.text('重试'));
    await tester.pump();
    courses.requests.last.result.complete(_result('刷新后的课程', id: 3));
    await tester.pumpAndSettle();
    expect(find.text('刷新后的课程'), findsOneWidget);
    expect(find.text('保留的课程'), findsNothing);
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets(
    'filter search preserves hidden selections and applies the union',
    (tester) async {
      await pumpCatalog(tester);
      await deliverFirst(tester);
      await tester.pumpAndSettle();
      await tester.tap(find.text('院系'));
      await tester.pumpAndSettle();
      final input = find.descendant(
        of: find.byKey(const Key('course-filter-search')),
        matching: find.byType(TextField),
      );
      await tester.enterText(input, '数学');
      await tester.pumpAndSettle();
      await tester.tap(find.byType(CheckboxListTile));
      await tester.enterText(input, '物理');
      await tester.pumpAndSettle();
      await tester.tap(find.byType(CheckboxListTile));
      await tester.tap(find.text('完成'));
      await tester.pump();
      expect(courses.requests.last.departments, ['数学科学学院', '物理科学与工程学院']);
      courses.requests.last.result.complete(_result('筛选课程'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('重置搜索和筛选'));
      await tester.pump();
      expect(courses.requests.last.departments, isEmpty);
      courses.requests.last.result.complete(_result('全部课程'));
      await tester.pumpAndSettle();
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets(
    'filter option failure has its own retry without hiding courses',
    (tester) async {
      await pumpCatalog(tester, optionsFail: true);
      await deliverFirst(tester);
      await tester.pumpAndSettle();
      expect(find.text('保留的课程'), findsOneWidget);
      expect(find.text('筛选项加载失败，请重试。'), findsOneWidget);
      pages.fail = false;
      await tester.tap(find.text('重试'));
      await tester.pumpAndSettle();
      expect(pages.calls, 2);
      expect(find.text('筛选项加载失败，请重试。'), findsNothing);
      await tester.tap(find.text('院系'));
      await tester.pumpAndSettle();
      expect(find.byType(CheckboxListTile), findsNWidgets(2));
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets(
    'term filter searches both codes and labels and clears selection',
    (tester) async {
      await pumpCatalog(tester);
      await deliverFirst(tester);
      await tester.tap(find.text('学期'));
      await tester.pumpAndSettle();
      final input = find.descendant(
        of: find.byKey(const Key('course-filter-search')),
        matching: find.byType(TextField),
      );
      await tester.enterText(input, '2025-2026-1');
      await tester.pumpAndSettle();
      expect(find.text('2025-2026 第一学期'), findsOneWidget);
      await tester.tap(find.byType(CheckboxListTile));
      await tester.enterText(input, '第一学期');
      await tester.pumpAndSettle();
      expect(
        tester.widget<CheckboxListTile>(find.byType(CheckboxListTile)).value,
        isTrue,
      );
      await tester.tap(find.text('清空选择'));
      await tester.pump();
      expect(
        tester.widget<CheckboxListTile>(find.byType(CheckboxListTile)).value,
        isFalse,
      );
      await tester.tap(find.text('完成'));
      await tester.pump();
      expect(courses.requests.last.terms, isEmpty);
      courses.requests.last.result.complete(_result('未筛选学期'));
      await tester.pumpAndSettle();
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets(
    'session invalidation hides cached permissions and ignores pending results',
    (tester) async {
      await pumpCatalog(tester);
      container.read(offlineCacheEpochProvider.notifier).invalidate();
      courses.requests.first.result.complete(_result('旧会话课程'));
      await tester.pumpAndSettle();
      expect(find.text('旧会话课程'), findsNothing);
      expect(find.byType(GfSearchField), findsNothing);
      await tester.pumpWidget(const SizedBox.shrink());
    },
  );

  testWidgets('session invalidation cancels a queued keyword request', (
    tester,
  ) async {
    await pumpCatalog(tester);
    await deliverFirst(tester);
    await tester.enterText(find.byType(TextField).first, '旧账号的搜索');
    container.read(offlineCacheEpochProvider.notifier).invalidate();
    await tester.pump(const Duration(milliseconds: 310));
    expect(courses.requests, hasLength(1));
    await tester.pumpWidget(const SizedBox.shrink());
  });

  testWidgets('filter sheet handles narrow width, large text and keyboard', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 700);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.view.resetViewInsets);
    await pumpCatalog(tester, locale: const Locale('de'), textScale: 1.6);
    await deliverFirst(tester);
    await tester.pumpAndSettle();
    final l10n = AppLocalizations.of(
      tester.element(find.byType(CourseCatalogPage)),
    );
    await tester.tap(find.text(l10n.coursesFilterDepartment));
    await tester.pumpAndSettle();
    tester.view.viewInsets = const FakeViewPadding(bottom: 260);
    await tester.pumpAndSettle();
    final input = find.descendant(
      of: find.byKey(const Key('course-filter-search')),
      matching: find.byType(TextField),
    );
    await tester.enterText(input, '不存在');
    await tester.pumpAndSettle();
    expect(find.text(l10n.coursesFilterNoMatches), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox.shrink());
  });
}
