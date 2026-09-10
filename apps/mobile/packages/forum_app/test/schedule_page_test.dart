// 排课器 store + 页面测试（自包含 fakes，不依赖 pages_smoke_test）。
//
// - store 层：SharedPreferences mock 持久化 roundtrip / 消毒（坏安排丢弃、
//   周次夹取、损坏 JSON 回退空方案）/ 同课换班不冲突 / 容忍式加课 +
//   deriveConflicts ⚠ / 学期变更清空方案 / syncLatest 保留班级状态 /
//   自定义占位进 occupied 表。
// - 页面层：SchedulePage 在 fake repository + 预置 store 下渲染课程卡与统计；
//   周次过滤隐藏非匹配课程。
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:core/core.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/pages/schedule/schedule_page.dart';
import 'package:forum_app/src/schedule/schedule_store.dart';
import 'package:forum_app/src/providers.dart';

/// 测试用内存 TokenStorage（与 pages_smoke_test 同构，副本内联）。
class MemoryTokenStorage implements TokenStorage {
  String? _token;

  @override
  Future<String?> read() async => _token;

  @override
  Future<void> write(String token) async => _token = token;

  @override
  Future<void> clear() async => _token = null;
}

/// Fake PK repository：只喂 fixture，不发真实网络。
class FakePkRepository extends PkRepository {
  FakePkRepository(super.client);

  static const List<PkCalendarItem> calendarsFixture = <PkCalendarItem>[
    PkCalendarItem(
      calendarId: 119,
      calendarName: '2025-2026-1',
      startDate: '2025-09-08',
      endDate: '2026-01-16',
    ),
  ];

  @override
  Future<List<PkCalendarItem>> calendars() async => calendarsFixture;

  @override
  Future<SectionTimesPayload?> sectionTimes() async => null;

  @override
  Future<String?> latestUpdate() async => null;

  @override
  Future<PkGradeList> grades({required int calendarId}) async =>
      const PkGradeList(gradeList: <int>[2025, 2024, 2023]);

  @override
  Future<List<PkMajor>> majors({required int grade, int? calendarId}) async =>
      const <PkMajor>[
        PkMajor(code: 'm1', name: '软件工程'),
        PkMajor(code: 'm2', name: '土木工程'),
      ];

  List<PkCourseByMajorItem> coursesByMajorFixture =
      const <PkCourseByMajorItem>[];
  List<PkCourseDetailBrief> Function(String courseCode)? onCourseDetails;

  @override
  Future<List<PkCourseByMajorItem>> coursesByMajor({
    required int grade,
    required String code,
    required int calendarId,
  }) async => coursesByMajorFixture;

  @override
  Future<List<PkOptionalType>> optionalTypes({required int calendarId}) async =>
      const <PkOptionalType>[];

  @override
  Future<List<PkCourseByNatureItem>> coursesByNature({
    required int calendarId,
    required List<int> ids,
  }) async => const <PkCourseByNatureItem>[];

  @override
  Future<List<PkCourseDetailBrief>> courseDetails({
    required int calendarId,
    required String courseCode,
  }) async {
    if (onCourseDetails != null) return onCourseDetails!(courseCode);
    return const <PkCourseDetailBrief>[];
  }

  @override
  Future<Map<String, List<PkCourseDetailBrief>>> courseDetailsBatch({
    required int calendarId,
    required List<String> courseCodes,
  }) async => const <String, List<PkCourseDetailBrief>>{};

  @override
  Future<PkSearchResult> searchCourses({
    required int calendarId,
    String? courseName,
    String? courseCode,
    String? teacherCode,
    String? teacherName,
    String? campus,
    String? faculty,
  }) async =>
      const PkSearchResult(courses: <PkSearchCourseItem>[], sizeLimit: 0);

