import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';

import '../../theme/gf_theme.dart';
import '../gf_motion.dart';

/// Optical roles, independent of the site's semantic colour palette.
enum GfGlassWeight { regular, strong, menu, clear }

/// The app supplies platform Reduce Transparency here. MediaQuery supplies
/// high contrast, accessible navigation and reduced motion use the same fallback.
class GfGlassSettings extends InheritedWidget {
  const GfGlassSettings({
    super.key,
    this.reduceTransparency = false,
    required super.child,
  });

  final bool reduceTransparency;

  static bool opaqueOf(BuildContext context) =>
      (context
              .dependOnInheritedWidgetOfExactType<GfGlassSettings>()
              ?.reduceTransparency ??
          false) ||
      MediaQuery.highContrastOf(context) ||
      MediaQuery.accessibleNavigationOf(context) ||
      GfMotion.reducedOf(context);

  @override
  bool updateShouldNotify(GfGlassSettings oldWidget) =>
      reduceTransparency != oldWidget.reduceTransparency;
}

/// A bounded optical control layer. Impeller refracts the real backdrop;
/// other renderers use frosted glass. It never filters its own text or icons.
///
/// No repeating animation, screen capture or image readback is involved. Each
/// mounted surface owns a shader; the compiled program is shared. Contents
/// remain mounted when settings, theme, renderer or shader readiness changes.
class GfLiquidSurface extends StatefulWidget {
  const GfLiquidSurface({
    super.key,
    required this.child,
    this.radius = 28,
    this.padding = EdgeInsets.zero,
    this.weight = GfGlassWeight.regular,
    this.tint,
    this.pressable = false,
    this.elevated = true,
    this.forceOpaque = false,
    this.blurSigma,
  });

  final Widget child;
  final double radius;
  final EdgeInsetsGeometry padding;
  final GfGlassWeight weight;
  final Color? tint;
  final bool pressable;
  final bool elevated;
  final bool forceOpaque;
  final double? blurSigma;

  @override
  State<GfLiquidSurface> createState() => _GfLiquidSurfaceState();
}

class _GfLiquidSurfaceState extends State<GfLiquidSurface> {
  static Future<ui.FragmentProgram?>? _program;
  ui.FragmentShader? _shader;
  bool _loading = false;
  final Set<int> _pointers = {};

