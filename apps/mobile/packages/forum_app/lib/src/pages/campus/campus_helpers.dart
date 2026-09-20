import 'package:core/core.dart';
import '../../../l10n/app_localizations.dart';
import '../../schedule/schedule_grid.dart';

DateTime campusNow(DateTime date) => date.toUtc().add(const Duration(hours: 8));
String campusDateKey(DateTime date) =>
    campusNow(date).toIso8601String().substring(0, 10);
String campusGreeting(AppLocalizations l, DateTime date) {
  final h = campusNow(date).hour;
  return h < 5
      ? l.campusNight
      : h < 11
      ? l.campusMorning
      : h < 14
      ? l.campusNoon
      : h < 18
      ? l.campusAfternoon
      : l.campusEvening;
}

String campusMetric(CampusDataset? data, String label) {
  for (final m in data?.metrics ?? <CampusMetric>[]) {
    if (m.label == label) return m.value;
  }
  return '';
}

List<CampusEvent> campusCoursesForWeek(List<CampusEvent> events, int? week) =>
    events
        .where(
          (e) =>
              e.day >= 1 &&
              e.day <= 7 &&
              e.start >= 1 &&
              e.end >= e.start &&
              e.end <= 12 &&
              (week == null || e.weeks.isEmpty || e.weeks.contains(week)),
        )
        .toList();
ScheduleGridData campusGrid(List<CampusEvent> events, int week) {
  final visible = campusCoursesForWeek(events, week);
  final rows = events.any((e) => e.end == 12) ? 12 : 11;
  return buildScheduleGridData(
    [
      for (var i = 0; i < visible.length; i++)
        PkCourseOnTable(
          showText: visible[i].name,
          courseName: visible[i].name,
          // The API has no class ID. Never consolidate distinct official records.
          code: 'campus:$i:${visible[i].name}',
          occupyDay: visible[i].day,
          occupyTime: [
            for (var s = visible[i].start; s <= visible[i].end; s++) s,
          ],
          occupyWeek: visible[i].weeks,
          teacherAndCode: visible[i].teacher,
          occupyRoom: [
            visible[i].room,
            visible[i].campus,
          ].where((s) => s.isNotEmpty).join(' · '),
        ),
    ],
    const [],
    maxRows: rows,
  );
}

String campusError(AppLocalizations l, Object? e) {
  if (e is ApiException) {
    return switch (e.messageCode) {
      'campus.authorizationRequired' ||
      'campus.messageAuthorizationRequired' => l.campusAuthRequired,
      'campus.authorizationExpired' => l.campusAuthExpired,
      'campus.identityUnavailable' ||
      'campus.connectionChanged' => l.campusIdentityConflict,
      'campus.messageUnavailable' => l.campusMessageUnavailable,
      'campus.calendarIncomplete' => l.campusCalendarIncomplete,
      'campus.rulesUnavailable' ||
      'campus.rulesInvalid' => l.campusRulesUnavailable,
      'campus.calendarEmpty' => l.campusCalendarEmpty,
      'campus.disabled' => l.campusDisabled,
      _ => l.campusUnavailable,
    };
  }
  return l.campusUnavailable;
}