  @override
  Future<PkCoursesByTimeResult> coursesByTime({
    required int calendarId,
    required int day,
    required int section,
  }) async => PkCoursesByTimeResult(
    auxiliaryReady: true,
    courses: const <PkSearchCourseItem>[],
  );

  @override
  Future<Map<String, List<PkCourseDetailBrief>>> courseInfoSync({
    required int calendarId,
    List<String> majorCourseCodes = const <String>[],
    List<String> otherCourseCodes = const <String>[],
    (int, String)? majorInfo,
  }) async => const <String, List<PkCourseDetailBrief>>{};

  @override
  Future<PkReviewBrief> courseReviewBrief({
    required String courseCode,
    String? teacherName,
    int? calendarId,
    int? teachingClassId,
  }) async => PkReviewBrief(
    courseId: 0,
    courseCode: courseCode,
    courseName: '',
    teacherName: '',
    reviewCount: 0,
    classes: const <PkReviewBriefClass>[],
  );
}

// ---- fixture helpers ----

List<int> weeksAll() => List<int>.generate(16, (int i) => i + 1);

PkCourseDetail detail(
  String code, {
  int day = 1,
  List<int> sections = const <int>[1, 2],
  List<int>? weeks,
  String room = 'A101',
  String teacher = '张老师(T001)',
  int? teachingClassId,
}) {
  return PkCourseDetail(
    arrangementInfo: <PkArrangement>[
      PkArrangement(
        arrangementText: '$day 1-2节 $room',
        occupyDay: day,
        occupyTime: List<int>.from(sections),
        occupyWeek: List<int>.from(weeks ?? weeksAll()),
        occupyRoom: room,
        teacherAndCode: teacher,
      ),
    ],
    campus: '四平',
    code: code,
    teachingClassId: teachingClassId,
    status: 0,
    teachers: <PkTeacher>[PkTeacher(teacherName: '张老师', teacherCode: 'T001')],
    teachingLanguage: '',
  );
}

/// 手工构造持久化课程 JSON（绕过 store API，直接模拟旧/localStorage 数据）。
Map<String, dynamic> stagedCourseJson({
  required String baseCode,
  required String className,
  required String courseName,
  double credit = 4,
  int status = 2,
  List<Map<String, dynamic>>? arrangements,
}) {
  return <String, dynamic>{
    'courseCode': baseCode,
    'courseName': courseName,
    'courseNameReserved': courseName,
    'credit': credit,
    'courseType': '必',
    'teacher': <Map<String, String>>[
      <String, String>{'teacherName': '张老师', 'teacherCode': 'T001'},
    ],
    'status': status,
    'courseDetail': <Map<String, dynamic>>[
      <String, dynamic>{
        'code': className,
        'status': status,
        'campus': '四平',
        'teachers': <Map<String, String>>[
          <String, String>{'teacherName': '张老师', 'teacherCode': 'T001'},
        ],
        'arrangementInfo':
            arrangements ??
            <Map<String, dynamic>>[
              <String, dynamic>{
                'arrangementText': '1-16周 周一 1-2节 A101',
                'occupyDay': 1,
                'occupyTime': <int>[1, 2],
                'occupyWeek': weeksAll(),
                'occupyRoom': 'A101',
                'teacherAndCode': '张老师(T001)',
              },
            ],
      },
    ],
  };
}

Map<String, dynamic> planJson(
  String id,
  String name, {
  List<Map<String, dynamic>>? staged,
  List<String> selected = const <String>[],
}) {
  return <String, dynamic>{
    'id': id,
    'name': name,
    'createdAt': 1725000000000,
    'stagedCourses': staged ?? const <Map<String, dynamic>>[],
    'selectedCourses': selected,
    'customEvents': const <Map<String, dynamic>>[],
  };
}

