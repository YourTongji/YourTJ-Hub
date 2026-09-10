/// 排课器时间工具：周次掩码、交集判断、arrangeInfoText 文本解析。
///
/// 一比一移植自 web `resource/src/site/utils/pkArrange.ts`（2026-09-06 dev）。
/// 冲突检测以「同一天 + 同一节次 + 周次交集非空」为判据；文本解析器作为
/// 后端只返回 arrangeInfoText 时的降级手段。
library;

/// 学年周数上限（一系统 16 周/学期）。
const int kMaxWeek = 16;

/// 新 11 节课制的学期下界（calendarId >= 120 为 2025-2026 学年第 1 学期及以后）。
/// web 端 pkArrange.ts / timetable.ts 硬编码同值；三处共用此常量。
const int kNewSectionSystemMinCalendarId = 120;

/// 周次位掩码：bit i 置位 = 第 i+1 周上课。周次范围 [1,16]。
typedef WeekMask = int;

/// 从展开的周次数组构建位掩码。非法周次（非整数 / 越界）忽略。
WeekMask buildWeekMask(List<int> weeks) {
  var mask = 0;
  for (final week in weeks) {
    if (week >= 1 && week <= kMaxWeek) {
      mask |= 1 << (week - 1);
    }
  }
  return mask;
}

/// 两个周次数组是否有交集。
bool weeksOverlap(List<int>? a, List<int>? b) {
  if (a == null || b == null) return false;
  for (final week in a) {
    if (b.contains(week)) return true;
  }
  return false;
}

/// 两个周次掩码是否有交集。
bool weekMasksOverlap(WeekMask a, WeekMask b) => (a & b) != 0;

/// 周次奇偶性。
enum PkWeekParity { odd, even }

/// 排课文本解析结果（一段上课安排）。
class PkArrangementParse {
  PkArrangementParse({
    required this.weekStart,
    required this.weekEnd,
    this.weekParity,
    required this.specificWeeks,
    required this.day,
    required this.sectionStart,
    required this.sectionEnd,
    required this.location,
  });

  final int weekStart;
  final int weekEnd;
  final PkWeekParity? weekParity; // 单周 / 双周 / 全周
  final List<int> specificWeeks; // 非连续周显式列出；区间型为空数组
  final int day; // 星期 1-7
  final int sectionStart; // 1-12
  final int sectionEnd;
  final String location; // 地点原文（无则空串）
}

const Map<String, int> _dayMap = {
  '一': 1,
  '二': 2,
  '三': 3,
  '四': 4,
  '五': 5,
  '六': 6,
  '日': 7,
  '天': 7,
};

int? _parseDay(String text) {
  // 兼容「周一」与「星期一」
  final match = RegExp(r'(?:星期|周)([一二三四五六日天])').firstMatch(text);
  if (match == null) return null;
  return _dayMap[match.group(1)];
}

class _WeeksParse {
  _WeeksParse(
    this.weekStart,
    this.weekEnd,
    this.weekParity,
    this.specificWeeks,
  );
  final int weekStart;
  final int weekEnd;
  final PkWeekParity? weekParity;
  final List<int> specificWeeks;
}

final RegExp _parityOddRe = RegExp(r'(单周|\(单\)|（单）)');
final RegExp _parityEvenRe = RegExp(r'(双周|\(双\)|（双）)');
final RegExp _weekTokenRe = RegExp(
  r'\[?((?:\d{1,2}\s*[-~－—]\s*\d{1,2}|\d{1,2})(?:[、,，]\s*\d{1,2})*)\s*周\]?',
);
final RegExp _rangeRe = RegExp(r'(\d{1,2})\s*[-~－—]\s*(\d{1,2})');
final RegExp _numRe = RegExp(r'\d{1,2}');

int _clampWeek(num value) {
  if (!value.isFinite) return 1;
  return value.toInt().clamp(1, kMaxWeek);
}

int _clampSection(num value) {
  if (!value.isFinite) return 1;
  return value.toInt().clamp(1, 12);
}

