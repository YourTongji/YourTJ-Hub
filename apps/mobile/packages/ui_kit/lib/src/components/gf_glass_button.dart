import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter/physics.dart';

import '../theme/gf_theme.dart';
import 'gf_motion.dart';
import 'surfaces/gf_navigation_surface.dart';

// How far a press swells the glass on each side, as on the dock.
const double _pressSwell = 3;

/// A capsule of floating-layer glass that acts as a button: the compose
/// action, its menu items and close button.
///
/// It shares the dock's material ([GfNavigationSurface]) and its touch
/// language: a press swells the glass around a steady content that dips
/// slightly, and a soft light gathers under the finger and follows it, as a
/// lens catches light where it is touched. Without glass (high contrast,
/// reduced motion, accessible navigation) the surface is opaque and a press
/// tints it instead. Keyboard focus draws a ring outside the glass; Enter and
/// Space activate it.
class GfGlassButton extends StatefulWidget {
  const GfGlassButton({
    super.key,
    required this.onPressed,
    required this.child,
    this.tone = GfGlassTone.regular,
    this.tooltip,
    this.autofocus = false,
    this.padding = EdgeInsets.zero,
    this.minSize = const Size.square(56),
  });

  final VoidCallback? onPressed;
  final Widget child;
  final GfGlassTone tone;

  /// Long-press label and accessibility label of icon-only buttons.
  final String? tooltip;
  final bool autofocus;
  final EdgeInsetsGeometry padding;
  final Size minSize;

  @override
  State<GfGlassButton> createState() => _GfGlassButtonState();
}

class _GfGlassButtonState extends State<GfGlassButton>
    with SingleTickerProviderStateMixin {
  late final AnimationController _press = AnimationController.unbounded(
    vsync: this,
  );
  final ValueNotifier<Offset?> _light = ValueNotifier<Offset?>(null);
  bool _focused = false;
  int _pointers = 0;

  bool get _enabled => widget.onPressed != null;

  @override
  void dispose() {
    _press.dispose();
    _light.dispose();
    super.dispose();
  }

  void _springTo(double target, double stiffness, double dampingRatio) {
    if (GfMotion.reducedOf(context)) {
      _press.value = target;
      return;
    }
    _press
        .animateWith(
          SpringSimulation(
            SpringDescription.withDampingRatio(
              mass: 1,
              stiffness: stiffness,
              ratio: dampingRatio,
            ),
            _press.value,
            target,
            _press.velocity,
          ),
        )
        // A spring stops within its tolerance; rest exactly on the target.
        .then((_) {
          if (mounted) _press.value = target;
        });
  }

  void _down(PointerDownEvent event) {
    if (!_enabled) return;
    _pointers++;
    _light.value = event.localPosition;
    // The dock's press spring: quick and slightly lively in, calm out.
    _springTo(1, 520, 0.62);
  }

  void _move(PointerMoveEvent event) {
    if (_pointers > 0) _light.value = event.localPosition;
  }

  void _up(PointerEvent _) {
    if (_pointers == 0) return;
    _pointers--;
    if (_pointers == 0) _springTo(0, 300, 0.8);
  }

  @override
  Widget build(BuildContext context) {
    final GfColors colors = GfTheme.colorsOf(context);
    final bool prominent = widget.tone == GfGlassTone.prominent;
    final bool glass = GfNavigationSurface.usesGlass(context);
    final Color foreground = !_enabled
        ? colors.textDisabled
        : prominent
        ? colors.primaryContent
        : colors.textPrimary;
    final bool round = widget.minSize.width == widget.minSize.height;
    final Widget content = IconTheme.merge(
      data: IconThemeData(color: foreground),
      child: DefaultTextStyle.merge(
        style: TextStyle(color: foreground),
        child: Padding(padding: widget.padding, child: widget.child),
      ),
    );
    Widget button = ConstrainedBox(
      constraints: BoxConstraints(
        minWidth: widget.minSize.width,
        minHeight: widget.minSize.height,
      ),
      child: AnimatedBuilder(
        animation: _press,
        builder: (BuildContext context, Widget? child) {
          final double press = _press.value;
          // The swell only paints: layout and hit testing see the rest box.
          final double swell = glass ? press * _pressSwell : 0;
          return Stack(
            alignment: Alignment.center,
            clipBehavior: Clip.none,
            children: <Widget>[
              Positioned.fill(
                left: -swell,
                top: -swell,
                right: -swell,
                bottom: -swell,
                child: RepaintBoundary(
                  child: CustomPaint(
                    foregroundPainter: _GlassTouchPainter(
                      colors: colors,
                      prominent: prominent,
                      glass: glass,
                      press: press.clamp(0.0, 1.0),
                      light: _light,
                      origin: Offset(swell, swell),
                      focused: _focused,
                    ),
                    child: GfNavigationSurface(tone: widget.tone),
                  ),
                ),
              ),
              Transform.scale(
                scale: 1 - (round ? 0.06 : 0.03) * press,
                child: child,
              ),
            ],
          );
        },
        child: Opacity(opacity: _enabled ? 1 : 0.6, child: content),
      ),
    );
    button = Listener(
      onPointerDown: _down,
      onPointerMove: _move,
      onPointerUp: _up,
      onPointerCancel: _up,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: widget.onPressed,
        child: button,
      ),
    );
    button = FocusableActionDetector(
      enabled: _enabled,
      autofocus: widget.autofocus,
      mouseCursor: _enabled ? SystemMouseCursors.click : MouseCursor.defer,
      onShowFocusHighlight: (bool value) => setState(() => _focused = value),
      actions: <Type, Action<Intent>>{
        ActivateIntent: CallbackAction<ActivateIntent>(
          onInvoke: (_) {
            widget.onPressed?.call();
            return null;
          },
        ),
      },
      child: button,
    );
    button = Semantics(button: true, enabled: _enabled, child: button);
    if (widget.tooltip != null) {
      button = Tooltip(message: widget.tooltip, child: button);
    }
    // One button node carrying its tooltip, or a menu item's text.
    return MergeSemantics(child: button);
  }
}

