import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../campus_widget/schedule_widget_projection.dart';
import '../../../current_user.dart';
import '../../../local/writing_store.dart' show writingScope;
import '../../../offline/campus_snapshot_store.dart';
import '../../../providers.dart';
import '../../../schedule/schedule_grid.dart' show kCourseColorSlotCount;
import '../../../storage/user_work_database.dart';

/// User-authored timetable memos: course notes, hand-drawn blocks and display
/// preferences. They are local work (like drafts), scoped per site and forum
/// account, and never leave the device or enter the school snapshot.
const String kTimetableMemoDomain = 'campus-timetable';

/// Official records carry no class ID; a note follows the visible course name
/// and teacher across every session of that course.
String timetableCourseKey(String name, String teacher) => '$name\u001f$teacher';

class TimetableNote {
  const TimetableNote({required this.course, required this.text, this.week});

  /// [timetableCourseKey] of the course.
  final String course;

  /// Null applies to every week; otherwise only that teaching week.
  final int? week;
  final String text;

  Map<String, Object?> toJson() => {
    'course': course,
    'week': week,
    'text': text,
  };

  static TimetableNote? fromJson(Object? json) {
    if (json is! Map) return null;
    final course = json['course'];
    final text = json['text'];
    final week = json['week'];
    if (course is! String || text is! String || text.trim().isEmpty) {
      return null;
    }
    return TimetableNote(
      course: course,
      text: text,
      week: week is int && week > 0 ? week : null,
    );
  }
}

/// A block the user drew on the grid (club meeting, self-study, reminder).
class TimetableBlock {
  const TimetableBlock({
    required this.id,
    required this.title,
    required this.days,
    required this.start,
    required this.end,
    this.place = '',
    this.note = '',
    this.color = 0,
    this.weeks = const [],
  });

  final String id;
  final String title;
  final String place;
  final String note;

  /// Index into the shared course palette.
  final int color;

  /// Weekdays 1 (Monday) to 7.
  final List<int> days;
  final int start;
  final int end;

  /// Teaching weeks; empty repeats every week.
  final List<int> weeks;

  bool occursIn(int week) => weeks.isEmpty || weeks.contains(week);

  /// The fields a home-screen widget shows; notes stay in the app.
  ScheduleWidgetBlock toWidget() => ScheduleWidgetBlock(
    id: id,
    title: title,
    place: place,
    colorSlot: color % kCourseColorSlotCount + 1,
    days: days,
    start: start,
    end: end,
    weeks: weeks,
  );

  TimetableBlock copyWith({
    String? title,
    String? place,
    String? note,
    int? color,
    List<int>? days,
    int? start,
    int? end,
    List<int>? weeks,
  }) => TimetableBlock(
    id: id,
    title: title ?? this.title,
    place: place ?? this.place,
    note: note ?? this.note,
    color: color ?? this.color,
    days: days ?? this.days,
    start: start ?? this.start,
    end: end ?? this.end,
    weeks: weeks ?? this.weeks,
  );

  Map<String, Object?> toJson() => {
    'id': id,
    'title': title,
    'place': place,
    'note': note,
    'color': color,
    'days': days,
    'start': start,
    'end': end,
    'weeks': weeks,
  };

  static TimetableBlock? fromJson(Object? json) {
    if (json is! Map) return null;
    List<int> ints(Object? value) => value is List
        ? value.whereType<int>().toList(growable: false)
        : const [];
    final id = json['id'];
    final title = json['title'];
    final start = json['start'];
    final end = json['end'];
    final days = ints(json['days']).where((d) => d >= 1 && d <= 7).toList();
    if (id is! String ||
        title is! String ||
        start is! int ||
        end is! int ||
        start < 1 ||
        end < start ||
        days.isEmpty) {
      return null;
    }
    return TimetableBlock(
      id: id,
      title: title,
      place: json['place'] is String ? json['place'] as String : '',
      note: json['note'] is String ? json['note'] as String : '',
      color: json['color'] is int ? json['color'] as int : 0,
      days: days,
      start: start,
      end: end,
      weeks: ints(json['weeks']).where((w) => w > 0).toList(),
    );
  }
}

class TimetablePrefs {
  const TimetablePrefs({
    this.alwaysShowWeekend = false,
    this.showOtherWeeks = false,
    this.showSectionTimes = true,
    this.focusToday = false,
  });

