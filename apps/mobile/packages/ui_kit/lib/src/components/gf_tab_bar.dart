import 'dart:math' as math;
import 'dart:ui' show ImageFilter, lerpDouble;

import 'package:flutter/foundation.dart'
    show DiagnosticPropertiesBuilder, DoubleProperty, ValueListenable;
import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import '../twemoji/twemoji_text.dart';
import 'gf_motion.dart';
import 'gf_symbol.dart';

/// A single selectable tab in [GfTabBar].
class GfTab {
  const GfTab({required this.label, required this.value, this.symbol});

  final String label;
  final Object value;

  /// Optional leading ReIcon glyph; it follows the label's selected color.
  /// With [GfTabBar.selectedSymbolOnly], pass the Filled weight.
  final String? symbol;
}

@immutable
class GfTabSwipeProgress {
  const GfTabSwipeProgress({
    required this.originIndex,
    required this.offset,
    this.targetIndex,
  });

  final int originIndex;
  final double offset;
  final int? targetIndex;
}

class GfTabSwipeProgressScope extends InheritedWidget {
  const GfTabSwipeProgressScope({
    super.key,
    required this.progress,
    this.onTabSelected,
    required super.child,
  });

  final ValueListenable<GfTabSwipeProgress> progress;
  final ValueChanged<int>? onTabSelected;

  static ValueListenable<GfTabSwipeProgress>? maybeOf(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<GfTabSwipeProgressScope>()
          ?.progress;

  static ValueChanged<int>? onTabSelectedOf(BuildContext context) => context
      .getInheritedWidgetOfExactType<GfTabSwipeProgressScope>()
      ?.onTabSelected;

  @override
  bool updateShouldNotify(GfTabSwipeProgressScope oldWidget) =>
      progress != oldWidget.progress ||
      onTabSelected != oldWidget.onTabSelected;
}

/// Scrollable tab bar mirroring web `.gf-tab` semantics.
///
/// On mobile the bar scrolls horizontally when tabs overflow; the active tab
/// renders with a short brand underline; idle tabs use muted text. Overflowing
/// edges fade out so the hidden tabs read as reachable by scrolling.
class GfTabBar extends StatefulWidget {
  /// Use this for overlay toolbars so their content inset grows with text.
  static double heightFor(BuildContext context) =>
      math.max(48, MediaQuery.textScalerOf(context).scale(16) * 1.25 + 28);
  const GfTabBar({
    super.key,
    required this.tabs,
    required this.selected,
    required this.onSelected,
    this.mobile = true,
    this.distribute = false,
    this.selectedSymbolOnly = false,
  });

  final List<GfTab> tabs;
  final Object selected;
  final ValueChanged<Object> onSelected;

  /// When true and every tab fits, tabs share the width evenly. Otherwise
  /// (narrow windows, large text) the bar keeps scrolling horizontally.
  final bool distribute;

  /// When true (default) the bar scrolls horizontally on overflow; when false
  /// tabs wrap.
  final bool mobile;

  /// When true, only the selected tab shows its [GfTab.symbol]. The glyph
  /// follows the selection: the outgoing one shrinks toward the incoming tab
  /// while the incoming one grows in from the outgoing side, so the two meet
  /// in the middle. A page swipe scrubs the same handoff.
  final bool selectedSymbolOnly;

  @override
  State<GfTabBar> createState() => _GfTabBarState();
}

class _GfTabBarState extends State<GfTabBar> with TickerProviderStateMixin {
  late TabController _controller;
  late final AnimationController _settleController = AnimationController(
    vsync: this,
    duration: GfMotion.layout,
  )..addListener(_onSettleTick);
  ValueListenable<GfTabSwipeProgress>? _swipeProgress;
  GfTabSwipeProgress _lastSwipeProgress = const GfTabSwipeProgress(
    originIndex: 0,
    offset: 0,
  );
  double _settlingExtension = 0;

  /// Index direction (+1 or -1) of the travel [_settlingExtension] trails.
  double _settlingDirection = 1;
  double _returningOffset = 0;
  int? _returningIndex;
  bool _reduceMotion = false;
  bool _ready = false;

  /// Width of each evenly distributed tab, or null while the bar scrolls.
  ///
  /// Assigned inside [LayoutBuilder] during [build] (before the indicator is
  /// measured in that same frame) and read by the indicator/gesture paths as
  /// the value from the last completed build. It is a layout cache, not a
  /// source of truth, so it must be refreshed on every build.
  double? _cellWidth;

