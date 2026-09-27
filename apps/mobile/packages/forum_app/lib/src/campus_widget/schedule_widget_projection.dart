import 'dart:convert';

import 'package:core/core.dart';
import 'package:crypto/crypto.dart';

import '../offline/campus_snapshot_store.dart';
import '../schedule/schedule_grid.dart';

class ScheduleWidgetProjection {
  ScheduleWidgetProjection({
    required this.identity,
    required this.generatedAt,
    required String schoolDate,
    required this.semesterId,
    required int? week,
    required this.sectionTimes,
    required String dayKind,
    required String? adjustmentLabel,
    required List<ScheduleWidgetCourse> courses,
  }) : days = [
         ScheduleWidgetDay(
           date: schoolDate,
           week: week,
           source: 'server-adjusted',
           kind: dayKind,
           adjustmentLabel: _optionalText(adjustmentLabel),
           courses: courses,
         ),
       ];

  ScheduleWidgetProjection._({
    required this.identity,
    required this.generatedAt,
    required this.semesterId,
    required this.sectionTimes,
    required this.days,
  });

  static const schemaVersion = 2;
  static const timezone = 'Asia/Shanghai';
  static const staleAfter = Duration(days: 7);
  static const rollingDayCount = 8;

  final ScheduleWidgetIdentity identity;
  final DateTime generatedAt;
  final String semesterId;
  final List<SectionTime> sectionTimes;
  final List<ScheduleWidgetDay> days;

  String get schoolDate => days.first.date;
  int? get week => days.first.week;
  String get dayKind => days.first.kind;
  String? get adjustmentLabel => days.first.adjustmentLabel;
  List<ScheduleWidgetCourse> get courses => days.first.courses;

  factory ScheduleWidgetProjection.fromSnapshot(
    CampusSnapshot snapshot,
    CampusCacheScope scope, {
    List<SectionTime>? sectionTimeOverrides,
    CampusCalendarRules? calendarRules,
  }) {
    final today = snapshot.data['today']!;
    final calendar = snapshot.data['calendar']!;
    final timetable = snapshot.data['timetable']!;
    final teachingDay = today.teachingDay;
    if (teachingDay == null || _calendarDate(teachingDay.date) == null) {
      throw const FormatException('today.teachingDay is required');
    }
    final sectionCount = timetable.events.any((event) => event.end == 12)
        ? 12
        : teachingDay.sectionCount;
    final times = sectionTimesFor(sectionCount, sectionTimeOverrides);
    final officialCourses =
        today.events
            .map((event) => _courseFromEvent(event, teachingDay.date, times))
            .nonNulls
            .toList()
          ..sort((a, b) => a.startAt.compareTo(b.startAt));
    final days = <ScheduleWidgetDay>[
      ScheduleWidgetDay(
        date: teachingDay.date,
        week: int.tryParse(_metric(calendar, '教学周')),
        source: 'server-adjusted',
        kind: _dayKind(teachingDay.kind),
        adjustmentLabel: _optionalText(teachingDay.label),
        courses: officialCourses,
      ),
    ];
    final resolver = calendarRules == null
        ? null
        : _FutureDayResolver.tryCreate(
            calendar: calendar,
            timetable: timetable,
            rules: calendarRules,
            sectionTimes: times,
          );
    final start = _calendarDate(teachingDay.date)!;
    for (var offset = 1; offset < rollingDayCount; offset++) {
      final date = _dateOnly(start.add(Duration(days: offset)));
      days.add(resolver?.resolve(date) ?? ScheduleWidgetDay.unknown(date));
    }
    return ScheduleWidgetProjection._(
      identity: ScheduleWidgetIdentity(
        siteKey: _opaque(scope.site),
        accountScope: _opaque('${scope.site}:${scope.accountId}'),
        bindingRevision: snapshot.bindingRevision,
      ),
      generatedAt: snapshot.committedAt,
      semesterId: _optionalText(_metric(calendar, '当前学期')) ?? '',
      sectionTimes: times,
      days: days,
    );
  }