  /// Off: an empty weekend folds away so weekdays get the width.
  final bool alwaysShowWeekend;

  /// Ghost other-week sessions into free cells, as paper timetables do.
  final bool showOtherWeeks;
  final bool showSectionTimes;

  /// Open the current week with today's column focused.
  final bool focusToday;

  TimetablePrefs copyWith({
    bool? alwaysShowWeekend,
    bool? showOtherWeeks,
    bool? showSectionTimes,
    bool? focusToday,
  }) => TimetablePrefs(
    alwaysShowWeekend: alwaysShowWeekend ?? this.alwaysShowWeekend,
    showOtherWeeks: showOtherWeeks ?? this.showOtherWeeks,
    showSectionTimes: showSectionTimes ?? this.showSectionTimes,
    focusToday: focusToday ?? this.focusToday,
  );

  Map<String, Object?> toJson() => {
    'alwaysShowWeekend': alwaysShowWeekend,
    'showOtherWeeks': showOtherWeeks,
    'showSectionTimes': showSectionTimes,
    'focusToday': focusToday,
  };

  static TimetablePrefs fromJson(Object? json) {
    if (json is! Map) return const TimetablePrefs();
    bool read(String key, bool fallback) =>
        json[key] is bool ? json[key] as bool : fallback;
    return TimetablePrefs(
      alwaysShowWeekend: read('alwaysShowWeekend', false),
      showOtherWeeks: read('showOtherWeeks', false),
      showSectionTimes: read('showSectionTimes', true),
      focusToday: read('focusToday', false),
    );
  }
}

class TimetableMemoState {
  const TimetableMemoState({
    this.loaded = false,
    this.notes = const [],
    this.blocks = const [],
    this.prefs = const TimetablePrefs(),
    this.error,
  });

  final bool loaded;
  final List<TimetableNote> notes;
  final List<TimetableBlock> blocks;
  final TimetablePrefs prefs;
  final Object? error;

  /// The every-week note when [week] is null, otherwise that week's memo.
  String noteFor(String course, int? week) {
    for (final note in notes) {
      if (note.course == course && note.week == week) return note.text;
    }
    return '';
  }

  /// Notes visible in [week]: the week memo first, then the standing note.
  List<String> notesIn(String course, int week) => [
    if (noteFor(course, week) case final String text when text.isNotEmpty) text,
    if (noteFor(course, null) case final String text when text.isNotEmpty) text,
  ];

  TimetableMemoState copyWith({
    bool? loaded,
    List<TimetableNote>? notes,
    List<TimetableBlock>? blocks,
    TimetablePrefs? prefs,
    Object? error,
    bool clearError = false,
  }) => TimetableMemoState(
    loaded: loaded ?? this.loaded,
    notes: notes ?? this.notes,
    blocks: blocks ?? this.blocks,
    prefs: prefs ?? this.prefs,
    error: clearError ? null : error ?? this.error,
  );
}

class TimetableMemoStore extends StateNotifier<TimetableMemoState> {
  TimetableMemoStore({required UserWorkDatabase database, required this.scope})
    : _database = database,
      _generation = database.generation,
      super(const TimetableMemoState()) {
    _tail = _load();
  }

  final UserWorkDatabase _database;
  final int _generation;
  bool _loadFailed = false;
  late Future<void> _tail;

  bool get canEdit => mounted && state.loaded && !_loadFailed;

  /// Null without a signed-in account: memos stay in memory only.
  final String? scope;

  Future<void> _load() async {
    final target = scope;
    if (target == null) {
      state = state.copyWith(loaded: true);
      return;
    }
    try {
      final values = await _database.readDomain(target, kTimetableMemoDomain);
      if (!mounted) return;
      // Tolerate corrupt data: a bad key or entry drops itself, the rest of
      // the domain still loads, and the next successful write rewrites the
      // cleaned key. An unreadable domain (below) still blocks edits so a
      // failed read can never be overwritten by an empty one.
      Object? decoded(String key, String fallback) {
        try {
          return jsonDecode(values[key] ?? fallback);
        } catch (_) {
          return null;
        }
      }

      List<T> list<T>(String key, T? Function(Object?) decode) {
        final value = decoded(key, '[]');
        if (value is! List) return const [];
        return value.map(decode).whereType<T>().toList();
      }

      final prefsJson = decoded('prefs', '{}');
      _loadFailed = false;
      state = TimetableMemoState(
        loaded: true,
        notes: list('notes', TimetableNote.fromJson),
        blocks: list('blocks', TimetableBlock.fromJson),
        prefs: prefsJson is Map
            ? TimetablePrefs.fromJson(prefsJson)
            : const TimetablePrefs(),
      );
    } catch (error) {
      _loadFailed = true;
      if (mounted) state = state.copyWith(loaded: true, error: error);
    }
  }

