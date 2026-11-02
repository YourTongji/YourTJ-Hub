import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'tab_scroll_registry.dart';

/// Sections inside shell destinations: notification categories and campus
/// tabs. Pages report the section they show, so the dock's long-press menu
/// can mark it, and take the section a menu asked for when they next build.
class ShellSections extends ChangeNotifier {
  final Map<GfShellDestination, String> _current = {};
  (GfShellDestination, String)? _pending;

  String? currentOf(GfShellDestination destination) => _current[destination];

  /// Records what [destination] shows. Silent: the menu reads it on demand.
  void report(GfShellDestination destination, String section) =>
      _current[destination] = section;

  /// Asks [destination] to show [section]; a page that is not built yet takes
  /// it on its first build.
  void request(GfShellDestination destination, String section) {
    _pending = (destination, section);
    notifyListeners();
  }

  /// The section requested for [destination], consumed by this call.
  String? take(GfShellDestination destination) {
    final pending = _pending;
    if (pending == null || pending.$1 != destination) return null;
    _pending = null;
    return pending.$2;
  }
}

final shellSectionsProvider = ChangeNotifierProvider<ShellSections>(
  (ref) => ShellSections(),
);
