import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/services.dart';

import '../theme/gf_theme.dart';
import '../twemoji/twemoji_text.dart';
import 'gf_motion.dart';
import 'gf_symbol.dart';
import 'surfaces/gf_navigation_surface.dart';

class GfBottomNavigationItem {
  const GfBottomNavigationItem({
    required this.label,
    required this.symbol,
    required this.selectedSymbol,
    this.badge = false,
    this.badgeSemanticLabel,
  });

  final String label;
  final String symbol;
  final String selectedSymbol;
  final bool badge;
  final String? badgeSemanticLabel;

  String get semanticsLabel => badge && badgeSemanticLabel != null
      ? '$label, $badgeSemanticLabel'
      : label;
}

/// Shared geometry for the compact navigation shell.
///
/// Keeping these offsets here prevents page content and floating actions from
/// drifting when the bar switches between icon-only and labeled modes.
class GfBottomNavigationMetrics {
  const GfBottomNavigationMetrics({
    required this.barHeight,
    required this.contentBottomInset,
    required this.actionBottomInset,
  });

  final double barHeight;
  final double contentBottomInset;
  final double actionBottomInset;
}

// Dock geometry. The selection lens sits [_dockInset] inside the rim, so the
// dock and the lens stay concentric pills.
const double _dockHeight = 56;
const double _dockInset = 5;
const double _dockSideMargin = GfSpace.xl;
const double _maxSlotExtent = 80;
const double _labeledActionWidth = 64;
// How far a lifted lens grows past its resting box on each side.
const double _lensLift = 3;
// How much of the destinations a lifted lens magnifies.
const double _lensMagnify = 0.1;
// How far a press swells the dock on each side.
const double _pressSwell = 3;
const double _maxStretch = 0.26;
// Lens stretch per destination per second of travel.
const double _stretchPerSlot = 0.03;
// The furthest, in destinations, a drag can pull the lens past either end.
const double _overdrag = 0.3;
const double _flingProjection = 0.09;

/// Four accessible destinations on a floating glass dock, with optional
/// visible labels and a compose action beside it.
///
/// The selection is a lens that springs between destinations and stretches
/// while it travels; the destinations it covers take the selected colour and
/// filled glyph part by part as it passes. Dragging along the dock lifts the
/// lens and scrubs it under the finger, ticking at each destination, and
/// selects where it is let go.
class GfBottomNavigation extends StatelessWidget {
  const GfBottomNavigation({
    super.key,
    required this.currentIndex,
    required this.items,
    required this.onSelected,
    this.onAction,
    this.actionLabel = '发布',
    this.actionSymbol = 'plus',
    this.showLabels = true,
  });

  final int currentIndex;
  final List<GfBottomNavigationItem> items;
  final ValueChanged<int> onSelected;
  final VoidCallback? onAction;
  final String actionLabel;
  final String actionSymbol;
  final bool showLabels;

  /// Gap between the dock's foot and the screen edge. On iOS the dock floats
  /// over the home indicator, as the system tab bar does; elsewhere it clears
  /// the system navigation entirely.
  static double bottomMarginFor(
    double safeAreaBottom, {
    bool homeIndicator = false,
  }) => math.max(
    GfSpace.md,
    homeIndicator ? safeAreaBottom - GfSpace.md : safeAreaBottom,
  );

  static bool _floatsOverHomeIndicator(BuildContext context) =>
      Theme.of(context).platform == TargetPlatform.iOS;

  static GfBottomNavigationMetrics metrics({
    required double safeAreaBottom,
    required double barHeight,
    bool homeIndicator = false,
  }) {
    final double margin = bottomMarginFor(
      safeAreaBottom,
      homeIndicator: homeIndicator,
    );
    return GfBottomNavigationMetrics(
      barHeight: barHeight,
      contentBottomInset: margin + barHeight + GfSpace.md,
      // Callers place floating actions above the safe area.
      actionBottomInset: margin + barHeight + GfSpace.lg - safeAreaBottom,
    );
  }

  /// [metrics] for the device insets and platform of [context].
  static GfBottomNavigationMetrics metricsOf(
    BuildContext context, {
    required double barHeight,
  }) => metrics(
    safeAreaBottom: MediaQuery.paddingOf(context).bottom,
    barHeight: barHeight,
    homeIndicator: _floatsOverHomeIndicator(context),
  );

