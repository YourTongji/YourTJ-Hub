import 'dart:math' as math;
import 'package:core/core.dart';
import 'package:flutter/foundation.dart' show ValueListenable;
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

class ProfileTabsHeader extends SliverPersistentHeaderDelegate {
  const ProfileTabsHeader({required this.height, required this.child});
  final double height;
  final Widget child;
  @override
  double get minExtent => height;
  @override
  double get maxExtent => height;
  @override
  Widget build(
    BuildContext context,
    double shrinkOffset,
    bool overlapsContent,
  ) => Material(
    color: GfTheme.colorsOf(context).base100,
    child: Column(
      children: [
        Expanded(child: child),
        const GfDivider(),
      ],
    ),
  );
  @override
  bool shouldRebuild(covariant ProfileTabsHeader oldDelegate) => true;
}

class ProfileTabs extends StatefulWidget {
  const ProfileTabs({
    super.key,
    required this.tabs,
    required this.index,
    required this.onChanged,
  });

  final List<TabItemPayload> tabs;
  final int index;
  final ValueChanged<int> onChanged;

  @override
  State<ProfileTabs> createState() => _ProfileTabsState();
}

class _ProfileTabsState extends State<ProfileTabs>
    with SingleTickerProviderStateMixin {
  static const _animationDuration = GfMotion.layout;
  static const _animationCurve = GfLogarithmicEaseOutCurve();

  late final AnimationController _controller = AnimationController(
    vsync: this,
    duration: _animationDuration,
    value: 1,
  )..addStatusListener(_handleAnimationStatus);
  ValueListenable<GfTabSwipeProgress>? _swipeProgressNotifier;
  GfTabSwipeProgress _dragProgress = const GfTabSwipeProgress(
    originIndex: 0,
    offset: 0,
  );
  List<double> _fromShares = [];
  List<double> _displayedShares = [];
  List<double> _fromTapFractions = [];
  List<double> _displayedActiveFractions = [];
  Offset? _fromSegmentShares;
  Offset? _displayedSegmentShares;
  double? _fromIndicatorCenterShare;
  double? _fromIndicatorWidthShare;
  double? _displayedIndicatorCenterShare;
  double? _displayedIndicatorWidthShare;
  bool _disableAnimations = false;
  bool _tapSelection = false;

  void _handleAnimationStatus(AnimationStatus status) {
    if (status != AnimationStatus.completed) return;
    _fromIndicatorCenterShare = null;
    _fromIndicatorWidthShare = null;
    if (_tapSelection) {
      _tapSelection = false;
      _controller.duration = _animationDuration;
      _fromTapFractions = [];
    }
  }

  void _onSwipeProgressChanged() {
    final previous = _dragProgress;
    _dragProgress = _swipeProgressNotifier!.value;
    final progress = _dragProgress;
    if (progress.offset == 0 &&
        previous.offset != 0 &&
        progress.originIndex == widget.index) {
      _fromShares = List.of(_displayedShares);
      _fromSegmentShares = _displayedSegmentShares;
      _fromIndicatorCenterShare = _displayedIndicatorCenterShare;
      _fromIndicatorWidthShare = _displayedIndicatorWidthShare;
      if (_disableAnimations) {
        _controller.value = 1;
      } else {
        _controller.forward(from: 0);
      }
    } else if (progress.targetIndex != null || progress.offset != 0) {
      _tapSelection = false;
      _controller.duration = _animationDuration;
      _fromTapFractions = [];
      _controller.stop();
      _fromShares = [];
      _fromSegmentShares = null;
    }
    if (mounted) setState(() {});
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final progress = GfTabSwipeProgressScope.maybeOf(context);
    if (progress != _swipeProgressNotifier) {
      _swipeProgressNotifier?.removeListener(_onSwipeProgressChanged);
      _swipeProgressNotifier = progress;
      _swipeProgressNotifier?.addListener(_onSwipeProgressChanged);
      _dragProgress =
          progress?.value ??
          const GfTabSwipeProgress(originIndex: 0, offset: 0);
    }
    final disableAnimations = MediaQuery.disableAnimationsOf(context);
    if (disableAnimations && !_disableAnimations) {
      _fromShares = List.of(_displayedShares);
      _fromSegmentShares = _displayedSegmentShares;
      _controller.value = 1;
    }
    _disableAnimations = disableAnimations;
  }

  @override
  void didUpdateWidget(covariant ProfileTabs oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.index == oldWidget.index) return;
    _fromShares = List.of(_displayedShares);
    _fromSegmentShares = _displayedSegmentShares;
    _fromIndicatorCenterShare = _displayedIndicatorCenterShare;
    _fromIndicatorWidthShare = _displayedIndicatorWidthShare;
    if (_disableAnimations) {
      _controller.value = 1;
      _tapSelection = false;
      _controller.duration = _animationDuration;
    } else {
      _controller.duration = _tapSelection
          ? GfMotion.duration(context, GfMotion.selection)
          : _animationDuration;
      _controller.forward(from: 0);
    }
  }

  void _selectTab(int index) {
    if (index == widget.index) return;
    _tapSelection = true;
    _fromTapFractions = _displayedActiveFractions.length == widget.tabs.length
        ? List.of(_displayedActiveFractions)
        : [
            for (var i = 0; i < widget.tabs.length; i++)
              i == widget.index ? 1 : 0,
          ];
    final selectTab = GfTabSwipeProgressScope.onTabSelectedOf(context);
    if (selectTab == null) {
      widget.onChanged(index);
    } else {
      selectTab(index);
    }
  }

  @override
  void dispose() {
    _swipeProgressNotifier?.removeListener(_onSwipeProgressChanged);
    _controller.dispose();
    super.dispose();
  }

  String _iconFor(String key) => switch (key) {
    'timeline' => 'activity',
    'topics' => 'file-text',
    'replies' => 'message-circle',
    'likes' => 'heart',
    'bookmarks' => 'bookmark',
    'badges' => 'award',
    'following' => 'user-round-check',
    'followers' => 'users-round',
    _ => 'circle-user-round',
  };

  double _labelWidth(BuildContext context, String label) {
    final style = DefaultTextStyle.of(
      context,
    ).style.copyWith(fontSize: 16, fontWeight: FontWeight.w600);
    final painter = TextPainter(
      text: TextSpan(text: label, style: style),
      maxLines: 1,
      textDirection: Directionality.of(context),
      textScaler: MediaQuery.textScalerOf(context),
    )..layout();
    final width = painter.width;
    painter.dispose();
    return width;
  }

  @override
  Widget build(BuildContext context) {
    final selectedIndex = widget.index < 0 ? 0 : widget.index;
    return AnimatedBuilder(
      animation: _controller,
      builder: (context, _) => LayoutBuilder(
        builder: (context, constraints) {
          final colors = GfTheme.colorsOf(context);
          final animationDuration = _disableAnimations
              ? Duration.zero
              : _tapSelection
              ? GfMotion.duration(context, GfMotion.content)
              : _animationDuration;
          final animationCurve = _tapSelection
              ? GfMotion.enterCurve
              : _animationCurve;
          final labelWidths = [
            for (final tab in widget.tabs)
              _labelWidth(context, tab.label ?? tab.key),
          ];
          final expandedWidths = [
            for (final width in labelWidths) math.max(48.0, width + 42),
          ];
          final drag = _dragProgress;
          final dragTarget =
              drag.targetIndex ?? drag.originIndex + drag.offset.sign.toInt();
          final bool dragAnimating =
              (drag.offset != 0 || drag.targetIndex != null) &&
              drag.originIndex >= 0 &&
              drag.originIndex < widget.tabs.length &&
              dragTarget >= 0 &&
              dragTarget < widget.tabs.length;
          final double dragFraction = dragAnimating
              ? drag.offset.abs().clamp(0.0, 1.0)
              : 0;
          final tapProgress = _tapSelection
              ? animationCurve.transform(_controller.value)
              : 1.0;
          double activeFraction(int index) {
            if (dragAnimating) {
              if (index == drag.originIndex) return 1 - dragFraction;
              return index == dragTarget ? dragFraction : 0;
            }
            if (_tapSelection &&
                _fromTapFractions.length == widget.tabs.length) {
              final target = index == selectedIndex ? 1.0 : 0.0;
              return _fromTapFractions[index] +
                  (target - _fromTapFractions[index]) * tapProgress;
            }
            return index == selectedIndex ? 1 : 0;
          }

          _displayedActiveFractions = [
            for (var i = 0; i < widget.tabs.length; i++) activeFraction(i),
          ];
          final layoutIndex = dragAnimating ? drag.originIndex : selectedIndex;

          final rowWidth = math.max(
            constraints.maxWidth,
            widget.tabs.length <= 2
                ? expandedWidths.reduce((a, b) => a > b ? a : b) *
                      widget.tabs.length
                : math.max(
                        expandedWidths[layoutIndex],
                        dragAnimating ? expandedWidths[dragTarget] : 0,
                      ) +
                      48.0 * (widget.tabs.length - 1),
          );
          List<double> widthsForSelection(int activeIndex) =>
              widget.tabs.length <= 2
              ? List<double>.filled(
                  widget.tabs.length,
                  rowWidth / widget.tabs.length,
                )
              : [
                  for (int i = 0; i < widget.tabs.length; i++)
                    i == activeIndex
                        ? expandedWidths[i]
                        : (rowWidth - expandedWidths[activeIndex]) /
                              (widget.tabs.length - 1),
                ];
          final targetWidths = widthsForSelection(layoutIndex);
          final destinationWidths = dragAnimating
              ? widthsForSelection(dragTarget)
              : targetWidths;
          final targetShares = [
            for (final width in targetWidths) width / rowWidth,
          ];
          var targetSegmentLeft = 0.0;
          for (int i = 0; i < layoutIndex; i++) {
            targetSegmentLeft += targetShares[i];
          }
          final targetSegment = Offset(
            targetSegmentLeft,
            targetShares[selectedIndex],
          );
          if (_fromShares.length != targetShares.length ||
              _fromSegmentShares == null) {
            _fromShares = List.of(targetShares);
            _fromSegmentShares = targetSegment;
          }

          final progress = _disableAnimations
              ? 1.0
              : animationCurve.transform(_controller.value);
          final shares = [
            for (int i = 0; i < targetShares.length; i++)
              _fromShares[i] + (targetShares[i] - _fromShares[i]) * progress,
          ];
          final baseWidths = [for (final share in shares) share * rowWidth];
          final widths = [
            for (int i = 0; i < baseWidths.length; i++)
              baseWidths[i] +
                  (destinationWidths[i] - baseWidths[i]) * dragFraction,
          ];
          _displayedShares = [for (final width in widths) width / rowWidth];
          final fromSegment = _fromSegmentShares!;
          final activeSegmentShares = Offset(
            fromSegment.dx + (targetSegment.dx - fromSegment.dx) * progress,
            fromSegment.dy + (targetSegment.dy - fromSegment.dy) * progress,
          );
          _displayedSegmentShares = activeSegmentShares;
          final activeLeft = activeSegmentShares.dx * rowWidth;
          final activeWidth = activeSegmentShares.dy * rowWidth;
          final baseIndicatorWidth = math.min(
            64.0,
            math.max(40.0, activeWidth * .72),
          );
          var indicatorCenter = activeLeft + activeWidth / 2;
          var indicatorWidth = baseIndicatorWidth;
          if (dragAnimating) {
            final double targetCenter =
                destinationWidths
                    .take(dragTarget)
                    .fold(0.0, (sum, width) => sum + width) +
                destinationWidths[dragTarget] / 2;
            final double distance = (targetCenter - indicatorCenter).abs();
            indicatorCenter += (targetCenter - indicatorCenter) * dragFraction;
            indicatorWidth = baseIndicatorWidth + distance * dragFraction;
          } else if (_fromIndicatorCenterShare != null &&
              _fromIndicatorWidthShare != null) {
            final double targetCenter =
                (targetSegment.dx + targetSegment.dy / 2) * rowWidth;
            final double targetWidth = math.min(
              64.0,
              math.max(40.0, targetSegment.dy * rowWidth * .72),
            );
            indicatorCenter =
                _fromIndicatorCenterShare! * rowWidth +
                (targetCenter - _fromIndicatorCenterShare! * rowWidth) *
                    progress;
            indicatorWidth =
                _fromIndicatorWidthShare! * rowWidth +
                (targetWidth - _fromIndicatorWidthShare! * rowWidth) * progress;
          }
          _displayedIndicatorCenterShare = indicatorCenter / rowWidth;
          _displayedIndicatorWidthShare = indicatorWidth / rowWidth;
          final indicatorLeft = indicatorCenter - indicatorWidth / 2;
          final height = math.max(
            52.0,
            MediaQuery.textScalerOf(context).scale(16) * 1.4 + 24,
          );

          return SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: SizedBox(
              width: rowWidth,
              height: height,
              child: Stack(
                children: [
                  Row(
                    children: [
                      for (int i = 0; i < widget.tabs.length; i++)
                        SizedBox(
                          key: ValueKey('profile-tab-segment-$i'),
                          width: widths[i],
                          height: height,
                          child: Tooltip(
                            message: widget.tabs[i].label ?? widget.tabs[i].key,
                            excludeFromSemantics: true,
                            child: Semantics(
                              selected: i == selectedIndex,
                              button: true,
                              label: widget.tabs[i].label ?? widget.tabs[i].key,
                              child: InkWell(
                                borderRadius: BorderRadius.circular(8),
                                onTap: () => _selectTab(i),
                                child: Padding(
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 8,
                                  ),
                                  child: Row(
                                    mainAxisAlignment: MainAxisAlignment.center,
                                    mainAxisSize: MainAxisSize.max,
                                    children: [
                                      AnimatedScale(
                                        scale: .9 + .1 * activeFraction(i),
                                        duration: dragAnimating || _tapSelection
                                            ? Duration.zero
                                            : animationDuration,
                                        curve: animationCurve,
                                        child: GfSymbol(
                                          _iconFor(widget.tabs[i].key),
                                          size: 20,
                                          color: Color.lerp(
                                            colors.iconMuted,
                                            colors.baseContent,
                                            activeFraction(i),
                                          ),
                                        ),
                                      ),
                                      Flexible(
                                        fit: FlexFit.loose,
                                        child: AnimatedContainer(
                                          duration:
                                              dragAnimating || _tapSelection
                                              ? Duration.zero
                                              : animationDuration,
                                          curve: animationCurve,
                                          width:
                                              (labelWidths[i] + 6) *
                                              activeFraction(i),
                                          child: ClipRect(
                                            child: AnimatedOpacity(
                                              duration:
                                                  dragAnimating || _tapSelection
                                                  ? Duration.zero
                                                  : animationDuration,
                                              curve: animationCurve,
                                              opacity: activeFraction(i),
                                              child: Padding(
                                                padding: const EdgeInsets.only(
                                                  left: 6,
                                                ),
                                                child: Transform.translate(
                                                  offset: Offset(
                                                    8 * (1 - activeFraction(i)),
                                                    0,
                                                  ),
                                                  child: ExcludeSemantics(
                                                    child: Text(
                                                      widget.tabs[i].label ??
                                                          widget.tabs[i].key,
                                                      maxLines: 1,
                                                      softWrap: false,
                                                      style:
                                                          DefaultTextStyle.of(
                                                            context,
                                                          ).style.copyWith(
                                                            fontSize: 16,
                                                            fontWeight:
                                                                FontWeight.w600,
                                                            color: colors
                                                                .baseContent,
                                                          ),
                                                    ),
                                                  ),
                                                ),
                                              ),
                                            ),
                                          ),
                                        ),
                                      ),
                                    ],
                                  ),
                                ),
                              ),
                            ),
                          ),
                        ),
                    ],
                  ),
                  Positioned(
                    left: indicatorLeft,
                    bottom: 0,
                    width: indicatorWidth,
                    height: 3,
                    child: DecoratedBox(
                      key: const ValueKey('profile-tab-indicator'),
                      decoration: BoxDecoration(
                        color: colors.primary,
                        borderRadius: BorderRadius.circular(3),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}