  factory ScheduleWidgetProjection.fromJson(Map<String, dynamic> json) {
    final version = json['schemaVersion'];
    if ((version != 1 && version != schemaVersion) ||
        json['timezone'] != timezone) {
      throw const FormatException('unsupported widget projection');
    }
    final identity = ScheduleWidgetIdentity.fromJson(_map(json['identity']));
    final generatedAt = _dateTime(json['generatedAt']);
    final semester = _map(json['semester']);
    final sectionTimes = _list(
      json['sectionTimes'],
    ).map(_sectionTime).nonNulls.toList();
    final parsedDays = version == 1
        ? [
            ScheduleWidgetDay.fromJson({
              ..._map(json['today']),
              'date': json['schoolDate'],
              'week': semester['week'],
              'source': 'server-adjusted',
            }),
          ]
        : _list(
            json['days'],
          ).map((value) => ScheduleWidgetDay.fromJson(_map(value))).toList();
    if (generatedAt == null || parsedDays.isEmpty) {
      throw const FormatException('invalid widget projection');
    }
    return ScheduleWidgetProjection._(
      identity: identity,
      generatedAt: generatedAt,
      semesterId: _optionalText(semester['id']) ?? '',
      sectionTimes: sectionTimes,
      days: parsedDays,
    );
  }

  Map<String, Object?> toJson() => {
    'schemaVersion': schemaVersion,
    'identity': identity.toJson(),
    'generatedAt': _shanghaiIso(generatedAt),
    'schoolDate': schoolDate,
    'timezone': timezone,
    'semester': {'id': semesterId, 'week': week},
    'sectionTimes': [
      for (final time in sectionTimes)
        {'section': time.section, 'start': time.start, 'end': time.end},
    ],
    'days': [for (final day in days) day.toJson()],
  };

  String encode() => jsonEncode(toJson());

  ScheduleWidgetStatus statusAt(DateTime value) {
    final now = value.toUtc();
    if (now.difference(generatedAt.toUtc()) > staleAfter) {
      return const ScheduleWidgetStatus('stale');
    }
    final date = _schoolDate(now);
    final day = days.where((candidate) => candidate.date == date).firstOrNull;
    if (day == null || day.kind == 'unknown') {
      return const ScheduleWidgetStatus('needsRefresh');
    }
    if (day.kind == 'holiday') return const ScheduleWidgetStatus('holiday');
    if (day.courses.isEmpty) {
      return const ScheduleWidgetStatus('noClasses');
    }
    for (var index = 0; index < day.courses.length; index++) {
      final course = day.courses[index];
      if (!now.isBefore(course.startAt.toUtc()) &&
          now.isBefore(course.endAt.toUtc())) {
        return ScheduleWidgetStatus('inClass', course: course);
      }
      if (now.isBefore(course.startAt.toUtc())) {
        return ScheduleWidgetStatus(
          index == 0 ? 'upcoming' : 'break',
          course: course,
        );
      }
    }
    return const ScheduleWidgetStatus('finished');
  }