_WeeksParse? _parseWeeks(String text) {
  // 奇偶后缀：1-8周(单周) / 1-8周(单) / 单周 / 双周
  PkWeekParity? parity;
  if (_parityOddRe.hasMatch(text)) {
    parity = PkWeekParity.odd;
  } else if (_parityEvenRe.hasMatch(text)) {
    parity = PkWeekParity.even;
  }

  // 周次片段：兼容 [1-8周]、1-8周、1、3、5周、1-3,5周（「周」必需，避免误吞节次区间）
  final weekToken = _weekTokenRe.firstMatch(text);
  if (weekToken == null) return null;

  final content = weekToken.group(1)!;
  final weeks = <int>{};
  // 区间型（1-8）展开
  final rangeMatch = _rangeRe.firstMatch(content);
  if (rangeMatch != null) {
    final start = _clampWeek(num.tryParse(rangeMatch.group(1)!) ?? 1);
    final end = _clampWeek(num.tryParse(rangeMatch.group(2)!) ?? 1);
    for (var w = start; w <= end; w++) {
      weeks.add(w);
    }
  }
  // 枚举数字（含区间端点与 1-3,5 的尾部枚举）
  for (final m in _numRe.allMatches(content)) {
    weeks.add(_clampWeek(num.tryParse(m.group(0)!) ?? 1));
  }
  if (weeks.isEmpty) return null;

  var weekList = weeks.toList()..sort();
  if (parity == PkWeekParity.odd) {
    weekList = weekList.where((w) => w.isOdd).toList();
  }
  if (parity == PkWeekParity.even) {
    weekList = weekList.where((w) => w.isEven).toList();
  }
  if (weekList.isEmpty) return null;

  final isContiguous = weekList.last - weekList.first + 1 == weekList.length;
  return _WeeksParse(
    weekList.first,
    weekList.last,
    parity,
    isContiguous ? <int>[] : weekList,
  );
}

class _SectionsParse {
  _SectionsParse(this.sectionStart, this.sectionEnd);
  final int sectionStart;
  final int sectionEnd;
}

final RegExp _sectionRangeRe = RegExp(
  r'(?:第)?(\d{1,2})\s*[-~－—]\s*(\d{1,2})\s*节',
);
final RegExp _sectionSingleRe = RegExp(r'(?:第)?(\d{1,2})\s*节');

_SectionsParse? _parseSections(String text) {
  // 区间：3-4节 / 第3-4节 / 3～4节（「节」必需，避免误吞周次区间）
  final range = _sectionRangeRe.firstMatch(text);
  if (range != null) {
    final start = _clampSection(num.tryParse(range.group(1)!) ?? 1);
    final end = _clampSection(num.tryParse(range.group(2)!) ?? 1);
    return _SectionsParse(start < end ? start : end, start < end ? end : start);
  }
  // 单节：第3节 / 3节
  final single = _sectionSingleRe.firstMatch(text);
  if (single != null) {
    final section = _clampSection(num.tryParse(single.group(1)!) ?? 1);
    return _SectionsParse(section, section);
  }
  return null;
}

/// 解析一段（或多段）arrangeInfoText 为结构化安排。
///
/// 典型输入：`"1-8周 周一 3-4节 同济楼A201；9-16周 周三 5-6节 线上"`
/// 分段符：`；` `;` `|` 换行。解析失败的段被忽略（不抛错）。
List<PkArrangementParse> parseArrangeInfoText(String? text) {
  final result = <PkArrangementParse>[];
  final raw = (text ?? '').trim();
  if (raw.isEmpty) return result;

  final segments = raw
      .split(RegExp(r'[；;|\n]'))
      .map((s) => s.trim())
      .where((s) => s.isNotEmpty);
  for (final segment in segments) {
    final day = _parseDay(segment);
    if (day == null) continue; // 无星期信息无法上表

    final sections = _parseSections(segment);
    if (sections == null) continue;

    final weeks = _parseWeeks(segment);
    if (weeks == null) continue;

    var location = segment
        .replaceAll(RegExp(r'(?:星期|周)([一二三四五六日天])'), '')
        .replaceAll(RegExp(r'[（(](单|双)周?[)）]'), '')
        .replaceAll(RegExp(r'(?:第)?\d{1,2}\s*[-~－—]\s*\d{1,2}\s*周'), '')
        .replaceAll(RegExp(r'(?:第)?\d{1,2}(?:[、,，]\d{1,2})+\s*周'), '')
        .replaceAll(RegExp(r'(?:第)?\d{1,2}\s*周'), '')
        .replaceAll(RegExp(r'(?:第)?\d{1,2}\s*[-~－—]\s*\d{1,2}\s*节?'), '')
        .replaceAll(RegExp(r'(?:第)?\d{1,2}\s*节'), '')
        .replaceAll(RegExp(r'[\[\]]'), '')
        .trim();

    result.add(
      PkArrangementParse(
        weekStart: weeks.weekStart,
        weekEnd: weeks.weekEnd,
        weekParity: weeks.weekParity,
        specificWeeks: weeks.specificWeeks,
        day: day,
        sectionStart: sections.sectionStart,
        sectionEnd: sections.sectionEnd,
        location: location,
      ),
    );
  }
  return result;
}