  Future<void> retryLoad() {
    _tail = _tail.then((_) async {
      if (!mounted || !_loadFailed) return;
      state = state.copyWith(loaded: false);
      await _load();
    });
    return _tail;
  }

  // Loading and mutations share one queue. Build each new value from the last
  // committed state, then publish it only after durable storage succeeds.
  Future<bool> _update(
    String key,
    TimetableMemoState Function(TimetableMemoState) update,
    Object Function(TimetableMemoState) encode,
  ) {
    final operation = _tail.then((_) async {
      if (!canEdit) return false;
      final next = update(state);
      try {
        if (scope case final String target) {
          await _database.writeBatch(
            target,
            kTimetableMemoDomain,
            {key: jsonEncode(encode(next))},
            generation: _generation,
            isCurrent: () => mounted,
          );
        }
        if (!mounted) return false;
        state = next.copyWith(clearError: true);
        return true;
      } catch (error) {
        if (mounted) state = state.copyWith(error: error);
        return false;
      }
    });
    _tail = operation.then<void>((_) {});
    return operation;
  }

  Future<bool> setNote(String course, int? week, String text) {
    final trimmed = text.trim();
    return _update(
      'notes',
      (current) => current.copyWith(
        notes: [
          for (final note in current.notes)
            if (note.course != course || note.week != week) note,
          if (trimmed.isNotEmpty)
            TimetableNote(course: course, week: week, text: trimmed),
        ],
      ),
      (next) => [for (final note in next.notes) note.toJson()],
    );
  }

  Future<bool> saveBlock(TimetableBlock block) => _update(
    'blocks',
    (current) => current.copyWith(
      blocks: [
        for (final item in current.blocks)
          if (item.id != block.id) item,
        block,
      ],
    ),
    (next) => [for (final item in next.blocks) item.toJson()],
  );

  Future<bool> deleteBlock(String id) => _update(
    'blocks',
    (current) => current.copyWith(
      blocks: [
        for (final item in current.blocks)
          if (item.id != id) item,
      ],
    ),
    (next) => [for (final item in next.blocks) item.toJson()],
  );

  Future<bool> setPrefs(TimetablePrefs prefs) => _update(
    'prefs',
    (current) => current.copyWith(prefs: prefs),
    (next) => next.prefs.toJson(),
  );

  Future<bool> updatePrefs(TimetablePrefs Function(TimetablePrefs) update) =>
      _update(
        'prefs',
        (current) => current.copyWith(prefs: update(current.prefs)),
        (next) => next.prefs.toJson(),
      );
}

String newTimetableBlockId() {
  final random = Random.secure();
  return 'block_${List.generate(8, (_) => random.nextInt(256).toRadixString(16).padLeft(2, '0')).join()}';
}

final timetableMemoProvider =
    StateNotifierProvider<TimetableMemoStore, TimetableMemoState>((ref) {
      ref.watch(offlineCacheEpochProvider);
      final id = ref.watch(currentUserProvider).valueOrNull?.id;
      final site = Uri.parse(ref.watch(apiClientProvider).baseUrl).origin;
      final store = TimetableMemoStore(
        database: ref.watch(userWorkDatabaseProvider),
        scope: id == null ? null : writingScope(site, id),
      );
      if (id != null) {
        // Blocks also show on the home-screen widgets.
        final bridge = ref.watch(scheduleWidgetBridgeProvider);
        final account = ScheduleWidgetIdentity.accountScopeFor(
          CampusCacheScope(site: site, accountId: id),
        );
        store.addListener((state) {
          // A failed load must not wipe blocks already on the widget.
          if (!state.loaded || (state.error != null && state.blocks.isEmpty)) {
            return;
          }
          unawaited(
            bridge
                .setBlocks(account, [
                  for (final block in state.blocks) block.toWidget(),
                ])
                .catchError((Object _) {}),
          );
        });
      }
      return store;
    });