  static double _dockWidthFor(
    double availableWidth, {
    required bool hasAction,
    required bool showLabels,
  }) {
    final double action = hasAction
        ? (showLabels ? _labeledActionWidth : _dockHeight) + GfSpace.sm
        : 0;
    return math.max(
      _dockInset * 2 + 4,
      math.min(
        availableWidth - _dockSideMargin * 2 - action,
        4 * _maxSlotExtent + _dockInset * 2,
      ),
    );
  }

  static double heightFor(
    BuildContext context, {
    required Iterable<String> labels,
    required double availableWidth,
    bool showLabels = true,
    bool hasAction = false,
  }) {
    if (!showLabels) return _dockHeight;
    final TextStyle style = _navigationLabelStyle(
      context,
      color: GfTheme.colorsOf(context).baseContent,
    );
    final TextScaler scaler = MediaQuery.textScalerOf(context);
    final double dockWidth = _dockWidthFor(
      availableWidth,
      hasAction: hasAction,
      showLabels: showLabels,
    );
    final double labelWidth = math.max(1, _labelWidthFor(dockWidth));
    double maxLabelHeight = 0;
    for (final String label in labels) {
      final TextPainter painter = TextPainter(
        text: TextSpan(text: label, style: style),
        textDirection: Directionality.of(context),
        textScaler: scaler,
      )..layout(maxWidth: labelWidth);
      maxLabelHeight = math.max(maxLabelHeight, painter.height);
      painter.dispose();
    }
    return math.max(64, _labeledContentHeight + maxLabelHeight);
  }

  static double _labelWidthFor(double dockWidth) =>
      (dockWidth - _dockInset * 2) / 4 - GfSpace.sm;

  // Inset, glyph and gap above a label, and the air below it.
  static const double _labeledContentHeight =
      _dockInset * 2 + GfSpace.sm + GfIconSize.lg + GfSpace.xxs + GfSpace.xs;

  @override
  Widget build(BuildContext context) {
    if (items.length != 4) {
      throw FlutterError(
        'GfBottomNavigation requires exactly four destinations; '
        'received ${items.length}.',
      );
    }
    final EdgeInsets padding = MediaQuery.paddingOf(context);
    final double margin = bottomMarginFor(
      padding.bottom,
      homeIndicator: _floatsOverHomeIndicator(context),
    );
    // Glass is skipped wherever the platform asks for plainer surfaces.
    final bool glass = GfNavigationSurface.usesGlass(context);
    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        final double available = math.max(
          1,
          constraints.maxWidth - padding.horizontal,
        );
        final bool hasAction = onAction != null;
        final double barHeight = heightFor(
          context,
          labels: [
            ...items.map((item) => item.label),
            if (hasAction) actionLabel,
          ],
          availableWidth: available,
          showLabels: showLabels,
          hasAction: hasAction,
        );
        final double dockWidth = _dockWidthFor(
          available,
          hasAction: hasAction,
          showLabels: showLabels,
        );
        return Padding(
          padding: EdgeInsets.fromLTRB(
            padding.left + _dockSideMargin,
            0,
            padding.right + _dockSideMargin,
            margin,
          ),
          child: SizedBox(
            height: barHeight,
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: <Widget>[
                SizedBox(
                  width: dockWidth,
                  child: _Dock(
                    items: items,
                    currentIndex: currentIndex,
                    onSelected: onSelected,
                    showLabels: showLabels,
                    glass: glass,
                  ),
                ),
                if (hasAction) ...<Widget>[
                  const SizedBox(width: GfSpace.sm),
                  _DockAction(
                    symbol: actionSymbol,
                    label: actionLabel,
                    onTap: onAction!,
                    showLabel: showLabels,
                    width: showLabels ? _labeledActionWidth : barHeight,
                  ),
                ],
              ],
            ),
          ),
        );
      },
    );
  }
}

/// A spring whose target may move every frame.
///
/// [AnimationController.animateWith] restarts its clock on every call, so a
/// value retargeted each frame (the lens under a finger, or its stretch while
/// it travels) would hold still. This integrates continuously instead.
class _Spring extends ChangeNotifier implements ValueListenable<double> {
  _Spring(TickerProvider vsync, this._value) : _target = _value {
    _ticker = vsync.createTicker(_tick);
  }

  late final Ticker _ticker;
  double _value;
  double _target;
  double _velocity = 0;
  double _stiffness = 300;
  double _damping = 30;
  Duration? _last;

