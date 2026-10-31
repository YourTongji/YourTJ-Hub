import 'dart:async';
import 'dart:math' as math;
import 'dart:ui' show ImageFilter;

import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart' show Ticker;
import 'package:flutter/services.dart';

import '../theme/gf_theme.dart';
import '../twemoji/twemoji_text.dart';
import 'gf_motion.dart';
import 'gf_symbol.dart';
import 'surfaces/gf_navigation_surface.dart';

/// One shortcut in a dock destination's long-press menu.
@immutable
class GfDockMenuEntry {
  const GfDockMenuEntry({
    required this.label,
    required this.symbol,
    required this.onSelected,
    this.selected = false,
    this.opensPage = false,
    this.group = 0,
  });

  final String label;
  final String symbol;
  final VoidCallback onSelected;

  /// The section the destination currently shows.
  final bool selected;

  /// Marks a shortcut that pushes a page instead of switching a section.
  final bool opensPage;

  /// Adjacent entries with different groups are set apart.
  final int group;
}

/// Drives an open dock menu from the gesture that opened it, so a press can
/// slide straight onto a shortcut and select it on release.
class GfDockMenuHandle {
  GfDockMenuHandle._();

  final Completer<void> _closed = Completer<void>();
  _DockMenuState? _menu;

  /// Completes once the menu has been dismissed.
  Future<void> get closed => _closed.future;

  void track(Offset globalPosition) => _menu?.track(globalPosition);

  void release(Offset globalPosition) => _menu?.release(globalPosition);

  void close() => _menu?.close();
}

/// The menu stays above a bottom dock, or beside a vertical navigation rail.
enum GfDockMenuPlacement { above, beside }

/// The same held gesture opens and selects shortcuts on a dock or a rail.
/// Movement before recognition stays available to the parent's scrolling or scrub.
class GfDockMenuPress extends StatelessWidget {
  const GfDockMenuPress({
    super.key,
    required this.enabled,
    required this.onOpen,
    required this.onMove,
    required this.onRelease,
    required this.child,
  });

  final bool enabled;
  final VoidCallback onOpen;
  final ValueChanged<Offset> onMove;
  final ValueChanged<Offset> onRelease;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    if (!enabled) return child;
    return RawGestureDetector(
      gestures: <Type, GestureRecognizerFactory>{
        LongPressGestureRecognizer:
            GestureRecognizerFactoryWithHandlers<LongPressGestureRecognizer>(
              () => LongPressGestureRecognizer(
                duration: const Duration(milliseconds: 320),
              ),
              (LongPressGestureRecognizer recognizer) => recognizer
                ..onLongPressStart = ((_) => onOpen())
                ..onLongPressMoveUpdate = ((LongPressMoveUpdateDetails d) =>
                    onMove(d.globalPosition))
                ..onLongPressEnd = ((LongPressEndDetails d) =>
                    onRelease(d.globalPosition)),
            ),
      },
      child: child,
    );
  }
}

/// Opens [entries] from [source] (a destination's
/// global box). [dock] stays clear of the scrim so the pressed destination
/// keeps its place while everything else recedes.
///
/// The menu is an overlay rather than a route: pushing a route cancels the
/// pointer that is still holding the destination, and that press is what
/// slides onto a shortcut.
GfDockMenuHandle showGfDockMenu(
  BuildContext context, {
  required List<GfDockMenuEntry> entries,
  required Rect source,
  required RRect dock,
  required String label,
  GfDockMenuPlacement placement = GfDockMenuPlacement.above,
}) {
  final GfDockMenuHandle handle = GfDockMenuHandle._();
  late final OverlayEntry entry;
  entry = OverlayEntry(
    builder: (BuildContext context) => _DockMenu(
      handle: handle,
      entries: entries,
      source: source,
      dock: dock,
      label: label,
      placement: placement,
      onClosed: () {
        if (handle._closed.isCompleted) return;
        entry.remove();
        entry.dispose();
        if (!handle._closed.isCompleted) handle._closed.complete();
      },
    ),
  );
  Overlay.of(context, rootOverlay: true).insert(entry);
  return handle;
}

