import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

const drawerSwipeOpeningFraction = .55;

class DrawerGestureGate extends InheritedWidget {
  const DrawerGestureGate({
    super.key,
    required this.policy,
    required super.child,
  });

  final DrawerGesturePolicy policy;

  static DrawerGesturePolicy? maybeOf(BuildContext context) =>
      context.dependOnInheritedWidgetOfExactType<DrawerGestureGate>()?.policy;

  @override
  bool updateShouldNotify(DrawerGestureGate oldWidget) =>
      !identical(policy, oldWidget.policy);
}

class DrawerGesturePolicy {
  final Map<int, bool> _firstTabPointers = {};

  void track(int pointer, {required bool isFirstTab}) {
    _firstTabPointers[pointer] = isFirstTab;
  }

  bool canOpen(int pointer) => _firstTabPointers[pointer] ?? true;

  void release(int pointer) => _firstTabPointers.remove(pointer);
}

/// Lets a page's active horizontal tab change from a body swipe. The drawer
/// only reserves its opening direction within the leading edge band on the
/// first tab; later tabs retain both horizontal directions.
class TabSwipeSurface extends StatefulWidget {
  const TabSwipeSurface({
    super.key,
    required this.index,
    required this.length,
    required this.onChanged,
    required this.child,
    this.tabSelectionDuration,
  });

  final int index;
  final int length;
  final ValueChanged<int> onChanged;
  final Widget child;
  final Duration? tabSelectionDuration;

  @override
  State<TabSwipeSurface> createState() => _TabSwipeSurfaceState();
}

class _TabSwipeSurfaceState extends State<TabSwipeSurface>
    with SingleTickerProviderStateMixin {
  double _distance = 0;
  late final ValueNotifier<GfTabSwipeProgress> _progress = ValueNotifier(
    GfTabSwipeProgress(originIndex: widget.index, offset: 0),
  );
  AnimationController? _settleController;
  AnimationController get _settle => _settleController ??=
      AnimationController(vsync: this, duration: GfMotion.layout)
        ..addListener(_tickSettle)
        ..addStatusListener(_finishSettle);
  int _settleOrigin = 0;
  int? _settleTarget;
  double _settleFrom = 0;
  double _settleTo = 0;
  Curve _settleCurve = GfMotion.layoutCurve;

  void _publishProgress(int originIndex, double offset, {int? targetIndex}) {
    _progress.value = GfTabSwipeProgress(
      originIndex: originIndex,
      offset: offset,
      targetIndex: targetIndex,
    );
  }

  void _tickSettle() {
    final t = _settleCurve.transform(_settle.value);
    _publishProgress(
      _settleOrigin,
      _settleFrom + (_settleTo - _settleFrom) * t,
      targetIndex: _settleTarget,
    );
  }

  void _finishSettle(AnimationStatus status) {
    if (status != AnimationStatus.completed) return;
    final index = _settleTarget != null && widget.index == _settleTarget
        ? _settleTarget!
        : _settleOrigin;
    _publishProgress(index, 0);
    _settleTarget = null;
  }

  void _settleProgress({
    required int originIndex,
    required int? targetIndex,
    required double from,
    required double to,
    Duration? duration,
    Curve? curve,
  }) {
    _settle.stop();
    _settle.duration = duration ?? GfMotion.layout;
    _settleCurve = curve ?? GfMotion.layoutCurve;
    _settleOrigin = originIndex;
    _settleTarget = targetIndex;
    _settleFrom = from;
    _settleTo = to;
    if (GfMotion.reducedOf(context) || from == to) {
      _publishProgress(targetIndex ?? originIndex, 0);
      _settleTarget = null;
      return;
    }
    _publishProgress(originIndex, from, targetIndex: targetIndex);
    _settle.forward(from: 0);
  }

  void _selectTab(int targetIndex) {
    if (targetIndex < 0 || targetIndex >= widget.length) return;
    if (targetIndex == widget.index) return;
    final originIndex = widget.index;
    final direction = targetIndex > originIndex ? 1.0 : -1.0;
    if (GfMotion.reducedOf(context)) {
      widget.onChanged(targetIndex);
      _publishProgress(targetIndex, 0);
      return;
    }
    _settleProgress(
      originIndex: originIndex,
      targetIndex: targetIndex,
      from: 0,
      to: direction,
      duration: widget.tabSelectionDuration,
      curve: GfMotion.enterCurve,
    );
    widget.onChanged(targetIndex);
  }

  @override
  void dispose() {
    _settleController?.dispose();
    _progress.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final policy = DrawerGestureGate.maybeOf(context);
    final isFirstTab = widget.length <= 1 || widget.index <= 0;
    return GfTabSwipeProgressScope(
      progress: _progress,
      onTabSelected: _selectTab,
      child: Listener(
        behavior: HitTestBehavior.translucent,
        onPointerDown: (event) =>
            policy?.track(event.pointer, isFirstTab: isFirstTab),
        onPointerUp: (event) => policy?.release(event.pointer),
        onPointerCancel: (event) => policy?.release(event.pointer),
        child: RawGestureDetector(
          behavior: HitTestBehavior.translucent,
          excludeFromSemantics: true,
          gestures: {
            _TabSwipeGestureRecognizer:
                GestureRecognizerFactoryWithHandlers<
                  _TabSwipeGestureRecognizer
                >(
                  _TabSwipeGestureRecognizer.new,
                  (recognizer) => recognizer
                    ..screenWidth = MediaQuery.sizeOf(context).width
                    ..drawerEdgeWidth =
                        MediaQuery.sizeOf(context).width *
                        drawerSwipeOpeningFraction
                    ..drawerOpenDirection =
                        Directionality.of(context) == TextDirection.ltr ? 1 : -1
                    ..drawerSwipeEnabled = isFirstTab
                    ..onStart = (_) {
                      _settle.stop();
                      _distance = 0;
                      _publishProgress(widget.index, 0);
                    }
                    ..onUpdate = (details) {
                      _distance += details.primaryDelta ?? 0;
                      final int next =
                          widget.index +
                          (_distance * recognizer.drawerOpenDirection < 0
                              ? 1
                              : -1);
                      final double offset = next >= 0 && next < widget.length
                          ? (-_distance *
                                    recognizer.drawerOpenDirection /
                                    recognizer.screenWidth)
                                .clamp(-.95, .95)
                                .toDouble()
                          : 0;
                      _publishProgress(
                        widget.index,
                        offset,
                        targetIndex: offset == 0 ? null : next,
                      );
                    }
                    ..onCancel = () {
                      _settleProgress(
                        originIndex: widget.index,
                        targetIndex: null,
                        from: _progress.value.offset,
                        to: 0,
                      );
                    }
                    ..onEnd = (details) {
                      final velocity = details.velocity.pixelsPerSecond.dx;
                      final distance = _distance.abs() >= 48
                          ? _distance
                          : velocity.abs() >= 500
                          ? velocity
                          : 0.0;
                      if (distance == 0 ||
                          widget.length < 2 ||
                          widget.index < 0 ||
                          widget.index >= widget.length) {
                        _settleProgress(
                          originIndex: widget.index,
                          targetIndex: null,
                          from: _progress.value.offset,
                          to: 0,
                        );
                        return;
                      }
                      final int next =
                          widget.index +
                          (distance * recognizer.drawerOpenDirection < 0
                              ? 1
                              : -1);
                      if (next < 0 || next >= widget.length) {
                        _settleProgress(
                          originIndex: widget.index,
                          targetIndex: null,
                          from: _progress.value.offset,
                          to: 0,
                        );
                        return;
                      }
                      final int originIndex = widget.index;
                      final offset = _progress.value.offset;
                      final direction = next > originIndex ? 1.0 : -1.0;
                      _settleProgress(
                        originIndex: originIndex,
                        targetIndex: next,
                        from: offset,
                        to: direction,
                      );
                      widget.onChanged(next);
                    },
                ),
          },
          child: widget.child,
        ),
      ),
    );
  }
}