  @override
  double get value => _value;
  double get velocity => _velocity;
  double get target => _target;

  void tune(double stiffness, double dampingRatio) {
    _stiffness = stiffness;
    _damping = 2 * dampingRatio * math.sqrt(stiffness);
  }

  void moveTo(double target, {bool immediate = false}) {
    _target = target;
    if (immediate) {
      _ticker.stop();
      _velocity = 0;
      if (_value != target) {
        _value = target;
        notifyListeners();
      }
      return;
    }
    if (!_ticker.isActive && (_value != target || _velocity != 0)) {
      _last = null;
      _ticker.start();
    }
  }

  void _tick(Duration elapsed) {
    double dt = _last == null
        ? 1 / 60
        : (elapsed - _last!).inMicroseconds / Duration.microsecondsPerSecond;
    _last = elapsed;
    dt = math.min(dt, 1 / 20);
    while (dt > 0) {
      final double step = math.min(dt, 1 / 240);
      final double force =
          -_stiffness * (_value - _target) - _damping * _velocity;
      _velocity += force * step;
      _value += _velocity * step;
      dt -= step;
    }
    if ((_value - _target).abs() < 1e-3 && _velocity.abs() < 1e-2) {
      _value = _target;
      _velocity = 0;
      _ticker.stop();
    }
    notifyListeners();
  }

  @override
  void dispose() {
    _ticker.dispose();
    super.dispose();
  }
}

class _DockGeometry {
  const _DockGeometry(this.size, this.count, this.rtl);

  final Size size;
  final int count;
  final bool rtl;

  double get pitch => (size.width - _dockInset * 2) / count;

  double centerOf(double position) {
    final double x = _dockInset + (position + 0.5) * pitch;
    return rtl ? size.width - x : x;
  }

  double positionAt(double x) =>
      ((rtl ? size.width - x : x) - _dockInset) / pitch - 0.5;

  /// The lens over [position]: [stretch] lengthens it along its travel and
  /// thins it a little, and [lift] raises it past the dock's rim.
  RRect lens(double position, double stretch, double lift) {
    final double grow = _lensLift * 2 * lift;
    final double width = (pitch + grow) * (1 + stretch);
    final double height =
        (size.height - _dockInset * 2 + grow) * (1 - stretch * 0.3);
    return RRect.fromRectAndRadius(
      Rect.fromCenter(
        center: Offset(centerOf(position), size.height / 2),
        width: width,
        height: height,
      ),
      Radius.circular(height / 2),
    );
  }
}

class _Dock extends StatefulWidget {
  const _Dock({
    required this.items,
    required this.currentIndex,
    required this.onSelected,
    required this.showLabels,
    required this.glass,
  });

  final List<GfBottomNavigationItem> items;
  final int currentIndex;
  final ValueChanged<int> onSelected;
  final bool showLabels;
  final bool glass;

  @override
  State<_Dock> createState() => _DockState();
}

class _DockState extends State<_Dock> with TickerProviderStateMixin {
  late final _Spring _travel = _Spring(this, widget.currentIndex.toDouble())
    ..addListener(_onTravel);
  late final _Spring _stretch = _Spring(this, 0);
  late final _Spring _lift = _Spring(this, 0);
  late final _Spring _press = _Spring(this, 0);
  late final Listenable _lensMotion = Listenable.merge(<Listenable>[
    _travel,
    _stretch,
    _lift,
  ]);
  _DockGeometry _geometry = const _DockGeometry(Size(1, 1), 4, false);
  bool _reduced = false;
  bool _dragging = false;
  // Set while a lifted lens is on its way down to a destination.
  bool _landing = false;
  int _pressedSlot = -1;
  int _dragSlot = 0;