// Row and panel geometry. The highlight sits [_panelPadding] inside the panel
// rim, so the two stay concentric.
const double _panelPadding = 6;
const double _highlightRadius = 14;
const double _panelRadius = _highlightRadius + _panelPadding;
const double _rowMinHeight = 44;
const double _rowInset = 12;
const double _iconGap = 12;
const double _trailing = 16;
const double _groupGap = 9;
const double _sourceGap = 12;
const double _edge = 16;
const double _minWidth = 196;
const double _maxWidth = 280;
// Holding the opening press this close to the edge of a panel whose rows
// overflow scrolls them, at up to [_edgeScrollSpeed] logical pixels a second.
const double _edgeScrollZone = 44;
const double _edgeScrollSpeed = 720;

class _DockMenu extends StatefulWidget {
  const _DockMenu({
    required this.handle,
    required this.entries,
    required this.source,
    required this.dock,
    required this.label,
    required this.placement,
    required this.onClosed,
  });

  final GfDockMenuHandle handle;
  final List<GfDockMenuEntry> entries;
  final Rect source;
  final RRect dock;
  final String label;
  final GfDockMenuPlacement placement;
  final VoidCallback onClosed;

  @override
  State<_DockMenu> createState() => _DockMenuState();
}

class _DockMenuState extends State<_DockMenu> with TickerProviderStateMixin {
  late final AnimationController _animation = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 320),
    reverseDuration: GfMotion.content,
  );
  final ValueNotifier<int> hovered = ValueNotifier<int>(-1);
  final FocusScopeNode _focus = FocusScopeNode(debugLabel: 'gf-dock-menu');
  _DockMenuLayout? _layout;
  bool _closing = false;
  bool _anchorInvalid = false;
  MediaQueryData? _window;
  // Rows that do not fit above the dock scroll; null while they all fit.
  ScrollController? _scroll;
  late final Ticker _edgeScroll = createTicker(_scrollTowardEdge);
  Duration _edgeElapsed = Duration.zero;
  double _edgeSpeed = 0;
  Offset? _finger;
  double _pressOffset = 0;

  List<GfDockMenuEntry> get entries => widget.entries;

  @override
  void initState() {
    super.initState();
    widget.handle._menu = this;
    // Autofocus would defer to the page that already holds focus; the menu
    // takes it so Escape and the arrow keys reach it.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && !_closing) _focus.requestFocus();
    });
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final window = MediaQuery.of(context);
    final previous = _window;
    _window = window;
    // A fold, rotation or keyboard changes the source's global coordinates.
    // Cancel before another pointer-up can select against that old snapshot.
    if (previous != null &&
        (previous.size != window.size ||
            previous.padding != window.padding ||
            previous.viewPadding != window.viewPadding ||
            previous.viewInsets != window.viewInsets ||
            previous.textScaler != window.textScaler ||
            !listEquals(previous.displayFeatures, window.displayFeatures))) {
      cancelForLayoutChange();
      return;
    }
    if (_animation.status == AnimationStatus.dismissed && !_closing) {
      if (GfMotion.reducedOf(context)) {
        _animation.value = 1;
      } else {
        _animation.forward();
      }
    }
  }

  void cancelForLayoutChange() {
    if (_anchorInvalid) return;
    _closing = true;
    _anchorInvalid = true;
    _animation.stop();
    _stopEdge();
    // Overlay removal cannot mark an ancestor dirty during a rebuild.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) widget.onClosed();
    });
  }

  @override
  void dispose() {
    if (identical(widget.handle._menu, this)) widget.handle._menu = null;
    _animation.dispose();
    _edgeScroll.dispose();
    _scroll?.dispose();
    hovered.dispose();
    _focus.dispose();
    super.dispose();
  }

  double get _offset {
    final ScrollController? scroll = _scroll;
    return scroll != null && scroll.hasClients ? scroll.offset : 0;
  }

  /// The row under [local], a point in the panel's own coordinates.
  int _rowAtPanel(Offset local) {
    final _DockMenuLayout? layout = _layout;
    if (layout == null || local.dy < 0 || local.dy > layout.panel.height) {
      return -1;
    }
    return layout.rowAt(local.translate(0, _offset));
  }

  int _rowAt(Offset globalPosition) {
    final _DockMenuLayout? layout = _layout;
    if (layout == null) return -1;
    return _rowAtPanel(globalPosition - layout.toGlobal - layout.panel.topLeft);
  }

  void hover(int index, {required bool haptic}) {
    if (_closing || hovered.value == index) return;
    hovered.value = index;
    if (index >= 0 && haptic) HapticFeedback.selectionClick();
  }

  void track(Offset globalPosition) {
    _finger = globalPosition;
    _followEdge(globalPosition);
    hover(_rowAt(globalPosition), haptic: true);
  }

  /// While the opening press holds near the top or bottom of rows that
  /// overflow, scrolls them toward that edge, faster the closer it gets.
  void _followEdge(Offset globalPosition) {
    final _DockMenuLayout? layout = _layout;
    double speed = 0;
    if (layout != null && layout.scrolls && !_closing) {
      final Offset local =
          globalPosition - layout.toGlobal - layout.panel.topLeft;
      final double bottom = layout.panel.height;
      if (local.dx >= 0 && local.dx <= layout.panel.width) {
        if (local.dy > -_edgeScrollZone && local.dy < _edgeScrollZone) {
          speed = -((_edgeScrollZone - local.dy) / _edgeScrollZone).clamp(
            0.0,
            1.0,
          );
        } else if (local.dy > bottom - _edgeScrollZone && local.dy <= bottom) {
          speed = (local.dy - bottom + _edgeScrollZone) / _edgeScrollZone;
        }
      }
    }
    _edgeSpeed = speed * _edgeScrollSpeed;
    if (_edgeSpeed == 0) {
      _edgeScroll.stop();
    } else if (!_edgeScroll.isActive) {
      _edgeElapsed = Duration.zero;
      _edgeScroll.start();
    }
  }

  void _scrollTowardEdge(Duration elapsed) {
    final double seconds =
        (elapsed - _edgeElapsed).inMicroseconds /
        Duration.microsecondsPerSecond;
    _edgeElapsed = elapsed;
    if (seconds == 0) return;
    final ScrollController? scroll = _scroll;
    if (scroll == null || !scroll.hasClients) return _edgeScroll.stop();
    final ScrollPosition position = scroll.position;
    final double next = (position.pixels + _edgeSpeed * seconds).clamp(
      position.minScrollExtent,
      position.maxScrollExtent,
    );
    if (next == position.pixels) return _edgeScroll.stop();
    position.jumpTo(next);
    final Offset? finger = _finger;
    if (finger != null) hover(_rowAt(finger), haptic: true);
  }

  void _stopEdge() {
    _edgeSpeed = 0;
    _edgeScroll.stop();
  }

  /// Releasing over a shortcut picks it; anywhere else leaves the menu open
  /// for a tap, as a press that only opened it should.
  void release(Offset globalPosition) {
    _stopEdge();
    final int index = _rowAt(globalPosition);
    if (index < 0) {
      hover(-1, haptic: false);
      return;
    }
    select(index);
  }

  void pressRow(Offset local) {
    _pressOffset = _offset;
    hover(_rowAtPanel(local), haptic: false);
  }

  void moveRow(Offset local) => hover(_rowAtPanel(local), haptic: true);

  /// A lift picks the row under it, unless the press scrolled the rows.
  void liftRow(Offset local) {
    final int index = _rowAtPanel(local);
    if (index >= 0 && (_offset - _pressOffset).abs() < 1) {
      select(index);
    } else {
      hover(-1, haptic: false);
    }
  }

  void select(int index) {
    if (_closing) return;
    hovered.value = index;
    close();
    entries[index].onSelected();
  }

  void close() {
    if (_closing) return;
    _closing = true;
    _stopEdge();
    if (GfMotion.reducedOf(context)) {
      widget.onClosed();
      return;
    }
    _animation.reverse().whenComplete(widget.onClosed);
  }

  @override
  Widget build(BuildContext context) {
    if (_anchorInvalid) return const SizedBox.shrink();
    final Widget menu = CallbackShortcuts(
      bindings: <ShortcutActivator, VoidCallback>{
        const SingleActivator(LogicalKeyboardKey.escape): close,
      },
      child: FocusScope(
        node: _focus,
        child: _DockMenuView(state: this, animation: _animation),
      ),
    );
    final Router<Object?>? router = Router.maybeOf(context);
    return BlockSemantics(
      child: router?.backButtonDispatcher == null
          ? menu
          : BackButtonListener(
              onBackButtonPressed: () async {
                close();
                return true;
              },
              child: menu,
            ),
    );
  }
}