/// 构建带一门已选课的 store（页面 smoke 用）。
Future<ScheduleStoreNotifier> seededNotifier({
  int weeksSeed = 16,
  String courseName = '高等数学',
  bool withCustomEvent = false,
}) async {
  final ScheduleStoreNotifier notifier = ScheduleStoreNotifier();
  await notifier.ready;
  notifier.selectClass(
    detail('X.01', weeks: weeksSeed >= 16 ? weeksAll() : <int>[1]),
    courseName,
  );
  if (withCustomEvent) {
    notifier.addCustomEvent(
      label: '有事',
      day: 1,
      sections: <int>[3],
      weeks: <int>[1, 2],
    );
  }
  await notifier.flush;
  return notifier;
}

ProviderContainer makeContainer(
  ScheduleStoreNotifier notifier, {
  FakePkRepository? repository,
}) {
  final MemoryTokenStorage storage = MemoryTokenStorage();
  final GfApiClient client = GfApiClient(
    dio: Dio(),
    tokenStorage: storage,
    baseUrl: 'http://fake.local',
  );
  final FakePkRepository repo = repository ?? FakePkRepository(client);
  return ProviderContainer(
    overrides: <Override>[
      tokenStorageProvider.overrideWithValue(storage),
      pkRepositoryProvider.overrideWithValue(repo),
      scheduleStoreProvider.overrideWith((ref) => notifier),
    ],
  );
}

