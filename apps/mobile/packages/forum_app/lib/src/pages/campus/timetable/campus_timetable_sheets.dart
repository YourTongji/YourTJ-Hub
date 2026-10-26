import 'dart:async';

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart' show HapticFeedback;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../../l10n/app_localizations.dart';
import '../../../campus_widget/schedule_widget_bridge.dart';
import '../../../providers.dart';
import '../../../schedule/schedule_grid.dart';
import '../../settings/schedule_widget_settings_page.dart'
    show confirmScheduleWidgetPrivacy;
import '../campus_calendar_export.dart';
import '../campus_helpers.dart';
import '../campus_state.dart';
import 'timetable_grid.dart';
import 'timetable_memo_store.dart';

// A modal route outlives the campus page. Capture the store that owns its
// draft so replacing the account/site/session cannot expose or submit it.
Future<void> _showMemoSheet(
  BuildContext context, {
  required WidgetBuilder builder,
  bool keyboardAware = false,
}) {
  final store = ProviderScope.containerOf(
    context,
    listen: false,
  ).read(timetableMemoProvider.notifier);
  return showGfBottomSheet<void>(
    context,
    keyboardAware: keyboardAware,
    builder: (_) => _MemoSessionSurface(store: store, builder: builder),
  );
}

class _MemoSessionSurface extends ConsumerWidget {
  const _MemoSessionSurface({required this.store, required this.builder});
  final TimetableMemoStore store;
  final WidgetBuilder builder;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (!identical(store, ref.watch(timetableMemoProvider.notifier))) {
      final route = ModalRoute.of(context);
      final navigator = Navigator.of(context);
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (navigator.mounted && route != null && route.isActive) {
          navigator.removeRoute(route);
        }
      });
      return const SizedBox.shrink();
    }
    return builder(context);
  }
}

Widget _memoError(BuildContext context) => Padding(
  padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
  child: Semantics(
    liveRegion: true,
    child: GfText(
      AppLocalizations.of(context).draftLocalSaveFailed,
      style: GfTheme.typographyOf(
        context,
      ).small.copyWith(color: GfTheme.colorsOf(context).error),
    ),
  ),
);

Widget _fieldLabel(BuildContext context, String text) => Padding(
  padding: const EdgeInsets.only(bottom: 8),
  child: GfText(
    text,
    style: GfTheme.typographyOf(
      context,
    ).small.copyWith(color: GfTheme.colorsOf(context).iconMuted),
  ),
);

// ---------------------------------------------------------------------------
// Week picker

Future<int?> showTimetableWeekPicker(
  BuildContext context, {
  required int week,
  required int currentWeek,
  required int maxWeek,
  required Set<int> busyWeeks,
}) => showGfBottomSheet<int>(
  context,
  builder: (sheet) {
    final l = AppLocalizations.of(sheet);
    final colors = GfTheme.colorsOf(sheet);
    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(20, 0, 20, 24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          GfText(
            l.timetableWeekPicker,
            style: GfTheme.typographyOf(sheet).title3,
          ),
          const SizedBox(height: 16),
          LayoutBuilder(
            builder: (context, constraints) {
              const spacing = 8.0;
              final perRow = constraints.maxWidth >= 480 ? 8 : 6;
              final size =
                  ((constraints.maxWidth - spacing * (perRow - 1)) / perRow)
                      .clamp(44.0, 64.0);
              return Wrap(
                spacing: spacing,
                runSpacing: spacing,
                children: [
                  for (var w = 1; w <= maxWeek; w++)
                    _WeekCell(
                      week: w,
                      size: size,
                      selected: w == week,
                      current: w == currentWeek,
                      busy: busyWeeks.contains(w),
                      colors: colors,
                      onTap: () => Navigator.of(sheet).pop(w),
                    ),
                ],
              );
            },
          ),
        ],
      ),
    );
  },
);