/// Every row's place, fixed before the first frame so the bloom can start at
/// the destination and a finger can be hit-tested without waiting for layout.
class _DockMenuLayout {
  _DockMenuLayout({
    required this.panel,
    required this.content,
    required this.rows,
    required this.origin,
    required this.toGlobal,
    required this.initialOffset,
  });

  /// The visible panel; shorter than [content] when the rows scroll.
  final Rect panel;
  final double content;

  /// Each row in content coordinates.
  final List<Rect> rows;
  final Offset origin;
  final Offset toGlobal;

  /// Scrolls the current section into view when the rows overflow.
  final double initialOffset;

  bool get scrolls => content > panel.height;

  /// The row under [point] in content coordinates. The panel's rim and the
  /// gaps between groups belong to the nearest row, so a finger sliding from
  /// row to row never falls between them.
  int rowAt(Offset point) {
    if (point.dx < 0 ||
        point.dx > panel.width ||
        point.dy < 0 ||
        point.dy > content) {
      return -1;
    }
    for (int i = 0; i < rows.length - 1; i++) {
      if (point.dy < (rows[i].bottom + rows[i + 1].top) / 2) return i;
    }
    return rows.length - 1;
  }
}

double _menuIconSize(BuildContext context) =>
    MediaQuery.textScalerOf(context).scale(GfIconSize.md).clamp(20.0, 28.0);