  ScheduleWidgetStatus nextClassAt(DateTime value) {
    final now = value.toUtc();
    if (now.difference(generatedAt.toUtc()) > staleAfter) {
      return const ScheduleWidgetStatus('stale');
    }
    final date = _schoolDate(now);
    final today = days.where((candidate) => candidate.date == date).firstOrNull;
    if (today == null || today.kind == 'unknown') {
      return const ScheduleWidgetStatus('needsRefresh');
    }
    for (final course in today.courses) {
      if (!now.isBefore(course.startAt.toUtc()) &&
          now.isBefore(course.endAt.toUtc())) {
        return ScheduleWidgetStatus('inClass', course: course, day: today);
      }
    }
    final future = <(ScheduleWidgetDay, ScheduleWidgetCourse)>[
      for (final day in days.where(
        (candidate) => candidate.date.compareTo(date) >= 0,
      ))
        for (final course in day.courses)
          if (course.startAt.toUtc().isAfter(now)) (day, course),
    ]..sort((left, right) => left.$2.startAt.compareTo(right.$2.startAt));
    if (future.isNotEmpty) {
      return ScheduleWidgetStatus(
        'upcoming',
        course: future.first.$2,
        day: future.first.$1,
      );
    }
    if (days.length < rollingDayCount ||
        days.any(
          (day) => day.date.compareTo(date) >= 0 && day.kind == 'unknown',
        )) {
      return const ScheduleWidgetStatus('needsRefresh');
    }
    return const ScheduleWidgetStatus('noneUpcoming');
  }

  List<DateTime> updateTimesAfter(DateTime value) {
    final now = value.toUtc();
    return <DateTime>{
      for (final day in days)
        for (final course in day.courses) course.startAt.toUtc(),
      for (final day in days)
        for (final course in day.courses) course.endAt.toUtc(),
      for (final day in days) _schoolInstant(day.date, '00:00').toUtc(),
      generatedAt.toUtc().add(staleAfter),
    }.where((time) => time.isAfter(now)).toList()..sort();
  }
}

class ScheduleWidgetDay {
  ScheduleWidgetDay({
    required this.date,
    required this.week,
    required this.source,
    required this.kind,
    required this.adjustmentLabel,
    required List<ScheduleWidgetCourse> courses,
  }) : courses = List.unmodifiable(courses);

  factory ScheduleWidgetDay.unknown(String date) => ScheduleWidgetDay(
    date: date,
    week: null,
    source: 'unknown',
    kind: 'unknown',
    adjustmentLabel: null,
    courses: const [],
  );

  factory ScheduleWidgetDay.fromJson(Map<String, dynamic> json) {
    final date = _optionalText(json['date']);
    if (_calendarDate(date ?? '') == null) {
      throw const FormatException('invalid widget day');
    }
    final courses =
        _list(json['courses'])
            .map((value) => ScheduleWidgetCourse.tryFromJson(_map(value)))
            .nonNulls
            .toList()
          ..sort((a, b) => a.startAt.compareTo(b.startAt));
    return ScheduleWidgetDay(
      date: date!,
      week: json['week'] is int && (json['week'] as int) > 0
          ? json['week'] as int
          : null,
      source: _source(json['source']),
      kind: _dayKind(json['kind']),
      adjustmentLabel: _optionalText(json['adjustmentLabel']),
      courses: courses,
    );
  }

  final String date;
  final int? week;
  final String source;
  final String kind;
  final String? adjustmentLabel;
  final List<ScheduleWidgetCourse> courses;

  Map<String, Object?> toJson() => {
    'date': date,
    'week': week,
    'source': source,
    'kind': kind,
    'holiday': kind == 'holiday',
    if (adjustmentLabel != null) 'adjustmentLabel': adjustmentLabel,
    'courses': [for (final course in courses) course.toJson()],
  };
}

class ScheduleWidgetIdentity {
  const ScheduleWidgetIdentity({
    required this.siteKey,
    required this.accountScope,
    required this.bindingRevision,
  });

  final String siteKey;
  final String accountScope;
  final String bindingRevision;

  factory ScheduleWidgetIdentity.fromJson(Map<String, dynamic> json) {
    final value = ScheduleWidgetIdentity(
      siteKey: _optionalText(json['siteKey']) ?? '',
      accountScope: _optionalText(json['accountScope']) ?? '',
      bindingRevision: _optionalText(json['bindingRevision']) ?? '',
    );
    if (value.siteKey.isEmpty ||
        value.accountScope.isEmpty ||
        value.bindingRevision.isEmpty) {
      throw const FormatException('invalid widget identity');
    }
    return value;
  }