  int get _last => widget.items.length - 1;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _reduced = GfMotion.reducedOf(context);
    if (_reduced) {
      // Reduced motion takes effect at once, even mid-flight.
      _travel.moveTo(_travel.target, immediate: true);
      for (final _Spring spring in <_Spring>[_stretch, _lift, _press]) {
        spring.moveTo(0, immediate: true);
      }
      _landing = false;
    }
  }

  @override
  void didUpdateWidget(_Dock oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!_dragging && _travel.target != widget.currentIndex) {
      _travelTo(widget.currentIndex, lifted: false);
    }
  }

  @override
  void dispose() {
    for (final _Spring spring in <_Spring>[_travel, _stretch, _lift, _press]) {
      spring.dispose();
    }
    super.dispose();
  }

  RRect get _lens => _geometry.lens(_travel.value, _stretch.value, _lift.value);

  void _onTravel() {
    _stretch
      ..tune(320, 0.42)
      ..moveTo(
        _reduced
            ? 0
            : math.min(_maxStretch, _travel.velocity.abs() * _stretchPerSlot),
        immediate: _reduced,
      );
    if (_landing && (_travel.value - _travel.target).abs() < 0.12) {
      _landing = false;
      _lift
        ..tune(260, 1)
        ..moveTo(0, immediate: _reduced);
    }
  }

  void _travelTo(int index, {required bool lifted}) {
    _travel
      ..tune(260, 0.78)
      ..moveTo(index.toDouble(), immediate: _reduced);
    if (lifted && !_reduced) {
      _lift
        ..tune(420, 0.7)
        ..moveTo(1);
      _landing = true;
    }
  }

  /// Follows the parent if it did not take the selection we sent it.
  void _reconcile() {
    SchedulerBinding.instance
      ..addPostFrameCallback((_) {
        if (mounted && !_dragging && _travel.target != widget.currentIndex) {
          _travelTo(widget.currentIndex, lifted: false);
        }
      })
      ..ensureVisualUpdate();
  }

  void _select(int index) {
    if (index != widget.currentIndex) _travelTo(index, lifted: true);
    // Selecting the current destination still reaches the parent, which
    // scrolls it back to the top.
    widget.onSelected(index);
    _reconcile();
  }

  void _pressDown(PointerDownEvent event) {
    _pressedSlot = _geometry
        .positionAt(event.localPosition.dx)
        .round()
        .clamp(0, _last);
    _press
      ..tune(520, 0.62)
      ..moveTo(1, immediate: _reduced);
  }

  void _pressUp([PointerEvent? _]) {
    if (_dragging) return;
    _press
      ..tune(300, 0.8)
      ..moveTo(0, immediate: _reduced);
  }

  void _dragStart(DragStartDetails details) {
    _dragging = true;
    _landing = false;
    _dragSlot = _travel.target.round().clamp(0, _last);
    _lift
      ..tune(420, 0.7)
      ..moveTo(1, immediate: _reduced);
    _press.moveTo(1, immediate: _reduced);
    _follow(details.localPosition.dx);
  }

  void _follow(double x) {
    final double raw = _geometry.positionAt(x);
    final double clamped = raw.clamp(0.0, _last.toDouble());
    final double over = raw - clamped;
    // Past either end the lens resists, approaching [_overdrag] at most.
    final double pull = _overdrag * (1 - 1 / (over.abs() / _overdrag + 1));
    _travel
      ..tune(1100, 0.92)
      ..moveTo(clamped + pull * over.sign, immediate: _reduced);
    final int slot = clamped.round();
    if (slot != _dragSlot) {
      _dragSlot = slot;
      HapticFeedback.selectionClick();
    }
  }

  void _dragEnd(DragEndDetails details) {
    final double velocity = (details.primaryVelocity ?? 0) / _geometry.pitch;
    final double projected =
        _travel.value +
        (_geometry.rtl ? -velocity : velocity) * _flingProjection;
    // A flick carries on at most one destination past the finger.
    _settle(
      projected
          .round()
          .clamp(_dragSlot - 1, _dragSlot + 1)
          .clamp(0, _last)
          .toInt(),
    );
  }

  void _settle(int index) {
    _dragging = false;
    _travel
      ..tune(260, 0.72)
      ..moveTo(index.toDouble(), immediate: _reduced);
    _landing = true;
    // The landing check must run even when moveTo above was a no-op (the
    // travel spring already converged on this destination): otherwise the
    // lift never gets the signal to come back down.
    _onTravel();
    _pressUp();
    if (index != widget.currentIndex) widget.onSelected(index);
    _reconcile();
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        _geometry = _DockGeometry(
          constraints.biggest,
          widget.items.length,
          Directionality.of(context) == TextDirection.rtl,
        );
        final Size size = constraints.biggest;
        return AnimatedBuilder(
          animation: _press,
          builder: (BuildContext context, Widget? child) {
            final double swell = _press.value * _pressSwell * 2;
            // The swell only paints: layout and hit testing see the dock at rest.
            return Transform.scale(
              scaleX: 1 + swell / size.width,
              scaleY: 1 + swell / size.height,
              transformHitTests: false,
              child: child,
            );
          },
          child: Listener(
            onPointerDown: _pressDown,
            onPointerUp: _pressUp,
            onPointerCancel: _pressUp,
            child: GestureDetector(
              behavior: HitTestBehavior.opaque,
              onHorizontalDragStart: _dragStart,
              onHorizontalDragUpdate: (DragUpdateDetails details) =>
                  _follow(details.localPosition.dx),
              onHorizontalDragEnd: _dragEnd,
              onHorizontalDragCancel: () => _settle(widget.currentIndex),
              child: Stack(
                clipBehavior: Clip.none,
                children: <Widget>[
                  Positioned.fill(child: const GfNavigationSurface()),
                  AnimatedBuilder(
                    animation: _lensMotion,
                    builder: (BuildContext context, Widget? child) =>
                        Positioned.fromRect(
                          rect: _lens.outerRect,
                          child: child!,
                        ),
                    child: _Lens(
                      key: const ValueKey<String>('gf-bottom-navigation-lens'),
                      colors: colors,
                      glass: widget.glass,
                      lift: _lift,
                    ),
                  ),
                  Positioned.fill(
                    child: ExcludeSemantics(
                      child: ClipPath(
                        clipper: _LensClipper(this, inside: false),
                        child: _destinations(colors, selected: false),
                      ),
                    ),
                  ),
                  Positioned.fill(
                    child: ExcludeSemantics(
                      child: ClipPath(
                        clipper: _LensClipper(this, inside: true),
                        child: AnimatedBuilder(
                          animation: _lensMotion,
                          builder: (BuildContext context, Widget? child) {
                            // A lifted lens magnifies what it carries.
                            final Offset center = _lens.center;
                            final double scale = 1 + _lensMagnify * _lift.value;
                            return Transform(
                              transform: Matrix4.identity()
                                ..translateByDouble(center.dx, center.dy, 0, 1)
                                ..scaleByDouble(scale, scale, 1, 1)
                                ..translateByDouble(
                                  -center.dx,
                                  -center.dy,
                                  0,
                                  1,
                                ),
                              child: child,
                            );
                          },
                          child: _destinations(colors, selected: true),
                        ),
                      ),
                    ),
                  ),
                  Positioned.fill(child: _targets(colors)),
                ],
              ),
            ),
          ),
        );
      },
    );
  }

  /// Every destination drawn in one state; the lens decides which parts of
  /// each layer show.
  Widget _destinations(GfColors colors, {required bool selected}) => Padding(
    key: ValueKey<String>(
      selected
          ? 'gf-bottom-navigation-selected-layer'
          : 'gf-bottom-navigation-resting-layer',
    ),
    padding: const EdgeInsets.symmetric(horizontal: _dockInset),
    child: Row(
      children: <Widget>[
        for (int i = 0; i < widget.items.length; i++)
          Expanded(
            child: AnimatedBuilder(
              animation: _press,
              builder: (BuildContext context, Widget? child) => Transform.scale(
                // The pressed destination gives a little under the finger.
                scale: i == _pressedSlot && !_dragging
                    ? 1 - 0.06 * _press.value
                    : 1,
                child: child,
              ),
              child: _DestinationContent(
                item: widget.items[i],
                selected: selected,
                showLabel: widget.showLabels,
              ),
            ),
          ),
      ],
    ),
  );

  Widget _targets(GfColors colors) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: _dockInset),
    child: Row(
      children: <Widget>[
        for (int i = 0; i < widget.items.length; i++)
          Expanded(
            child: Semantics(
              button: true,
              enabled: true,
              selected: widget.currentIndex == i,
              label: widget.items[i].semanticsLabel,
              onTap: () => _select(i),
              child: ExcludeSemantics(
                child: _LabelTooltip(
                  label: widget.items[i].label,
                  enabled: !widget.showLabels,
                  child: Material(
                    type: MaterialType.transparency,
                    child: InkWell(
                      key: ValueKey<String>('gf-bottom-navigation-target-$i'),
                      onTap: () => _select(i),
                      borderRadius: BorderRadius.circular(999),
                      splashFactory: NoSplash.splashFactory,
                      highlightColor: Colors.transparent,
                      hoverColor: Colors.transparent,
                      focusColor: colors.primary.withValues(alpha: 0.18),
                      child: _UnreadDot(
                        visible: widget.items[i].badge,
                        showLabel: widget.showLabels,
                        ring: colors.surfaceRaised,
                        color: colors.error,
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
      ],
    ),
  );
}

class _LensClipper extends CustomClipper<Path> {
  _LensClipper(this.dock, {required this.inside})
    : super(reclip: dock._lensMotion);

  final _DockState dock;
  final bool inside;

  @override
  Path getClip(Size size) {
    final Path lens = Path()..addRRect(dock._lens);
    if (inside) return lens;
    return Path.combine(
      PathOperation.difference,
      Path()..addRect((Offset.zero & size).inflate(_lensLift * 4)),
      lens,
    );
  }

  @override
  bool shouldReclip(_LensClipper oldClipper) =>
      oldClipper.dock != dock || oldClipper.inside != inside;
}

class _DestinationContent extends StatelessWidget {
  const _DestinationContent({
    required this.item,
    required this.selected,
    required this.showLabel,
  });

  final GfBottomNavigationItem item;
  final bool selected;
  final bool showLabel;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final Color color = selected ? colors.primary : colors.textSecondary;
    final Widget glyph = GfSymbol(
      selected ? item.selectedSymbol : item.symbol,
      size: GfIconSize.lg,
      color: color,
    );
    if (!showLabel) return Center(child: glyph);
    return Padding(
      padding: const EdgeInsets.fromLTRB(
        GfSpace.xs,
        _dockInset + GfSpace.sm,
        GfSpace.xs,
        0,
      ),
      child: Column(
        children: <Widget>[
          glyph,
          const SizedBox(height: GfSpace.xxs),
          // Both layers share one weight so a lens edge never splits a glyph
          // into two different shapes.
          GfText(
            item.label,
            textAlign: TextAlign.center,
            style: _navigationLabelStyle(context, color: color),
          ),
        ],
      ),
    );
  }
}

/// The unread mark rides above both layers so the lens never tints it, and
/// pops in when a destination gains unread content.
class _UnreadDot extends StatelessWidget {
  const _UnreadDot({
    required this.visible,
    required this.showLabel,
    required this.ring,
    required this.color,
  });

  final bool visible;
  final bool showLabel;
  final Color ring;
  final Color color;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (BuildContext context, BoxConstraints constraints) {
      final double glyphCenter = showLabel
          ? _dockInset + GfSpace.sm + GfIconSize.lg / 2
          : constraints.maxHeight / 2;
      return Stack(
        children: <Widget>[
          Positioned(
            left: constraints.maxWidth / 2 + 4,
            top: glyphCenter - 14,
            child: AnimatedScale(
              scale: visible ? 1 : 0,
              duration: GfMotion.duration(context, GfMotion.layout),
              curve: visible ? Curves.easeOutBack : GfMotion.exitCurve,
              child: Container(
                width: 10,
                height: 10,
                decoration: BoxDecoration(
                  color: color,
                  shape: BoxShape.circle,
                  border: Border.all(color: ring, width: 2),
                ),
              ),
            ),
          ),
        ],
      );
    },
  );
}

/// The resting lens is a soft primary platter; lifted, it clears into glass
/// that catches the light along its top edge.
class _Lens extends StatelessWidget {
  const _Lens({
    super.key,
    required this.colors,
    required this.glass,
    required this.lift,
  });

  final GfColors colors;
  final bool glass;
  final ValueListenable<double> lift;

  @override
  Widget build(BuildContext context) => CustomPaint(
    painter: _LensPainter(colors: colors, glass: glass, lift: lift),
  );
}

class _LensPainter extends CustomPainter {
  _LensPainter({required this.colors, required this.glass, required this.lift})
    : super(repaint: lift);

  final GfColors colors;
  final bool glass;
  final ValueListenable<double> lift;

  @override
  void paint(Canvas canvas, Size size) {
    final double t = lift.value.clamp(0, 1);
    final RRect lens = RRect.fromRectAndRadius(
      Offset.zero & size,
      Radius.circular(size.height / 2),
    );
    final bool dark = colors.isDark;
    if (!glass) {
      canvas.drawRRect(lens, Paint()..color = colors.primarySoft);
      return;
    }
    if (t > 0 && !dark) {
      canvas.drawRRect(
        lens.shift(const Offset(0, 2)),
        Paint()
          ..color = const Color(0xFF0F172A).withValues(alpha: 0.12 * t)
          ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 6),
      );
    }
    final Color resting = colors.primary.withValues(alpha: dark ? 0.2 : 0.11);
    final Color clear = Colors.white.withValues(alpha: dark ? 0.1 : 0.55);
    canvas.drawRRect(lens, Paint()..color = Color.lerp(resting, clear, t)!);
    canvas.drawRRect(
      lens.deflate(0.5),
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1
        ..shader = LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: <Color>[
            Colors.white.withValues(alpha: (dark ? 0.1 : 0.5) + 0.35 * t),
            Colors.white.withValues(alpha: dark ? 0.02 : 0.08),
          ],
        ).createShader(lens.outerRect),
    );
  }

  @override
  bool shouldRepaint(_LensPainter oldDelegate) =>
      oldDelegate.colors != colors ||
      oldDelegate.glass != glass ||
      oldDelegate.lift != lift;
}

/// Icon-only destinations reveal their name on long press; the destination
/// node already carries it for screen readers.
class _LabelTooltip extends StatelessWidget {
  const _LabelTooltip({
    required this.label,
    required this.enabled,
    required this.child,
  });

  final String label;
  final bool enabled;
  final Widget child;

  @override
  Widget build(BuildContext context) => enabled
      ? Tooltip(message: label, excludeFromSemantics: true, child: child)
      : child;
}

/// A compose action that floats beside the dock on the same material.
class _DockAction extends StatefulWidget {
  const _DockAction({
    required this.symbol,
    required this.label,
    required this.onTap,
    required this.showLabel,
    required this.width,
  });

  final String symbol;
  final String label;
  final VoidCallback onTap;
  final bool showLabel;
  final double width;

  @override
  State<_DockAction> createState() => _DockActionState();
}

class _DockActionState extends State<_DockAction> {
  bool _pressed = false;

  void _setPressed(bool pressed) {
    if (_pressed != pressed) setState(() => _pressed = pressed);
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final double disc = widget.showLabel
        ? GfControl.compact
        : _dockHeight - _dockInset * 2;
    return SizedBox(
      width: widget.width,
      child: Semantics(
        button: true,
        enabled: true,
        label: widget.label,
        onTap: widget.onTap,
        child: ExcludeSemantics(
          child: _LabelTooltip(
            label: widget.label,
            enabled: !widget.showLabel,
            child: AnimatedScale(
              scale: _pressed ? 0.94 : 1,
              duration: GfMotion.duration(context, GfMotion.press),
              curve: GfMotion.enterCurve,
              child: Stack(
                children: <Widget>[
                  Positioned.fill(child: const GfNavigationSurface()),
                  Center(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: <Widget>[
                        DecoratedBox(
                          decoration: BoxDecoration(
                            color: colors.primary,
                            shape: BoxShape.circle,
                          ),
                          child: SizedBox.square(
                            dimension: disc,
                            child: GfSymbol(
                              widget.symbol,
                              size: widget.showLabel
                                  ? GfIconSize.md
                                  : GfIconSize.lg,
                              color: colors.primaryContent,
                            ),
                          ),
                        ),
                        if (widget.showLabel) ...<Widget>[
                          const SizedBox(height: GfSpace.xxs),
                          GfText(
                            widget.label,
                            textAlign: TextAlign.center,
                            style: _navigationLabelStyle(
                              context,
                              color: colors.primary,
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                  // Paint keyboard focus above the glass and action glyph.
                  Positioned.fill(
                    child: Material(
                      type: MaterialType.transparency,
                      child: InkWell(
                        onTapDown: (_) => _setPressed(true),
                        onTapUp: (_) => _setPressed(false),
                        onTapCancel: () => _setPressed(false),
                        onTap: widget.onTap,
                        borderRadius: BorderRadius.circular(999),
                        splashFactory: NoSplash.splashFactory,
                        highlightColor: Colors.transparent,
                        hoverColor: Colors.transparent,
                        focusColor: colors.primary.withValues(alpha: 0.18),
                      ),
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

TextStyle _navigationLabelStyle(BuildContext context, {required Color color}) {
  final TextStyle materialStyle =
      Theme.of(context).textTheme.bodyMedium ?? const TextStyle();
  return materialStyle.merge(
    GfTheme.typographyOf(context).meta.copyWith(
      color: color,
      fontWeight: MediaQuery.boldTextOf(context)
          ? FontWeight.w700
          : FontWeight.w600,
    ),
  );
}