/// 课表行数：新 11 节制（calendarId >= 120）vs 旧 12 节制。
/// 数据不足（null）时按 12 节处理。
int maxRowsForCalendar(int? calendarId) {
  if (calendarId != null && calendarId >= kNewSectionSystemMinCalendarId) {
    return 11;
  }
  return 12;
}

/// 同列课程的节次区间簇（相交/包含的课程归入同一格渲染）。
class PkDayCluster<T> {
  PkDayCluster({required this.start, required this.end, required this.items});

  /// 簇内最早节次（1-based，格子锚定行）。
  final int start;

  /// 簇内最晚节次（rowspan 覆盖到该行）。
  final int end;
  final List<T> items;
}

/// 把同一天的课程按节次区间聚类：区间相交（含包含、部分重叠）的课归为同格。
/// 容忍式冲突下部分重叠的课（如 1-2 节与 2-3 节）必须同格可见。
/// 输入按 occupyTime 首节次稳定排序后分簇。
List<PkDayCluster<T>> clusterBySections<T>(
  List<T> courses,
  List<int> Function(T) occupyTimeOf,
) {
  final sorted = [...courses]
    ..sort((a, b) {
      final at = occupyTimeOf(a);
      final bt = occupyTimeOf(b);
      final af = at.isEmpty ? 0 : at.first;
      final bf = bt.isEmpty ? 0 : bt.first;
      return af.compareTo(bf);
    });
  final clusters = <PkDayCluster<T>>[];
  for (final course in sorted) {
    final time = occupyTimeOf(course);
    if (time.isEmpty) continue;
    final start = time.first;
    final end = time.last;
    final last = clusters.isEmpty ? null : clusters.last;
    if (last != null && start <= last.end) {
      clusters[clusters.length - 1] = PkDayCluster(
        start: last.start,
        end: last.end > end ? last.end : end,
        items: [...last.items, course],
      );
    } else {
      clusters.add(PkDayCluster(start: start, end: end, items: [course]));
    }
  }
  return clusters;
}

/// 由学期起始日期计算「今天」是第几周（1-based）；学期外/日期非法返回 null。
/// 周一为一周之始；endDate（可选）当天仍属学期内，之后返回 null；
/// 超出 [kMaxWeek] 同样返回 null（不夹取）。
int? currentWeekForDate(String startDate, DateTime today, [String? endDate]) {
  final start = DateTime.tryParse('${startDate}T00:00:00');
  if (start == null) return null;
  final todayUtc = DateTime.utc(today.year, today.month, today.day);
  final startUtc = DateTime.utc(start.year, start.month, start.day);
  final diffDays = todayUtc.difference(startUtc).inDays;
  if (diffDays < 0) return null;
  final week = diffDays ~/ 7 + 1;
  if (week > kMaxWeek) return null;
  if (endDate != null) {
    final end = DateTime.tryParse('${endDate}T00:00:00');
    if (end != null) {
      final endUtc = DateTime.utc(end.year, end.month, end.day);
      if (todayUtc.isAfter(endUtc)) return null;
    }
  }
  return week;
}

/// 周次数组 → 紧凑显示文本（如 [1,2,3]→"1-3"、[1,3,5]→"1,3,5"、[2]→"2"）。
String formatWeeksText(List<int>? weeks) {
  if (weeks == null || weeks.isEmpty) return '';
  final sorted = weeks.toSet().toList()..sort();
  final parts = <String>[];
  var runStart = sorted[0];
  var prev = sorted[0];
  for (var i = 1; i <= sorted.length; i++) {
    final current = i < sorted.length ? sorted[i] : null;
    if (current != null && current == prev + 1) {
      prev = current;
      continue;
    }
    parts.add(runStart == prev ? '$runStart' : '$runStart-$prev');
    if (current != null) {
      runStart = current;
      prev = current;
    }
  }
  return parts.join(',');
}

/// 判定周次集合的奇偶性：纯奇 → odd；纯偶 → even；混合或空 → null。
PkWeekParity? detectWeekParity(List<int>? weeks) {
  if (weeks == null || weeks.isEmpty) return null;
  final unique = weeks.toSet();
  if (unique.every((w) => w.isOdd)) return PkWeekParity.odd;
  if (unique.every((w) => w.isEven)) return PkWeekParity.even;
  return null;
}

