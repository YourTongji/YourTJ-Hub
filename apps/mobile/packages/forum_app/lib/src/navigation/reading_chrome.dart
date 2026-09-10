import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Direction hysteresis ignores bounce, horizontal gestures and tiny reversals.
/// Moving chrome never changes a scroll view's viewport or pixel offset.
class ReadingChrome extends ChangeNotifier {
  bool hidden = false;
  double _travel = 0;
  int _direction = 0;

  void show() {
    _travel = 0;
    _direction = 0;
    if (!hidden) return;
    hidden = false;
    notifyListeners();
  }

  void update(double delta, double pixels, {bool locked = false}) {
    if (locked || pixels <= 0) {
      show();
      return;
    }
    if (delta == 0) return;
    final direction = delta > 0 ? 1 : -1;
    if (direction != _direction) _travel = 0;
    _direction = direction;
    _travel += delta.abs();
    if ((!hidden && direction == 1 && _travel >= 48 && pixels >= 48) ||
        (hidden && direction == -1 && _travel >= 12)) {
      hidden = direction == 1;
      _travel = 0;
      notifyListeners();
    }
  }
}

final readingChromeProvider = ChangeNotifierProvider<ReadingChrome>(
  (ref) => ReadingChrome(),
);
