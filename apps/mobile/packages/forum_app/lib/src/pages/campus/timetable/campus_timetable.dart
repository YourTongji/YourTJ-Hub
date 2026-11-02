import 'dart:math' as math;

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart' show CustomSemanticsAction;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../../l10n/app_localizations.dart';
import '../campus_helpers.dart';
import '../campus_private_surface.dart';
import 'campus_timetable_sheets.dart';
import 'timetable_grid.dart';
import 'timetable_layout.dart';
import 'timetable_memo_store.dart';

/// Freshness line folded into the timetable header.
class CampusTimetableStatus {
  const CampusTimetableStatus(this.text, {this.detail, this.warning = false});

  /// One-line summary shown beside the dates.
  final String text;

  /// Full wording (snapshot time and any warning), shown on tap.
  final String? detail;
  final bool warning;
}

/// Header height for a given text scale; the grid takes the rest.
double campusTimetableHeaderHeight(double scale) => (44 + 26) * scale + 8;

/// Content wider than this shows the focused day's agenda beside the grid.
const double kTimetableAgendaBreakpoint = 720;

/// The campus week: a compact header (week switcher, freshness, options) and
/// a grid that fits the given [height]. Secondary tools live in the options
/// sheet so the timetable keeps the screen.
class CampusWeekTimetable extends ConsumerStatefulWidget {
  const CampusWeekTimetable({
    super.key,
    required this.data,
    required this.week,
    required this.maxWeek,
    required this.times,
    this.selectedWeek,
    this.onWeekChanged,
    this.height,
    this.status,
    this.refreshing = false,
    this.onRefresh,
  });
  final CampusDataset data;
  final int week;
  final int maxWeek;
  final List<SectionTime> times;
  final int? selectedWeek;
  final ValueChanged<int>? onWeekChanged;

  /// Height available to header and grid. Null keeps comfortable rows.
  final double? height;
  final CampusTimetableStatus? status;
  final bool refreshing;
  final VoidCallback? onRefresh;

  @override
  ConsumerState<CampusWeekTimetable> createState() =>
      _CampusWeekTimetableState();
}

class _CampusWeekTimetableState extends ConsumerState<CampusWeekTimetable> {
  late int week = (widget.selectedWeek ?? widget.week).clamp(1, widget.maxWeek);
  int? _focusedDay;
  bool _appliedFocusPref = false;

  void _selectWeek(int value) {
    setState(() => week = value.clamp(1, widget.maxWeek));
    widget.onWeekChanged?.call(week);
  }

  @override
  void didUpdateWidget(CampusWeekTimetable oldWidget) {
    super.didUpdateWidget(oldWidget);
    week = (widget.selectedWeek ?? week).clamp(1, widget.maxWeek);
  }

  int get _today => campusNow(DateTime.now()).weekday;

  Future<void> _pickWeek() async {
    final busy = <int>{
      for (final e in widget.data.events)
        ...(e.weeks.isEmpty
            ? [for (var w = 1; w <= widget.maxWeek; w++) w]
            : e.weeks),
      // Empty weeks repeat every week, like a course with no week filter.
      for (final b in ref.read(timetableMemoProvider).blocks)
        ...(b.weeks.isEmpty
            ? [for (var w = 1; w <= widget.maxWeek; w++) w]
            : b.weeks),
    };
    final picked = await showTimetableWeekPicker(
      context,
      week: week,
      currentWeek: widget.week,
      maxWeek: widget.maxWeek,
      busyWeeks: busy,
    );
    if (picked != null && mounted) _selectWeek(picked);
  }

