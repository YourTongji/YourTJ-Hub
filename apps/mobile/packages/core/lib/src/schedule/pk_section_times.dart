/// 排课器节次时间表（纯展示用，可被后台设置覆盖）。
///
/// 一比一移植自 web `resource/src/site/utils/sectionTimes.ts`（2026-09-06 dev）。
/// 默认作息以四个锚点构建（45 分钟一节，节间 5 分钟，大课间 25 分钟）：
/// 第 3 节 10:00 / 第 5 节 13:30 / 第 7 节 15:30 / 第 10 节 18:30。
library;

/// 单个节次的起止时间。
class SectionTime {
  const SectionTime({
    required this.section,
    required this.start,
    required this.end,
  });

  /// 节次（1-based）。
  final int section;

  /// 开始时间 "HH:MM"。
  final String start;

  /// 结束时间 "HH:MM"。
  final String end;
}

/// 完整 12 节制默认作息（锚点：3=10:00 / 5=13:30 / 7=15:30 / 10=18:30）。
const List<SectionTime> kDefaultSectionTimes12 = [
  SectionTime(section: 1, start: '08:00', end: '08:45'),
  SectionTime(section: 2, start: '08:50', end: '09:35'),
  SectionTime(section: 3, start: '10:00', end: '10:45'),
  SectionTime(section: 4, start: '10:50', end: '11:35'),
  SectionTime(section: 5, start: '13:30', end: '14:15'),
  SectionTime(section: 6, start: '14:20', end: '15:05'),
  SectionTime(section: 7, start: '15:30', end: '16:15'),
  SectionTime(section: 8, start: '16:20', end: '17:05'),
  SectionTime(section: 9, start: '17:10', end: '17:55'),
  SectionTime(section: 10, start: '18:30', end: '19:15'),
  SectionTime(section: 11, start: '19:20', end: '20:05'),
  SectionTime(section: 12, start: '20:10', end: '20:55'),
];

/// 现行 11 节制默认作息（锚点：3=10:00 / 5=13:30 / 7=15:30 / 9=18:30），
/// 2025-2026 学年（calendarId>=120）起生效；晚间重新编号为 9/10/11 节。
const List<SectionTime> kDefaultSectionTimes11 = [
  SectionTime(section: 1, start: '08:00', end: '08:45'),
  SectionTime(section: 2, start: '08:50', end: '09:35'),
  SectionTime(section: 3, start: '10:00', end: '10:45'),
  SectionTime(section: 4, start: '10:50', end: '11:35'),
  SectionTime(section: 5, start: '13:30', end: '14:15'),
  SectionTime(section: 6, start: '14:20', end: '15:05'),
  SectionTime(section: 7, start: '15:30', end: '16:15'),
  SectionTime(section: 8, start: '16:20', end: '17:05'),
  SectionTime(section: 9, start: '18:30', end: '19:15'),
  SectionTime(section: 10, start: '19:20', end: '20:05'),
  SectionTime(section: 11, start: '20:10', end: '20:55'),
];

/// 按节次制取作息表：11 节制（现行）优先使用后台覆盖（overrides，按 section
/// 对齐补齐缺口，未知 section 忽略），缺失时回退默认表；12 节制为历史学期，
/// 恒返回内置历史表、忽略覆盖（web sectionTimesFor 同语义）。
List<SectionTime> sectionTimesFor(int maxRows, List<SectionTime>? overrides) {
  if (maxRows != 11) return kDefaultSectionTimes12;
  if (overrides == null || overrides.isEmpty) return kDefaultSectionTimes11;
  final Map<int, SectionTime> bySection = {
    for (final SectionTime item in overrides) item.section: item,
  };
  return kDefaultSectionTimes11
      .map((SectionTime item) => bySection[item.section] ?? item)
      .toList();
}

/// 解析 "HH:MM" 为分钟数；非法返回 null（分组推导容错用）。
int? parseHHMM(String? value) {
  final match = RegExp(r'^(\d{1,2}):(\d{2})$').firstMatch((value ?? '').trim());
  if (match == null) return null;
  final hour = int.tryParse(match.group(1)!);
  final minute = int.tryParse(match.group(2)!);
  if (hour == null || minute == null || hour > 23 || minute > 59) return null;
  return hour * 60 + minute;
}

/// 时段分组键（morning/afternoon/evening）。
typedef DayPart = String;

/// 按开始时间推导时段分组：<12:00 上午、<18:00 下午、否则晚上。
DayPart dayPartOfStart(String start) {
  final minutes = parseHHMM(start);
  if (minutes == null) return 'morning';
  if (minutes < 12 * 60) return 'morning';
  if (minutes < 18 * 60) return 'afternoon';
  return 'evening';
}

/// 分组切分点：每个 DayPart 首个节次（1-based）；无时间数据返回空。
Map<DayPart, int> dayPartBoundaries(List<SectionTime> times) {
  final boundaries = <DayPart, int>{};
  for (final time in times) {
    final part = dayPartOfStart(time.start);
    boundaries.putIfAbsent(part, () => time.section);
  }
  return boundaries;
}
