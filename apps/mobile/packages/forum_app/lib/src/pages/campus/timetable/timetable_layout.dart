import 'package:core/core.dart';

import '../../../schedule/schedule_grid.dart';
import '../campus_helpers.dart';
import 'timetable_memo_store.dart';

enum TimetableItemKind { course, block, otherWeek }

/// One rendered session: an official course, a user block, or (optionally)
/// a course from another week ghosted into a free cell.
class TimetableItem {
  const TimetableItem({
    required this.kind,
    required this.day,
    required this.start,
    required this.end,
    required this.title,
    this.place = '',
    this.campus = '',
    this.teacher = '',
    this.weeks = const [],
    this.credits = '',
    this.eventIndex,
    this.block,
    required this.colorSlot,
  });

  final TimetableItemKind kind;
  final int day;
  final int start;
  final int end;
  final String title;
  final String place;
  final String campus;
  final String teacher;
  final List<int> weeks;
  final String credits;

  /// Index into [campusCoursesForWeek] for the shown week (course details);
  /// other-week ghosts index the unfiltered list (`week: null`).
  final int? eventIndex;
  final TimetableBlock? block;

  /// Zero-based palette slot; a course keeps its color in every week.
  final int colorSlot;

  String get courseKey => timetableCourseKey(title, teacher);
  List<int> get sections => [for (var s = start; s <= end; s++) s];
}

class TimetableCluster {
  const TimetableCluster(this.start, this.end, this.items);
  final int start;
  final int end;
  final List<TimetableItem> items;
}

class TimetableWeekLayout {
  const TimetableWeekLayout({
    required this.rows,
    required this.days,
    required this.clusters,
    required this.courseCount,
  });

  /// Section rows: 11 for the current schedule, 12 for historical terms.
  final int rows;

  /// Visible weekdays (1-7) in display order.
  final List<int> days;

  /// Clusters per weekday (index 0 = Monday) with overlapping items stacked.
  final List<List<TimetableCluster>> clusters;

  /// Official sessions in this week (empty-week messaging).
  final int courseCount;

  /// Whether a free cell exists at [day]/[section] (no clustered item).
  /// Ghosted other-week sessions do not occupy a cell.
  bool isFree(int day, int section) => !clusters[day - 1].any(
    (c) =>
        section >= c.start &&
        section <= c.end &&
        c.items.any(
          (item) =>
              item.kind != TimetableItemKind.otherWeek &&
              section >= item.start &&
              section <= item.end,
        ),
  );
}

int timetableColorSlot(String seed) =>
    (courseColorSlotFor(seed) - 1) % kCourseColorSlotCount;

TimetableWeekLayout buildTimetableWeek({
  required List<CampusEvent> events,
  required int week,
  required List<TimetableBlock> blocks,
  required TimetablePrefs prefs,
}) {
  final rows = events.any((e) => e.end == 12) ? 12 : 11;
  final visible = campusCoursesForWeek(events, week);
  final byDay = List.generate(7, (_) => <TimetableItem>[]);
  for (var i = 0; i < visible.length; i++) {
    final e = visible[i];
    if (e.end > rows) continue;
    byDay[e.day - 1].add(
      TimetableItem(
        kind: TimetableItemKind.course,
        day: e.day,
        start: e.start,
        end: e.end,
        title: e.name,
        place: e.room,
        campus: e.campus,
        teacher: e.teacher,
        weeks: e.weeks,
        credits: e.credits,
        eventIndex: i,
        colorSlot: timetableColorSlot(e.name),
      ),
    );
  }
  for (final block in blocks) {
    if (!block.occursIn(week)) continue;
    if (block.start > rows) continue;
    final end = block.end.clamp(block.start, rows);
    for (final day in block.days) {
      byDay[day - 1].add(
        TimetableItem(
          kind: TimetableItemKind.block,
          day: day,
          start: block.start,
          end: end,
          title: block.title,
          place: block.place,
          weeks: block.weeks,
          block: block,
          colorSlot: block.color % kCourseColorSlotCount,
        ),
      );
    }
  }
  if (prefs.showOtherWeeks) {
    final seen = <String>{};
    final all = campusCoursesForWeek(events, null);
    for (var i = 0; i < all.length; i++) {
      final e = all[i];
      if (e.weeks.isEmpty || e.weeks.contains(week) || e.end > rows) continue;
      final taken = byDay[e.day - 1].any(
        (item) =>
            item.kind != TimetableItemKind.otherWeek &&
            item.start <= e.end &&
            e.start <= item.end,
      );
      if (taken || !seen.add('${e.day}:${e.start}:${e.end}:${e.name}')) {
        continue;
      }
      byDay[e.day - 1].add(
        TimetableItem(
          kind: TimetableItemKind.otherWeek,
          day: e.day,
          start: e.start,
          end: e.end,
          title: e.name,
          place: e.room,
          campus: e.campus,
          teacher: e.teacher,
          weeks: e.weeks,
          credits: e.credits,
          eventIndex: i,
          colorSlot: timetableColorSlot(e.name),
        ),
      );
    }
  }
  final clusters = [
    for (final items in byDay)
      [
        for (final c in clusterBySections(items, (item) => item.sections))
          TimetableCluster(c.start, c.end, c.items),
      ],
  ];
  final weekendUsed = clusters[5].isNotEmpty || clusters[6].isNotEmpty;
  return TimetableWeekLayout(
    rows: rows,
    days: [
      for (var d = 1; d <= 5; d++) d,
      if (prefs.alwaysShowWeekend || weekendUsed) ...[6, 7],
    ],
    clusters: clusters,
    courseCount: visible.length,
  );
}

/// Campus-local calendar dates of [week], anchored on today's teaching week.
/// The returned values are UTC-tagged wall dates in campus time.
List<DateTime> timetableWeekDates(int week, int currentWeek, DateTime now) {
  final today = campusNow(now);
  final monday = DateTime.utc(
    today.year,
    today.month,
    today.day,
  ).subtract(Duration(days: today.weekday - 1));
  return [
    for (var d = 0; d < 7; d++)
      monday.add(Duration(days: (week - currentWeek) * 7 + d)),
  ];
}

/// Minutes since midnight in campus time.
int timetableMinuteOfDay(DateTime now) {
  final campus = campusNow(now);
  return campus.hour * 60 + campus.minute;
}

/// Fractional section position of [minute] (0 = top of section 1), or null
/// outside the teaching day. Breaks snap to the boundary that follows them.
double? timetableNowPosition(List<SectionTime> times, int rows, int minute) {
  final count = rows < times.length ? rows : times.length;
  for (var i = 0; i < count; i++) {
    final start = parseHHMM(times[i].start);
    final end = parseHHMM(times[i].end);
    if (start == null || end == null || end <= start) return null;
    if (i == 0 && minute < start) return null;
    if (minute < start) return i.toDouble();
    if (minute <= end) return i + (minute - start) / (end - start);
  }
  return null;
}