  Future<void> _loadShader() async {
    _loading = true;
    final program = await (_program ??=
        ui.FragmentProgram.fromAsset(
          'packages/ui_kit/shaders/liquid_glass.frag',
        ).then<ui.FragmentProgram?>((value) => value).catchError((
          Object error,
        ) {
          debugPrint('GfLiquidSurface: frosted fallback ($error)');
          return null;
        }));
    if (!mounted || program == null) return;
    setState(() => _shader = program.fragmentShader());
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_loading &&
        ui.ImageFilter.isShaderFilterSupported &&
        !GfGlassSettings.opaqueOf(context)) {
      _loadShader();
    }
  }

  @override
  void didUpdateWidget(GfLiquidSurface oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!widget.pressable) _pointers.clear();
  }

  @override
  void dispose() {
    _shader?.dispose();
    super.dispose();
  }

  void _pointer(int pointer, bool down) {
    if (!widget.pressable && down) return;
    if (!down && !_pointers.contains(pointer)) return;
    setState(() {
      if (down) {
        _pointers.add(pointer);
      } else {
        _pointers.remove(pointer);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final colors = GfTheme.colorsOf(context);
    final dark = Theme.of(context).brightness == Brightness.dark;
    final opaque = widget.forceOpaque || GfGlassSettings.opaqueOf(context);
    final clear = widget.weight == GfGlassWeight.clear;
    final strong = widget.weight == GfGlassWeight.strong;
    final menu = widget.weight == GfGlassWeight.menu;
    final base =
        widget.tint ??
        (clear
            ? const Color(0xff182334)
            : (strong || menu || opaque ? colors.base100 : colors.base300));
    final alpha = opaque
        ? 1.0
        : (menu
              ? (dark ? .46 : .42)
              : (clear ? .58 : (strong ? .88 : (dark ? .70 : .62))));
    final radius = BorderRadius.circular(widget.radius);
    final fill = base.withValues(alpha: alpha);
    Color illuminated(double light) =>
        Color.alphaBlend(Colors.white.withValues(alpha: light), fill);
    return Listener(
      onPointerDown: (event) => _pointer(event.pointer, true),
      onPointerUp: (event) => _pointer(event.pointer, false),
      onPointerCancel: (event) => _pointer(event.pointer, false),
      child: TweenAnimationBuilder<double>(
        tween: Tween(
          begin: 1,
          end: !opaque && widget.pressable && _pointers.isNotEmpty ? .97 : 1,
        ),
        duration: GfMotion.duration(context, GfMotion.press),
        curve: Curves.easeOutCubic,
        builder: (context, scale, child) => Transform.scale(
          key: const Key('gf-liquid-press-transform'),
          scale: scale,
          transformHitTests: false,
          child: child,
        ),
        child: DecoratedBox(
          decoration: BoxDecoration(
            borderRadius: radius,
            boxShadow: widget.elevated
                ? [
                    BoxShadow(
                      color: Colors.black.withValues(alpha: dark ? .25 : .08),
                      blurRadius: 24,
                      offset: const Offset(0, 8),
                    ),
                    BoxShadow(
                      color: Colors.black.withValues(alpha: dark ? .15 : .04),
                      blurRadius: 3,
                      offset: const Offset(0, 1),
                    ),
                  ]
                : null,
          ),
          child: ClipRRect(
            borderRadius: radius,
            child: _OpticalBackdrop(
              enabled: !opaque,
              shader: _shader,
              radius: widget.radius,
              blurSigma:
                  widget.blurSigma ??
                  (menu ? 6 : (strong ? 14 : (clear ? 1.5 : 3))),
              pixelRatio: MediaQuery.devicePixelRatioOf(context),
              child: DecoratedBox(
                decoration: BoxDecoration(
                  color: opaque ? fill : null,
                  gradient: opaque
                      ? null
                      : LinearGradient(
                          begin: Alignment.topLeft,
                          end: Alignment.bottomRight,
                          colors: [
                            illuminated(
                              dark || clear ? .08 : (menu ? .12 : .24),
                            ),
                            illuminated(0),
                            illuminated(
                              dark || clear ? .02 : (menu ? .04 : .10),
                            ),
                          ],
                        ),
                ),
                child: CustomPaint(
                  foregroundPainter: _GlassRim(
                    radius: widget.radius,
                    dark: dark || clear,
                    opaque: opaque,
                    line: colors.line,
                  ),
                  child: Material(
                    type: MaterialType.transparency,
                    child: Padding(
                      padding: widget.padding,
                      child: widget.child,
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

class _GlassRim extends CustomPainter {
  const _GlassRim({
    required this.radius,
    required this.dark,
    required this.opaque,
    required this.line,
  });
  final double radius;
  final bool dark;
  final bool opaque;
  final Color line;

  @override
  void paint(Canvas canvas, Size size) {
    if (radius == 0) return;
    final rect = (Offset.zero & size).deflate(.6);
    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1;
    if (opaque) {
      paint.color = line;
    } else {
      paint.shader = LinearGradient(
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
        colors: [
          Colors.white.withValues(alpha: dark ? .38 : .95),
          dark
              ? Colors.white.withValues(alpha: .05)
              : Colors.black.withValues(alpha: .10),
          Colors.white.withValues(alpha: dark ? .18 : .70),
        ],
        stops: const [0, .55, 1],
      ).createShader(rect);
    }
    canvas.drawRRect(
      RRect.fromRectAndRadius(rect, Radius.circular(radius)),
      paint,
    );
  }

  @override
  bool shouldRepaint(_GlassRim oldDelegate) =>
      radius != oldDelegate.radius ||
      dark != oldDelegate.dark ||
      opaque != oldDelegate.opaque ||
      line != oldDelegate.line;
}

/// ImageFilter.shader samples a backdrop texture in scene coordinates, not
/// widget coordinates. Refresh the lens bounds at paint time so scrolling,
/// ancestor transforms and interrupted route transitions keep it attached.
class _OpticalBackdrop extends SingleChildRenderObjectWidget {
  const _OpticalBackdrop({
    required this.enabled,
    required this.shader,
    required this.radius,
    required this.pixelRatio,
    required this.blurSigma,
    required super.child,
  });
  final bool enabled;
  final ui.FragmentShader? shader;
  final double radius;
  final double pixelRatio;
  final double blurSigma;

  @override
  _RenderOpticalBackdrop createRenderObject(BuildContext context) =>
      _RenderOpticalBackdrop(
        enabled: enabled,
        shader: shader,
        radius: radius,
        pixelRatio: pixelRatio,
        blurSigma: blurSigma,
      );
  @override
  void updateRenderObject(
    BuildContext context,
    _RenderOpticalBackdrop renderObject,
  ) {
    renderObject
      ..enabled = enabled
      ..shader = shader
      ..radius = radius
      ..pixelRatio = pixelRatio
      ..blurSigma = blurSigma
      ..markNeedsPaint();
  }
}

class _RenderOpticalBackdrop extends RenderProxyBox {
  _RenderOpticalBackdrop({
    required this.enabled,
    required this.shader,
    required this.radius,
    required this.pixelRatio,
    required this.blurSigma,
  });
  bool enabled;
  ui.FragmentShader? shader;
  double radius;
  double pixelRatio;
  double blurSigma;

  @override
  bool get alwaysNeedsCompositing => child != null;

  @override
  void paint(PaintingContext context, Offset offset) {
    if (!enabled || child == null) {
      layer = null;
      super.paint(context, offset);
      return;
    }
    final effect = shader;
    ui.ImageFilter filter;
    if (effect != null) {
      final bounds = MatrixUtils.transformRect(
        getTransformTo(null),
        Offset.zero & size,
      );
      effect
        ..setFloat(2, bounds.left * pixelRatio)
        ..setFloat(3, bounds.top * pixelRatio)
        ..setFloat(4, bounds.width * pixelRatio)
        ..setFloat(5, bounds.height * pixelRatio)
        ..setFloat(6, radius)
        ..setFloat(7, pixelRatio);
      filter = ui.ImageFilter.compose(
        outer: ui.ImageFilter.shader(effect),
        inner: ui.ImageFilter.blur(sigmaX: blurSigma, sigmaY: blurSigma),
      );
    } else {
      filter = ui.ImageFilter.blur(sigmaX: blurSigma, sigmaY: blurSigma);
    }
    final backdrop = (layer ??= BackdropFilterLayer()) as BackdropFilterLayer;
    backdrop.filter = filter;
    context.pushLayer(backdrop, super.paint, offset);
  }
}