class _DockMenuView extends StatelessWidget {
  const _DockMenuView({required this.state, required this.animation});

  final _DockMenuState state;
  final Animation<double> animation;

  _DockMenuLayout? _measure(
    BuildContext context,
    Size screen,
    Offset toGlobal,
  ) {
    final TextScaler scaler = MediaQuery.textScalerOf(context);
    final EdgeInsets padding = MediaQuery.paddingOf(context);
    final TextStyle style = GfTheme.typographyOf(context).body;
    double textWidth = 0;
    final List<double> heights = <double>[];
    for (final GfDockMenuEntry entry in state.entries) {
      final TextPainter painter = TextPainter(
        text: TextSpan(text: entry.label, style: style),
        textDirection: Directionality.of(context),
        textScaler: scaler,
        maxLines: 1,
      )..layout();
      textWidth = math.max(textWidth, painter.width);
      heights.add(math.max(_rowMinHeight, painter.height + 20));
      painter.dispose();
    }
    final Rect source = state.widget.source.shift(-toGlobal);
    final Rect dock = state.widget.dock.outerRect.shift(-toGlobal);
    final bool beside = state.widget.placement == GfDockMenuPlacement.beside;
    final bool toRight = dock.center.dx < screen.width / 2;
    final double minLeft = beside && toRight
        ? math.max(padding.left + _edge, dock.right + _sourceGap)
        : padding.left + _edge;
    final double maxRight = beside && !toRight
        ? math.min(screen.width - padding.right - _edge, dock.left - _sourceGap)
        : screen.width - padding.right - _edge;
    final double availableWidth = maxRight - minLeft;
    if (availableWidth < _minWidth) return null;
    final double width =
        (_panelPadding * 2 +
                _rowInset * 2 +
                _menuIconSize(context) +
                _iconGap * 2 +
                textWidth +
                _trailing)
            .clamp(
              math.min(_minWidth, availableWidth),
              math.min(_maxWidth, availableWidth),
            );
    final List<Rect> rows = <Rect>[];
    double y = _panelPadding;
    for (int i = 0; i < state.entries.length; i++) {
      if (i > 0 && state.entries[i].group != state.entries[i - 1].group) {
        y += _groupGap;
      }
      rows.add(
        Rect.fromLTWH(_panelPadding, y, width - _panelPadding * 2, heights[i]),
      );
      y += heights[i];
    }
    final double content = y + _panelPadding;
    final double maxBottom = beside
        ? screen.height -
              math.max(
                padding.bottom,
                MediaQuery.viewInsetsOf(context).bottom,
              ) -
              _edge
        : dock.top - _sourceGap;
    // Side menus use the content's full safe height; bottom menus stop at the dock.
    final double height = math.min(
      content,
      math.max(
        maxBottom - padding.top - _edge,
        _rowMinHeight + _panelPadding * 2,
      ),
    );
    final int current = state.entries.indexWhere(
      (GfDockMenuEntry entry) => entry.selected,
    );
    final double left =
        (beside
                ? (toRight ? minLeft : maxRight - width)
                : source.center.dx - width / 2)
            .clamp(minLeft, math.max(minLeft, maxRight - width));
    final double top =
        (beside ? source.center.dy - height / 2 : maxBottom - height).clamp(
          padding.top + _edge,
          math.max(padding.top + _edge, maxBottom - height),
        );
    return _DockMenuLayout(
      panel: Rect.fromLTWH(left, top, width, height),
      content: content,
      rows: rows,
      origin: beside
          ? Offset(toRight ? dock.right : dock.left, source.center.dy)
          : Offset(source.center.dx, dock.top),
      toGlobal: toGlobal,
      initialOffset: current < 0
          ? 0
          : (rows[current].center.dy - height / 2).clamp(0, content - height),
    );
  }

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (BuildContext context, BoxConstraints constraints) {
      // The overlay's origin is the screen's unless the app is embedded in
      // a larger window.
      final RenderObject? overlay = Overlay.of(
        context,
        rootOverlay: true,
      ).context.findRenderObject();
      final Offset toGlobal = overlay is RenderBox && overlay.hasSize
          ? overlay.localToGlobal(Offset.zero)
          : Offset.zero;
      final _DockMenuLayout? layout = state._layout = _measure(
        context,
        constraints.biggest,
        toGlobal,
      );
      if (layout == null) {
        state.cancelForLayoutChange();
        return const SizedBox.shrink();
      }
      if (layout.scrolls) {
        state._scroll ??= ScrollController(
          initialScrollOffset: layout.initialOffset,
        );
      }
      final Rect panel = layout.panel;
      final Alignment origin = Alignment(
        (layout.origin.dx - panel.left) / panel.width * 2 - 1,
        (layout.origin.dy - panel.top) / panel.height * 2 - 1,
      );
      return Stack(
        children: <Widget>[
          Positioned.fill(
            child: Semantics(
              label: MaterialLocalizations.of(context).modalBarrierDismissLabel,
              button: true,
              onTap: state.close,
              child: GestureDetector(
                behavior: HitTestBehavior.opaque,
                onTap: state.close,
              ),
            ),
          ),
          Positioned.fill(
            child: IgnorePointer(
              child: _Scrim(
                animation: animation,
                hole: state.widget.dock.shift(-toGlobal),
              ),
            ),
          ),
          Positioned.fromRect(
            rect: panel,
            child: AnimatedBuilder(
              animation: animation,
              builder: (BuildContext context, Widget? child) {
                final bool leaving =
                    animation.status == AnimationStatus.reverse;
                final double t = leaving
                    ? Curves.easeOutCubic.transform(animation.value)
                    : GfMotion.enterCurve.transform(animation.value);
                return IgnorePointer(
                  ignoring: leaving,
                  child: Opacity(
                    opacity: math.min(1, t * 1.6),
                    child: Transform.scale(
                      // Grows out of the destination; folds back softer.
                      scale: leaving ? 0.85 + 0.15 * t : 0.45 + 0.55 * t,
                      alignment: origin,
                      child: child,
                    ),
                  ),
                );
              },
              child: _Panel(state: state, layout: layout),
            ),
          ),
        ],
      );
    },
  );
}

