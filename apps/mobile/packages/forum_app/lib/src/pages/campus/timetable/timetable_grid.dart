import 'dart:math' as math;

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:intl/intl.dart';
import 'package:ui_kit/ui_kit.dart';

import '../../../../l10n/app_localizations.dart';
import '../../../schedule/schedule_grid.dart';
import 'timetable_layout.dart';

/// Inclusive cell range picked on the grid (weekdays and 1-based sections).
class TimetableCellRange {
  const TimetableCellRange({
    required this.days,
    required this.start,
    required this.end,
  });
  final List<int> days;
  final int start;
  final int end;

  bool contains(int day, int section) =>
      days.contains(day) && section >= start && section <= end;
}

List<String> timetableWeekdayLabels(AppLocalizations l) => [
  l.scheduleDayMon,
  l.scheduleDayTue,
  l.scheduleDayWed,
  l.scheduleDayThu,
  l.scheduleDayFri,
  l.scheduleDaySat,
  l.scheduleDaySun,
];

/// Largest ambient scale among the grid's small text sizes; nonlinear
/// accessibility scaling enlarges captions more than titles.
double timetableTextScale(BuildContext context) {
  final scaler = MediaQuery.textScalerOf(context);
  return [
    9.0,
    10.0,
    11.0,
  ].map((size) => scaler.scale(size) / size).fold(1.0, math.max);
}

/// Palette pair for a course color slot: card fill, stripe accent and ink.
({Color fill, Color accent, Color ink}) timetableSlotColors(
  BuildContext context,
  int slot,
) {
  final dark = Theme.of(context).brightness == Brightness.dark;
  final fill = (dark
      ? kCourseSlotColorsDark
      : kCourseSlotColorsLight)[slot % kCourseColorSlotCount];
  final hsl = HSLColor.fromColor(fill);
  return (
    fill: fill,
    accent: hsl.withLightness(dark ? .72 : .42).toColor(),
    ink: hsl.withLightness(dark ? .88 : .22).toColor(),
  );
}

enum TimetableSessionPhase { upcoming, now, past }

/// Full-week timetable sized to a target height: every row and visible day
/// fits without inner scrolling. A focused day widens while the other days
/// compress, so one column can be read in full without leaving the week.
///
/// Only taps and long presses are claimed: horizontal drags stay with the
/// page's tab swipe and vertical drags with the page scroll.
class TimetableGrid extends StatefulWidget {
  const TimetableGrid({
    super.key,
    required this.layout,
    required this.times,
    required this.week,
    this.height,
    this.dates,
    this.today,
    this.nowMinute,
    this.focusedDay,
    this.expandFocus = true,
    this.showSectionTimes = true,
    this.notesOf,
    this.onFocusDay,
    this.onTapItem,
    this.onCreate,
    this.placeholder,
    this.refreshing = false,
    this.emptyLabel,
  });

  final TimetableWeekLayout layout;
  final List<SectionTime> times;

  /// Shown week; a pending cell selection does not survive a week change.
  final int week;

  /// Total height including the weekday header. Null uses comfortable rows.
  final double? height;
  final List<DateTime>? dates;

  /// Weekday of today when the shown week contains it.
  final int? today;
  final int? nowMinute;
  final int? focusedDay;

  /// Phones widen the focused day; wide layouts show an agenda instead.
  final bool expandFocus;
  final bool showSectionTimes;
  final List<String> Function(TimetableItem item)? notesOf;
  final ValueChanged<int>? onFocusDay;
  final ValueChanged<TimetableItem>? onTapItem;

  /// Opens the block editor; the selection stays drawn until it completes.
  final Future<void> Function(TimetableCellRange range)? onCreate;

  /// Skeleton mode: paints shimmer bones through this wrapper, no items.
  final Widget Function(Widget bone)? placeholder;
  final bool refreshing;
  final String? emptyLabel;

  @override
  State<TimetableGrid> createState() => _TimetableGridState();
}

class _Geometry {
  _Geometry({
    required this.axis,
    required this.header,
    required this.rowHeight,
    required this.rowTops,
    required this.columnLefts,
    required this.columnWidths,
    required this.height,
    required this.partStarts,
  });
  final double axis;
  final double header;
  final double rowHeight;
  final List<double> rowTops;
  final List<double> columnLefts;
  final List<double> columnWidths;
  final double height;
  final Set<int> partStarts;