  Map<String, String> toJson() => {
    'siteKey': siteKey,
    'accountScope': accountScope,
    'bindingRevision': bindingRevision,
  };
}

class ScheduleWidgetCourse {
  const ScheduleWidgetCourse({
    required this.stableId,
    required this.name,
    required this.teacher,
    required this.room,
    required this.campus,
    required this.startSection,
    required this.endSection,
    required this.startAt,
    required this.endAt,
    required this.colorSlot,
  });

  final String stableId;
  final String name;
  final String teacher;
  final String room;
  final String campus;
  final int startSection;
  final int endSection;
  final DateTime startAt;
  final DateTime endAt;
  final int colorSlot;

  static ScheduleWidgetCourse? tryFromJson(Map<String, dynamic> json) {
    final stableId = _optionalText(json['stableId']);
    final name = _optionalText(json['name']);
    final startAt = _dateTime(json['startAt']);
    final endAt = _dateTime(json['endAt']);
    final startSection = json['startSection'];
    final endSection = json['endSection'];
    final colorSlot = json['colorSlot'];
    if (stableId == null ||
        name == null ||
        startAt == null ||
        endAt == null ||
        !endAt.isAfter(startAt) ||
        startSection is! int ||
        endSection is! int ||
        startSection < 1 ||
        endSection < startSection ||
        colorSlot is! int ||
        colorSlot < 1 ||
        colorSlot > 8) {
      return null;
    }
    return ScheduleWidgetCourse(
      stableId: stableId,
      name: name,
      teacher: _optionalText(json['teacher']) ?? '',
      room: _optionalText(json['room']) ?? '',
      campus: _optionalText(json['campus']) ?? '',
      startSection: startSection,
      endSection: endSection,
      startAt: startAt,
      endAt: endAt,
      colorSlot: colorSlot,
    );
  }

  Map<String, Object> toJson() => {
    'stableId': stableId,
    'name': name,
    'teacher': teacher,
    'room': room,
    'campus': campus,
    'startSection': startSection,
    'endSection': endSection,
    'startAt': _shanghaiIso(startAt),
    'endAt': _shanghaiIso(endAt),
    'colorSlot': colorSlot,
  };
}

class ScheduleWidgetStatus {
  const ScheduleWidgetStatus(this.kind, {this.course, this.day});

  final String kind;
  final ScheduleWidgetCourse? course;
  final ScheduleWidgetDay? day;
}

class _FutureDayResolver {
  const _FutureDayResolver({
    required this.begin,
    required this.end,
    required this.weeks,
    required this.timetable,
    required this.rules,
    required this.sectionTimes,
  });

  final DateTime begin;
  final DateTime end;
  final int weeks;
  final CampusDataset timetable;
  final CampusCalendarRules rules;
  final List<SectionTime> sectionTimes;

  static _FutureDayResolver? tryCreate({
    required CampusDataset calendar,
    required CampusDataset timetable,
    required CampusCalendarRules rules,
    required List<SectionTime> sectionTimes,
  }) {
    final begin = _calendarDate(_metric(calendar, '学期开始'));
    final end = _calendarDate(_metric(calendar, '学期结束'));
    final weeks = int.tryParse(_metric(calendar, '学期周数'));
    if (begin == null ||
        end == null ||
        weeks == null ||
        begin.weekday != DateTime.monday ||
        end.isBefore(begin) ||
        end.difference(begin).inDays >= 371 ||
        weeks < 1 ||
        weeks > 53 ||
        !_validRules(rules, begin, end) ||
        timetable.events.any(
          (event) => !_validEvent(event, weeks, begin, end, sectionTimes),
        )) {
      return null;
    }
    return _FutureDayResolver(
      begin: begin,
      end: end,
      weeks: weeks,
      timetable: timetable,
      rules: rules,
      sectionTimes: sectionTimes,
    );
  }