/// Dims and, on glass, frosts everything but the dock.
class _Scrim extends StatelessWidget {
  const _Scrim({required this.animation, required this.hole});

  final Animation<double> animation;
  final RRect hole;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final bool glass = GfNavigationSurface.usesGlass(context);
    return AnimatedBuilder(
      animation: animation,
      builder: (BuildContext context, _) {
        final double t = animation.value.clamp(0.0, 1.0);
        final Widget dim = CustomPaint(
          painter: _ScrimPainter(
            color: glass
                ? Colors.black.withValues(
                    alpha: (colors.isDark ? 0.45 : 0.16) * t,
                  )
                : colors.scrim.withValues(alpha: colors.scrim.a * t),
            hole: hole,
          ),
          child: const SizedBox.expand(),
        );
        if (!glass || t == 0) return dim;
        return ClipPath(
          clipper: _HoleClipper(hole),
          child: BackdropFilter(
            filter: ImageFilter.blur(sigmaX: 6 * t, sigmaY: 6 * t),
            child: dim,
          ),
        );
      },
    );
  }
}

Path _withoutHole(Size size, RRect hole) => Path.combine(
  PathOperation.difference,
  Path()..addRect(Offset.zero & size),
  Path()..addRRect(hole),
);

class _ScrimPainter extends CustomPainter {
  const _ScrimPainter({required this.color, required this.hole});