  double rowBottom(int row) => rowTops[row] + rowHeight;

  /// (visible day index, zero-based row) under [p], or null.
  (int, int)? cellAt(Offset p) {
    var column = -1;
    for (var i = 0; i < columnLefts.length; i++) {
      if (p.dx >= columnLefts[i] && p.dx < columnLefts[i] + columnWidths[i]) {
        column = i;
        break;
      }
    }
    if (column < 0 || p.dy < header) return null;
    for (var row = 0; row < rowTops.length; row++) {
      if (p.dy < rowBottom(row)) return (column, row);
    }
    return null;
  }
}

class _TimetableGridState extends State<TimetableGrid>
    with SingleTickerProviderStateMixin {
  late final AnimationController _focus = AnimationController(
    vsync: this,
    duration: GfMotion.layout,
    value: 1,
  );
  late List<double> _from = _target();
  late List<double> _to = _from;
  _Geometry? _geometry;
  TimetableCellRange? _selection;
  (int, int)? _anchor;

  List<double> _target() => [
    for (var day = 1; day <= 7; day++)
      widget.expandFocus && widget.focusedDay == day ? 1.0 : 0.0,
  ];

  List<double> _current() {
    final t = GfMotion.layoutCurve.transform(_focus.value);
    return [for (var i = 0; i < 7; i++) _from[i] + (_to[i] - _from[i]) * t];
  }

  @override
  void didUpdateWidget(TimetableGrid oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.week != widget.week) _selection = null;
    if (oldWidget.focusedDay != widget.focusedDay ||
        oldWidget.expandFocus != widget.expandFocus) {
      _from = _current();
      _to = _target();
      _focus.duration = GfMotion.duration(context, GfMotion.layout);
      _focus.forward(from: 0);
    }
  }

  @override
  void dispose() {
    _focus.dispose();
    super.dispose();
  }

  _Geometry _measure(BoxConstraints constraints, double scale) {
    final layout = widget.layout;
    final width = constraints.maxWidth;
    final axis = math.min((width >= 600 ? 44.0 : 30.0) * scale, width * .2);
    final header = 40.0 * scale;
    const gap = 6.0;
    final partStarts = {
      for (final start in dayPartBoundaries(widget.times).values)
        if (start > 1 && start <= layout.rows) start - 1,
    };
    final minRow = 34.0 * scale;
    final fitted = widget.height == null
        ? 52.0 * scale
        : (widget.height! - header - partStarts.length * gap - 2) / layout.rows;
    final rowHeight = fitted.clamp(minRow, 120.0 * scale);
    final rowTops = <double>[];
    var y = header;
    for (var row = 0; row < layout.rows; row++) {
      if (partStarts.contains(row)) y += gap;
      rowTops.add(y);
      y += rowHeight;
    }
    final days = layout.days;
    final available = math.max(0.0, width - axis);
    final n = days.length;
    final base = available / n;
    final focused = math
        .max(available * .38, 112.0 * scale)
        .clamp(base, math.max(base, available * .6));
    final factor = n > 1 && available > focused
        ? math.max(1.0, focused * (n - 1) / (available - focused))
        : 1.0;
    final progress = _current();
    final weights = [
      for (final day in days) 1 + (factor - 1) * progress[day - 1],
    ];
    final total = weights.fold<double>(0, (sum, w) => sum + w);
    final lefts = <double>[];
    final widths = <double>[];
    var x = axis;
    for (final w in weights) {
      lefts.add(x);
      widths.add(available * w / total);
      x += available * w / total;
    }
    return _Geometry(
      axis: axis,
      header: header,
      rowHeight: rowHeight,
      rowTops: rowTops,
      columnLefts: lefts,
      columnWidths: widths,
      height: y + 2,
      partStarts: partStarts,
    );
  }

  TimetableCellRange _rangeOf((int, int) a, (int, int) b) {
    final days = widget.layout.days;
    final from = math.min(a.$1, b.$1);
    final to = math.max(a.$1, b.$1);
    return TimetableCellRange(
      days: [for (var i = from; i <= to; i++) days[i]],
      start: math.min(a.$2, b.$2) + 1,
      end: math.max(a.$2, b.$2) + 1,
    );
  }

  void _tap(Offset position) {
    final cell = _geometry?.cellAt(position);
    if (cell == null) {
      if (_selection != null) setState(() => _selection = null);
      return;
    }
    final day = widget.layout.days[cell.$1];
    final section = cell.$2 + 1;
    if (!widget.layout.isFree(day, section)) {
      if (_selection != null) setState(() => _selection = null);
      return;
    }
    HapticFeedback.selectionClick();
    setState(
      () => _selection = TimetableCellRange(
        days: [day],
        start: section,
        end: section,
      ),
    );
  }

  void _paintStart(Offset position) {
    final cell = _geometry?.cellAt(position);
    if (cell == null) return;
    HapticFeedback.mediumImpact();
    setState(() {
      _anchor = cell;
      _selection = _rangeOf(cell, cell);
    });
  }

  void _paintMove(Offset position) {
    final anchor = _anchor;
    final cell = _geometry?.cellAt(position);
    if (anchor == null || cell == null) return;
    final next = _rangeOf(anchor, cell);
    final current = _selection;
    if (current != null &&
        current.start == next.start &&
        current.end == next.end &&
        current.days.length == next.days.length &&
        current.days.first == next.days.first) {
      return;
    }
    HapticFeedback.selectionClick();
    setState(() => _selection = next);
  }

  void _paintEnd() {
    final selection = _selection;
    _anchor = null;
    if (selection != null) _create(selection);
  }

  bool _creating = false;

  /// Keeps the selection visible behind the editor, then clears it whether
  /// the block was saved or the editor dismissed.
  Future<void> _create(TimetableCellRange selection) async {
    final create = widget.onCreate;
    if (create == null || _creating) return;
    _creating = true;
    try {
      await create(selection);
    } finally {
      _creating = false;
      if (mounted) setState(() => _selection = null);
    }
  }

  void _clearSelection() {
    _anchor = null;
    if (_selection != null && !_creating) setState(() => _selection = null);
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final l = AppLocalizations.of(context);
    final scale = timetableTextScale(context);
    final radius = GfTheme.radiiOf(context).box + 4;
    return LayoutBuilder(
      builder: (context, constraints) => AnimatedBuilder(
        animation: _focus,
        builder: (context, _) {
          final g = _geometry = _measure(constraints, scale);
          // A tap anywhere outside the grid drops a pending selection.
          return TapRegion(
            onTapOutside: (_) => _clearSelection(),
            child: RepaintBoundary(
              child: DecoratedBox(
                position: DecorationPosition.foreground,
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(radius),
                  border: Border.all(color: colors.line),
                ),
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(radius),
                  child: ColoredBox(
                    color: colors.base100,
                    child: SizedBox(
                      height: g.height,
                      width: constraints.maxWidth,
                      // Header ripples paint on this layer, above the fill.
                      child: Material(
                        type: MaterialType.transparency,
                        child: Stack(children: _layers(context, l, g, scale)),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  List<Widget> _layers(
    BuildContext context,
    AppLocalizations l,
    _Geometry g,
    double scale,
  ) {
    final colors = GfTheme.colorsOf(context);
    final layout = widget.layout;
    final days = layout.days;
    final labels = timetableWeekdayLabels(l);
    final todayIndex = widget.today == null ? -1 : days.indexOf(widget.today!);
    final focusIndex = widget.focusedDay == null
        ? -1
        : days.indexOf(widget.focusedDay!);
    final interactive = widget.placeholder == null && widget.onCreate != null;
    final nowSection = widget.nowMinute == null
        ? null
        : timetableNowPosition(widget.times, layout.rows, widget.nowMinute!);
    final dates = widget.dates;
    final month = dates == null
        ? l.scheduleTimeAxis
        : DateFormat.MMM(l.localeName).format(dates[days.first - 1]);
    final children = <Widget>[
      Positioned.fill(
        child: CustomPaint(
          painter: _GridPainter(
            geometry: g,
            line: colors.line,
            band: colors.base200,
            today: todayIndex,
            focus: focusIndex,
            highlight: colors.primary,
          ),
        ),
      ),
      if (interactive)
        Positioned.fill(
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            excludeFromSemantics: true,
            onTapUp: (d) => _tap(d.localPosition),
            onLongPressStart: (d) => _paintStart(d.localPosition),
            onLongPressMoveUpdate: (d) => _paintMove(d.localPosition),
            onLongPressEnd: (_) => _paintEnd(),
            onLongPressCancel: _clearSelection,
          ),
        ),
    ];

    // Sessions, stacked inside overlapping clusters.
    for (var i = 0; i < days.length; i++) {
      final day = days[i];
      if (widget.placeholder != null) {
        for (final (boneDay, start, end) in const [
          (1, 1, 2),
          (2, 3, 4),
          (3, 5, 6),
          (4, 1, 2),
          (5, 7, 8),
        ]) {
          if (boneDay != day || end > layout.rows) continue;
          children.add(
            Positioned(
              left: g.columnLefts[i] + 2,
              top: g.rowTops[start - 1] + 2,
              width: g.columnWidths[i] - 4,
              height: g.rowBottom(end - 1) - g.rowTops[start - 1] - 4,
              child: widget.placeholder!(
                const GfSkeleton(height: double.infinity, radius: 7),
              ),
            ),
          );
        }
        continue;
      }
      for (final cluster in layout.clusters[day - 1]) {
        final top = g.rowTops[cluster.start - 1];
        final slice =
            (g.rowBottom(cluster.end - 1) - top) / cluster.items.length;
        for (var k = 0; k < cluster.items.length; k++) {
          final item = cluster.items[k];
          var phase = TimetableSessionPhase.upcoming;
          if (day == widget.today && widget.nowMinute != null) {
            final start = item.start <= widget.times.length
                ? parseHHMM(widget.times[item.start - 1].start)
                : null;
            final end = item.end <= widget.times.length
                ? parseHHMM(widget.times[item.end - 1].end)
                : null;
            if (end != null && widget.nowMinute! > end) {
              phase = TimetableSessionPhase.past;
            } else if (start != null && widget.nowMinute! >= start) {
              phase = TimetableSessionPhase.now;
            }
          }
          children.add(
            Positioned(
              key: ValueKey(
                '${item.kind.name}:${item.eventIndex ?? item.block?.id}:$day:${item.start}',
              ),
              left: g.columnLefts[i],
              top: top + slice * k,
              width: g.columnWidths[i],
              height: slice,
              child: TimetableSessionCard(
                item: item,
                times: widget.times,
                expanded: i == focusIndex && widget.expandFocus,
                phase: phase,
                notes: widget.notesOf?.call(item) ?? const [],
                onTap: widget.onTapItem == null
                    ? null
                    : () {
                        _clearSelection();
                        widget.onTapItem!(item);
                      },
              ),
            ),
          );
        }
      }
    }

    final selection = _selection;
    if (interactive && selection != null) {
      final first = days.indexOf(selection.days.first);
      final last = days.indexOf(selection.days.last);
      if (first >= 0 && last >= 0 && selection.end <= layout.rows) {
        final left = g.columnLefts[first];
        final width = g.columnLefts[last] + g.columnWidths[last] - left;
        final top = g.rowTops[selection.start - 1];
        final height = g.rowBottom(selection.end - 1) - top;
        children.add(
          Positioned(
            left: left,
            top: top,
            width: width,
            height: height,
            child: Padding(
              padding: const EdgeInsets.all(1.5),
              child: Semantics(
                button: true,
                label: l.timetableAddBlockHere,
                excludeSemantics: true,
                onTap: () => _create(selection),
                child: Material(
                  color: colors.primary.withValues(alpha: .08),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(7),
                    side: BorderSide(color: colors.primary, width: 1.5),
                  ),
                  clipBehavior: Clip.antiAlias,
                  child: InkWell(
                    excludeFromSemantics: true,
                    onTap: () => _create(selection),
                    child: Center(
                      child: width >= 22 && height >= 22
                          ? GfSymbol('plus', size: 16, color: colors.primary)
                          : null,
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
      }
    }

    if (todayIndex >= 0 && nowSection != null) {
      final row = nowSection.floor().clamp(0, layout.rows - 1);
      final y = g.rowTops[row] + (nowSection - row) * g.rowHeight;
      children.add(
        Positioned(
          left: g.columnLefts[todayIndex] - 3,
          top: y - 3,
          width: g.columnWidths[todayIndex] + 3,
          height: 6,
          child: IgnorePointer(
            child: CustomPaint(painter: _NowPainter(colors.primary)),
          ),
        ),
      );
    }

    // Corner, weekday header and section axis.
    children.add(
      Positioned(
        left: 0,
        top: 0,
        width: g.axis,
        height: g.header,
        child: Center(
          child: FittedBox(
            fit: BoxFit.scaleDown,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 2),
              child: GfText(
                month,
                style: TextStyle(
                  fontSize: 10,
                  fontWeight: FontWeight.w600,
                  color: colors.iconMuted,
                ),
              ),
            ),
          ),
        ),
      ),
    );
    for (var i = 0; i < days.length; i++) {
      final day = days[i];
      children.add(
        Positioned(
          left: g.columnLefts[i],
          top: 0,
          width: g.columnWidths[i],
          height: g.header,
          child: _DayHeader(
            label: labels[day - 1],
            date: dates?[day - 1],
            today: i == todayIndex,
            focused: i == focusIndex,
            onTap: widget.onFocusDay == null
                ? null
                : () => widget.onFocusDay!(day),
          ),
        ),
      );
    }
    for (var row = 0; row < layout.rows; row++) {
      final time = row < widget.times.length ? widget.times[row] : null;
      children.add(
        Positioned(
          left: 0,
          top: g.rowTops[row],
          width: g.axis,
          height: g.rowHeight,
          child: IgnorePointer(
            child: Center(
              child: FittedBox(
                fit: BoxFit.scaleDown,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    GfText(
                      '${row + 1}',
                      style: TextStyle(
                        fontSize: 11,
                        height: 1.1,
                        fontWeight: FontWeight.w600,
                        fontFeatures: const [FontFeature.tabularFigures()],
                        color: colors.baseContent.withValues(alpha: .72),
                      ),
                    ),
                    if (time != null &&
                        widget.showSectionTimes &&
                        g.rowHeight >= 36 * scale)
                      GfText(
                        g.rowHeight >= 56 * scale
                            ? '${time.start}\n${time.end}'
                            : time.start,
                        textAlign: TextAlign.center,
                        style: TextStyle(
                          fontSize: 8.5,
                          height: 1.15,
                          fontFeatures: const [FontFeature.tabularFigures()],
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

    if (widget.emptyLabel != null) {
      children.add(
        Positioned(
          left: g.axis,
          right: 0,
          top: g.header,
          bottom: 0,
          child: IgnorePointer(
            child: Center(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: GfText(
                  widget.emptyLabel!,
                  textAlign: TextAlign.center,
                  style: GfTheme.typographyOf(
                    context,
                  ).small.copyWith(color: colors.iconMuted),
                ),
              ),
            ),
          ),
        ),
      );
    }
    if (widget.refreshing) {
      children.add(
        Positioned(
          left: 0,
          right: 0,
          top: 0,
          height: 2,
          child: LinearProgressIndicator(
            minHeight: 2,
            semanticsLabel: l.commonLoading,
          ),
        ),
      );
    }
    return children;
  }
}

class _DayHeader extends StatelessWidget {
  const _DayHeader({
    required this.label,
    required this.date,
    required this.today,
    required this.focused,
    required this.onTap,
  });
  final String label;
  final DateTime? date;
  final bool today;
  final bool focused;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final l = AppLocalizations.of(context);
    final dateText = date == null ? null : '${date!.day}';
    final content = Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        GfText(
          label,
          maxLines: 1,
          style: TextStyle(
            fontSize: 11,
            height: 1.1,
            fontWeight: focused || today ? FontWeight.w700 : FontWeight.w500,
            color: focused || today ? colors.primary : colors.baseContent,
          ),
        ),
        if (dateText != null) ...[
          const SizedBox(height: 2),
          Container(
            constraints: const BoxConstraints(minWidth: 18),
            height: 16,
            padding: const EdgeInsets.symmetric(horizontal: 3),
            alignment: Alignment.center,
            decoration: today
                ? BoxDecoration(
                    color: colors.primary,
                    borderRadius: BorderRadius.circular(8),
                  )
                : null,
            child: GfText(
              dateText,
              style: TextStyle(
                fontSize: 10,
                height: 1,
                fontWeight: FontWeight.w600,
                fontFeatures: const [FontFeature.tabularFigures()],
                color: today ? colors.primaryContent : colors.iconMuted,
              ),
            ),
          ),
        ],
      ],
    );
    return Semantics(
      button: onTap != null,
      selected: focused,
      label: [
        label,
        if (date != null) DateFormat.MMMd(l.localeName).format(date!),
      ].join(' '),
      hint: onTap == null
          ? null
          : focused
          ? l.timetableUnfocusDay
          : l.timetableFocusDay,
      excludeSemantics: true,
      onTap: onTap,
      child: InkWell(
        excludeFromSemantics: true,
        onTap: onTap,
        child: FittedBox(fit: BoxFit.scaleDown, child: content),
      ),
    );
  }
}

/// Session card. Content tiers follow the card's own size: compact cells
/// keep the title and room; wider or focused cells add teacher, weeks and
/// notes. Text never overflows; it is clipped to the card.
class TimetableSessionCard extends StatelessWidget {
  const TimetableSessionCard({
    super.key,
    required this.item,
    required this.times,
    this.expanded = false,
    this.phase = TimetableSessionPhase.upcoming,
    this.notes = const [],
    this.onTap,
  });

  final TimetableItem item;
  final List<SectionTime> times;
  final bool expanded;
  final TimetableSessionPhase phase;
  final List<String> notes;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final l = AppLocalizations.of(context);
    final palette = timetableSlotColors(context, item.colorSlot);
    final ghost = item.kind == TimetableItemKind.otherWeek;
    final block = item.kind == TimetableItemKind.block;
    final fade = phase == TimetableSessionPhase.past ? .5 : 1.0;
    final now = phase == TimetableSessionPhase.now;
    final dark = Theme.of(context).brightness == Brightness.dark;
    final radius = BorderRadius.circular(7);
    final fill = ghost
        ? colors.base200.withValues(alpha: .6)
        : palette.fill.withValues(alpha: (block ? .78 : 1) * fade);
    final ink = ghost ? colors.iconMuted : palette.ink.withValues(alpha: fade);
    final accent = ghost ? colors.line : palette.accent.withValues(alpha: fade);
    final weeks = formatDisplayWeeks(
      item.weeks,
      (parity) => parity == PkWeekParity.odd
          ? l.scheduleParityOdd
          : l.scheduleParityEven,
      l.scheduleWeeksN,
    );
    final time = item.end <= times.length
        ? '${times[item.start - 1].start}–${times[item.end - 1].end}'
        : '';
    final place = expanded
        ? [item.place, item.campus].where((s) => s.isNotEmpty).join(' · ')
        : item.place;
    final label = [
      item.title,
      timetableWeekdayLabels(l)[item.day - 1],
      l.scheduleSectionsN('${item.start}–${item.end}'),
      time,
      item.place,
      item.campus,
      item.teacher,
      weeks,
      if (ghost) l.timetableOtherWeek,
      if (block) l.timetableBlock,
      if (phase == TimetableSessionPhase.now) l.timetableInProgress,
      ...notes,
    ].where((s) => s.isNotEmpty).join(' · ');

    return Padding(
      padding: const EdgeInsets.all(1.5),
      child: Semantics(
        button: onTap != null,
        label: label,
        excludeSemantics: true,
        onTap: onTap,
        child: DecoratedBox(
          decoration: BoxDecoration(
            borderRadius: radius,
            // Lit from above: a slightly lighter top reads as a soft material
            // rather than a flat swatch. Ink colours stay unchanged.
            gradient: ghost
                ? null
                : LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                    colors: [
                      Color.lerp(
                        fill,
                        Colors.white.withValues(alpha: fill.a),
                        dark ? .05 : .28,
                      )!,
                      fill,
                    ],
                  ),
            color: ghost ? fill : null,
            border: Border.all(
              color: now
                  ? palette.accent
                  : ghost
                  ? colors.line
                  : palette.accent.withValues(alpha: (dark ? .28 : .2) * fade),
              width: now ? 1.5 : 1,
            ),
            boxShadow: ghost || phase == TimetableSessionPhase.past
                ? null
                : [
                    BoxShadow(
                      color: palette.accent.withValues(
                        alpha: now ? .28 : (dark ? .1 : .12),
                      ),
                      blurRadius: now ? 10 : 5,
                      offset: Offset(0, now ? 2 : 1.5),
                      spreadRadius: now ? -1 : -2,
                    ),
                  ],
          ),
          child: Material(
            type: MaterialType.transparency,
            shape: RoundedRectangleBorder(borderRadius: radius),
            clipBehavior: Clip.antiAlias,
            child: InkWell(
              excludeFromSemantics: true,
              onTap: onTap,
              splashColor: palette.accent.withValues(alpha: .16),
              highlightColor: palette.accent.withValues(alpha: .08),
              child: LayoutBuilder(
                builder: (context, box) => CustomPaint(
                  painter: _CardDecorPainter(
                    sheen: ghost
                        ? null
                        : Colors.white.withValues(alpha: dark ? .07 : .55),
                    hatch: block ? palette.accent.withValues(alpha: .14) : null,
                    dogEar: notes.isEmpty
                        ? null
                        : palette.accent.withValues(alpha: .6 * fade),
                  ),
                  child: _content(
                    context,
                    box,
                    ink,
                    place,
                    weeks,
                    ghost,
                    block || ghost ? null : accent,
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _content(
    BuildContext context,
    BoxConstraints box,
    Color ink,
    String place,
    String weeks,
    bool ghost,
    Color? accent,
  ) {
    final l = AppLocalizations.of(context);
    final scale = timetableTextScale(context);
    final w = box.maxWidth;
    final h = box.maxHeight;
    final roomy = expanded || w >= 92 * scale;
    final narrow = w < 40 * scale;
    // Course colour as a short centred cap, matching the web timetable.
    final barWidth = roomy && h >= 56 * scale ? 28.0 : 20.0;
    final barThickness = barWidth > 20 ? 3.0 : 2.5;
    final barSpace = accent == null ? 0.0 : barThickness + 3;
    final titleSize = narrow ? 10.0 : 11.0;
    final titleLine = titleSize * 1.22 * scale;
    final metaLine = 9.5 * 1.2 * scale;
    final showPlace = place.isNotEmpty && h >= 30 * scale;
    final showTeacher =
        item.teacher.isNotEmpty &&
        (roomy || w >= 72 * scale) &&
        h >= 64 * scale;
    final showWeeks = roomy && weeks.isNotEmpty && h >= 84 * scale;
    final showNote = notes.isNotEmpty && roomy && h >= 56 * scale;
    final reserved =
        (showPlace ? metaLine * (roomy ? 2 : 3) : 0) +
        (showTeacher ? metaLine : 0) +
        (showWeeks ? metaLine : 0) +
        (showNote ? metaLine * 2 : 0) +
        (ghost ? metaLine : 0);
    final titleLines = math.max(
      1,
      ((h - 8 - barSpace - reserved) / titleLine).floor(),
    );
    final meta = TextStyle(
      fontSize: 9.5,
      height: 1.2,
      fontWeight: FontWeight.w500,
      color: ink.withValues(alpha: ink.a * .78),
    );
    return ClipRect(
      child: OverflowBox(
        alignment: Alignment.topLeft,
        maxHeight: double.infinity,
        child: Padding(
          padding: EdgeInsets.fromLTRB(narrow ? 3 : 5, 4, narrow ? 3 : 5, 3),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (accent != null)
                Center(
                  child: Container(
                    width: math.min(barWidth, w * .6),
                    height: barThickness,
                    margin: const EdgeInsets.only(bottom: 3),
                    decoration: BoxDecoration(
                      color: accent.withValues(alpha: accent.a * .7),
                      borderRadius: BorderRadius.circular(barThickness),
                    ),
                  ),
                ),
              GfText(
                item.title,
                maxLines: titleLines,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontSize: titleSize,
                  height: 1.22,
                  fontWeight: FontWeight.w600,
                  color: ink,
                ),
              ),
              if (showPlace) ...[
                const SizedBox(height: 2),
                GfText(
                  roomy ? place : '@$place',
                  maxLines: roomy ? 2 : 3,
                  overflow: TextOverflow.ellipsis,
                  style: meta,
                ),
              ],
              if (showTeacher)
                GfText(
                  compactTeacherName(item.teacher, 2),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: meta,
                ),
              if (showWeeks)
                GfText(
                  weeks,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: meta,
                ),
              if (ghost)
                GfText(
                  l.timetableOtherWeek,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: meta,
                ),
              if (showNote) ...[
                const SizedBox(height: 2),
                GfText(
                  notes.first,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: meta.copyWith(fontStyle: FontStyle.italic),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _CardDecorPainter extends CustomPainter {
  const _CardDecorPainter({this.sheen, this.hatch, this.dogEar});

  /// One-pixel inner highlight along the top edge.
  final Color? sheen;
  final Color? hatch;
  final Color? dogEar;

  @override
  void paint(Canvas canvas, Size size) {
    if (hatch case final Color color) {
      final paint = Paint()
        ..color = color
        ..strokeWidth = 1;
      for (var x = -size.height; x < size.width; x += 7) {
        canvas.drawLine(
          Offset(x, size.height),
          Offset(x + size.height, 0),
          paint,
        );
      }
    }
    if (sheen case final Color color) {
      canvas.drawLine(
        // Just inside the 1px border, clear of the rounded corners.
        Offset(5, 1.5),
        Offset(size.width - 5, 1.5),
        Paint()
          ..color = color
          ..strokeWidth = 1,
      );
    }
    if (dogEar case final Color color) {
      const ear = 7.0;
      canvas.drawPath(
        Path()
          ..moveTo(size.width - ear, 0)
          ..lineTo(size.width, 0)
          ..lineTo(size.width, ear)
          ..close(),
        Paint()..color = color,
      );
    }
  }

  @override
  bool shouldRepaint(_CardDecorPainter old) =>
      old.sheen != sheen || old.hatch != hatch || old.dogEar != dogEar;
}

class _GridPainter extends CustomPainter {
  const _GridPainter({
    required this.geometry,
    required this.line,
    required this.band,
    required this.today,
    required this.focus,
    required this.highlight,
  });
  final _Geometry geometry;
  final Color line;
  final Color band;
  final int today;
  final int focus;
  final Color highlight;

  @override
  void paint(Canvas canvas, Size size) {
    final g = geometry;
    final bodyBottom = g.rowBottom(g.rowTops.length - 1);
    void column(int index, double alpha) {
      if (index < 0) return;
      canvas.drawRect(
        Rect.fromLTWH(
          g.columnLefts[index],
          g.header,
          g.columnWidths[index],
          bodyBottom - g.header,
        ),
        Paint()..color = highlight.withValues(alpha: alpha),
      );
    }

    column(today, .05);
    if (focus != today) column(focus, .035);
    final strong = Paint()
      ..color = line.withValues(alpha: .6)
      ..strokeWidth = 1;
    final faint = Paint()
      ..color = line.withValues(alpha: .35)
      ..strokeWidth = 1;
    final bandPaint = Paint()..color = band.withValues(alpha: .7);
    canvas.drawLine(Offset(0, g.header), Offset(size.width, g.header), strong);
    for (var row = 0; row < g.rowTops.length; row++) {
      if (g.partStarts.contains(row)) {
        canvas.drawRect(
          Rect.fromLTRB(0, g.rowTops[row] - 6, size.width, g.rowTops[row]),
          bandPaint,
        );
      } else if (row > 0) {
        canvas.drawLine(
          Offset(g.axis, g.rowTops[row]),
          Offset(size.width, g.rowTops[row]),
          faint,
        );
      }
    }
    canvas.drawLine(
      Offset(g.axis, g.header),
      Offset(g.axis, bodyBottom),
      strong,
    );
  }

  @override
  bool shouldRepaint(_GridPainter old) => true;
}

class _NowPainter extends CustomPainter {
  const _NowPainter(this.color);
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()..color = color;
    final y = size.height / 2;
    canvas.drawCircle(Offset(3, y), 3, paint);
    canvas.drawRect(Rect.fromLTRB(3, y - .75, size.width, y + .75), paint);
  }

  @override
  bool shouldRepaint(_NowPainter old) => old.color != color;
}
