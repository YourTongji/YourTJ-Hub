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
  });

  final int index;
  final int length;
  final ValueChanged<int> onChanged;
  final Widget child;

  @override
  State<TabSwipeSurface> createState() => _TabSwipeSurfaceState();
}

class _TabSwipeSurfaceState extends State<TabSwipeSurface> {
  double _distance = 0;
  late final ValueNotifier<GfTabSwipeProgress> _progress = ValueNotifier(
    GfTabSwipeProgress(originIndex: widget.index, offset: 0),
  );

  void _publishProgress(int originIndex, double offset) {
    _progress.value = GfTabSwipeProgress(
      originIndex: originIndex,
      offset: offset,
    );
  }

  @override
  void dispose() {
    _progress.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final policy = DrawerGestureGate.maybeOf(context);
    final isFirstTab = widget.length <= 1 || widget.index <= 0;
    return GfTabSwipeProgressScope(
      progress: _progress,
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
                      _publishProgress(widget.index, offset);
                    }
                    ..onCancel = () {
                      _publishProgress(widget.index, 0);
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
                        _publishProgress(widget.index, 0);
                        return;
                      }
                      final int next =
                          widget.index +
                          (distance * recognizer.drawerOpenDirection < 0
                              ? 1
                              : -1);
                      if (next < 0 || next >= widget.length) {
                        _publishProgress(widget.index, 0);
                        return;
                      }
                      final int originIndex = widget.index;
                      widget.onChanged(next);
                      WidgetsBinding.instance.addPostFrameCallback((_) {
                        if (!mounted) return;
                        final bool changed = widget.index == next;
                        _publishProgress(changed ? next : originIndex, 0);
                      });
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
