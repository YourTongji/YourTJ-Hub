import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';

/// Only a stable, freshly measured viewport can qualify IDs. This class never
/// infers visibility from the highest message ID or from list construction.
class VisibleChatReads {
  VisibleChatReads({
    required this.isEnabled,
    required this.visibleIds,
    required this.acknowledge,
    required this.onAcknowledged,
    required this.onFailure,
    this.canRetry,
  });

  final bool Function() isEnabled;
  final Set<int> Function() visibleIds;
  final Future<Set<int>> Function(List<int>) acknowledge;
  final void Function(Set<int>) onAcknowledged;
  final void Function(Object) onFailure;
  final bool Function(Object)? canRetry;
  Timer? _dwell;
  Timer? _retry;
  Set<int> _candidates = {};
  final Set<int> _qualified = {};
  final Set<int> _acknowledged = {};
  bool _scheduled = false;
  bool _sending = false;
  bool _disposed = false;
  int _generation = 0;
  int _failures = 0;

  void changed({bool restartDwell = false}) {
    if (_disposed) return;
    if (restartDwell) {
      _dwell?.cancel();
      _dwell = null;
      _candidates.clear();
    }
    if (_scheduled) return;
    _scheduled = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _scheduled = false;
      if (!_disposed) _sample();
    });
    WidgetsBinding.instance.ensureVisualUpdate();
  }

  void suspend() {
    _generation++;
    _dwell?.cancel();
    _dwell = null;
    _retry?.cancel();
    _retry = null;
    _candidates.clear();
    _qualified.clear();
  }

  void retry() {
    _failures = 0;
    changed(restartDwell: true);
  }

  Set<int> _visible() => visibleIds().difference(_acknowledged);

  void _sample() {
    if (!isEnabled()) {
      suspend();
      return;
    }
    final visible = _visible();
    _qualified.retainAll(visible);
    if (setEquals(visible, _candidates)) return;
    _dwell?.cancel();
    _candidates = visible;
    if (visible.isEmpty || _failures >= 2) return;
    _dwell = Timer(const Duration(milliseconds: 350), () {
      _dwell = null;
      if (_disposed || !isEnabled()) return;
      _qualified.addAll(_candidates.intersection(_visible()));
      unawaited(_flush());
    });
  }

  Future<void> _flush() async {
    if (_disposed || _sending || !isEnabled() || _failures >= 2) return;
    _qualified.retainAll(_visible());
    if (_qualified.isEmpty) return;
    final ids = _qualified.take(100).toList();
    _qualified.removeAll(ids);
    _sending = true;
    final generation = _generation;
    try {
      final confirmed = await acknowledge(ids);
      if (_disposed || generation != _generation || !isEnabled()) return;
      final accepted = confirmed.intersection(ids.toSet());
      _acknowledged.addAll(accepted);
      _failures = 0;
      onAcknowledged(accepted);
    } catch (error) {
      if (_disposed || generation != _generation || !isEnabled()) return;
      _failures = canRetry?.call(error) == false ? 2 : _failures + 1;
      onFailure(error);
      // One bounded automatic retry, then an explicit retry in the UI. A new
      // visibility boundary always requires a fresh dwell before another write.
      if (_failures < 2) {
        _retry = Timer(const Duration(seconds: 1), () {
          if (!_disposed && isEnabled()) changed(restartDwell: true);
        });
      }
    } finally {
      _sending = false;
      if (!_disposed && _failures == 0) {
        // A newer visibility generation may have completed its own dwell while
        // this old request was in flight. Its newly qualified IDs can now flush.
        unawaited(_flush());
      }
    }
  }

  void dispose() {
    _disposed = true;
    suspend();
  }
}

/// A long bubble can qualify without fitting wholly in the viewport. Geometry
/// uses the actual bubble and the keyboard-clipped list, in one coordinate space.
bool bubbleIsVisible(Rect bubble, Rect viewport) {
  if (bubble.isEmpty || viewport.isEmpty) return false;
  final overlap = bubble.intersect(viewport);
  if (overlap.isEmpty) return false;
  final height = bubble.height < viewport.height
      ? bubble.height
      : viewport.height;
  final width = bubble.width < viewport.width ? bubble.width : viewport.width;
  return overlap.height >= height * .5 && overlap.width >= width * .5;
}