  ScheduleWidgetDay resolve(String actualDate) {
    final actual = _calendarDate(actualDate)!;
    var source = actual;
    var kind = 'none';
    String? label;
    for (final move in rules.moves) {
      if (move.toDate == actualDate) {
        source = _calendarDate(move.fromDate)!;
        kind = 'makeup';
        label = _optionalText(move.name);
        break;
      }
    }
    if (kind == 'none') {
      for (final holiday in rules.holidays) {
        if (actualDate.compareTo(holiday.startDate) >= 0 &&
            actualDate.compareTo(holiday.endDate) <= 0) {
          kind = 'holiday';
          label = _optionalText(holiday.name);
          break;
        }
      }
    }
    if (kind == 'none') {
      for (final move in rules.moves) {
        if (move.fromDate == actualDate) {
          kind = 'moved';
          label = _optionalText(move.name);
          break;
        }
      }
    }
    if (kind == 'holiday' || kind == 'moved') {
      return ScheduleWidgetDay(
        date: actualDate,
        week: null,
        source: 'local-resolved',
        kind: kind,
        adjustmentLabel: label,
        courses: const [],
      );
    }
    if (source.isBefore(begin) || source.isAfter(end)) {
      return kind == 'makeup'
          ? ScheduleWidgetDay.unknown(actualDate)
          : ScheduleWidgetDay(
              date: actualDate,
              week: null,
              source: 'local-resolved',
              kind: 'none',
              adjustmentLabel: null,
              courses: const [],
            );
    }
    final week = source.difference(begin).inDays ~/ 7 + 1;
    final courses =
        timetable.events
            .where(
              (event) =>
                  event.day == source.weekday && event.weeks.contains(week),
            )
            .map(
              (event) => _courseFromEvent(
                event,
                actualDate,
                sectionTimes,
                seedDate: _dateOnly(source),
              ),
            )
            .nonNulls
            .toList()
          ..sort((a, b) => a.startAt.compareTo(b.startAt));
    return ScheduleWidgetDay(
      date: actualDate,
      week: week,
      source: 'local-resolved',
      kind: kind,
      adjustmentLabel: label,
      courses: courses,
    );
  }
}

ScheduleWidgetCourse? _courseFromEvent(
  CampusEvent event,
  String date,
  List<SectionTime> times, {
  String? seedDate,
}) {
  final name = _optionalText(event.name);
  if (name == null ||
      event.start < 1 ||
      event.end < event.start ||
      event.end > times.length) {
    return null;
  }
  final seed = [
    ?seedDate,
    name,
    event.day,
    event.start,
    event.end,
    _optionalText(event.room) ?? '',
  ].join('|');
  return ScheduleWidgetCourse(
    stableId: sha256.convert(utf8.encode(seed)).toString().substring(0, 16),
    name: name,
    teacher: _optionalText(event.teacher) ?? '',
    room: _optionalText(event.room) ?? '',
    campus: _optionalText(event.campus) ?? '',
    startSection: event.start,
    endSection: event.end,
    startAt: _schoolInstant(date, times[event.start - 1].start),
    endAt: _schoolInstant(date, times[event.end - 1].end),
    colorSlot: courseColorSlotFor(name),
  );
}

bool _validEvent(
  CampusEvent event,
  int weeks,
  DateTime begin,
  DateTime end,
  List<SectionTime> times,
) {
  if (_optionalText(event.name) == null ||
      event.day < 1 ||
      event.day > 7 ||
      event.start < 1 ||
      event.end < event.start ||
      event.end > times.length ||
      event.weeks.isEmpty) {
    return false;
  }
  return event.weeks.every((week) {
    if (week < 1 || week > weeks) return false;
    return !begin
        .add(Duration(days: (week - 1) * 7 + event.day - 1))
        .isAfter(end);
  });
}

