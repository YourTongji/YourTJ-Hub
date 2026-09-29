import 'dart:math' as math;

import 'package:flutter/foundation.dart'
    show DiagnosticPropertiesBuilder, DoubleProperty, ValueListenable;
import 'package:flutter/material.dart';

import '../theme/gf_theme.dart';
import 'gf_motion.dart';

/// A single selectable tab in [GfTabBar].
class GfTab {
  const GfTab({required this.label, required this.value});

  final String label;
  final Object value;
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
/// renders with a short brand underline; idle tabs use muted text.
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
  });

  final List<GfTab> tabs;
  final Object selected;
  final ValueChanged<Object> onSelected;

  /// When true (default) the bar scrolls horizontally on overflow; when false
  /// tabs wrap.
  final bool mobile;

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
  double _returningOffset = 0;
  int? _returningIndex;
  bool _reduceMotion = false;
  bool _ready = false;

  static const Curve _indicatorCurve = GfLogarithmicEaseOutCurve();

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
    final double dragExtension =
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
            width: _itemWidth(context, tabs[index], measureStyle),
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
                  child: Text(tabs[index].label, maxLines: 1, softWrap: false),
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

    return SizedBox(
      height: height,
      child: TabBar(
        key: const ValueKey('gf-tab-bar'),
        controller: _controller,
        isScrollable: true,
        tabAlignment: TabAlignment.start,
        indicatorAnimation: TabIndicatorAnimation.elastic,
        indicatorSize: TabBarIndicatorSize.label,
        indicator: _FixedWidthTabIndicator(
          color: colors.primary,
          width: 28 + (dragExtension > 0 ? dragExtension : settleExtension),
        ),
        dividerColor: Colors.transparent,
        dividerHeight: 0,
        labelColor: colors.baseContent,
        labelStyle: measureStyle.copyWith(fontWeight: FontWeight.w600),
        labelPadding: const EdgeInsets.symmetric(horizontal: 16),
        unselectedLabelColor: colors.iconMuted,
        unselectedLabelStyle: measureStyle.copyWith(
          fontWeight: FontWeight.w400,
        ),
        tabs: [
          for (final tab in tabs)
            Tab(
              height: height,
              child: Text(tab.label, maxLines: 1, softWrap: false),
            ),
        ],
        onTap: selectTab,
      ),
    );
  }

  double _itemWidth(BuildContext context, GfTab tab, TextStyle style) {
    final painter = TextPainter(
      text: TextSpan(text: tab.label, style: style),
      maxLines: 1,
      textDirection: Directionality.of(context),
      textScaler: MediaQuery.textScalerOf(context),
    )..layout();
    final width = painter.width + 32;
    painter.dispose();
    return width;
  }
}

class _FixedWidthTabIndicator extends Decoration {
  const _FixedWidthTabIndicator({required this.color, required this.width});

  static const double _height = 4;
  static const double _radius = 2;

  final Color color;
  final double width;

  @override
  BoxPainter createBoxPainter([VoidCallback? onChanged]) =>
      _FixedWidthTabIndicatorPainter(this, onChanged);

  @override
  void debugFillProperties(DiagnosticPropertiesBuilder properties) {
    super.debugFillProperties(properties);
    properties.add(DoubleProperty('width', width));
  }
}

class _FixedWidthTabIndicatorPainter extends BoxPainter {
  _FixedWidthTabIndicatorPainter(this.decoration, super.onChanged);

  final _FixedWidthTabIndicator decoration;

  @override
  void paint(Canvas canvas, Offset offset, ImageConfiguration configuration) {
    final Size? size = configuration.size;
    if (size == null) return;

    final Rect rect = Rect.fromLTWH(
      offset.dx + (size.width - decoration.width) / 2,
      offset.dy + size.height - _FixedWidthTabIndicator._height,
      decoration.width,
      _FixedWidthTabIndicator._height,
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
