import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../theme/gf_theme.dart';
import '../gf_motion.dart';

/// A navigation glyph with an unread mark tucked into its top-trailing corner.
///
/// The mark overlaps the glyph the way system tab bars place theirs, and
/// instead of a coloured ring it knocks a thin gap out of the glyph behind
/// it, so it reads cleanly on glass, tint or any background. A [count] shows
/// tabular digits that roll up or down as it changes; without one it is a
/// dot. The mark never changes the glyph's layout box.
class GfBadgedGlyph extends StatefulWidget {
  const GfBadgedGlyph({
    super.key,
    required this.glyph,
    required this.size,
    required this.visible,
    this.count,
    this.max = 99,
  });

  final Widget glyph;

  /// The glyph's square extent; the mark is placed relative to it.
  final double size;
  final bool visible;

  /// Null or zero shows a dot.
  final int? count;
  final int max;

  static const double _pill = 16;
  static const double _dot = 8;
  static const double _countGap = 2;
  static const double _dotGap = 1.5;
  static const double _padding = 4.5;

  static String labelFor(int count, int max) =>
      count > max ? '$max+' : '$count';

  @override
  State<GfBadgedGlyph> createState() => _GfBadgedGlyphState();
}

class _GfBadgedGlyphState extends State<GfBadgedGlyph>
    with TickerProviderStateMixin {
  late final AnimationController _presence = AnimationController(
    vsync: this,
    duration: GfMotion.layout,
    reverseDuration: GfMotion.content,
    value: widget.visible ? 1 : 0,
  );
  late final CurvedAnimation _presenceCurve = CurvedAnimation(
    parent: _presence,
    curve: GfMotion.enterCurve,
    reverseCurve: GfMotion.exitCurve,
  );
  late final AnimationController _resize = AnimationController(
    vsync: this,
    duration: GfMotion.layout,
    value: 1,
  );
  late final AnimationController _bump = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 280),
  );
  late final Listenable _motion = Listenable.merge(<Listenable>[
    _presence,
    _resize,
    _bump,
  ]);

  double? _fromWidth;
  double? _toWidth;
  // The last mark shown, kept while it scales away.
  int? _shownCount;
  bool _rollsUp = true;

  bool get _hasCount => (widget.count ?? 0) > 0;

  @override
  void initState() {
    super.initState();
    _shownCount = _hasCount ? widget.count : null;
  }

  @override
  void didUpdateWidget(GfBadgedGlyph oldWidget) {
    super.didUpdateWidget(oldWidget);
    final bool reduced = GfMotion.reducedOf(context);
    if (widget.visible != oldWidget.visible) {
      if (reduced) {
        _presence.value = widget.visible ? 1 : 0;
      } else if (widget.visible) {
        _presence.forward();
      } else {
        _presence.reverse();
      }
    }
    if (!widget.visible) return;
    final int? next = _hasCount ? widget.count : null;
    if (next == _shownCount) return;
    final int? previous = _shownCount;
    _fromWidth = _currentWidth;
    _shownCount = next;
    _toWidth = null;
    _rollsUp = (next ?? 0) >= (previous ?? 0);
    if (reduced || !oldWidget.visible) {
      _resize.value = 1;
      return;
    }
    _resize.forward(from: 0);
    // A new arrival nudges the mark; reading things down stays quiet.
    if (next != null && previous != null && next > previous) {
      _bump.forward(from: 0);
    }
  }

  @override
  void dispose() {
    _presenceCurve.dispose();
    _presence.dispose();
    _resize.dispose();
    _bump.dispose();
    super.dispose();
  }

  TextStyle _style(GfColors colors) =>
      GfTheme.typographyOf(context).label.copyWith(
        color: colors.errorContent,
        fontSize: 11,
        height: 1,
        letterSpacing: 0,
        leadingDistribution: TextLeadingDistribution.even,
      );

  double _widthFor(int? count) {
    if (count == null) return GfBadgedGlyph._dot;
    final TextPainter painter = TextPainter(
      text: TextSpan(
        text: GfBadgedGlyph.labelFor(count, widget.max),
        style: _style(GfTheme.colorsOf(context)),
      ),
      textDirection: TextDirection.ltr,
      maxLines: 1,
    )..layout();
    final double width = math.max(
      GfBadgedGlyph._pill,
      painter.width + GfBadgedGlyph._padding * 2,
    );
    painter.dispose();
    return width.ceilToDouble();
  }

  double get _currentWidth {
    final double to = _toWidth ??= _widthFor(_shownCount);
    final double? from = _fromWidth;
    if (from == null) return to;
    final double t = GfMotion.enterCurve.transform(_resize.value);
    return from + (to - from) * t;
  }

  /// The mark's resting box in glyph coordinates.
  Rect _markRect(TextDirection direction) {
    final double s = widget.size;
    final Rect rect;
    if (_shownCount == null) {
      // A dot sits on the corner, mostly outside the glyph's ink.
      rect = Rect.fromCircle(
        center: Offset(s * 0.86, s * 0.14),
        radius: GfBadgedGlyph._dot / 2,
      );
    } else {
      // A count starts just past the glyph's middle and rides its top edge,
      // growing toward the trailing side as digits are added.
      rect = Rect.fromLTWH(
        s * 0.55,
        s * 0.16 - GfBadgedGlyph._pill / 2,
        _currentWidth,
        GfBadgedGlyph._pill,
      );
    }
    if (direction == TextDirection.ltr) return rect;
    return Rect.fromLTRB(s - rect.right, rect.top, s - rect.left, rect.bottom);
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final TextDirection direction = Directionality.of(context);
    final Widget glyph = SizedBox.square(
      dimension: widget.size,
      child: widget.glyph,
    );
    return AnimatedBuilder(
      animation: _motion,
      child: glyph,
      builder: (BuildContext context, Widget? glyph) {
        final double presence = _presenceCurve.value;
        if (presence == 0) return glyph!;
        final Rect rest = _markRect(direction);
        final double scale = 0.5 + 0.5 * presence;
        final Rect mark = Rect.fromCenter(
          center: rest.center,
          width: rest.width * scale,
          height: rest.height * scale,
        );
        final double gap = _shownCount == null
            ? GfBadgedGlyph._dotGap
            : GfBadgedGlyph._countGap;
        final double bump = math.sin(math.pi * _bump.value) * 0.14;
        return Stack(
          clipBehavior: Clip.none,
          children: <Widget>[
            ClipPath(
              clipper: _KnockoutClipper(
                RRect.fromRectAndRadius(
                  mark.inflate(gap * presence),
                  Radius.circular(mark.height / 2 + gap),
                ),
              ),
              child: glyph,
            ),
            Positioned.fromRect(
              rect: mark,
              child: Opacity(
                opacity: presence.clamp(0.0, 1.0),
                child: Transform.scale(
                  scale: 1 + bump,
                  child: _shownCount == null
                      ? DecoratedBox(
                          decoration: BoxDecoration(
                            color: colors.error,
                            shape: BoxShape.circle,
                          ),
                        )
                      : _CountPill(
                          label: GfBadgedGlyph.labelFor(
                            _shownCount!,
                            widget.max,
                          ),
                          rollsUp: _rollsUp,
                          scale: scale,
                          color: colors.error,
                          style: _style(colors),
                        ),
                ),
              ),
            ),
          ],
        );
      },
    );
  }
}