  static const Curve _indicatorCurve = GfLogarithmicEaseOutCurve();
  static const double _symbolSize = 18;
  static const double _symbolGap = 6;
  static const double _fillLabelPadding = 4;
  // Horizontal travel of a selected-only glyph while it hands off.
  static const double _symbolTravel = 10;
  // The 4px enter blur expressed as a Gaussian sigma.
  static const double _symbolBlurSigma = 2;

  int get _selectedIndex =>
      widget.tabs.indexWhere((tab) => tab.value == widget.selected);

  int get _controllerLength => widget.tabs.isEmpty ? 1 : widget.tabs.length;

  int get _initialIndex {
    final index = _selectedIndex;
    return index >= 0 && index < _controllerLength ? index : 0;
  }

  TabController _createController() => TabController(
    length: _controllerLength,
    initialIndex: _initialIndex,
    vsync: this,
    animationDuration: _reduceMotion ? Duration.zero : GfMotion.layout,
  );

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final progress = GfTabSwipeProgressScope.maybeOf(context);
    if (progress != _swipeProgress) {
      _swipeProgress?.removeListener(_onSwipeProgressChanged);
      _swipeProgress = progress;
      _swipeProgress?.addListener(_onSwipeProgressChanged);
      _lastSwipeProgress =
          progress?.value ??
          const GfTabSwipeProgress(originIndex: 0, offset: 0);
    }
    final reduceMotion = MediaQuery.disableAnimationsOf(context);
    _settleController.duration = reduceMotion ? Duration.zero : GfMotion.layout;
    if (!_ready || reduceMotion != _reduceMotion) {
      _reduceMotion = reduceMotion;
      if (_ready) {
        final oldController = _controller;
        _controller = _createController();
        oldController.dispose();
      } else {
        _controller = _createController();
        _ready = true;
      }
    }
  }

  @override
  void didUpdateWidget(GfTabBar oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.tabs.length != widget.tabs.length) {
      final oldController = _controller;
      _controller = _createController();
      oldController.dispose();
      return;
    }

