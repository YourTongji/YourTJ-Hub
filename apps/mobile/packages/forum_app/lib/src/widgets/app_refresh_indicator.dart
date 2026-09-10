import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';

/// Native refresh animation with a short, viewport-independent pull gesture.
/// [edgeOffset] keeps the indicator below overlaid navigation controls.
class AppRefreshIndicator extends StatefulWidget {
  const AppRefreshIndicator({
    super.key,
    required this.child,
    required this.onRefresh,
    this.edgeOffset = 0,
  });
  final Widget child;
  final RefreshCallback onRefresh;
  final double edgeOffset;

  @override
  State<AppRefreshIndicator> createState() => _AppRefreshIndicatorState();
}

class _AppRefreshIndicatorState extends State<AppRefreshIndicator> {
  static const double _pullDistance = 72;
  final _indicator = GlobalKey<RefreshIndicatorState>();
  bool _tracking = false;
  bool _requested = false;
  double _distance = 0;

  void _release() {
    final refresh = _tracking && _distance >= _pullDistance;
    _tracking = false;
    _distance = 0;
    if (!refresh || _requested) return;
    final indicator = _indicator.currentState;
    if (indicator == null) return;
    _requested = true;
    // show() coalesces with an already-armed native indicator. This retains
    // native loading/completion semantics instead of invoking the API twice.
    unawaited(indicator.show().whenComplete(() => _requested = false));
  }

  bool _onScroll(ScrollNotification notification) {
    if (notification.depth != 0 ||
        notification.metrics.axisDirection != AxisDirection.down) {
      return false;
    }
    if (notification is ScrollStartNotification) {
      _tracking =
          notification.dragDetails != null &&
          notification.metrics.extentBefore == 0 &&
          !_requested;
      _distance = 0;
    } else if (_tracking) {
      if (notification.metrics.extentBefore > 0) {
        _tracking = false;
        _distance = 0;
      } else if (notification is ScrollEndNotification) {
        _release();
      } else {
        final details = switch (notification) {
          ScrollUpdateNotification() => notification.dragDetails,
          OverscrollNotification() => notification.dragDetails,
          _ => null,
        };
        if (details != null) {
          // Finger travel, not the rubber-band displacement: iOS damping must
          // not require a much longer pull than Android or a shorter viewport.
          _distance = math.max(0, _distance + details.delta.dy);
        } else if (notification is ScrollUpdateNotification ||
            notification is OverscrollNotification) {
          // iOS begins its ballistic return before emitting ScrollEnd.
          _release();
        }
      }
    }
    return false;
  }

  @override
  Widget build(BuildContext context) => RefreshIndicator(
    key: _indicator,
    edgeOffset: widget.edgeOffset,
    displacement: 32,
    onRefresh: widget.onRefresh,
    child: NotificationListener<ScrollNotification>(
      onNotification: _onScroll,
      child: widget.child,
    ),
  );
}