  final Color color;
  final RRect hole;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.drawPath(_withoutHole(size, hole), Paint()..color = color);
  }

  @override
  bool shouldRepaint(_ScrimPainter oldDelegate) =>
      oldDelegate.color != color || oldDelegate.hole != hole;
}

class _HoleClipper extends CustomClipper<Path> {
  const _HoleClipper(this.hole);

  final RRect hole;

  @override
  Path getClip(Size size) => _withoutHole(size, hole);

  @override
  bool shouldReclip(_HoleClipper oldClipper) => oldClipper.hole != hole;
}

/// The rows on the dock's glass, with one highlight that follows the finger
/// from row to row the way the dock's lens follows it between destinations.
class _Panel extends StatelessWidget {
  const _Panel({required this.state, required this.layout});

  final _DockMenuState state;
  final _DockMenuLayout layout;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final List<GfDockMenuEntry> entries = state.entries;
    final Color highlight = colors.baseContent.withValues(
      alpha: colors.isDark ? 0.12 : 0.07,
    );
    final Widget rows = SizedBox(
      width: layout.panel.width,
      height: layout.content,
      child: Stack(
        children: <Widget>[
          ValueListenableBuilder<int>(
            valueListenable: state.hovered,
            builder: (BuildContext context, int hovered, _) =>
                _Highlight(row: hovered, rows: layout.rows, color: highlight),
          ),
          for (int i = 0; i < entries.length; i++) ...<Widget>[
            if (i > 0 && entries[i].group != entries[i - 1].group)
              Positioned(
                left: _panelPadding + _rowInset,
                right: _panelPadding + _rowInset,
                top: layout.rows[i].top - (_groupGap + 1) / 2,
                height: 1,
                child: ColoredBox(
                  color: colors.separator.withValues(alpha: 0.6),
                ),
              ),
            Positioned.fromRect(
              rect: layout.rows[i],
              child: _Row(
                key: ValueKey<String>('gf-dock-menu-row-$i'),
                entry: entries[i],
                onFocus: () => state.hover(i, haptic: false),
                onActivate: () => state.select(i),
              ),
            ),
          ],
        ],
      ),
    );
    return Semantics(
      scopesRoute: true,
      namesRoute: true,
      explicitChildNodes: true,
      label: state.widget.label,
      child: Material(
        key: const ValueKey<String>('gf-dock-menu-panel'),
        type: MaterialType.transparency,
        child: Listener(
          behavior: HitTestBehavior.opaque,
          onPointerDown: (PointerDownEvent event) =>
              state.pressRow(event.localPosition),
          onPointerMove: (PointerMoveEvent event) =>
              state.moveRow(event.localPosition),
          onPointerUp: (PointerUpEvent event) =>
              state.liftRow(event.localPosition),
          onPointerCancel: (_) => state.hover(-1, haptic: false),
          child: Stack(
            children: <Widget>[
              const Positioned.fill(
                child: GfNavigationSurface(
                  borderRadius: BorderRadius.all(Radius.circular(_panelRadius)),
                ),
              ),
              Positioned.fill(
                child: layout.scrolls
                    ? ClipRRect(
                        borderRadius: const BorderRadius.all(
                          Radius.circular(_panelRadius),
                        ),
                        child: SingleChildScrollView(
                          controller: state._scroll,
                          child: rows,
                        ),
                      )
                    : rows,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// The pressed row's tint. It slides from row to row while a finger stays on
/// the rows, fades where it is when the finger leaves, and comes back under
/// the finger instead of travelling from the row it last left.
class _Highlight extends StatefulWidget {
  const _Highlight({
    required this.row,
    required this.rows,
    required this.color,
  });

  /// The pressed row, or -1.
  final int row;
  final List<Rect> rows;
  final Color color;

  @override
  State<_Highlight> createState() => _HighlightState();
}

class _HighlightState extends State<_Highlight>
    with SingleTickerProviderStateMixin {
  late final AnimationController _slide = AnimationController(
    vsync: this,
    value: 1,
  );
  late int _shown = math.max(0, widget.row);
  Rect? _from;

  Rect get _rect {
    final Rect to = widget.rows[_shown];
    final Rect? from = _from;
    if (from == null) return to;
    return Rect.lerp(from, to, GfMotion.enterCurve.transform(_slide.value))!;
  }

  @override
  void didUpdateWidget(_Highlight oldWidget) {
    super.didUpdateWidget(oldWidget);
    final int row = widget.row;
    if (row < 0 || row == oldWidget.row) return;
    final Duration duration = GfMotion.duration(context, GfMotion.press);
    if (oldWidget.row < 0 || duration == Duration.zero) {
      _from = null;
      _slide.value = 1;
    } else {
      _from = _rect;
      _slide.duration = duration;
      _slide.forward(from: 0);
    }
    _shown = row;
  }

  @override
  void dispose() {
    _slide.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AnimatedBuilder(
    animation: _slide,
    builder: (BuildContext context, Widget? child) =>
        Positioned.fromRect(rect: _rect, child: child!),
    child: AnimatedOpacity(
      key: const ValueKey<String>('gf-dock-menu-highlight'),
      opacity: widget.row < 0 ? 0 : 1,
      duration: GfMotion.duration(context, GfMotion.press),
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: widget.color,
          borderRadius: BorderRadius.circular(_highlightRadius),
        ),
      ),
    ),
  );
}

class _Row extends StatelessWidget {
  const _Row({
    super.key,
    required this.entry,
    required this.onFocus,
    required this.onActivate,
  });

  final GfDockMenuEntry entry;
  final VoidCallback onFocus;
  final VoidCallback onActivate;

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    return Semantics(
      button: true,
      selected: entry.selected,
      label: entry.label,
      onTap: onActivate,
      child: ExcludeSemantics(
        child: FocusableActionDetector(
          // Keyboard focus moves the same highlight a finger does.
          onFocusChange: (bool focused) {
            if (focused) onFocus();
          },
          actions: <Type, Action<Intent>>{
            ActivateIntent: CallbackAction<ActivateIntent>(
              onInvoke: (_) {
                onActivate();
                return null;
              },
            ),
          },
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: _rowInset),
            child: Row(
              children: <Widget>[
                GfSymbol(
                  entry.symbol,
                  size: _menuIconSize(context),
                  color: entry.selected ? colors.primary : colors.textSecondary,
                ),
                const SizedBox(width: _iconGap),
                Expanded(
                  child: GfText(
                    entry.label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: GfTheme.typographyOf(context).body.copyWith(
                      color: entry.selected
                          ? colors.primary
                          : colors.textPrimary,
                      fontWeight: entry.selected
                          ? FontWeight.w600
                          : FontWeight.w400,
                    ),
                  ),
                ),
                const SizedBox(width: _iconGap),
                SizedBox.square(
                  dimension: _trailing,
                  child: entry.selected
                      ? GfSymbol(
                          'check',
                          size: _trailing,
                          color: colors.primary,
                        )
                      : entry.opensPage
                      ? GfSymbol(
                          'chevron-right',
                          size: _trailing,
                          color: colors.iconMuted,
                        )
                      : null,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