  void _openItem(TimetableItem item, List<SectionTime> times, int rows) {
    switch (item.kind) {
      case TimetableItemKind.block:
        // Read first, like a course; editing is one deliberate tap away.
        showTimetableBlockDetails(
          context,
          block: item.block!,
          times: times,
          onEdit: () => showTimetableBlockEditor(
            context,
            block: item.block,
            week: week,
            rows: rows,
            maxWeek: widget.maxWeek,
            times: times,
          ),
        );
      case TimetableItemKind.course || TimetableItemKind.otherWeek:
        final index = item.eventIndex;
        if (index == null) return;
        final shownWeek = week;
        final ghost = item.kind == TimetableItemKind.otherWeek;
        showGfBottomSheet<void>(
          context,
          keyboardAware: true,
          builder: (_) => CampusPrivateSurface(
            builder: (_) => CampusCourseDetails(
              week: ghost ? null : shownWeek,
              index: index,
              noteWeek: ghost ? null : shownWeek,
              times: times,
            ),
          ),
        );
    }
  }

  /// The focused day (or today) at its first free pair of sections.
  TimetableCellRange _defaultRange(TimetableWeekLayout layout) {
    var day =
        _focusedDay ??
        (week == widget.week && layout.days.contains(_today)
            ? _today
            : layout.days.first);
    // The focused weekend may have folded away with its last block.
    if (!layout.days.contains(day)) day = layout.days.first;
    for (var s = 1; s < layout.rows; s++) {
      if (layout.isFree(day, s) && layout.isFree(day, s + 1)) {
        return TimetableCellRange(days: [day], start: s, end: s + 1);
      }
    }
    return TimetableCellRange(days: [day], start: 1, end: 2);
  }

  Future<void> _create(
    TimetableCellRange range,
    List<SectionTime> times,
    int rows,
  ) => showTimetableBlockEditor(
    context,
    range: range,
    week: week,
    rows: rows,
    maxWeek: widget.maxWeek,
    times: times,
  );

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final memo = ref.watch(timetableMemoProvider);
    final isCurrent = week == widget.week;
    if (!_appliedFocusPref && memo.loaded && isCurrent) {
      _appliedFocusPref = true;
      if (memo.prefs.focusToday) _focusedDay = _today;
    }
    final layout = buildTimetableWeek(
      events: widget.data.events,
      week: week,
      blocks: memo.blocks,
      prefs: memo.prefs,
    );
    final times = sectionTimesFor(
      layout.rows,
      widget.times.isEmpty ? null : widget.times,
    );
    final now = DateTime.now();
    final dates = timetableWeekDates(week, widget.week, now);
    final focused = layout.days.contains(_focusedDay) ? _focusedDay : null;
    final scale = timetableTextScale(context);
    final headerHeight = campusTimetableHeaderHeight(scale);
    final hasItems = layout.clusters.any((day) => day.isNotEmpty);
    List<String> notesOf(TimetableItem item) => switch (item.kind) {
      TimetableItemKind.block => [
        if (item.block!.note.isNotEmpty) item.block!.note,
      ],
      TimetableItemKind.course => memo.notesIn(item.courseKey, week),
      TimetableItemKind.otherWeek => const [],
    };

