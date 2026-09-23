import 'dart:convert';

import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/campus_widget/schedule_widget_bridge.dart';
import 'package:forum_app/src/campus_widget/schedule_widget_projection.dart';
import 'package:forum_app/src/offline/campus_snapshot_store.dart';
import 'package:forum_app/src/offline/drift_cache.dart';
import 'package:forum_app/src/pages/campus/campus_state.dart';
import 'package:forum_app/src/schedule/schedule_grid.dart';

import 'fixtures/campus_fixtures.dart';

void main() {
  late AppDatabase database;
  late CampusSnapshotStore store;
  const scope = CampusCacheScope(site: 'https://forum.example', accountId: 7);

  setUp(() {
    database = AppDatabase(NativeDatabase.memory());
    store = CampusSnapshotStore(database);
  });

  tearDown(() => database.close());

  Map<String, dynamic> snapshotData(DateTime date) => {
    'profile': campusFixture('profile', now: date),
    'calendar': campusFixture('calendar', now: date),
    'timetable': campusFixture('timetable', now: date),
    'today': campusFixture('today', now: date),
  };

  test('Drift snapshot is atomic, scoped and drops corrupt schema', () async {
    final data = snapshotData(DateTime.parse('2026-09-22T00:00:00Z'));
    await store.write(
      scope,
      'binding-a',
      data.cast(),
      committedAt: DateTime.parse('2026-09-22T00:00:00Z'),
    );
    expect(
      (await store.read(scope, 'binding-a'))?.data.keys,
      campusPersistentKeys,
    );
    expect(
      await store.read(
        const CampusCacheScope(site: 'https://forum.example', accountId: 8),
      ),
      isNull,
    );
    expect(await store.read(scope, 'another-binding'), isNull);
    expect(
      () => store.write(scope, 'binding-a', {'today': data['today']!}.cast()),
      throwsArgumentError,
    );
    expect(await store.read(scope), isNotNull);
    await database.customStatement(
      'UPDATE campus_snapshots SET schema_version = 99',
    );
    expect(await store.read(scope), isNull);
    expect(await store.read(scope), isNull);
  });

  test(
    'projection is minimal and advances class, stale and midnight states',
    () async {
      final snapshot = await store.write(
        scope,
        'binding-a',
        snapshotData(DateTime.parse('2026-09-22T00:00:00Z')).cast(),
        committedAt: DateTime.parse('2026-09-22T00:00:00Z'),
      );
      final projection = ScheduleWidgetProjection.fromSnapshot(snapshot, scope);
      final encoded = projection.encode();
      final json = jsonDecode(encoded) as Map<String, dynamic>;
      expect(json['schemaVersion'], 2);
      expect(json['days'], isA<List<dynamic>>());
      for (final forbidden in [
        '演示同学',
        'token',
        'cookie',
        'studentId',
        'email',
        'grades',
        'messages',
      ]) {
        expect(encoded.toLowerCase(), isNot(contains(forbidden.toLowerCase())));
      }
      expect(
        projection.statusAt(DateTime.parse('2026-09-21T23:30:00Z')).kind,
        'upcoming',
      );
      expect(
        projection.statusAt(DateTime.parse('2026-09-22T00:10:00Z')).kind,
        'inClass',
      );
      expect(
        projection.statusAt(DateTime.parse('2026-09-22T16:00:00Z')).kind,
        'needsRefresh',
      );
      expect(
        projection.statusAt(DateTime.parse('2026-09-30T00:00:00Z')).kind,
        'stale',
      );
      expect(
        projection.updateTimesAfter(DateTime.parse('2026-09-21T23:00:00Z')),
        isNotEmpty,
      );
      expect(
        ScheduleWidgetProjection.fromJson(json).courses.single.stableId,
        projection.courses.single.stableId,
      );
      expect(
        projection.courses.single.colorSlot,
        courseColorSlotFor('第四周周二的数学'),
      );
      expect(projection.courses.single.room, 'A101');
    },
  );

  test(
    'v2 keeps server today authoritative and resolves an eight-day window',
    () async {
      final data = snapshotData(DateTime.parse('2026-09-22T00:00:00Z'));
      data['calendar'] = _withCalendarBounds(data['calendar']!);
      final timetable = data['timetable']!;
      data['timetable'] = _withEvents(timetable, [
        ...timetable.events,
        const CampusEvent(
          name: '不可覆盖服务器今日',
          teacher: '教师',
          room: 'A102',
          campus: '四平路',
          day: 2,
          start: 3,
          end: 4,
          weeks: [4],
          credits: '2',
        ),
        const CampusEvent(
          name: '明日物理',
          teacher: '李老师',
          room: 'B201',
          campus: '嘉定',
          day: 3,
          start: 1,
          end: 2,
          weeks: [4],
          credits: '2',
        ),
        const CampusEvent(
          name: '错误周次课程',
          teacher: '',
          room: '',
          campus: '',
          day: 3,
          start: 5,
          end: 6,
          weeks: [5],
          credits: '',
        ),
      ]);
      final snapshot = await store.write(
        scope,
        'binding-a',
        data.cast(),
        committedAt: DateTime.parse('2026-09-22T00:00:00Z'),
      );
      final projection = _projectionWithRules(
        snapshot,
        scope,
        const CampusCalendarRules(holidays: [], moves: []),
      );
      final json = projection.toJson();
      final days = (json['days']! as List).cast<Map<String, dynamic>>();

      expect(json['schemaVersion'], 2);
      expect(days, hasLength(8));
      expect(days.first['source'], 'server-adjusted');
      expect(days.first['date'], '2026-09-22');
      final todayCourses = (days.first['courses']! as List)
          .cast<Map<String, dynamic>>();
      expect(
        todayCourses.map((course) => course['name']),
        contains('第四周周二的数学'),
      );
      expect(
        todayCourses.map((course) => course['name']),
        isNot(contains('不可覆盖服务器今日')),
      );
      final tomorrow = days[1];
      expect(tomorrow['date'], '2026-09-23');
      expect(tomorrow['source'], 'local-resolved');
      expect(
        (tomorrow['courses']! as List).cast<Map<String, dynamic>>().map(
          (course) => course['name'],
        ),
        containsAll(<String>['明日物理']),
      );
      expect(
        (tomorrow['courses']! as List).cast<Map<String, dynamic>>().map(
          (course) => course['name'],
        ),
        isNot(contains('错误周次课程')),
      );
      final afterMidnight = projection.statusAt(
        DateTime.parse('2026-09-22T16:10:00Z'),
      );
      expect(afterMidnight.kind, 'upcoming');
      expect(afterMidnight.course?.name, '明日物理');

      final afterToday = projection.nextClassAt(
        DateTime.parse('2026-09-22T10:00:00Z'),
      );
      expect(afterToday.kind, 'upcoming');
      expect(afterToday.course?.name, '明日物理');
      expect(afterToday.day?.date, '2026-09-23');
    },
  );

  test('v2 applies holiday and makeup rules without guessing', () async {
    final data = snapshotData(DateTime.parse('2026-09-22T00:00:00Z'));
    data['calendar'] = _withCalendarBounds(data['calendar']!);
    final timetable = data['timetable']!;
    data['timetable'] = _withEvents(timetable, [
      ...timetable.events,
      const CampusEvent(
        name: '周五课程',
        teacher: '教师',
        room: 'C301',
        campus: '四平路',
        day: 5,
        start: 1,
        end: 2,
        weeks: [4],
        credits: '2',
      ),
    ]);
    final snapshot = await store.write(scope, 'binding-a', data.cast());
    final projection = _projectionWithRules(
      snapshot,
      scope,
      const CampusCalendarRules(
        holidays: [
          CampusHoliday(
            name: '校庆日',
            startDate: '2026-09-24',
            endDate: '2026-09-24',
          ),
        ],
        moves: [
          CampusCalendarMove(
            name: '教学调整',
            fromDate: '2026-09-25',
            toDate: '2026-09-26',
          ),
        ],
      ),
    );
    final days = (projection.toJson()['days']! as List)
        .cast<Map<String, dynamic>>();
    final byDate = {for (final day in days) day['date']: day};

    expect(byDate['2026-09-24']?['kind'], 'holiday');
    expect(byDate['2026-09-24']?['adjustmentLabel'], '校庆日');
    expect(byDate['2026-09-25']?['kind'], 'moved');
    expect(byDate['2026-09-26']?['kind'], 'makeup');
    expect(
      (byDate['2026-09-26']?['courses'] as List)
          .cast<Map<String, dynamic>>()
          .map((course) => course['name']),
      contains('周五课程'),
    );
  });

  test('v1 fallback and every optional text form sanitize safely', () async {
    final snapshot = await store.write(
      scope,
      'binding-a',
      snapshotData(DateTime.parse('2026-09-22T00:00:00Z')).cast(),
      committedAt: DateTime.parse('2026-09-22T00:00:00Z'),
    );
    final current = ScheduleWidgetProjection.fromSnapshot(snapshot, scope);
    final course = current.courses.single;
    final v1 = <String, dynamic>{
      'schemaVersion': 1,
      'identity': current.identity.toJson(),
      'generatedAt': '2026-09-22T08:00:00+08:00',
      'schoolDate': '2026-09-22',
      'timezone': 'Asia/Shanghai',
      'semester': {'id': ' undefined ', 'week': 4},
      'sectionTimes': [
        {'section': 1, 'start': '08:00', 'end': '08:45'},
      ],
      'today': {
        'kind': 'none',
        'adjustmentLabel': ' null ',
        'courses': [
          {
            ...course.toJson(),
            'teacher': null,
            'room': 'undefined',
            'campus': '   ',
          },
          {...course.toJson(), 'stableId': 'bad-name', 'name': 'null'},
        ],
      },
    };

    final decoded = ScheduleWidgetProjection.fromJson(v1);
    final normalized = decoded.toJson();
    final days = (normalized['days']! as List).cast<Map<String, dynamic>>();
    final courses = (days.single['courses']! as List)
        .cast<Map<String, dynamic>>();
    expect(normalized['schemaVersion'], 2);
    expect(days, hasLength(1));
    expect(decoded.semesterId, isEmpty);
    expect(decoded.adjustmentLabel, isNull);
    expect(courses, hasLength(1));
    expect(decoded.courses.single.teacher, isEmpty);
    expect(decoded.courses.single.room, isEmpty);
    expect(decoded.courses.single.campus, isEmpty);
    expect(
      decoded.statusAt(DateTime.parse('2026-09-22T16:00:00Z')).kind,
      'needsRefresh',
    );
  });

  test(
    'projection uses 11/12-section fallbacks and all local states',
    () async {
      final data = snapshotData(DateTime.parse('2026-09-22T00:00:00Z'));
      final today = data['today']!;
      data['today'] = CampusDataset(
        key: today.key,
        status: today.status,
        updatedAt: today.updatedAt,
        metrics: today.metrics,
        columns: today.columns,
        rows: today.rows,
        events: today.events,
        series: today.series,
        messages: today.messages,
        teachingDay: CampusTeachingDay(
          date: today.teachingDay!.date,
          sourceDate: today.teachingDay!.sourceDate,
          kind: today.teachingDay!.kind,
          label: today.teachingDay!.label,
          sectionCount: 12,
        ),
      );
      final snapshot = await store.write(scope, 'binding-a', data.cast());
      expect(
        ScheduleWidgetProjection.fromSnapshot(
          snapshot,
          scope,
        ).sectionTimes.length,
        12,
      );

      final projection = ScheduleWidgetProjection(
        identity: const ScheduleWidgetIdentity(
          siteKey: 'site',
          accountScope: 'account',
          bindingRevision: 'binding',
        ),
        generatedAt: DateTime.parse('2026-09-22T00:00:00Z'),
        schoolDate: '2026-09-22',
        semesterId: '2026-fall',
        week: 2,
        sectionTimes: const [],
        dayKind: 'normal',
        adjustmentLabel: null,
        courses: [
          ScheduleWidgetCourse(
            stableId: 'one',
            name: 'One',
            teacher: '',
            room: '',
            campus: '',
            startSection: 1,
            endSection: 1,
            startAt: DateTime.parse('2026-09-22T01:00:00Z'),
            endAt: DateTime.parse('2026-09-22T02:00:00Z'),
            colorSlot: 1,
          ),
          ScheduleWidgetCourse(
            stableId: 'two',
            name: 'Two',
            teacher: '',
            room: '',
            campus: '',
            startSection: 2,
            endSection: 2,
            startAt: DateTime.parse('2026-09-22T03:00:00Z'),
            endAt: DateTime.parse('2026-09-22T04:00:00Z'),
            colorSlot: 2,
          ),
        ],
      );
      expect(
        projection.statusAt(DateTime.parse('2026-09-22T00:30:00Z')).kind,
        'upcoming',
      );
      expect(
        projection.statusAt(DateTime.parse('2026-09-22T01:30:00Z')).kind,
        'inClass',
      );
      expect(
        projection.statusAt(DateTime.parse('2026-09-22T02:30:00Z')).kind,
        'break',
      );
      expect(
        projection.statusAt(DateTime.parse('2026-09-22T04:30:00Z')).kind,
        'finished',
      );
      expect(
        projection.statusAt(DateTime.parse('2026-09-22T16:00:00Z')).kind,
        'needsRefresh',
      );
      expect(
        projection.statusAt(DateTime.parse('2026-09-30T00:00:00Z')).kind,
        'stale',
      );
      expect(
        ScheduleWidgetProjection(
          identity: projection.identity,
          generatedAt: projection.generatedAt,
          schoolDate: projection.schoolDate,
          semesterId: projection.semesterId,
          week: projection.week,
          sectionTimes: projection.sectionTimes,
          dayKind: 'normal',
          adjustmentLabel: null,
          courses: const [],
        ).statusAt(DateTime.parse('2026-09-22T01:00:00Z')).kind,
        'noClasses',
      );
      expect(
        ScheduleWidgetProjection(
          identity: projection.identity,
          generatedAt: projection.generatedAt,
          schoolDate: projection.schoolDate,
          semesterId: projection.semesterId,
          week: projection.week,
          sectionTimes: projection.sectionTimes,
          dayKind: 'holiday',
          adjustmentLabel: '假期',
          courses: const [],
        ).statusAt(DateTime.parse('2026-09-22T01:00:00Z')).kind,
        'holiday',
      );
    },
  );

  test(
    'network failure restores and retains the last successful snapshot',
    () async {
      final bridge = RecordingWidgetBridge();
      final first = CampusController(
        FakeCampusRepository(),
        persistentStore: store,
        widgetBridge: bridge,
        scope: scope,
      );
      await first.refresh();
      expect(await store.read(scope), isNotNull);
      expect(bridge.writes, 1);
      first.dispose();

      final offline = CampusController(
        UnavailableCampusRepository(),
        persistentStore: store,
        widgetBridge: bridge,
        scope: scope,
      );
      await offline.refresh(reuseCache: true);
      expect(offline.state.data.keys, containsAll(campusPersistentKeys));
      expect(offline.state.error, isNotNull);
      expect(await store.read(scope), isNotNull);
      expect(bridge.writes, 1);
      expect(bridge.clears, 0);
      offline.dispose();
    },
  );

  test('binding revision change clears the old snapshot and widget', () async {
    await store.write(
      scope,
      'old-binding',
      snapshotData(DateTime.parse('2026-09-22T00:00:00Z')).cast(),
    );
    final bridge = RecordingWidgetBridge();
    final controller = CampusController(
      FakeCampusRepository(),
      persistentStore: store,
      widgetBridge: bridge,
      scope: scope,
    );
    await controller.refresh(reuseCache: true);
    expect((await store.read(scope))?.bindingRevision, testBinding.revision);
    expect(bridge.clears, 1);
    controller.dispose();
  });

  test(
    'calendar-rules failure keeps snapshot and writes unknown future days',
    () async {
      final repository = CalendarRulesUnavailableRepository();
      final bridge = RecordingWidgetBridge();
      final controller = CampusController(
        repository,
        persistentStore: store,
        widgetBridge: bridge,
        scope: scope,
      );

      await controller.refresh();

      expect(repository.calendarRulesCalls, 1);
      expect(await store.read(scope), isNotNull);
      expect(bridge.writes, 1);
      final days = (bridge.lastProjection!.toJson()['days']! as List)
          .cast<Map<String, dynamic>>();
      expect(days.first['source'], 'server-adjusted');
      expect(days.skip(1).map((day) => day['kind']), everyElement('unknown'));
      controller.dispose();
    },
  );
}