Widget wrapApp(ProviderContainer container) {
  return UncontrolledProviderScope(
    container: container,
    child: MaterialApp(
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      locale: const Locale('zh'),
      home: const SchedulePage(),
    ),
  );
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('ScheduleStore 持久化与消毒', () {
    test('roundtrip：方案/课程持久化后重建相等', () async {
      SharedPreferences.setMockInitialValues(<String, Object>{});
      final ScheduleStoreNotifier first = ScheduleStoreNotifier();
      await first.ready;
      first.selectClass(detail('X.01'), '高等数学');
      first.createPlan();
      await first.flush;

      final SharedPreferences prefs = await SharedPreferences.getInstance();
      expect(prefs.getString(ScheduleStorageKeys.plans), isNotNull);

      final ScheduleStoreNotifier second = ScheduleStoreNotifier();
      await second.ready;
      expect(second.state.plans.length, first.state.plans.length);
      expect(second.state.activePlanId, first.state.activePlanId);
      // 结构相等断言：消毒会把缺失 courseNature 归一为空数组，属预期
      // schema 归一，不逐字节比较 JSON。
      for (int i = 0; i < first.state.plans.length; i++) {
        final PkPlan a = first.state.plans[i];
        final PkPlan b = second.state.plans[i];
        expect(b.id, a.id);
        expect(b.name, a.name);
        expect(b.selectedCourses, a.selectedCourses);
        expect(
          b.stagedCourses.map((PkStagedCourse c) => c.courseCode).toList(),
          a.stagedCourses.map((PkStagedCourse c) => c.courseCode).toList(),
        );
        for (int j = 0; j < a.stagedCourses.length; j++) {
          final PkStagedCourse ca = a.stagedCourses[j];
          final PkStagedCourse cb = b.stagedCourses[j];
          expect(cb.status, ca.status);
          expect(
            cb.courseDetail.map((PkCourseDetail d) => d.code).toList(),
            ca.courseDetail.map((PkCourseDetail d) => d.code).toList(),
          );
          for (int k = 0; k < ca.courseDetail.length; k++) {
            final PkCourseDetail da = ca.courseDetail[k];
            final PkCourseDetail db = cb.courseDetail[k];
            expect(db.status, da.status);
            expect(db.code, da.code);
            expect(db.arrangementInfo.length, da.arrangementInfo.length);
            expect(
              db.arrangementInfo.first.occupyDay,
              da.arrangementInfo.first.occupyDay,
            );
            expect(
              db.arrangementInfo.first.occupyTime,
              da.arrangementInfo.first.occupyTime,
            );
            expect(
              db.arrangementInfo.first.occupyWeek,
              da.arrangementInfo.first.occupyWeek,
            );
          }
        }
      }
      addTearDown(first.dispose);
      addTearDown(second.dispose);
    });

    test('sanitize：丢弃非法安排、夹取周次', () async {
      SharedPreferences.setMockInitialValues(<String, Object>{
        ScheduleStorageKeys.plans: jsonEncode(<dynamic>[
          planJson(
            'plan-a',
            '方案 1',
            staged: <Map<String, dynamic>>[
              stagedCourseJson(
                baseCode: 'X',
                className: 'X.01',
                courseName: '高等数学',
                arrangements: <Map<String, dynamic>>[
                  <String, dynamic>{
                    'occupyDay': 1,
                    'occupyTime': <int>[1, 2],
                    'occupyWeek': weeksAll(),
                    'occupyRoom': 'A101',
                  },
                  // 非法星期 → 整段丢弃。
                  <String, dynamic>{
                    'occupyDay': 9,
                    'occupyTime': <int>[5],
                    'occupyWeek': <int>[1],
                  },
                  // 周次越界 → 夹取 1..16。
                  <String, dynamic>{
                    'occupyDay': 2,
                    'occupyTime': <int>[3, 4],
                    'occupyWeek': <int>[0, 99],
                    'occupyRoom': 'B202',
                  },
                ],
              ),
            ],
          ),
        ]),
      });
      final ScheduleStoreNotifier notifier = ScheduleStoreNotifier();
      await notifier.ready;
      addTearDown(notifier.dispose);

      final PkPlan plan = notifier.activePlan;
      final PkCourseDetail detailX =
          plan.stagedCourses.first.courseDetail.first;
      expect(detailX.arrangementInfo.length, 2, reason: '非法星期安排应被丢弃');
      final PkArrangement clamped = detailX.arrangementInfo[1];
      expect(clamped.occupyWeek, <int>[1, 16]);
    });

    test('corrupted JSON → 空「方案 1」', () async {
      SharedPreferences.setMockInitialValues(<String, Object>{
        ScheduleStorageKeys.plans: 'not-json{{{',
        ScheduleStorageKeys.activePlanId: '"missing"',
      });
      final ScheduleStoreNotifier notifier = ScheduleStoreNotifier();
      await notifier.ready;
      addTearDown(notifier.dispose);
      expect(notifier.state.plans.length, 1);
      expect(notifier.state.plans.first.name, '方案 1');
      expect(notifier.state.activePlanId, notifier.state.plans.first.id);
    });
  });

  group('ScheduleStore 课程语义', () {
    test('同基础课号换班：旧班被替换且不报冲突', () async {
      SharedPreferences.setMockInitialValues(<String, Object>{});
      final ScheduleStoreNotifier notifier = ScheduleStoreNotifier();
      await notifier.ready;
      addTearDown(notifier.dispose);

      notifier.selectClass(detail('X.01'), '高等数学');
      final StageCourseResult result = notifier.selectClass(
        detail('X.02', teacher: '李老师(T002)', room: 'B202'),
        '高等数学',
      );
      expect(result.conflicts, isEmpty, reason: '同课换班不计冲突');
      expect(notifier.activePlan.selectedCourses, <String>['X.02']);
      // occupied[0][0]（周一 1 节）只剩新班。
      final List<PkOccupyCell> cells = notifier.state.occupied[0][0];
      expect(cells.length, 1);
      expect(cells.single.code, 'X.02');
      expect(notifier.state.grid.conflicts, isEmpty);
    });

    test('容忍式加课：冲突课程仍入表，deriveConflicts 标注双方', () async {
      SharedPreferences.setMockInitialValues(<String, Object>{});
      final ScheduleStoreNotifier notifier = ScheduleStoreNotifier();
      await notifier.ready;
      addTearDown(notifier.dispose);

      notifier.selectClass(detail('X.01'), '高等数学');
      final StageCourseResult result = notifier.selectClass(
        detail('Y.01', teacher: '王老师(T003)', room: 'C303'),
        '大学物理',
      );
      expect(result.added, isTrue);
      expect(result.conflicts, isNotEmpty, reason: '容忍式冲突应返回冲突列表');

      final Map<String, List<PkConflictItem>> conflicts =
          notifier.state.grid.conflicts;
      expect(conflicts.containsKey('X'), isTrue);
      expect(conflicts.containsKey('Y'), isTrue);
      // 冲突统计（非 custom）为 2。
      expect(notifier.state.stats.conflictCount, 2);
    });

    test('major 变更清空全部方案的课程与占位', () async {
      SharedPreferences.setMockInitialValues(<String, Object>{});
      final ScheduleStoreNotifier notifier = ScheduleStoreNotifier();
      await notifier.ready;
      addTearDown(notifier.dispose);

      notifier.selectClass(detail('X.01'), '高等数学');
      notifier.createPlan();
      notifier.selectClass(detail('Y.01'), '大学物理');
      expect(notifier.state.plans.length, 2);
      notifier.setMajorSelection(
        PkMajorSelection(
          calendarId: 120,
          grade: 2025,
          major: 'm9',
          majorName: '新专业',
        ),
      );
      for (final PkPlan plan in notifier.state.plans) {
        expect(plan.stagedCourses, isEmpty);
        expect(plan.selectedCourses, isEmpty);
        expect(plan.customEvents, isEmpty);
      }
      expect(notifier.state.majorSelected.calendarId, 120);
    });

    test('syncLatest 保留每班状态并推进 updateTime', () async {
      SharedPreferences.setMockInitialValues(<String, Object>{});
      final ScheduleStoreNotifier notifier = ScheduleStoreNotifier();
      await notifier.ready;
      addTearDown(notifier.dispose);

      notifier.selectClass(detail('X.01'), '高等数学');
      notifier.setLatestUpdateTime('2026-09-02');
      final PkCourseDetail freshOld = detail('X.01', room: '新教室');
      final PkCourseDetail freshNew = detail('X.02', teacher: '李老师(T002)');
      final ScheduleState synced = notifier.syncLatest(
        <String, List<PkCourseDetail>>{
          'X': <PkCourseDetail>[freshOld, freshNew],
        },
        syncDate: '2026-09-02',
      );
      expect(synced.updateTime, '2026-09-02');
      final PkPlan active = notifier.activePlan;
      PkCourseDetail? foundOld;
      PkCourseDetail? foundNew;
      for (final PkCourseDetail d in active.stagedCourses.first.courseDetail) {
        if (d.code == 'X.01') foundOld = d;
        if (d.code == 'X.02') foundNew = d;
      }
      expect(foundOld, isNotNull);
      expect(foundOld!.status, CourseStatus.selected, reason: '旧班已选状态保留');
      expect(foundOld.arrangementInfo.first.occupyRoom, '新教室');
      expect(foundNew, isNotNull);
      expect(foundNew!.status, CourseStatus.unselected, reason: '新班默认未选');
    });

    test('自定义占位进入 occupied 表与网格', () async {
      SharedPreferences.setMockInitialValues(<String, Object>{});
      final ScheduleStoreNotifier notifier = ScheduleStoreNotifier();
      await notifier.ready;
      addTearDown(notifier.dispose);

      final PkCustomEvent? event = notifier.addCustomEvent(
        label: '有事',
        day: 3,
        sections: <int>[3, 4],
        weeks: <int>[1, 2],
      );
      expect(event, isNotNull);
      // occupied[2][2] = 周三 3 节。
      final List<PkOccupyCell> cells = notifier.state.occupied[2][2];
      expect(cells.length, 1);
      expect(cells.single.code.startsWith(kCustomEventCodePrefix), isTrue);
      // 网格：第 3 行（row 2）周三（day 2）出现占位课程。
      final List<PkCourseOnTable> gridCourses =
          notifier.state.grid.cellCourses[2][2];
      expect(gridCourses, isNotEmpty);
      expect(gridCourses.first.courseName, '有事');
    });
  });

  group('SchedulePage', () {
    testWidgets('渲染课程格 + 统计', (tester) async {
      SharedPreferences.setMockInitialValues(<String, Object>{});
      final ScheduleStoreNotifier notifier = await seededNotifier();
      // 容器 dispose 会一并释放被 override 的 notifier，勿重复 dispose。
      final ProviderContainer container = makeContainer(notifier);
      addTearDown(container.dispose);

      await tester.pumpWidget(wrapApp(container));
      await tester.pumpAndSettle();
      expect(find.text('完整版排课器，请到网页端体验'), findsOneWidget);
      await tester.ensureVisible(find.text('方案预览'));
      await tester.tap(find.text('方案预览'));
      await tester.pumpAndSettle();
      await tester.drag(find.byType(ListView).first, const Offset(0, -260));
      await tester.pumpAndSettle();

      // 网格课程卡：高等数学（周一 1-2 节 → 第一行第一列）。
      expect(find.text('高等数学'), findsWidgets);
      await tester.drag(find.byType(ListView).first, const Offset(0, 600));
      await tester.pumpAndSettle();
      // 统计卡：1 门。
      expect(find.text('1 门'), findsOneWidget);
      // 方案条 + 默认方案名。
      expect(find.text('方案 1'), findsOneWidget);
    });

    testWidgets('周次过滤隐藏非匹配课程', (tester) async {
      SharedPreferences.setMockInitialValues(<String, Object>{});
      final ScheduleStoreNotifier notifier = await seededNotifier(weeksSeed: 1);
      // 容器 dispose 会一并释放被 override 的 notifier，勿重复 dispose。
      final ProviderContainer container = makeContainer(notifier);
      addTearDown(container.dispose);

      await tester.pumpWidget(wrapApp(container));
      await tester.pumpAndSettle();
      expect(find.text('完整版排课器，请到网页端体验'), findsOneWidget);
      await tester.ensureVisible(find.text('方案预览'));
      await tester.tap(find.text('方案预览'));
      await tester.pumpAndSettle();
      await tester.drag(find.byType(ListView).first, const Offset(0, -260));
      await tester.pumpAndSettle();
      expect(find.text('高等数学'), findsWidgets);

      // 切到第 2 周（课程只占第 1 周）。
      await tester.tap(find.byType(DropdownButton<int?>));
      await tester.pumpAndSettle();
      await tester.tap(find.text('第 2 周').last);
      await tester.pumpAndSettle();

      expect(find.text('高等数学'), findsNothing);
      await tester.drag(find.byType(ListView).first, const Offset(0, 600));
      await tester.pumpAndSettle();
      // 统计仍按方案计（周次过滤不影响统计）。
      expect(find.text('1 门'), findsOneWidget);
    });

    testWidgets('自定义占位渲染为灰色格', (tester) async {
      SharedPreferences.setMockInitialValues(<String, Object>{});
      final ScheduleStoreNotifier notifier = await seededNotifier(
        withCustomEvent: true,
      );
      // 容器 dispose 会一并释放被 override 的 notifier，勿重复 dispose。
      final ProviderContainer container = makeContainer(notifier);
      addTearDown(container.dispose);

      await tester.pumpWidget(wrapApp(container));
      await tester.pumpAndSettle();
      expect(find.text('完整版排课器，请到网页端体验'), findsOneWidget);
      await tester.ensureVisible(find.text('方案预览'));
      await tester.tap(find.text('方案预览'));
      await tester.pumpAndSettle();
      await tester.drag(find.byType(ListView).first, const Offset(0, -260));
      await tester.pumpAndSettle();

      expect(find.text('有事'), findsWidgets);
    });

    testWidgets('选班列表呈现预选冲突提示且非阻塞加课', (tester) async {
      SharedPreferences.setMockInitialValues(<String, Object>{});
      final ScheduleStoreNotifier notifier = ScheduleStoreNotifier();
      await notifier.ready;

      // 设置专业信息使得选课 tab 处于就绪状态
      notifier.setMajorSelection(
        PkMajorSelection(
          calendarId: 119,
          grade: 2025,
          major: 'm1',
          majorName: '软件工程',
        ),
      );
      // 预先加入一门占用周一 1-2 节的已选课程
      notifier.selectClass(
        detail('110099.01', day: 1, sections: const <int>[1, 2]),
        '大学物理',
      );
      await notifier.flush;

      final MemoryTokenStorage storage = MemoryTokenStorage();
      final GfApiClient client = GfApiClient(
        dio: Dio(),
        tokenStorage: storage,
        baseUrl: 'http://fake.local',
      );
      final FakePkRepository repo = FakePkRepository(client);
      repo.coursesByMajorFixture = <PkCourseByMajorItem>[
        const PkCourseByMajorItem(
          courseCode: '110001',
          courseName: '高等数学',
          faculty: '数学系',
          facultyI18n: '',
          credit: 3.0,
          grade: 2025,
          courseNature: <String>['必修'],
          courses: <PkCourseClassItem>[],
        ),
      ];
      repo.onCourseDetails = (String courseCode) => <PkCourseDetailBrief>[
        const PkCourseDetailBrief(
          code: '110001.01',
          teachers: <PkTeacherRef>[PkTeacherRef(teacherName: '张老师', teacherCode: 'T1')],
          campus: '四平路校区',
          teachingLanguage: '中文',
          arrangementInfo: <PkArrangementInfo>[
            PkArrangementInfo(
              arrangementText: '周一 1-2节',
              occupyDay: 1,
              occupyTime: <int>[1, 2],
              occupyWeek: <int>[1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16],
            ),
          ],
        ),
        const PkCourseDetailBrief(
          code: '110001.02',
          teachers: <PkTeacherRef>[PkTeacherRef(teacherName: '王老师', teacherCode: 'T2')],
          campus: '四平路校区',
          teachingLanguage: '中文',
          arrangementInfo: <PkArrangementInfo>[
            PkArrangementInfo(
              arrangementText: '周二 3-4节',
              occupyDay: 2,
              occupyTime: <int>[3, 4],
              occupyWeek: <int>[1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16],
            ),
          ],
        ),
      ];

      final ProviderContainer container = makeContainer(notifier, repository: repo);
      addTearDown(container.dispose);

      await tester.pumpWidget(wrapApp(container));
      await tester.pumpAndSettle();

      // 点击切换到「选课」tab
      await tester.tap(find.text('选课'));
      await tester.pumpAndSettle();

      // 在必修课列表看见「高等数学」，点击唤起教学班 sheet
      expect(find.text('高等数学'), findsOneWidget);
      await tester.tap(find.text('高等数学'));
      await tester.pumpAndSettle();

      // 验证冲突标记：110001.01 与「大学物理」冲突，显示警示文案与「仍可加入」胶囊
      expect(find.text('110001.01'), findsOneWidget);
      expect(find.text('与「大学物理」时间冲突'), findsOneWidget);
      expect(find.text('仍可加入'), findsOneWidget);

      // 验证 110001.02 属于正常班级，无冲突
      expect(find.text('110001.02'), findsOneWidget);

      // 非阻塞测试：点击冲突班级 110001.01
      await tester.tap(find.text('110001.01'));
      await tester.pumpAndSettle();

      // 验证已被成功加入方案中（非阻塞加入）
      final PkPlan activePlan = notifier.activePlan;
      expect(activePlan.selectedCourses.contains('110001.01'), isTrue);
    });
  });
}
