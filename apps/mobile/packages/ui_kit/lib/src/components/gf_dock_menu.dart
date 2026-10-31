import 'dart:async';
import 'dart:math' as math;
import 'dart:ui' show ImageFilter;

import 'package:flutter/material.dart';
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

/// Opens [entries] above the dock, blooming out of [source] (a destination's
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
      onClosed: () {
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

class _DockMenu extends StatefulWidget {
  const _DockMenu({
    required this.handle,
    required this.entries,
    required this.source,
    required this.dock,
    required this.label,
    required this.onClosed,
  });

  final GfDockMenuHandle handle;
  final List<GfDockMenuEntry> entries;
  final Rect source;
  final RRect dock;
  final String label;
  final VoidCallback onClosed;

  @override
  State<_DockMenu> createState() => _DockMenuState();
}

class _DockMenuState extends State<_DockMenu>
    with SingleTickerProviderStateMixin {
  late final AnimationController _animation = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 320),
    reverseDuration: GfMotion.content,
  );
  final ValueNotifier<int> hovered = ValueNotifier<int>(-1);
  final FocusScopeNode _focus = FocusScopeNode(debugLabel: 'gf-dock-menu');
  _DockMenuLayout? _layout;
  bool _closing = false;

  List<GfDockMenuEntry> get entries => widget.entries;

  @override
  void initState() {
    super.initState();
    widget.handle._menu = this;
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_animation.status == AnimationStatus.dismissed && !_closing) {
      if (GfMotion.reducedOf(context)) {
        _animation.value = 1;
      } else {
        _animation.forward();
      }
    }
  }

  @override
  void dispose() {
    if (identical(widget.handle._menu, this)) widget.handle._menu = null;
    _animation.dispose();
    hovered.dispose();
    _focus.dispose();
    super.dispose();
  }

  int _rowAt(Offset globalPosition) => _layout?.rowAt(globalPosition) ?? -1;

  void hover(int index, {required bool haptic}) {
    if (_closing || hovered.value == index) return;
    hovered.value = index;
    if (index >= 0 && haptic) HapticFeedback.selectionClick();
  }

  void track(Offset globalPosition) =>
      hover(_rowAt(globalPosition), haptic: true);

  /// Releasing over a shortcut picks it; anywhere else leaves the menu open
  /// for a tap, as a press that only opened it should.
  void release(Offset globalPosition) {
    final int index = _rowAt(globalPosition);
    if (index < 0) {
      hover(-1, haptic: false);
      return;
    }
    select(index);
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
    if (GfMotion.reducedOf(context)) {
      widget.onClosed();
      return;
    }
    _animation.reverse().whenComplete(widget.onClosed);
  }

  @override
  Widget build(BuildContext context) {
    final Widget menu = CallbackShortcuts(
      bindings: <ShortcutActivator, VoidCallback>{
        const SingleActivator(LogicalKeyboardKey.escape): close,
      },
      child: FocusScope(
        node: _focus,
        autofocus: true,
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
    required this.rows,
    required this.origin,
    required this.toGlobal,
  });

  final Rect panel;
  final List<Rect> rows;
  final Offset origin;
  final Offset toGlobal;

  int rowAt(Offset globalPosition) {
    final Offset local = globalPosition - toGlobal - panel.topLeft;
    for (int i = 0; i < rows.length; i++) {
      if (rows[i].contains(local)) return i;
    }
    return -1;
  }
}

double _menuIconSize(BuildContext context) =>
    MediaQuery.textScalerOf(context).scale(GfIconSize.md).clamp(20.0, 28.0);

class _DockMenuView extends StatelessWidget {
  const _DockMenuView({required this.state, required this.animation});

  final _DockMenuState state;
  final Animation<double> animation;

  _DockMenuLayout _measure(BuildContext context, Size screen, Offset toGlobal) {
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
    final double width =
        (_panelPadding * 2 +
                _rowInset * 2 +
                _menuIconSize(context) +
                _iconGap * 2 +
                textWidth +
                _trailing)
            .clamp(_minWidth, math.min(_maxWidth, screen.width - _edge * 2))
            .ceilToDouble();
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
    final double height = y + _panelPadding;
    final Rect source = state.widget.source.shift(-toGlobal);
    final Rect dock = state.widget.dock.outerRect.shift(-toGlobal);
    final double minLeft = _edge + padding.left;
    final double left = (source.center.dx - width / 2).clamp(
      minLeft,
      math.max(minLeft, screen.width - _edge - padding.right - width),
    );
    final double top = math.max(
      padding.top + _edge,
      dock.top - _sourceGap - height,
    );
    return _DockMenuLayout(
      panel: Rect.fromLTWH(left, top, width, height),
      rows: rows,
      origin: Offset(source.center.dx, dock.top),
      toGlobal: toGlobal,
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
      final _DockMenuLayout layout = state._layout = _measure(
        context,
        constraints.biggest,
        toGlobal,
      );
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

  int _rowAt(Offset local) {
    for (int i = 0; i < layout.rows.length; i++) {
      if (layout.rows[i].contains(local)) return i;
    }
    return -1;
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final List<GfDockMenuEntry> entries = state.entries;
    final Color highlight = colors.baseContent.withValues(
      alpha: colors.isDark ? 0.12 : 0.07,
    );
    return Semantics(
      scopesRoute: true,
      namesRoute: true,
      explicitChildNodes: true,
      label: state.widget.label,
      child: Material(
        type: MaterialType.transparency,
        child: Listener(
          behavior: HitTestBehavior.opaque,
          onPointerDown: (PointerDownEvent event) =>
              state.hover(_rowAt(event.localPosition), haptic: false),
          onPointerMove: (PointerMoveEvent event) =>
              state.hover(_rowAt(event.localPosition), haptic: true),
          onPointerUp: (PointerUpEvent event) {
            final int index = _rowAt(event.localPosition);
            if (index >= 0) {
              state.select(index);
            } else {
              state.hover(-1, haptic: false);
            }
          },
          onPointerCancel: (_) => state.hover(-1, haptic: false),
          child: Stack(
            children: <Widget>[
              const Positioned.fill(
                child: GfNavigationSurface(
                  borderRadius: BorderRadius.all(Radius.circular(_panelRadius)),
                ),
              ),
              ValueListenableBuilder<int>(
                valueListenable: state.hovered,
                builder: (BuildContext context, int hovered, _) =>
                    AnimatedPositioned.fromRect(
                      rect: layout.rows[math.max(0, hovered)],
                      duration: GfMotion.duration(context, GfMotion.selection),
                      curve: GfMotion.enterCurve,
                      child: AnimatedOpacity(
                        opacity: hovered < 0 ? 0 : 1,
                        duration: GfMotion.duration(context, GfMotion.press),
                        child: DecoratedBox(
                          decoration: BoxDecoration(
                            color: highlight,
                            borderRadius: BorderRadius.circular(
                              _highlightRadius,
                            ),
                          ),
                        ),
                      ),
                    ),
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
        ),
      ),
    );
  }
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