/// The light under the finger, the opaque press tint and the focus ring.
class _GlassTouchPainter extends CustomPainter {
  _GlassTouchPainter({
    required this.colors,
    required this.prominent,
    required this.glass,
    required this.press,
    required this.light,
    required this.origin,
    required this.focused,
  }) : super(repaint: light);

  final GfColors colors;
  final bool prominent;
  final bool glass;
  final double press;
  final ValueNotifier<Offset?> light;

  /// Where the button's own origin sits inside the swollen glass.
  final Offset origin;
  final bool focused;

  @override
  void paint(Canvas canvas, Size size) {
    final Rect bounds = Offset.zero & size;
    final RRect capsule = RRect.fromRectAndRadius(
      bounds,
      Radius.circular(size.shortestSide / 2),
    );
    if (focused) {
      canvas.drawRRect(
        capsule.inflate(3),
        Paint()
          ..style = PaintingStyle.stroke
          ..strokeWidth = 2
          ..color = colors.primary,
      );
    }
    if (press <= 0) return;
    if (!glass) {
      canvas.drawRRect(
        capsule,
        Paint()
          ..color = (prominent ? Colors.black : colors.baseContent).withValues(
            alpha: (colors.isDark ? 0.12 : 0.08) * press,
          ),
      );
      return;
    }
    final Offset point = (light.value ?? bounds.center - origin) + origin;
    final bool dark = colors.isDark;
    final double peak = prominent ? (dark ? 0.22 : 0.3) : (dark ? 0.1 : 0.5);
    canvas
      ..save()
      ..clipRRect(capsule);
    canvas.drawRect(
      bounds,
      Paint()
        ..shader = ui.Gradient.radial(
          point,
          // Local to the finger, so a long capsule lights where it is held.
          size.shortestSide * 1.1,
          <Color>[
            Colors.white.withValues(alpha: peak * press),
            Colors.white.withValues(alpha: 0),
          ],
        ),
    );
    canvas.restore();
  }

  @override
  bool shouldRepaint(_GlassTouchPainter oldDelegate) =>
      oldDelegate.colors != colors ||
      oldDelegate.prominent != prominent ||
      oldDelegate.glass != glass ||
      oldDelegate.press != press ||
      oldDelegate.origin != origin ||
      oldDelegate.focused != focused;
}