/// 合并前的源条目形态（consolidateSameClassArrangements 的输入）。
/// 对齐 web 版泛型字段：code/courseName/occupyDay/occupyTime/occupyWeek/
/// teacherAndCode/arrangementText/occupyRoom/showText。
class PkConsolidatableCourse {
  PkConsolidatableCourse({
    required this.code,
    required this.courseName,
    required this.occupyDay,
    required this.occupyTime,
    this.occupyWeek,
    this.teacherAndCode,
    this.arrangementText,
    this.occupyRoom,
    this.showText,
  });

  String code;
  String courseName;
  int occupyDay;
  List<int> occupyTime;
  List<int>? occupyWeek;
  String? teacherAndCode;
  String? arrangementText;
  String? occupyRoom;
  String? showText;
}

final RegExp _teacherCodeSuffixRe = RegExp(r'\([^)]*\)$');
final RegExp _teacherSplitRe = RegExp(r'[,，、]');

/// 合并同一节次簇内属于同一教学班的多段安排（web
/// consolidateSameClassArrangements 一比一移植）。
///
/// 合并规则：同一天、同教学班课号（无 code 用 courseName）；occupyWeek/occupyTime
/// 取并集排序去重；occupyRoom 非重复地点以 ' / ' 拼接；teacherAndCode 提取
/// 教师姓名去重拼接；arrangementText/showText 以合并后字段重新生成。
/// 输出保持组间首次出现顺序。
List<PkConsolidatableCourse> consolidateSameClassArrangements(
  List<PkConsolidatableCourse> courses,
) {
  if (courses.length <= 1) return courses;

  final groups = <String, List<PkConsolidatableCourse>>{};
  final keyOrder = <String>[];
  for (final item in courses) {
    final key = item.code.isNotEmpty
        ? item.code
        : item.courseName.isNotEmpty
        ? item.courseName
        : 'unknown';
    final existing = groups[key];
    if (existing != null) {
      existing.add(item);
    } else {
      groups[key] = [item];
      keyOrder.add(key);
    }
  }

  final result = <PkConsolidatableCourse>[];
  for (final key in keyOrder) {
    final group = groups[key]!;
    if (group.length == 1) {
      result.add(group[0]);
      continue;
    }

    final first = group[0];
    final allWeeks =
        group.expand((g) => g.occupyWeek ?? <int>[]).toSet().toList()..sort();

    final allTimes = group.expand((g) => g.occupyTime).toSet().toList()..sort();

    final rooms = group
        .map((g) => g.occupyRoom?.trim())
        .whereType<String>()
        .where((s) => s.isNotEmpty)
        .toSet()
        .toList();
    final consolidatedRoom = rooms.isNotEmpty
        ? rooms.join(' / ')
        : (first.occupyRoom ?? '');

    final teacherNames = <String>{};
    for (final t in group) {
      final raw = t.teacherAndCode?.trim();
      if (raw == null || raw.isEmpty) continue;
      for (final part in raw.split(_teacherSplitRe)) {
        final name = part.replaceAll(_teacherCodeSuffixRe, '').trim();
        if (name.isNotEmpty) teacherNames.add(name);
      }
    }
    final consolidatedTeacher = teacherNames.isNotEmpty
        ? teacherNames.join('、')
        : (first.teacherAndCode ?? '');

    final weeksText = formatWeeksText(allWeeks);
    String timeSpan;
    if (allTimes.isEmpty) {
      timeSpan = '';
    } else if (allTimes.length == 1) {
      timeSpan = '第${allTimes[0]}节';
    } else {
      timeSpan = '第${allTimes[0]}-${allTimes[allTimes.length - 1]}节';
    }

    final consolidatedArrangement = [
      weeksText.isNotEmpty ? '[$weeksText周]' : '',
      '周${first.occupyDay}',
      timeSpan,
      consolidatedRoom,
    ].where((s) => s.isNotEmpty).join(' ');

    final consolidatedShowText = [
      consolidatedTeacher,
      '${first.courseName}(${first.code})',
      consolidatedArrangement,
    ].where((s) => s.isNotEmpty).join(' ');

    result.add(
      PkConsolidatableCourse(
        code: first.code,
        courseName: first.courseName,
        occupyDay: first.occupyDay,
        occupyTime: allTimes,
        occupyWeek: allWeeks,
        teacherAndCode: consolidatedTeacher,
        arrangementText: consolidatedArrangement,
        occupyRoom: consolidatedRoom,
        showText: consolidatedShowText,
      ),
    );
  }

  return result;
}
