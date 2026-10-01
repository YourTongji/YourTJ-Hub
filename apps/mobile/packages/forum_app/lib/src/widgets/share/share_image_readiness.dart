import 'dart:async';

import 'package:flutter/widgets.dart';

class ShareImageReadiness
    extends InheritedNotifier<ShareImageReadinessTracker> {
  const ShareImageReadiness({
    super.key,
    required ShareImageReadinessTracker tracker,
    required super.child,
  }) : super(notifier: tracker);
  ShareImageReadinessTracker get tracker => notifier!;

  static ShareImageReadinessTracker? maybeOf(BuildContext context) => context
      .dependOnInheritedWidgetOfExactType<ShareImageReadiness>()
      ?.tracker;

  @override
  bool updateShouldNotify(ShareImageReadiness oldWidget) =>
      tracker != oldWidget.tracker;
}

class ShareImageReadinessTracker extends ChangeNotifier {
  ShareImageReadinessTracker({this.timeout = const Duration(seconds: 20)});
  final Duration timeout;
  final Map<String, Completer<void>> _pending = {};
  final Set<String> _frozen = {};
  bool _disposed = false;

  void begin(String key) {
    if (!_disposed) _pending.putIfAbsent(key, Completer<void>.new);
  }

  bool isFrozen(String key) => _frozen.contains(key);
  void reset() {
    if (_disposed) return;
    _pending.clear();
    _frozen.clear();
    notifyListeners();
  }

  void finish(String key) {
    final completer = _pending[key];
    if (completer != null && !completer.isCompleted) completer.complete();
  }

  Future<void> wait() async {
    if (_pending.isEmpty) return;
    try {
      await Future.wait(
        _pending.values.map((item) => item.future),
      ).timeout(timeout);
    } on TimeoutException {
      for (final entry in _pending.entries) {
        if (!entry.value.isCompleted) _frozen.add(entry.key);
      }
      if (!_disposed) notifyListeners();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    for (final completer in _pending.values) {
      if (!completer.isCompleted) completer.complete();
    }
    _pending.clear();
    _frozen.clear();
    super.dispose();
  }
}
