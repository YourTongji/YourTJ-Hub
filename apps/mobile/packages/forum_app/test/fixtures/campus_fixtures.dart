import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';

class CampusMemoryTokenStorage implements TokenStorage {
  String? value;
  @override
  Future<String?> read() async => value;
  @override
  Future<void> write(String token) async {
    value = token;
  }

  @override
  Future<void> clear() async {
    value = null;
  }
}

const testBinding = CampusBinding(
  maskedId: 'DE••••MO',
  boundAt: '2026-09-01T00:00:00Z',
  revision: 'demo-revision',
  needsAuthorization: false,
);
const testStatus = CampusStatus(
  enabled: true,
  binding: testBinding,
  candidate: null,
);
CampusDataset campusFixture(String key) {
  final metrics = <CampusMetric>[];
  final rows = <List<String>>[];
  final columns = <String>[];
  final series = <CampusPoint>[];
  if (key == 'profile') {
    metrics.add(const CampusMetric(label: '姓名', value: '演示同学', unit: ''));
  }
  if (key == 'calendar') {
    metrics.addAll(const [
      CampusMetric(label: '教学周', value: '1', unit: '周'),
      CampusMetric(label: '学期周数', value: '18', unit: '周'),
      CampusMetric(label: '当前学期', value: '2026 秋季 · 演示数据', unit: ''),
    ]);
  }
  if (key == 'summary') {
    metrics.addAll(const [
      CampusMetric(label: '综合 GPA', value: '4.20', unit: ''),
      CampusMetric(label: '已修学分', value: '72', unit: '学分'),
      CampusMetric(label: '要求学分', value: '160', unit: '学分'),
    ]);
  }
  if (key == 'grades') {
    columns.addAll(['学期', '课程', '成绩', '学分', '绩点', '考试性质']);
    rows.addAll([
      ['2026 春', '高等数学（演示）', '95', '4', '4.5', '考试'],
      ['2026 春', '程序设计（演示）', '—', '3', '', '考试'],
    ]);
    series.addAll(const [
      CampusPoint(label: '2025 秋', value: 4),
      CampusPoint(label: '2026 春', value: 4.2),
    ]);
  }
  if (key == 'cet') {
    columns.addAll(['考试学期', '考试科目', '笔试成绩', '口试成绩']);
    rows.add(['2026 春', 'CET-4（演示）', '580', '']);
    series.add(const CampusPoint(label: 'CET-4', value: 580));
  }
  if (key == 'terms') {
    columns.addAll(['学期', '开始日期', '结束日期', '周数']);
    rows.add(['2026 秋 · 演示数据', '2026-09-14', '2027-01-17', '18']);
  }
  return CampusDataset(
    teachingDay: key == 'today'
        ? CampusTeachingDay(
            date: DateTime.now()
                .toUtc()
                .add(const Duration(hours: 8))
                .toIso8601String()
                .substring(0, 10),
            sourceDate: '2026-10-06',
            kind: 'makeup',
            label: '国庆补课',
            sectionCount: 11,
          )
        : null,
    key: key,
    status: 'ready',
    updatedAt: '',
    metrics: metrics,
    columns: columns,
    rows: rows,
    events: key == 'today'
        ? [
            const CampusEvent(
              name: '第四周周二的数学',
              teacher: '',
              room: 'A101',
              campus: '',
              day: 2,
              start: 1,
              end: 2,
              weeks: [4],
              credits: '',
            ),
          ]
        : key == 'timetable'
        ? [
            for (var day = 1; day <= 7; day++)
              CampusEvent(
                name: '课程 $day（演示）',
                teacher: '示例教师',
                room: '教学楼 A101',
                campus: '四平路',
                day: day,
                start: 1,
                end: 2,
                weeks: [1, 3],
                credits: '2',
              ),
            const CampusEvent(
              name: '程序设计（演示）',
              teacher: '示例教师',
              room: '教学楼 B202',
              campus: '嘉定',
              day: 2,
              start: 5,
              end: 6,
              weeks: [1, 2],
              credits: '3',
            ),
          ]
        : [],
    series: series,
    messages: key == 'messages'
        ? [
            for (var i = 0; i < 7; i++)
              CampusMessageSummary(
                id: '${9000000000000000 + i}',
                title: [
                  '图书馆开放时间调整（演示）',
                  '校园文化节报名开始（演示）',
                  '秋季学期教学安排（演示）',
                ][i % 3],
                publisher: '校园服务 · 演示',
                publishedAt: '2026-09-${19 - i} 10:00:00',
              ),
          ]
        : [],
  );
}