    return LayoutBuilder(
      builder: (context, constraints) {
        final wide = constraints.maxWidth >= kTimetableAgendaBreakpoint;
        final gridHeight = widget.height == null
            ? null
            : math.max(0.0, widget.height! - headerHeight);
        final grid = TimetableGrid(
          layout: layout,
          times: times,
          week: week,
          height: gridHeight,
          dates: dates,
          today: isCurrent ? _today : null,
          nowMinute: isCurrent ? timetableMinuteOfDay(now) : null,
          focusedDay: focused,
          expandFocus: !wide,
          showSectionTimes: memo.prefs.showSectionTimes,
          notesOf: notesOf,
          refreshing: widget.refreshing,
          emptyLabel: hasItems ? null : l.timetableEmptyWeek,
          onFocusDay: (day) => setState(
            () => _focusedDay = !wide && _focusedDay == day ? null : day,
          ),
          onTapItem: (item) => _openItem(item, times, layout.rows),
          onCreate: (range) => _create(range, times, layout.rows),
        );
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            SizedBox(
              height: headerHeight - 8,
              child: _Header(
                week: week,
                maxWeek: widget.maxWeek,
                isCurrent: isCurrent,
                range: l.timetableDateRange(
                  DateFormat.MMMd(l.localeName).format(dates.first),
                  DateFormat.MMMd(l.localeName).format(dates.last),
                ),
                status: widget.status,
                refreshing: widget.refreshing,
                onPrevious: week > 1 ? () => _selectWeek(week - 1) : null,
                onNext: week < widget.maxWeek
                    ? () => _selectWeek(week + 1)
                    : null,
                onCurrent: () => _selectWeek(widget.week),
                onPick: _pickWeek,
                onRefresh: widget.onRefresh,
                onAdd: () => _create(_defaultRange(layout), times, layout.rows),
                onOptions: () => showTimetableOptions(
                  context,
                  exportEnabled: widget.data.events.isNotEmpty,
                  onAddBlock: () =>
                      _create(_defaultRange(layout), times, layout.rows),
                  onEditBlock: (block) => _openItem(
                    TimetableItem(
                      kind: TimetableItemKind.block,
                      day: block.days.first,
                      start: block.start,
                      end: block.end,
                      title: block.title,
                      block: block,
                      colorSlot: block.color,
                    ),
                    times,
                    layout.rows,
                  ),
                ),
              ),
            ),
            const SizedBox(height: 8),
            if (!wide)
              grid
            else
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(child: grid),
                  const SizedBox(width: 16),
                  SizedBox(
                    width: 300,
                    // Scrolls on its own when the day outgrows the grid.
                    height: math.max(gridHeight ?? 480, 320.0),
                    child: _DayAgenda(
                      day: focused ?? (isCurrent ? _today : layout.days.first),
                      date:
                          dates[(focused ??
                                  (isCurrent ? _today : layout.days.first)) -
                              1],
                      layout: layout,
                      times: times,
                      notesOf: notesOf,
                      onTapItem: (item) => _openItem(item, times, layout.rows),
                    ),
                  ),
                ],
              ),
          ],
        );
      },
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({
    required this.week,
    required this.maxWeek,
    required this.isCurrent,
    required this.range,
    required this.status,
    required this.refreshing,
    required this.onPrevious,
    required this.onNext,
    required this.onCurrent,
    required this.onPick,
    required this.onRefresh,
    required this.onAdd,
    required this.onOptions,
  });
  final int week;
  final int maxWeek;
  final bool isCurrent;
  final String range;
  final CampusTimetableStatus? status;
  final bool refreshing;
  final VoidCallback? onPrevious;
  final VoidCallback? onNext;
  final VoidCallback onCurrent;
  final VoidCallback onPick;

  /// Pull-to-refresh is the visible path; screen readers get an action.
  final VoidCallback? onRefresh;
  final VoidCallback onAdd;
  final VoidCallback onOptions;

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final material = MaterialLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final status = this.status;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Expanded(
          flex: 44,
          child: Row(
            children: [
              GfIconButton(
                symbol: 'chevron-left',
                tooltip: material.previousPageTooltip,
                onPressed: onPrevious,
              ),
              Flexible(
                child: Semantics(
                  button: true,
                  hint: l.timetableWeekPicker,
                  child: InkWell(
                    onTap: onPick,
                    borderRadius: BorderRadius.circular(22),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 8,
                        vertical: 6,
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Flexible(
                            child: FittedBox(
                              fit: BoxFit.scaleDown,
                              child: GfText(
                                l.scheduleWeekN(week),
                                style: type.title3,
                              ),
                            ),
                          ),
                          const SizedBox(width: 4),
                          GfSymbol(
                            'chevron-down',
                            size: 16,
                            color: colors.iconMuted,
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
              GfIconButton(
                symbol: 'chevron-right',
                tooltip: material.nextPageTooltip,
                onPressed: onNext,
              ),
              const Spacer(),
              _GlassActions(
                children: [
                  GfIconButton(
                    symbol: 'plus',
                    tooltip: l.timetableAddBlock,
                    color: colors.primary,
                    onPressed: onAdd,
                  ),
                  GfIconButton(
                    symbol: 'settings-duotone',
                    tooltip: l.timetableOptions,
                    color: colors.primary,
                    iconSize: 21,
                    onPressed: onOptions,
                  ),
                ],
              ),
            ],
          ),
        ),
        Expanded(
          flex: 26,
          child: Padding(
            padding: const EdgeInsetsDirectional.only(start: 12),
            child: Row(
              children: [
                Expanded(
                  child: Tooltip(
                    message: status?.detail ?? '',
                    triggerMode: TooltipTriggerMode.tap,
                    excludeFromSemantics: true,
                    child: Semantics(
                      label: [
                        range,
                        status?.detail ?? status?.text ?? '',
                      ].where((s) => s.isNotEmpty).join(' · '),
                      customSemanticsActions: {
                        if (onRefresh != null && !refreshing)
                          CustomSemanticsAction(label: l.commonRefresh):
                              onRefresh!,
                      },
                      excludeSemantics: true,
                      child: Row(
                        children: [
                          if (status?.warning ?? false)
                            Padding(
                              padding: const EdgeInsetsDirectional.only(end: 4),
                              child: GfSymbol(
                                'circle-alert',
                                size: 13,
                                color: colors.baseContent,
                              ),
                            ),
                          Expanded(
                            child: Text.rich(
                              TextSpan(
                                children: [
                                  TextSpan(text: range),
                                  if (status != null) ...[
                                    const TextSpan(text: '  ·  '),
                                    TextSpan(
                                      text: status.text,
                                      // Neutral emphasis: amber text on the
                                      // page surface would lose contrast.
                                      style: status.warning
                                          ? TextStyle(
                                              color: colors.baseContent,
                                              fontWeight: FontWeight.w600,
                                            )
                                          : null,
                                    ),
                                  ],
                                ],
                              ),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: type.caption.copyWith(
                                color: colors.iconMuted,
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
                if (!isCurrent)
                  TextButton(
                    onPressed: onCurrent,
                    style: TextButton.styleFrom(
                      minimumSize: const Size(0, 26),
                      padding: const EdgeInsets.symmetric(horizontal: 8),
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      visualDensity: VisualDensity.compact,
                    ),
                    child: GfText(l.scheduleCurrentWeek),
                  ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}

/// Wide layouts: the focused day in full, beside the week.
class _DayAgenda extends StatelessWidget {
  const _DayAgenda({
    required this.day,
    required this.date,
    required this.layout,
    required this.times,
    required this.notesOf,
    required this.onTapItem,
  });
  final int day;
  final DateTime date;
  final TimetableWeekLayout layout;
  final List<SectionTime> times;
  final List<String> Function(TimetableItem) notesOf;
  final ValueChanged<TimetableItem> onTapItem;

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final items = [
      for (final cluster in layout.clusters[day - 1]) ...cluster.items,
    ]..sort((a, b) => a.start.compareTo(b.start));
    return DecoratedBox(
      decoration: BoxDecoration(
        color: colors.base200.withValues(alpha: .5),
        borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).box + 4),
      ),
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          GfText(
            '${timetableWeekdayLabels(l)[day - 1]} · ${DateFormat.MMMd(l.localeName).format(date)}',
            style: type.title3,
          ),
          const SizedBox(height: 16),
          if (items.isEmpty)
            GfText(
              l.timetableDayFree,
              style: type.small.copyWith(color: colors.iconMuted),
            ),
          for (final item in items)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: _AgendaRow(
                item: item,
                times: times,
                notes: notesOf(item),
                onTap: () => onTapItem(item),
              ),
            ),
        ],
      ),
    );
  }
}

/// The header's own actions on a primary-tinted glass capsule, so they read
/// as controls against the neutral page. Static tint and highlight only: no
/// backdrop blur over a flat surface.
class _GlassActions extends StatelessWidget {
  const _GlassActions({required this.children});
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final dark = Theme.of(context).brightness == Brightness.dark;
    final tint = colors.primary;
    return DecoratedBox(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(22),
        gradient: LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [
            Color.alphaBlend(
              tint.withValues(alpha: dark ? .2 : .11),
              colors.base100,
            ),
            Color.alphaBlend(
              tint.withValues(alpha: dark ? .13 : .07),
              colors.base100,
            ),
          ],
        ),
        border: Border.all(color: tint.withValues(alpha: dark ? .3 : .16)),
        boxShadow: [
          BoxShadow(
            color: tint.withValues(alpha: dark ? .16 : .12),
            blurRadius: 10,
            offset: const Offset(0, 3),
            spreadRadius: -4,
          ),
        ],
      ),
      // Inner top highlight: the light edge that makes the tint read as glass.
      child: DecoratedBox(
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(22),
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.center,
            colors: [
              Colors.white.withValues(alpha: dark ? .07 : .5),
              Colors.white.withValues(alpha: 0),
            ],
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            for (var i = 0; i < children.length; i++) ...[
              if (i > 0)
                SizedBox(
                  width: 1,
                  height: 18,
                  child: ColoredBox(color: tint.withValues(alpha: .18)),
                ),
              children[i],
            ],
          ],
        ),
      ),
    );
  }
}

class _AgendaRow extends StatelessWidget {
  const _AgendaRow({
    required this.item,
    required this.times,
    required this.notes,
    required this.onTap,
  });
  final TimetableItem item;
  final List<SectionTime> times;
  final List<String> notes;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final type = GfTheme.typographyOf(context);
    final palette = timetableSlotColors(context, item.colorSlot);
    final ghost = item.kind == TimetableItemKind.otherWeek;
    final start = item.start <= times.length ? times[item.start - 1].start : '';
    final end = item.end <= times.length ? times[item.end - 1].end : '';
    final meta = TextStyle(color: colors.iconMuted);
    return Material(
      color: colors.base100,
      borderRadius: BorderRadius.circular(GfTheme.radiiOf(context).box),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Short colour pill beside the title, as in course details.
            Container(
              width: 3,
              height: 16,
              margin: const EdgeInsetsDirectional.only(start: 10, top: 12),
              decoration: BoxDecoration(
                color: ghost ? colors.line : palette.accent,
                borderRadius: BorderRadius.circular(1.5),
              ),
            ),
            Padding(
              padding: const EdgeInsetsDirectional.fromSTEB(8, 10, 0, 10),
              child: SizedBox(
                width: 44,
                child: GfText(
                  '$start\n$end',
                  style: type.caption.copyWith(
                    fontFeatures: const [FontFeature.tabularFigures()],
                  ),
                ),
              ),
            ),
            Expanded(
              child: Padding(
                padding: const EdgeInsetsDirectional.fromSTEB(8, 10, 12, 10),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    GfText(
                      item.title,
                      style: type.bodyStrong.copyWith(
                        color: ghost ? colors.iconMuted : null,
                      ),
                    ),
                    GfText(
                      [
                        l.schedulePeriodRange('${item.start}–${item.end}'),
                        item.place,
                        item.campus,
                        item.teacher,
                        if (ghost) l.timetableOtherWeek,
                      ].where((s) => s.isNotEmpty).join(' · '),
                      style: type.small.merge(meta),
                    ),
                    for (final note in notes)
                      Padding(
                        padding: const EdgeInsets.only(top: 4),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Padding(
                              padding: const EdgeInsetsDirectional.only(
                                top: 2,
                                end: 6,
                              ),
                              child: GfSymbol(
                                'pen-line',
                                size: 14,
                                color: palette.accent,
                              ),
                            ),
                            Expanded(child: GfText(note, style: type.small)),
                          ],
                        ),
                      ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