ScheduleWidgetProjection _projectionWithRules(
  CampusSnapshot snapshot,
  CampusCacheScope scope,
  CampusCalendarRules rules,
) =>
    Function.apply(
          ScheduleWidgetProjection.fromSnapshot,
          [snapshot, scope],
          {#calendarRules: rules},
        )
        as ScheduleWidgetProjection;

CampusDataset _withEvents(CampusDataset source, List<CampusEvent> events) =>
    CampusDataset(
      key: source.key,
      status: source.status,
      updatedAt: source.updatedAt,
      metrics: source.metrics,
      columns: source.columns,
      rows: source.rows,
      events: events,
      series: source.series,
      messages: source.messages,
      teachingDay: source.teachingDay,
    );

CampusDataset _withCalendarBounds(CampusDataset source) => CampusDataset(
  key: source.key,
  status: source.status,
  updatedAt: source.updatedAt,
  metrics: [
    ...source.metrics,
    const CampusMetric(label: '学期开始', value: '2026-08-31', unit: ''),
    const CampusMetric(label: '学期结束', value: '2027-01-03', unit: ''),
  ],
  columns: source.columns,
  rows: source.rows,
  events: source.events,
  series: source.series,
  messages: source.messages,
  teachingDay: source.teachingDay,
);

class UnavailableCampusRepository extends FakeCampusRepository {
  @override
  Future<CampusStatus> status({CancelToken? cancelToken}) =>
      throw StateError('offline');
}

class CalendarRulesUnavailableRepository extends FakeCampusRepository {
  int calendarRulesCalls = 0;

  @override
  Future<CampusCalendarSettings> calendarRules({CancelToken? cancelToken}) {
    calendarRulesCalls++;
    throw StateError('calendar rules unavailable');
  }
}

class RecordingWidgetBridge extends ScheduleWidgetBridge {
  int clears = 0;
  int writes = 0;
  String? state;
  ScheduleWidgetProjection? lastProjection;

  @override
  Future<void> write(ScheduleWidgetProjection projection) async {
    writes++;
    lastProjection = projection;
  }

  @override
  Future<void> clear({String state = 'needsData'}) async {
    clears++;
    this.state = state;
  }
}