class FakeCampusRepository extends CampusRepository {
  FakeCampusRepository()
    : super(
        GfApiClient(
          dio: Dio(),
          tokenStorage: CampusMemoryTokenStorage(),
          baseUrl: 'https://forum.example',
        ),
      );
  CampusStatus current = testStatus;
  final requested = <String>[];
  final cancellations = <CancelToken>[];
  Completer<CampusDataset>? pendingProfile;
  Completer<CampusCalendarExport>? pendingExport;
  @override
  Future<CampusCalendarSettings> calendarRules({
    CancelToken? cancelToken,
  }) async {
    requested.add('calendar-rules');
    return const CampusCalendarSettings(
      revision: 'demo-rules',
      rules: CampusCalendarRules(
        holidays: [],
        moves: [
          CampusCalendarMove(
            name: '国庆补课',
            fromDate: '2026-10-06',
            toDate: '2026-09-20',
          ),
        ],
      ),
    );
  }

  bool? lastApplyAdjustments;
  CampusDataset? todayOverride;
  Object? todayError;
  Object? exportError;
  Object? messageError;
  Object? confirmError;
  int confirmed = 0;
  String? unbound;
  @override
  Future<CampusCalendarExport> exportCalendar({
    bool applyAdjustments = true,
    CancelToken? cancelToken,
  }) async {
    lastApplyAdjustments = applyAdjustments;
    requested.add('calendar-export');
    if (cancelToken != null) cancellations.add(cancelToken);
    if (exportError != null) throw exportError!;
    if (pendingExport != null) return pendingExport!.future;
    return const CampusCalendarExport(
      filename: 'yourtj-courses-2026-09-14.ics',
      content: 'BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n',
      eventCount: 2,
    );
  }

  @override
  Future<CampusStatus> status({CancelToken? cancelToken}) async {
    if (cancelToken != null) cancellations.add(cancelToken);
    return current;
  }

  @override
  Future<CampusDataset> dataset(String key, {CancelToken? cancelToken}) async {
    requested.add(key);
    if (cancelToken != null) cancellations.add(cancelToken);
    if (key == 'profile' && pendingProfile != null) {
      return pendingProfile!.future;
    }
    if (key == 'today') {
      if (todayError != null) throw todayError!;
      if (todayOverride != null) return todayOverride!;
    }
    return campusFixture(key);
  }

  @override
  Future<CampusMessageDetail> message(
    String id, {
    CancelToken? cancelToken,
  }) async {
    requested.add('detail');
    if (cancelToken != null) cancellations.add(cancelToken);
    if (messageError != null) throw messageError!;
    return CampusMessageDetail(
      id: id,
      title: '通知正文（演示）',
      publisher: '校园服务',
      publishedAt: '2026-09-19',
      content: '这是用于验证原生通知阅读的演示正文。\n\n正文使用可选择的纯文本，不执行 HTML，也不自动加载远程图片。',
      links: [],
    );
  }

  @override
  Future<void> confirm({CancelToken? cancelToken}) async {
    if (confirmError != null) throw confirmError!;
    confirmed++;
    current = testStatus;
    messageError = null;
  }

  @override
  Future<void> unbind(String revision, {CancelToken? cancelToken}) async {
    unbound = revision;
    current = const CampusStatus(enabled: true, binding: null, candidate: null);
  }
}