class _WeekCell extends StatelessWidget {
  const _WeekCell({
    required this.week,
    required this.size,
    required this.selected,
    required this.current,
    required this.busy,
    required this.colors,
    required this.onTap,
  });
  final int week;
  final double size;
  final bool selected;
  final bool current;
  final bool busy;
  final GfColors colors;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final foreground = selected
        ? colors.primaryContent
        : busy
        ? colors.baseContent
        : colors.iconMuted;
    return Semantics(
      button: true,
      selected: selected,
      label: [
        l.scheduleWeekN(week),
        if (current) l.scheduleCurrentWeek,
      ].join(' · '),
      excludeSemantics: true,
      child: Material(
        color: selected
            ? colors.primary
            : busy
            ? colors.base200
            : Colors.transparent,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
          side: selected || busy
              ? BorderSide.none
              : BorderSide(color: colors.line),
        ),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onTap,
          excludeFromSemantics: true,
          child: SizedBox.square(
            dimension: size,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                GfText(
                  '$week',
                  style: TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w600,
                    fontFeatures: const [FontFeature.tabularFigures()],
                    color: foreground,
                  ),
                ),
                const SizedBox(height: 3),
                Container(
                  width: 4,
                  height: 4,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: current
                        ? (selected ? colors.primaryContent : colors.primary)
                        : Colors.transparent,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Course details with notes

/// The modal retains only a numeric selection, never a captured school record.
/// Its content follows the same session/background boundary as the campus page.
/// [week] null indexes the unfiltered event list (other-week sessions);
/// [noteWeek] is the teaching week whose memo the sheet edits.
class CampusCourseDetails extends ConsumerWidget {
  const CampusCourseDetails({
    super.key,
    required this.week,
    required this.index,
    this.noteWeek,
    this.times = const [],
  });
  final int? week;
  final int index;
  final int? noteWeek;
  final List<SectionTime> times;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    final state = ref.watch(campusControllerProvider);
    final events = campusCoursesForWeek(
      state.data['timetable']?.events ?? [],
      week,
    );
    final event = index >= 0 && index < events.length ? events[index] : null;
    if (event == null) {
      return Padding(
        padding: const EdgeInsets.all(24),
        child: GfText(l.campusNoData),
      );
    }
    final palette = timetableSlotColors(
      context,
      (courseColorSlotFor(event.name) - 1) % kCourseColorSlotCount,
    );
    final time = event.end <= times.length
        ? '${times[event.start - 1].start}–${times[event.end - 1].end}'
        : '';
    final rows = <(String, String)>[
      (
        'clock',
        [
          timetableWeekdayLabels(l)[event.day - 1],
          l.schedulePeriodRange('${event.start}–${event.end}'),
          time,
        ].where((s) => s.isNotEmpty).join(' · '),
      ),
      (
        'university',
        [event.room, event.campus].where((s) => s.isNotEmpty).join(' · '),
      ),
      ('user-round', event.teacher),
      (
        'calendar-days',
        event.weeks.isEmpty
            ? ''
            : l.scheduleWeeksN(formatWeeksText(event.weeks)),
      ),
      (
        'graduation-cap',
        event.credits.isEmpty ? '' : l.timetableCredits(event.credits),
      ),
    ];
    final noteWeek = this.noteWeek ?? week;
    final memo = ref.watch(timetableMemoProvider);
    final store = ref.read(timetableMemoProvider.notifier);
    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(20, 0, 20, 24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 4,
                height: 24,
                margin: const EdgeInsets.only(top: 2, right: 12),
                decoration: BoxDecoration(
                  color: palette.accent,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
              Expanded(
                child: GfText(
                  event.name,
                  style: GfTheme.typographyOf(context).title2,
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          for (final (symbol, text) in rows)
            if (text.isNotEmpty) _InfoRow(symbol: symbol, text: text),
          const SizedBox(height: 20),
          if (!memo.loaded)
            const Center(child: CircularProgressIndicator())
          else if (!store.canEdit) ...[
            _memoError(context),
            TextButton(
              onPressed: store.retryLoad,
              child: GfText(l.commonRetry),
            ),
          ] else
            _CourseNotes(
              key: ValueKey((
                timetableCourseKey(event.name, event.teacher),
                noteWeek,
              )),
              course: timetableCourseKey(event.name, event.teacher),
              week: noteWeek,
            ),
        ],
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow({required this.symbol, required this.text});
  final String symbol;
  final String text;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 10),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(top: 1, right: 12),
          child: GfSymbol(
            symbol,
            size: 18,
            color: GfTheme.colorsOf(context).iconMuted,
          ),
        ),
        Expanded(
          child: GfText(text, style: GfTheme.typographyOf(context).body),
        ),
      ],
    ),
  );
}

/// Notes save as you type; the last edit is flushed when the sheet closes.
class _CourseNotes extends ConsumerStatefulWidget {
  const _CourseNotes({super.key, required this.course, required this.week});
  final String course;
  final int? week;

  @override
  ConsumerState<_CourseNotes> createState() => _CourseNotesState();
}

class _CourseNotesState extends ConsumerState<_CourseNotes> {
  late final TimetableMemoStore _store;
  late final _standing = TextEditingController(
    text: ref.read(timetableMemoProvider).noteFor(widget.course, null),
  );
  late final _weekly = TextEditingController(
    text: widget.week == null
        ? ''
        : ref.read(timetableMemoProvider).noteFor(widget.course, widget.week),
  );
  Timer? _debounce;
  final _dirty = <int?>{};
  bool _saved = false;
  int _editRevision = 0;

  @override
  void initState() {
    super.initState();
    _store = ref.read(timetableMemoProvider.notifier);
  }