class _TabSwipeGestureRecognizer extends HorizontalDragGestureRecognizer {
  _TabSwipeGestureRecognizer()
    : super(supportedDevices: const {PointerDeviceKind.touch}) {
    onlyAcceptDragOnThreshold = true;
  }

  double drawerEdgeWidth = 0;
  double screenWidth = 0;
  int drawerOpenDirection = 1;
  bool drawerSwipeEnabled = true;
  final Map<int, Offset> _origins = {};
  final Set<int> _acceptedPointers = {};

  @override
  void addAllowedPointer(PointerDownEvent event) {
    _origins[event.pointer] = event.position;
    super.addAllowedPointer(event);
  }

  @override
  void handleEvent(PointerEvent event) {
    final origin = _origins[event.pointer];
    if (origin != null && event is PointerMoveEvent) {
      final delta = event.position - origin;
      final slop = computeHitSlop(event.kind, gestureSettings);
      final inDrawerZone = drawerOpenDirection > 0
          ? origin.dx <= drawerEdgeWidth
          : origin.dx >= screenWidth - drawerEdgeWidth;
      if (drawerSwipeEnabled &&
          inDrawerZone &&
          delta.dx * drawerOpenDirection > slop) {
        // Once this recognizer won, rejecting is a no-op; keep its moves flowing.
        if (!_acceptedPointers.contains(event.pointer)) {
          resolve(GestureDisposition.rejected);
          return;
        }
      }
    }
    if (event is PointerUpEvent || event is PointerCancelEvent) {
      _origins.remove(event.pointer);
      _acceptedPointers.remove(event.pointer);
    }
    super.handleEvent(event);
  }

  @override
  void acceptGesture(int pointer) {
    _acceptedPointers.add(pointer);
    super.acceptGesture(pointer);
  }

  @override
  void rejectGesture(int pointer) {
    _origins.remove(pointer);
    _acceptedPointers.remove(pointer);
    super.rejectGesture(pointer);
  }
}