    final selectedIndex = _selectedIndex;
    if (selectedIndex >= 0 && selectedIndex != _controller.index) {
      final progress = _lastSwipeProgress;
      if (progress.originIndex == _controller.index && progress.offset != 0) {
        _settlingExtension = _indicatorExtension(progress);
        _settlingDirection = progress.offset.sign;
        _returningIndex = null;
        _settleController.forward(from: 0);
      }
      _controller.animateTo(
        selectedIndex,
        duration: GfMotion.duration(context, GfMotion.layout),
        curve: _indicatorCurve,
      );
    }
  }

  void _onSwipeProgressChanged() {
    if (!_ready || !mounted) return;
    final progress = _swipeProgress!.value;
    final previous = _lastSwipeProgress;
    _lastSwipeProgress = progress;
    if (progress.offset == 0) {
      if (previous.offset != 0 &&
          previous.originIndex == progress.originIndex &&
          progress.originIndex == _controller.index &&
          !_controller.indexIsChanging) {
        _settlingExtension = _indicatorExtension(previous);
        _settlingDirection = previous.offset.sign;
        _returningOffset = previous.offset;
        _returningIndex = previous.originIndex;
        _settleController.forward(from: 0);
      }
      return;
    }
    if (progress.originIndex != _controller.index ||
        _controller.indexIsChanging ||
        progress.originIndex + progress.offset.sign < 0 ||
        progress.originIndex + progress.offset.sign >= widget.tabs.length) {
      return;
    }
    _settleController.stop();
    _settlingExtension = 0;
    _returningIndex = null;
    _controller.offset = progress.offset.clamp(-1.0, 1.0).toDouble();
    setState(() {});
  }

  void _onSettleTick() {
    if (!mounted) return;
    if (_returningIndex == _controller.index && !_controller.indexIsChanging) {
      _controller.offset =
          _returningOffset *
          (1 - _indicatorCurve.transform(_settleController.value));
    }
    setState(() {});
  }

  double _indicatorExtension(GfTabSwipeProgress progress) {
    final int targetIndex = progress.originIndex + progress.offset.sign.toInt();
    if (progress.originIndex < 0 ||
        progress.originIndex >= widget.tabs.length ||
        targetIndex < 0 ||
        targetIndex >= widget.tabs.length) {
      return 0;
    }
    if (_cellWidth case final cell?) return cell * progress.offset.abs();
    final style = DefaultTextStyle.of(
      context,
    ).style.copyWith(fontSize: 16, height: 1.25, fontWeight: FontWeight.w600);
    return (_itemWidth(context, widget.tabs[progress.originIndex], style) +
            _itemWidth(context, widget.tabs[targetIndex], style)) /
        2 *
        progress.offset.abs();
  }

  @override
  void dispose() {
    _swipeProgress?.removeListener(_onSwipeProgressChanged);
    _settleController.dispose();
    _controller.dispose();
    super.dispose();
  }

  /// How present the selected-only glyph of [index] is, from 0 to 1.
  ///
  /// A tap that skips tabs only hands off between its two ends; a drag or an
  /// idle bar follows the controller value, so neighbours share the glyph.
  double _symbolPresence(int index) {
    final double value = _controller.animation!.value;
    if (_controller.indexIsChanging) {
      final int from = _controller.previousIndex;
      final int to = _controller.index;
      final double progress = ((value - from) / (to - from)).clamp(0.0, 1.0);
      if (index == to) return progress;
      if (index == from) return 1 - progress;
      return 0;
    }
    return (1 - (value - index).abs()).clamp(0.0, 1.0);
  }

  Widget _selectedOnlySymbol(int index, String symbol) => AnimatedBuilder(
    animation: _controller.animation!,
    builder: (context, _) {
      final double presence = _symbolPresence(index);
      if (presence == 0) return const SizedBox.shrink();
      const double slot = _symbolSize + _symbolGap;
      final double hidden = 1 - presence;
      // Toward the other tab of the handoff: positive while selection moves
      // to higher indexes, mirrored for right-to-left reading.
      final double direction =
          (_controller.animation!.value - index).sign *
          (Directionality.of(context) == TextDirection.rtl ? -1 : 1);
      Widget glyph = Transform.translate(
        offset: Offset(_symbolTravel * hidden * direction, 0),
        child: Transform.scale(
          scale: lerpDouble(.25, 1, presence)!,
          child: GfSymbol(symbol, size: _symbolSize),
        ),
      );
      if (hidden > 0) {
        final double sigma = _symbolBlurSigma * hidden;
        glyph = ImageFiltered(
          imageFilter: ImageFilter.blur(sigmaX: sigma, sigmaY: sigma),
          child: glyph,
        );
      }
      return SizedBox(
        width: slot * presence,
        height: _symbolSize,
        child: OverflowBox(
          minWidth: slot,
          maxWidth: slot,
          child: Padding(
            padding: const EdgeInsetsDirectional.only(end: _symbolGap),
            child: Opacity(opacity: presence, child: glyph),
          ),
        ),
      );
    },
  );

  Widget _label(int index, {required bool active}) {
    final GfTab tab = widget.tabs[index];
    final text = GfText(tab.label, maxLines: 1, softWrap: false);
    final String? symbol = tab.symbol;
    if (symbol == null ||
        (widget.selectedSymbolOnly && !widget.mobile && !active)) {
      return text;
    }
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (widget.selectedSymbolOnly && widget.mobile)
          _selectedOnlySymbol(index, symbol)
        else ...[
          GfSymbol(symbol, size: _symbolSize),
          const SizedBox(width: _symbolGap),
        ],
        Flexible(child: text),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final widget = this.widget;
    final tabs = widget.tabs;
    final selected = widget.selected;
    final onSelected = widget.onSelected;
    final mobile = widget.mobile;
    final colors = GfTheme.colorsOf(context);
    const duration = GfMotion.layout;
    const curve = GfMotion.layoutCurve;
    final noAnimation = MediaQuery.disableAnimationsOf(context);
    final height = GfTabBar.heightFor(context);
    final selectedIndex = tabs.indexWhere((tab) => tab.value == selected);
    final measureStyle = DefaultTextStyle.of(
      context,
    ).style.copyWith(fontSize: 16, height: 1.25, fontWeight: FontWeight.w600);
    final currentProgress = _swipeProgress?.value;
    double dragExtension() =>
        currentProgress != null &&
            currentProgress.originIndex == _controller.index &&
            !_controller.indexIsChanging
        ? _indicatorExtension(currentProgress)
        : 0;
    final double settleExtension =
        _settlingExtension *
        (1 - _indicatorCurve.transform(_settleController.value));

    void selectTab(int index) {
      final onTabSelected = GfTabSwipeProgressScope.onTabSelectedOf(context);
      if (onTabSelected == null) {
        onSelected(tabs[index].value);
      } else {
        onTabSelected(index);
      }
    }

    Widget item(int index) {
      final active = index == selectedIndex;
      return Semantics(
        selected: active,
        button: true,
        child: InkWell(
          onTap: () => selectTab(index),
          child: SizedBox(
            width: _itemWidth(
              context,
              tabs[index],
              measureStyle,
              withSymbol: !widget.selectedSymbolOnly || active,
            ),
            height: height,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const SizedBox(height: 12),
                AnimatedDefaultTextStyle(
                  duration: noAnimation ? Duration.zero : duration,
                  curve: curve,
                  style: measureStyle.copyWith(
                    color: active ? colors.baseContent : colors.iconMuted,
                    fontWeight: active ? FontWeight.w600 : FontWeight.w400,
                  ),
                  // The glyph shares the label's color transition.
                  child: TweenAnimationBuilder<Color?>(
                    tween: ColorTween(
                      end: active ? colors.baseContent : colors.iconMuted,
                    ),
                    duration: noAnimation ? Duration.zero : duration,
                    curve: curve,
                    builder: (context, color, child) => IconTheme.merge(
                      data: IconThemeData(color: color),
                      child: child!,
                    ),
                    child: _label(index, active: active),
                  ),
                ),
                const Spacer(),
                AnimatedContainer(
                  duration: noAnimation ? Duration.zero : duration,
                  curve: curve,
                  width: 28,
                  height: 4,
                  decoration: BoxDecoration(
                    color: active ? colors.primary : Colors.transparent,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    }

    if (!mobile) {
      return Wrap(children: [for (var i = 0; i < tabs.length; i++) item(i)]);
    }
    if (tabs.isEmpty) return const SizedBox.shrink();

    return LayoutBuilder(
      builder: (context, constraints) {
        // Equal cells must each hold the widest label at the fill padding.
        final bool fill =
            widget.distribute &&
            constraints.hasBoundedWidth &&
            tabs.fold<double>(
                      0,
                      (widest, tab) => math.max(
                        widest,
                        _itemWidth(
                          context,
                          tab,
                          measureStyle,
                          padding: _fillLabelPadding,
                        ),
                      ),
                    ) *
                    tabs.length <=
                constraints.maxWidth;
        _cellWidth = fill ? constraints.maxWidth / tabs.length : null;
        final double drag = dragExtension();
        final bool dragging = drag > 0;
        // The underline's centre tracks the controller linearly; the extra
        // length only reaches back toward the tab it left, so the trailing
        // edge holds during a drag and catches up after release.
        final double travel =
            (dragging ? currentProgress!.offset.sign : _settlingDirection) *
            (Directionality.of(context) == TextDirection.rtl ? -1 : 1);
        final Widget bar = SizedBox(
          height: height,
          child: TabBar(
            key: const ValueKey('gf-tab-bar'),
            controller: _controller,
            isScrollable: !fill,
            tabAlignment: fill ? TabAlignment.fill : TabAlignment.start,
            indicatorAnimation: TabIndicatorAnimation.linear,
            indicatorSize: TabBarIndicatorSize.label,
            indicator: _FixedWidthTabIndicator(
              color: colors.primary,
              width: 28,
              trailing: (dragging ? drag : settleExtension) * -travel,
            ),
            dividerColor: Colors.transparent,
            dividerHeight: 0,
            labelColor: colors.baseContent,
            labelStyle: measureStyle.copyWith(fontWeight: FontWeight.w600),
            labelPadding: EdgeInsets.symmetric(
              horizontal: fill ? _fillLabelPadding : 16,
            ),
            unselectedLabelColor: colors.iconMuted,
            unselectedLabelStyle: measureStyle.copyWith(
              fontWeight: FontWeight.w400,
            ),
            tabs: [
              for (var i = 0; i < tabs.length; i++)
                Tab(
                  height: height,
                  child: _label(i, active: i == selectedIndex),
                ),
            ],
            onTap: selectTab,
          ),
        );
        // Evenly distributed tabs always fit. The scrolling strip keeps one
        // tree so its offset survives the fade appearing and disappearing.
        return fill ? bar : _GfTabEdgeFade(child: bar);
      },
    );
  }

  double _itemWidth(
    BuildContext context,
    GfTab tab,
    TextStyle style, {
    double padding = 16,
    bool withSymbol = true,
  }) {
    final painter = TextPainter(
      text: TextSpan(text: tab.label, style: style),
      maxLines: 1,
      textDirection: Directionality.of(context),
      textScaler: MediaQuery.textScalerOf(context),
    )..layout();
    final width =
        painter.width +
        padding * 2 +
        (tab.symbol == null || !withSymbol ? 0 : _symbolSize + _symbolGap);
    painter.dispose();
    return width;
  }
}

/// Fades whichever end of a horizontal strip still has content to scroll to.
class _GfTabEdgeFade extends StatefulWidget {
  const _GfTabEdgeFade({required this.child});

  final Widget child;

  @override
  State<_GfTabEdgeFade> createState() => _GfTabEdgeFadeState();
}

class _GfTabEdgeFadeState extends State<_GfTabEdgeFade> {
  static const double _fadeExtent = 32;

  bool _start = false;
  bool _end = false;

  bool _onNotification(Notification notification) {
    final ScrollMetrics? metrics = switch (notification) {
      ScrollNotification(depth: 0, :final metrics) => metrics,
      ScrollMetricsNotification(depth: 0, :final metrics) => metrics,
      _ => null,
    };
    if (metrics == null || metrics.axis != Axis.horizontal) return false;
    final bool start = metrics.extentBefore > .5;
    final bool end = metrics.extentAfter > .5;
    if (start != _start || end != _end) {
      setState(() {
        _start = start;
        _end = end;
      });
    }
    return false;
  }

  @override
  Widget build(BuildContext context) {
    final TextDirection direction = Directionality.of(context);
    return NotificationListener<Notification>(
      onNotification: _onNotification,
      child: TweenAnimationBuilder<Offset>(
        tween: Tween<Offset>(end: Offset(_start ? 1 : 0, _end ? 1 : 0)),
        duration: GfMotion.duration(context, GfMotion.selection),
        curve: GfMotion.layoutCurve,
        child: widget.child,
        builder: (context, fade, child) => ShaderMask(
          blendMode: BlendMode.dstIn,
          shaderCallback: (Rect bounds) {
            final double stop = bounds.width <= 0
                ? 0
                : math.min(.5, _fadeExtent / bounds.width);
            return LinearGradient(
              begin: AlignmentDirectional.centerStart,
              end: AlignmentDirectional.centerEnd,
              colors: <Color>[
                Color.fromRGBO(0, 0, 0, 1 - fade.dx),
                const Color(0xFF000000),
                const Color(0xFF000000),
                Color.fromRGBO(0, 0, 0, 1 - fade.dy),
              ],
              stops: <double>[0, stop, 1 - stop, 1],
            ).createShader(bounds, textDirection: direction);
          },
          child: child,
        ),
      ),
    );
  }
}

class _FixedWidthTabIndicator extends Decoration {
  const _FixedWidthTabIndicator({
    required this.color,
    required this.width,
    this.trailing = 0,
  });

  static const double _height = 4;
  static const double _radius = 2;

  final Color color;
  final double width;

  /// Extra length beyond [width], in logical pixels along the x axis:
  /// negative extends the left end, positive the right end.
  final double trailing;

  @override
  BoxPainter createBoxPainter([VoidCallback? onChanged]) =>
      _FixedWidthTabIndicatorPainter(this, onChanged);

  @override
  void debugFillProperties(DiagnosticPropertiesBuilder properties) {
    super.debugFillProperties(properties);
    properties.add(DoubleProperty('width', width));
    properties.add(DoubleProperty('trailing', trailing, defaultValue: 0));
  }
}

class _FixedWidthTabIndicatorPainter extends BoxPainter {
  _FixedWidthTabIndicatorPainter(this.decoration, super.onChanged);

  final _FixedWidthTabIndicator decoration;

  @override
  void paint(Canvas canvas, Offset offset, ImageConfiguration configuration) {
    final Size? size = configuration.size;
    if (size == null) return;

    final double center = offset.dx + size.width / 2;
    final double trailing = decoration.trailing;
    final Rect rect = Rect.fromLTRB(
      center - decoration.width / 2 + math.min(0, trailing),
      offset.dy + size.height - _FixedWidthTabIndicator._height,
      center + decoration.width / 2 + math.max(0, trailing),
      offset.dy + size.height,
    );
    final RRect rounded = RRect.fromRectAndRadius(
      rect,
      const Radius.circular(_FixedWidthTabIndicator._radius),
    );
    canvas.drawRRect(
      rounded,
      Paint()
        ..color = decoration.color.withValues(alpha: .18)
        ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 3),
    );
    canvas.drawRRect(rounded, Paint()..color = decoration.color);
  }
}

class GfLogarithmicEaseOutCurve extends Curve {
  const GfLogarithmicEaseOutCurve();

  static const double _strength = 4;

  @override
  double transformInternal(double t) =>
      math.log(1 + (math.exp(_strength) - 1) * t) / _strength;
}