  void _changed(int? week) {
    _dirty.add(week);
    _editRevision++;
    setState(() => _saved = false);
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 500), _flush);
  }

  Future<void> _flush({bool notify = true}) async {
    final revision = _editRevision;
    final edits = {
      for (final week in _dirty)
        week: week == null ? _standing.text : _weekly.text,
    };
    _dirty.clear();
    var saved = true;
    for (final edit in edits.entries) {
      final success = await _store.setNote(widget.course, edit.key, edit.value);
      if (!success) _dirty.add(edit.key);
      saved = success && saved;
    }
    if (edits.isNotEmpty && notify && mounted) {
      // A new keystroke while a write was pending is not saved by that write.
      setState(
        () => _saved = saved && _dirty.isEmpty && revision == _editRevision,
      );
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    unawaited(_flush(notify: false));
    _standing.dispose();
    _weekly.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final error = ref.watch(timetableMemoProvider.select((s) => s.error));
    final week = widget.week;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        GfTextarea(
          controller: _standing,
          labelText: l.timetableCourseNote,
          hintText: l.timetableCourseNoteHint,
          minLines: 2,
          maxLines: 5,
          maxLength: 200,
          onChanged: (_) => _changed(null),
        ),
        if (week != null) ...[
          const SizedBox(height: 12),
          GfTextarea(
            controller: _weekly,
            labelText: l.timetableWeekNote(week),
            hintText: l.timetableWeekNoteHint,
            minLines: 2,
            maxLines: 5,
            maxLength: 200,
            onChanged: (_) => _changed(week),
          ),
        ],
        const SizedBox(height: 4),
        Semantics(
          liveRegion: true,
          child: GfText(
            error != null
                ? l.timetableNotesUnsaved
                : _saved
                ? l.timetableNotesSaved
                : l.timetableNotesLocal,
            style: GfTheme.typographyOf(context).caption.copyWith(
              color: error != null
                  ? GfTheme.colorsOf(context).error
                  : GfTheme.colorsOf(context).iconMuted,
            ),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Block editor

enum _WeekMode { only, every, odd, even, custom }

Future<void> showTimetableBlockEditor(
  BuildContext context, {
  TimetableBlock? block,
  TimetableCellRange? range,
  required int week,
  required int rows,
  required int maxWeek,
  required List<SectionTime> times,
}) => _showMemoSheet(
  context,
  keyboardAware: true,
  builder: (_) => _MemoReady(
    builder: (_) => _BlockEditor(
      block: block,
      range: range,
      week: week,
      rows: rows,
      maxWeek: maxWeek,
      times: times,
    ),
  ),
);

/// Memo write surfaces gate on an editable store: a failed load offers the
/// error and a retry instead of a form whose saves always fail.
class _MemoReady extends ConsumerWidget {
  const _MemoReady({required this.builder});
  final WidgetBuilder builder;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final memo = ref.watch(timetableMemoProvider);
    final store = ref.read(timetableMemoProvider.notifier);
    if (!memo.loaded) return const Center(child: CircularProgressIndicator());
    if (!store.canEdit) {
      final l = AppLocalizations.of(context);
      return Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          _memoError(context),
          TextButton(onPressed: store.retryLoad, child: GfText(l.commonRetry)),
        ],
      );
    }
    return builder(context);
  }
}

class _BlockEditor extends ConsumerStatefulWidget {
  const _BlockEditor({
    required this.block,
    required this.range,
    required this.week,
    required this.rows,
    required this.maxWeek,
    required this.times,
  });
  final TimetableBlock? block;
  final TimetableCellRange? range;
  final int week;
  final int rows;
  final int maxWeek;
  final List<SectionTime> times;

  @override
  ConsumerState<_BlockEditor> createState() => _BlockEditorState();
}

class _BlockEditorState extends ConsumerState<_BlockEditor> {
  late final TimetableBlock? _block = widget.block;
  late final _title = TextEditingController(text: _block?.title ?? '');
  late final _place = TextEditingController(text: _block?.place ?? '');
  late final _note = TextEditingController(text: _block?.note ?? '');
  late Set<int> _days = {
    ...(_block?.days ?? widget.range?.days ?? const <int>[]),
  };
  late int _start = (_block?.start ?? widget.range?.start ?? 1).clamp(
    1,
    widget.rows,
  );
  late int _end = (_block?.end ?? widget.range?.end ?? _start).clamp(
    _start,
    widget.rows,
  );
  late int _color = _block?.color ?? 0;
  late _WeekMode _mode = _modeOf(_block?.weeks);
  late final List<int> _customWeeks = _block?.weeks ?? const [];
  bool _whenOpen = false;
  bool _noteOpen = false;
  bool _saving = false;
  bool _saveFailed = false;
  late final _store = ref.read(timetableMemoProvider.notifier);

  _WeekMode _modeOf(List<int>? weeks) {
    if (weeks == null) return _WeekMode.only;
    if (weeks.isEmpty) return _WeekMode.every;
    if (weeks.length == 1 && weeks.first == widget.week) return _WeekMode.only;
    if (_listEquals(weeks, _parity(1))) return _WeekMode.odd;
    if (_listEquals(weeks, _parity(0))) return _WeekMode.even;
    return _WeekMode.custom;
  }

  static bool _listEquals(List<int> a, List<int> b) =>
      a.length == b.length &&
      [for (var i = 0; i < a.length; i++) a[i] == b[i]].every((e) => e);

  List<int> _parity(int remainder) => [
    for (var w = 1; w <= widget.maxWeek; w++)
      if (w % 2 == remainder) w,
  ];

  List<int> get _weeks => switch (_mode) {
    _WeekMode.only => [widget.week],
    _WeekMode.every => const [],
    _WeekMode.odd => _parity(1),
    _WeekMode.even => _parity(0),
    _WeekMode.custom => _customWeeks,
  };

  bool get _valid => _title.text.trim().isNotEmpty && _days.isNotEmpty;

  /// An untouched block has nothing to save; Save waits for a change.
  bool get _changed {
    final block = _block;
    if (block == null) return true;
    final days = _days.toList()..sort();
    return block.title != _title.text.trim() ||
        block.place != _place.text.trim() ||
        block.note != _note.text.trim() ||
        block.color != _color ||
        block.start != _start ||
        block.end != _end ||
        !_listEquals(block.days, days) ||
        !_listEquals(block.weeks, _weeks);
  }

  Future<void> _save() async {
    if (!_valid || _saving) return;
    final block = TimetableBlock(
      id: _block?.id ?? newTimetableBlockId(),
      title: _title.text.trim(),
      place: _place.text.trim(),
      note: _note.text.trim(),
      color: _color,
      days: _days.toList()..sort(),
      start: _start,
      end: _end,
      weeks: _weeks,
    );
    FocusScope.of(context).unfocus();
    unawaited(HapticFeedback.lightImpact());
    setState(() {
      _saving = true;
      _saveFailed = false;
    });
    final saved = await _store.saveBlock(block);
    if (!mounted) return;
    if (saved) {
      Navigator.of(context).pop();
    } else {
      setState(() {
        _saving = false;
        _saveFailed = true;
      });
    }
  }

  Future<bool> _delete() async {
    final id = _block?.id;
    if (id == null || _saving) return false;
    setState(() {
      _saving = true;
      _saveFailed = false;
    });
    final deleted = await _store.deleteBlock(id);
    if (!mounted) return false;
    if (deleted) {
      Navigator.of(context).pop();
      return true;
    }
    setState(() {
      _saving = false;
      _saveFailed = true;
    });
    return false;
  }

  @override
  void dispose() {
    _title.dispose();
    _place.dispose();
    _note.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final labels = timetableWeekdayLabels(l);
    final time = _blockTimeText(widget.times, _start, _end);
    final weeksLabel = switch (_mode) {
      _WeekMode.only => l.timetableWeeksOnly(widget.week),
      _WeekMode.every => l.timetableWeeksEvery,
      _WeekMode.odd => l.timetableWeeksOdd,
      _WeekMode.even => l.timetableWeeksEven,
      _WeekMode.custom => _blockWeeksText(l, _customWeeks),
    };
    final whenOpen = _whenOpen || _days.isEmpty;
    final noteOpen = _noteOpen || _note.text.isNotEmpty;

    return AbsorbPointer(
      absorbing: _saving,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Flexible(
            child: SingleChildScrollView(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 8),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: GfText(
                          _block == null
                              ? l.timetableAddBlock
                              : l.timetableEditBlock,
                          style: type.title3,
                        ),
                      ),
                      if (_block != null) _DeleteAction(onDelete: _delete),
                    ],
                  ),
                  SizedBox(height: _block == null ? 16 : 8),
                  GfInput(
                    controller: _title,
                    labelText: l.timetableBlockTitle,
                    hintText: l.timetableBlockTitleHint,
                    autofocus: _block == null,
                    maxLength: 40,
                    textInputAction: TextInputAction.next,
                    onChanged: (_) => setState(() {}),
                  ),
                  const SizedBox(height: 4),
                  GfInput(
                    controller: _place,
                    labelText: l.timetableBlockPlace,
                    hintText: l.timetableBlockPlaceHint,
                    maxLength: 40,
                    textInputAction: TextInputAction.done,
                    onChanged: (_) => setState(() {}),
                  ),
                  const SizedBox(height: 8),
                  // When: one summary line; the controls open on request.
                  _WhenRow(
                    label: l.timetableBlockWhen,
                    days: _blockDaysText(l, _days),
                    time: time.isEmpty
                        ? l.schedulePeriodRange('$_start–$_end')
                        : time,
                    weeks: weeksLabel,
                    open: whenOpen,
                    onTap: _days.isEmpty
                        ? null
                        : () => setState(() => _whenOpen = !_whenOpen),
                  ),
                  AnimatedSize(
                    duration: GfMotion.duration(context, GfMotion.layout),
                    curve: GfMotion.layoutCurve,
                    alignment: Alignment.topCenter,
                    child: !whenOpen
                        ? const SizedBox(width: double.infinity)
                        : Padding(
                            padding: const EdgeInsets.only(top: 16),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.stretch,
                              children: [
                                _fieldLabel(context, l.timetableBlockDays),
                                // One row of seven: the week reads as a week.
                                Row(
                                  children: [
                                    for (var day = 1; day <= 7; day++) ...[
                                      if (day > 1) const SizedBox(width: 6),
                                      Expanded(
                                        child: _Toggle(
                                          label: labels[day - 1],
                                          selected: _days.contains(day),
                                          onTap: () => setState(() {
                                            HapticFeedback.selectionClick();
                                            _days = {..._days};
                                            if (!_days.remove(day)) {
                                              _days.add(day);
                                            }
                                          }),
                                        ),
                                      ),
                                    ],
                                  ],
                                ),
                                const SizedBox(height: 16),
                                _fieldLabel(context, l.schedulePeriods),
                                // Each stepper shows the clock time it lands on.
                                Row(
                                  children: [
                                    Expanded(
                                      child: _Stepper(
                                        value: _start,
                                        min: 1,
                                        max: widget.rows,
                                        label: l.timetableBlockStart,
                                        caption: _start <= widget.times.length
                                            ? widget.times[_start - 1].start
                                            : null,
                                        onChanged: (v) => setState(() {
                                          _start = v;
                                          if (_end < v) _end = v;
                                        }),
                                      ),
                                    ),
                                    Padding(
                                      padding: const EdgeInsets.symmetric(
                                        horizontal: 6,
                                      ),
                                      child: GfText(
                                        '–',
                                        style: type.body.copyWith(
                                          color: colors.iconMuted,
                                        ),
                                      ),
                                    ),
                                    Expanded(
                                      child: _Stepper(
                                        value: _end,
                                        min: _start,
                                        max: widget.rows,
                                        label: l.timetableBlockEnd,
                                        caption: _end <= widget.times.length
                                            ? widget.times[_end - 1].end
                                            : null,
                                        onChanged: (v) =>
                                            setState(() => _end = v),
                                      ),
                                    ),
                                  ],
                                ),
                                const SizedBox(height: 16),
                                _fieldLabel(context, l.scheduleWeeksLabel),
                                GfSegmented<_WeekMode>(
                                  segments: [
                                    (
                                      l.timetableWeeksOnly(widget.week),
                                      _WeekMode.only,
                                    ),
                                    (l.timetableWeeksEvery, _WeekMode.every),
                                    (l.timetableWeeksOdd, _WeekMode.odd),
                                    (l.timetableWeeksEven, _WeekMode.even),
                                    if (_customWeeks.isNotEmpty &&
                                        _modeOf(_customWeeks) ==
                                            _WeekMode.custom)
                                      (
                                        _blockWeeksText(l, _customWeeks),
                                        _WeekMode.custom,
                                      ),
                                  ],
                                  selected: _mode,
                                  onSelected: (mode) =>
                                      setState(() => _mode = mode),
                                ),
                              ],
                            ),
                          ),
                  ),
                  const SizedBox(height: 20),
                  _fieldLabel(context, l.timetableBlockColor),
                  Wrap(
                    spacing: 10,
                    runSpacing: 10,
                    children: [
                      for (var slot = 0; slot < kCourseColorSlotCount; slot++)
                        _Swatch(
                          slot: slot,
                          selected: slot == _color,
                          label: l.timetableColorN(slot + 1),
                          onTap: () => setState(() => _color = slot),
                        ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  if (noteOpen)
                    GfTextarea(
                      controller: _note,
                      labelText: l.timetableBlockNote,
                      autofocus: _noteOpen && _note.text.isEmpty,
                      minLines: 2,
                      maxLines: 4,
                      maxLength: 200,
                      onChanged: (_) => setState(() {}),
                    )
                  else
                    Align(
                      alignment: AlignmentDirectional.centerStart,
                      child: TextButton.icon(
                        onPressed: () => setState(() => _noteOpen = true),
                        icon: GfSymbol('plus', size: 18, color: colors.primary),
                        label: GfText(l.timetableAddNote),
                      ),
                    ),
                ],
              ),
            ),
          ),
          if (_saveFailed) _memoError(context),
          // Pinned so the keyboard or open sections never hide the choice:
          // leave without changes, or keep them.
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 8, 20, 16),
            child: Row(
              children: [
                Expanded(
                  child: GfButton(
                    label: l.commonCancel,
                    variant: GfButtonVariant.secondary,
                    expanded: true,
                    onPressed: _saving
                        ? null
                        : () => Navigator.of(context).pop(),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: GfButton(
                    label: l.timetableSaveBlock,
                    expanded: true,
                    loading: _saving,
                    onPressed: _valid && _changed && !_saving ? _save : null,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// Delete asks twice in place: the first tap arms it, a second confirms, and
/// it settles back on its own if the user moves on.
class _DeleteAction extends StatefulWidget {
  const _DeleteAction({required this.onDelete});
  final Future<bool> Function() onDelete;

  @override
  State<_DeleteAction> createState() => _DeleteActionState();
}

class _DeleteActionState extends State<_DeleteAction> {
  bool _armed = false;
  Timer? _disarm;

  void _tap() {
    _disarm?.cancel();
    if (_armed) {
      unawaited(_confirm());
      return;
    }
    unawaited(HapticFeedback.selectionClick());
    setState(() => _armed = true);
    _disarm = Timer(const Duration(seconds: 3), () {
      if (mounted) setState(() => _armed = false);
    });
  }

  Future<void> _confirm() async {
    unawaited(HapticFeedback.mediumImpact());
    final deleted = await widget.onDelete();
    if (!mounted || deleted) return;
    // The delete failed: ask again instead of staying armed.
    setState(() => _armed = false);
  }

  @override
  void dispose() {
    _disarm?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final duration = GfMotion.duration(context, GfMotion.selection);
    return AnimatedSize(
      duration: duration,
      curve: GfMotion.layoutCurve,
      alignment: AlignmentDirectional.centerEnd,
      child: AnimatedSwitcher(
        duration: duration,
        switchInCurve: GfMotion.enterCurve,
        switchOutCurve: GfMotion.exitCurve,
        transitionBuilder: (child, animation) => FadeTransition(
          opacity: animation,
          child: ScaleTransition(
            scale: Tween(begin: .96, end: 1.0).animate(animation),
            child: child,
          ),
        ),
        child: _armed
            ? TextButton.icon(
                key: const ValueKey('armed'),
                onPressed: _tap,
                style: TextButton.styleFrom(
                  foregroundColor: colors.error,
                  backgroundColor: colors.error.withValues(alpha: .1),
                  minimumSize: const Size(44, 40),
                  shape: const StadiumBorder(),
                ),
                icon: GfSymbol('trash-2', size: 16, color: colors.error),
                label: GfText(l.timetableConfirmDelete),
              )
            : GfIconButton(
                key: const ValueKey('idle'),
                symbol: 'trash-2',
                iconSize: 19,
                tooltip: l.timetableDeleteBlock,
                color: colors.iconMuted,
                onPressed: _tap,
              ),
      ),
    );
  }
}

String _blockWeeksText(AppLocalizations l, List<int> weeks) {
  if (weeks.isEmpty) return l.timetableWeeksEvery;
  final sorted = weeks.toSet().toList()..sort();
  if (sorted.length == 1) return l.scheduleWeekN(sorted.first);
  final alternating =
      sorted.length > 2 &&
      sorted.first <= 2 &&
      [
        for (var i = 1; i < sorted.length; i++) sorted[i] - sorted[i - 1],
      ].every((gap) => gap == 2);
  if (alternating) {
    return sorted.first == 1 ? l.timetableWeeksOdd : l.timetableWeeksEven;
  }
  return l.timetableWeeksRange(formatWeeksText(sorted));
}

/// Days as one short phrase; three or more in a row collapse ("周一–周五").
String _blockDaysText(AppLocalizations l, Iterable<int> days) {
  final labels = timetableWeekdayLabels(l);
  final sorted = days.toSet().toList()..sort();
  final parts = <String>[];
  for (var i = 0; i < sorted.length;) {
    var j = i;
    while (j + 1 < sorted.length && sorted[j + 1] == sorted[j] + 1) {
      j++;
    }
    if (j - i >= 2) {
      parts.add('${labels[sorted[i] - 1]}–${labels[sorted[j] - 1]}');
    } else {
      for (var k = i; k <= j; k++) {
        parts.add(labels[sorted[k] - 1]);
      }
    }
    i = j + 1;
  }
  return parts.join(l.timetableListSeparator);
}

String _blockTimeText(List<SectionTime> times, int start, int end) =>
    end <= times.length
    ? '${times[start - 1].start}–${times[end - 1].end}'
    : '';

/// Collapsed "when" line of the block editor: day and time lead in one line,
/// the week rule trails in a quieter tone.
class _WhenRow extends StatelessWidget {
  const _WhenRow({
    required this.label,
    required this.days,
    required this.time,
    required this.weeks,
    required this.open,
    required this.onTap,
  });
  final String label;
  final String days;
  final String time;
  final String weeks;
  final bool open;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final radius = BorderRadius.circular(GfTheme.radiiOf(context).box);
    return Semantics(
      button: onTap != null,
      expanded: open,
      label: label,
      child: Material(
        color: colors.base200,
        borderRadius: radius,
        child: InkWell(
          onTap: onTap,
          borderRadius: radius,
          child: ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 52),
            child: Padding(
              padding: const EdgeInsets.fromLTRB(14, 8, 10, 8),
              child: Row(
                children: [
                  GfSymbol('clock', size: 18, color: colors.iconMuted),
                  const SizedBox(width: 12),
                  Expanded(
                    child: GfText.rich(
                      TextSpan(
                        children: [
                          if (days.isNotEmpty) TextSpan(text: '$days '),
                          TextSpan(text: time),
                          TextSpan(
                            text: '  ·  $weeks',
                            style: type.small.copyWith(color: colors.iconMuted),
                          ),
                        ],
                      ),
                      style: type.bodyStrong.copyWith(
                        fontFeatures: const [FontFeature.tabularFigures()],
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  if (onTap != null)
                    AnimatedRotation(
                      turns: open ? .5 : 0,
                      duration: GfMotion.duration(context, GfMotion.selection),
                      child: GfSymbol(
                        'chevron-down',
                        size: 18,
                        color: colors.iconMuted,
                      ),
                    ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Block details: a read view like course details, with editing one tap away

Future<void> showTimetableBlockDetails(
  BuildContext context, {
  required TimetableBlock block,
  required List<SectionTime> times,
  required VoidCallback onEdit,
}) => _showMemoSheet(
  context,
  builder: (sheet) => _BlockDetails(
    block: block,
    times: times,
    onEdit: () {
      Navigator.of(sheet).pop();
      onEdit();
    },
  ),
);

class _BlockDetails extends ConsumerWidget {
  const _BlockDetails({
    required this.block,
    required this.times,
    required this.onEdit,
  });
  final TimetableBlock block;
  final List<SectionTime> times;
  final VoidCallback onEdit;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    // Show the saved copy so an edit elsewhere is reflected here.
    final current =
        ref
            .watch(timetableMemoProvider)
            .blocks
            .where((b) => b.id == block.id)
            .firstOrNull ??
        block;
    final time = _blockTimeText(times, current.start, current.end);
    final rows = [
      (
        'clock',
        [
          [
            _blockDaysText(l, current.days),
            time,
          ].where((s) => s.isNotEmpty).join(' '),
          l.schedulePeriodRange('${current.start}–${current.end}'),
        ].join(' · '),
      ),
      ('university', current.place),
      ('calendar-days', _blockWeeksText(l, current.weeks)),
      ('pen-line', current.note),
    ];
    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 4,
                height: 24,
                margin: const EdgeInsetsDirectional.only(top: 2, end: 12),
                decoration: BoxDecoration(
                  color: timetableSlotColors(context, current.color).accent,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    GfText(current.title, style: type.title2),
                    const SizedBox(height: 2),
                    GfText(
                      l.timetableBlock,
                      style: type.small.copyWith(color: colors.iconMuted),
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          for (final (symbol, text) in rows)
            if (text.isNotEmpty) _InfoRow(symbol: symbol, text: text),
          const SizedBox(height: 16),
          GfButton(
            label: l.timetableEditBlock,
            variant: GfButtonVariant.secondary,
            expanded: true,
            icon: GfSymbol('square-pen', size: 18, color: colors.baseContent),
            onPressed: onEdit,
          ),
        ],
      ),
    );
  }
}

class _Toggle extends StatelessWidget {
  const _Toggle({
    required this.label,
    required this.selected,
    required this.onTap,
  });
  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    return Semantics(
      button: true,
      toggled: selected,
      label: label,
      excludeSemantics: true,
      child: Material(
        color: selected ? colors.primary.withValues(alpha: .1) : colors.base200,
        shape: StadiumBorder(
          side: BorderSide(
            color: selected ? colors.primary : Colors.transparent,
          ),
        ),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onTap,
          excludeFromSemantics: true,
          child: ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 44),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 4),
              child: Center(
                child: FittedBox(
                  fit: BoxFit.scaleDown,
                  child: GfText(
                    label,
                    maxLines: 1,
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: selected ? FontWeight.w600 : FontWeight.w500,
                      color: selected ? colors.primary : colors.baseContent,
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _Stepper extends StatelessWidget {
  const _Stepper({
    required this.value,
    required this.min,
    required this.max,
    required this.label,
    required this.onChanged,
    this.caption,
  });
  final int value;
  final int min;
  final int max;
  final String label;

  /// Clock time of the chosen section, shown beneath it.
  final String? caption;
  final ValueChanged<int> onChanged;

  void _step(int delta) {
    HapticFeedback.selectionClick();
    onChanged(value + delta);
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final l = AppLocalizations.of(context);
    const tabular = [FontFeature.tabularFigures()];
    return Semantics(
      label: label,
      value: [l.schedulePeriodRange('$value'), ?caption].join(' '),
      increasedValue: value < max
          ? l.schedulePeriodRange('${value + 1}')
          : null,
      decreasedValue: value > min
          ? l.schedulePeriodRange('${value - 1}')
          : null,
      onIncrease: value < max ? () => _step(1) : null,
      onDecrease: value > min ? () => _step(-1) : null,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: colors.base200,
          borderRadius: BorderRadius.circular(22),
        ),
        child: Row(
          children: [
            ExcludeSemantics(
              child: GfIconButton(
                symbol: 'minus',
                iconSize: 16,
                onPressed: value > min ? () => _step(-1) : null,
              ),
            ),
            Expanded(
              child: ExcludeSemantics(
                child: FittedBox(
                  fit: BoxFit.scaleDown,
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      GfText(
                        l.schedulePeriodRange('$value'),
                        maxLines: 1,
                        style: type.bodyStrong.copyWith(
                          height: 1.2,
                          fontFeatures: tabular,
                        ),
                      ),
                      if (caption case final String caption)
                        GfText(
                          caption,
                          maxLines: 1,
                          style: type.caption.copyWith(
                            height: 1.2,
                            color: colors.iconMuted,
                            fontFeatures: tabular,
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ),
            ExcludeSemantics(
              child: GfIconButton(
                symbol: 'plus',
                iconSize: 16,
                onPressed: value < max ? () => _step(1) : null,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _Swatch extends StatelessWidget {
  const _Swatch({
    required this.slot,
    required this.selected,
    required this.label,
    required this.onTap,
  });
  final int slot;
  final bool selected;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final palette = timetableSlotColors(context, slot);
    return Semantics(
      button: true,
      selected: selected,
      label: label,
      excludeSemantics: true,
      child: InkResponse(
        onTap: onTap,
        excludeFromSemantics: true,
        radius: 24,
        child: SizedBox.square(
          dimension: 44,
          child: Center(
            child: AnimatedContainer(
              duration: GfMotion.duration(context, GfMotion.selection),
              width: 32,
              height: 32,
              padding: const EdgeInsets.all(3),
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                border: Border.all(
                  color: selected ? colors.baseContent : Colors.transparent,
                  width: 2,
                ),
              ),
              child: DecoratedBox(
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: palette.fill,
                  border: Border.all(
                    color: palette.accent.withValues(alpha: .5),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Options: display, memos, widgets and calendar export

Future<void> showTimetableOptions(
  BuildContext context, {
  required bool exportEnabled,
  required VoidCallback onAddBlock,
  required ValueChanged<TimetableBlock> onEditBlock,
}) => _showMemoSheet(
  context,
  builder: (_) => _TimetableOptions(
    exportEnabled: exportEnabled,
    onAddBlock: onAddBlock,
    onEditBlock: onEditBlock,
  ),
);

/// Closes the options sheet and anything stacked on it, then runs [then] so
/// its result lands on the visible timetable.
void _closeSheets(BuildContext context, VoidCallback then) {
  Navigator.of(
    context,
    rootNavigator: true,
  ).popUntil((route) => route is! PopupRoute);
  then();
}

/// Sheet heading with an optional muted lead line, inset to the row edge.
Widget _sheetHeading(BuildContext context, String title, [String? lead]) {
  final type = GfTheme.typographyOf(context);
  return Padding(
    padding: const EdgeInsets.fromLTRB(20, 0, 20, 16),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        GfText(title, style: type.title2),
        if (lead != null) ...[
          const SizedBox(height: 6),
          GfText(
            lead,
            style: type.small.copyWith(
              color: GfTheme.colorsOf(context).iconMuted,
            ),
          ),
        ],
      ],
    ),
  );
}

Widget _groupTitle(BuildContext context, String text) => Padding(
  padding: const EdgeInsets.fromLTRB(20, 16, 20, 4),
  child: GfText(text, style: GfTheme.typographyOf(context).heading),
);

/// Display switches up front; blocks, widgets and the calendar each open
/// their own sheet so this one stays short.
class _TimetableOptions extends ConsumerWidget {
  const _TimetableOptions({
    required this.exportEnabled,
    required this.onAddBlock,
    required this.onEditBlock,
  });
  final bool exportEnabled;
  final VoidCallback onAddBlock;
  final ValueChanged<TimetableBlock> onEditBlock;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    final memo = ref.watch(timetableMemoProvider);
    final store = ref.read(timetableMemoProvider.notifier);
    final prefs = memo.prefs;
    if (!memo.loaded) return const Center(child: CircularProgressIndicator());
    if (!store.canEdit) {
      return Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          _memoError(context),
          TextButton(onPressed: store.retryLoad, child: GfText(l.commonRetry)),
        ],
      );
    }
    Widget toggle(
      String title,
      bool value,
      TimetablePrefs Function(TimetablePrefs, bool) next, {
      String? hint,
    }) => GfSwitchRow(
      title: title,
      description: hint,
      value: value,
      onChanged: (v) => store.updatePrefs((prefs) => next(prefs, v)),
    );

    return ListView(
      shrinkWrap: true,
      padding: const EdgeInsets.only(bottom: 16),
      children: [
        _sheetHeading(context, l.timetableOptions, l.timetableGestureTip),
        if (memo.error != null) _memoError(context),
        _groupTitle(context, l.timetableDisplay),
        toggle(
          l.timetableAlwaysShowWeekend,
          prefs.alwaysShowWeekend,
          (prefs, v) => prefs.copyWith(alwaysShowWeekend: v),
          hint: l.timetableAlwaysShowWeekendHint,
        ),
        toggle(
          l.timetableShowOtherWeeks,
          prefs.showOtherWeeks,
          (prefs, v) => prefs.copyWith(showOtherWeeks: v),
          hint: l.timetableShowOtherWeeksHint,
        ),
        toggle(
          l.timetableShowSectionTimes,
          prefs.showSectionTimes,
          (prefs, v) => prefs.copyWith(showSectionTimes: v),
        ),
        toggle(
          l.timetableFocusToday,
          prefs.focusToday,
          (prefs, v) => prefs.copyWith(focusToday: v),
        ),
        _groupTitle(context, l.timetableMore),
        GfSettingRow(
          symbol: 'square-pen',
          title: l.timetableMemos,
          description: memo.blocks.isEmpty && memo.notes.isEmpty
              ? l.timetableMemosNone
              : l.timetableMemoSummary(memo.blocks.length, memo.notes.length),
          onTap: () => _showMemoSheet(
            context,
            builder: (_) =>
                _MemoSheet(onAddBlock: onAddBlock, onEditBlock: onEditBlock),
          ),
        ),
        GfSettingRow(
          symbol: 'layout-grid',
          title: l.timetableWidgets,
          description: l.timetableWidgetsHint,
          onTap: () => showGfBottomSheet<void>(
            context,
            builder: (_) => const _WidgetSheet(),
          ),
        ),
        GfSettingRow(
          symbol: 'calendar-days',
          title: l.campusExportCalendar,
          description: l.timetableExportHint,
          onTap: () => showGfBottomSheet<void>(
            context,
            builder: (_) => ListView(
              shrinkWrap: true,
              padding: const EdgeInsets.only(bottom: 24),
              children: [
                _sheetHeading(context, l.timetableCalendarSection),
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 20),
                  child: CampusCalendarExportButton(enabled: exportEnabled),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}

/// Every block and note in one list; the add action stays pinned below it.
class _MemoSheet extends ConsumerWidget {
  const _MemoSheet({required this.onAddBlock, required this.onEditBlock});
  final VoidCallback onAddBlock;
  final ValueChanged<TimetableBlock> onEditBlock;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final memo = ref.watch(timetableMemoProvider);
    final store = ref.read(timetableMemoProvider.notifier);
    final empty = memo.blocks.isEmpty && memo.notes.isEmpty;

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Flexible(
          child: ListView(
            shrinkWrap: true,
            padding: EdgeInsets.zero,
            children: [
              _sheetHeading(context, l.timetableMemos, l.timetableMemosIntro),
              if (memo.error != null) _memoError(context),
              if (empty)
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 8, 20, 8),
                  child: GfText(
                    l.timetableNoMemos,
                    style: type.body.copyWith(color: colors.iconMuted),
                  ),
                ),
              if (memo.blocks.isNotEmpty) ...[
                _groupTitle(context, l.timetableBlocksHeading),
                for (final block in memo.blocks)
                  _MemoRow(
                    color: timetableSlotColors(context, block.color).accent,
                    title: block.title,
                    subtitle: [
                      _blockDaysText(l, block.days),
                      l.schedulePeriodRange('${block.start}–${block.end}'),
                      _blockWeeksText(l, block.weeks),
                    ].join(' · '),
                    onTap: () => onEditBlock(block),
                  ),
              ],
              if (memo.notes.isNotEmpty) ...[
                _groupTitle(context, l.timetableNotesHeading),
                for (final note in memo.notes)
                  _MemoRow(
                    symbol: 'pen-line',
                    title: note.course.split('\u001f').first,
                    subtitle:
                        '${note.week == null ? l.timetableWeeksEvery : l.scheduleWeekN(note.week!)} · ${note.text}',
                    trailing: GfIconButton(
                      symbol: 'trash-2',
                      tooltip: l.timetableDeleteNote,
                      onPressed: () =>
                          store.setNote(note.course, note.week, ''),
                    ),
                  ),
              ],
            ],
          ),
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 16),
          child: GfButton(
            label: l.timetableAddBlock,
            expanded: true,
            icon: GfSymbol('plus', size: 18, color: colors.primaryContent),
            onPressed: () => _closeSheets(context, onAddBlock),
          ),
        ),
      ],
    );
  }
}

/// Pins a home-screen widget where the launcher allows it; otherwise shows
/// the platform's manual steps.
class _WidgetSheet extends ConsumerStatefulWidget {
  const _WidgetSheet();

  @override
  ConsumerState<_WidgetSheet> createState() => _WidgetSheetState();
}

class _WidgetSheetState extends ConsumerState<_WidgetSheet> {
  late final Future<bool> _canPin = ref
      .read(scheduleWidgetBridgeProvider)
      .canRequestPin();

  Future<void> _pin(ScheduleWidgetKind kind) async {
    final l = AppLocalizations.of(context);
    if (!await confirmScheduleWidgetPrivacy(context) || !mounted) return;
    try {
      await ref.read(scheduleWidgetBridgeProvider).requestPin(kind);
      if (mounted) showGfToast(context, l.timetableWidgetRequested);
    } catch (_) {
      if (mounted) showGfToast(context, l.timetableWidgetManual, error: true);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final iOS = Theme.of(context).platform == TargetPlatform.iOS;
    final router = GoRouter.maybeOf(context);
    Widget steps(String text) => Padding(
      padding: const EdgeInsets.fromLTRB(20, 0, 20, 8),
      child: GfText(text, style: type.body.copyWith(color: colors.iconMuted)),
    );
    Widget widgetRow(ScheduleWidgetKind kind, String title, String hint) =>
        GfSettingRow(
          title: title,
          description: hint,
          trailing: Semantics(
            label: l.timetableWidgetAddNamed(title),
            button: true,
            excludeSemantics: true,
            onTap: () => _pin(kind),
            child: GfButton(
              label: l.timetableWidgetAdd,
              size: GfButtonSize.small,
              variant: GfButtonVariant.secondary,
              icon: GfSymbol('plus', size: 16, color: colors.baseContent),
              onPressed: () => _pin(kind),
            ),
          ),
        );

    return ListView(
      shrinkWrap: true,
      padding: const EdgeInsets.only(bottom: 16),
      children: [
        _sheetHeading(context, l.timetableWidgets, l.timetableWidgetsHint),
        FutureBuilder<bool>(
          future: _canPin,
          builder: (context, snapshot) {
            if (iOS) return steps(l.timetableWidgetIosSteps);
            if (snapshot.data != true) return steps(l.timetableWidgetManual);
            return Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                widgetRow(
                  ScheduleWidgetKind.nextClass,
                  l.timetableWidgetNext,
                  l.timetableWidgetNextHint,
                ),
                widgetRow(
                  ScheduleWidgetKind.today,
                  l.timetableWidgetToday,
                  l.timetableWidgetTodayHint,
                ),
                widgetRow(
                  ScheduleWidgetKind.timeline,
                  l.timetableWidgetTimeline,
                  l.timetableWidgetTimelineHint,
                ),
              ],
            );
          },
        ),
        if (router != null) ...[
          const SizedBox(height: 8),
          GfSettingRow(
            symbol: 'settings',
            title: l.scheduleWidgetSettingsTitle,
            onTap: () =>
                _closeSheets(context, () => router.push('/settings/widgets')),
          ),
        ],
      ],
    );
  }
}

class _MemoRow extends StatelessWidget {
  const _MemoRow({
    required this.title,
    required this.subtitle,
    this.color,
    this.symbol,
    this.trailing,
    this.onTap,
  });
  final String title;
  final String subtitle;
  final Color? color;
  final String? symbol;
  final Widget? trailing;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(10),
      child: ConstrainedBox(
        constraints: const BoxConstraints(minHeight: 56),
        child: Row(
          children: [
            SizedBox(
              width: 28,
              child: symbol != null
                  ? GfSymbol(symbol!, size: 18, color: colors.iconMuted)
                  : Center(
                      child: Container(
                        width: 10,
                        height: 10,
                        decoration: BoxDecoration(
                          color: color,
                          borderRadius: BorderRadius.circular(3),
                        ),
                      ),
                    ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    GfText(
                      title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: type.bodyStrong,
                    ),
                    GfText(
                      subtitle,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: type.small.copyWith(color: colors.iconMuted),
                    ),
                  ],
                ),
              ),
            ),
            ?trailing,
            if (trailing == null && onTap != null)
              GfSymbol('chevron-right', size: 18, color: colors.iconMuted),
          ],
        ),
      ),
    );
  }
}