bool _validRules(CampusCalendarRules rules, DateTime begin, DateTime end) {
  if (rules.holidays.length > 60 || rules.moves.length > 120) return false;
  final holidayDays = <String>{};
  for (final holiday in rules.holidays) {
    final start = _calendarDate(holiday.startDate);
    final finish = _calendarDate(holiday.endDate);
    if (_optionalText(holiday.name) == null ||
        start == null ||
        finish == null ||
        finish.isBefore(start) ||
        finish.difference(start).inDays > 365) {
      return false;
    }
    for (
      var day = start;
      !day.isAfter(finish);
      day = day.add(const Duration(days: 1))
    ) {
      if (!holidayDays.add(_dateOnly(day))) return false;
    }
  }
  final from = <String>{};
  final to = <String>{};
  for (final move in rules.moves) {
    final source = _calendarDate(move.fromDate);
    final target = _calendarDate(move.toDate);
    if (_optionalText(move.name) == null ||
        source == null ||
        target == null ||
        move.fromDate == move.toDate ||
        !from.add(move.fromDate) ||
        !to.add(move.toDate) ||
        holidayDays.contains(move.toDate) ||
        ((source.isBefore(begin) || source.isAfter(end)) &&
            !target.isBefore(begin) &&
            !target.isAfter(end))) {
      return false;
    }
  }
  return from.intersection(to).isEmpty;
}

String _metric(CampusDataset data, String label) =>
    data.metrics
        .where((metric) => metric.label == label)
        .map((metric) => metric.value)
        .firstOrNull ??
    '';

String _opaque(String value) =>
    sha256.convert(utf8.encode(value)).toString().substring(0, 16);

String? _optionalText(Object? value) {
  if (value is! String) return null;
  final text = value.trim();
  if (text.isEmpty ||
      const {'null', 'undefined'}.contains(text.toLowerCase())) {
    return null;
  }
  return text;
}

String _dayKind(Object? value) {
  final kind = _optionalText(value);
  return const {
        'none',
        'normal',
        'holiday',
        'moved',
        'makeup',
        'unknown',
      }.contains(kind)
      ? kind!
      : 'unknown';
}

String _source(Object? value) {
  final source = _optionalText(value);
  return const {'server-adjusted', 'local-resolved', 'unknown'}.contains(source)
      ? source!
      : 'unknown';
}

Map<String, dynamic> _map(Object? value) => value is Map<String, dynamic>
    ? value
    : throw const FormatException('invalid widget object');

List<Object?> _list(Object? value) => value is List
    ? value.cast<Object?>()
    : throw const FormatException('invalid widget list');

SectionTime? _sectionTime(Object? value) {
  final json = _map(value);
  final section = json['section'];
  final start = _optionalText(json['start']);
  final end = _optionalText(json['end']);
  if (section is! int || section < 1 || start == null || end == null) {
    return null;
  }
  return SectionTime(section: section, start: start, end: end);
}

DateTime? _dateTime(Object? value) {
  final text = _optionalText(value);
  return text == null ? null : DateTime.tryParse(text);
}

DateTime? _calendarDate(String value) {
  if (!RegExp(r'^\d{4}-\d{2}-\d{2}$').hasMatch(value)) return null;
  final parsed = DateTime.tryParse('${value}T00:00:00Z');
  if (parsed == null ||
      parsed.year < 2000 ||
      parsed.year > 2100 ||
      _dateOnly(parsed) != value) {
    return null;
  }
  return parsed;
}

String _dateOnly(DateTime value) =>
    value.toUtc().toIso8601String().substring(0, 10);

DateTime _schoolInstant(String date, String time) =>
    DateTime.parse('${date}T${time.padLeft(5, '0')}:00+08:00');

String _schoolDate(DateTime value) => value
    .toUtc()
    .add(const Duration(hours: 8))
    .toIso8601String()
    .substring(0, 10);

String _shanghaiIso(DateTime value) {
  final school = value.toUtc().add(const Duration(hours: 8));
  return '${school.toIso8601String().substring(0, 19)}+08:00';
}