/// A capsule whose digits roll in from below as the count grows and from
/// above as it shrinks.
class _CountPill extends StatelessWidget {
  const _CountPill({
    required this.label,
    required this.rollsUp,
    required this.scale,
    required this.color,
    required this.style,
  });

  final String label;
  final bool rollsUp;
  final double scale;
  final Color color;
  final TextStyle style;

  @override
  Widget build(BuildContext context) => DecoratedBox(
    decoration: ShapeDecoration(color: color, shape: const StadiumBorder()),
    child: ClipRRect(
      borderRadius: BorderRadius.circular(GfRadius.full),
      child: AnimatedSwitcher(
        duration: GfMotion.duration(context, GfMotion.layout),
        switchInCurve: GfMotion.enterCurve,
        switchOutCurve: GfMotion.exitCurve,
        transitionBuilder: (Widget child, Animation<double> animation) {
          final bool incoming = child.key == ValueKey<String>(label);
          final double from = (rollsUp ? 1 : -1) * (incoming ? 0.7 : -0.7);
          return FadeTransition(
            opacity: animation,
            child: SlideTransition(
              position: Tween<Offset>(
                begin: Offset(0, from),
                end: Offset.zero,
              ).animate(animation),
              child: child,
            ),
          );
        },
        child: Center(
          key: ValueKey<String>(label),
          child: Transform.scale(
            scale: scale,
            child: Text(
              label,
              maxLines: 1,
              softWrap: false,
              textScaler: TextScaler.noScaling,
              textHeightBehavior: const TextHeightBehavior(
                applyHeightToFirstAscent: false,
                applyHeightToLastDescent: false,
              ),
              style: style,
            ),
          ),
        ),
      ),
    ),
  );
}

class _KnockoutClipper extends CustomClipper<Path> {
  const _KnockoutClipper(this.hole);

  final RRect hole;

  @override
  Path getClip(Size size) => Path.combine(
    PathOperation.difference,
    Path()..addRect((Offset.zero & size).inflate(size.shortestSide)),
    Path()..addRRect(hole),
  );

  @override
  bool shouldReclip(_KnockoutClipper oldClipper) => oldClipper.hole != hole;
}
