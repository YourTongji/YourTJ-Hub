import 'dart:async';
import 'package:core/core.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../providers.dart';

/// A bounded foreground-only cache, separate from disposable private views.
/// No credentials, grades, notice bodies or persistent storage belong here.
class CampusMemoryCache with WidgetsBindingObserver {
  CampusMemoryCache({DateTime Function()? now}) : now = now ?? DateTime.now;
  static const lifetime = Duration(minutes: 5);
  static const keys = {'profile', 'calendar', 'timetable', 'today', 'messages'};
  static const dailyKeys = {'calendar', 'timetable', 'today'};
  final DateTime Function() now;
  final _entries = <String, ({CampusDataset data, DateTime receivedAt})>{};
  final _expiry = <String, Timer>{};
  String? _revision;
  int generation = 0;
  bool _foreground = true;
  bool _disposed = false;

  bool isCurrent(int fence) => !_disposed && _foreground && fence == generation;

  static String schoolDate(DateTime value) => value
      .toUtc()
      .add(const Duration(hours: 8))
      .toIso8601String()
      .substring(0, 10);

  bool fresh(String key, DateTime receivedAt) =>
      !now().isBefore(receivedAt) &&
      now().difference(receivedAt) < lifetime &&
      (!dailyKeys.contains(key) || schoolDate(receivedAt) == schoolDate(now()));

  /// Called only after a fresh status request has verified this binding.
  Map<String, ({CampusDataset data, DateTime receivedAt})> restore(
    String? revision,
  ) {
    if (revision != _revision) clear();
    _revision = revision;
    for (final key in _entries.keys.toList()) {
      if (!fresh(key, _entries[key]!.receivedAt)) forget(key);
    }
    return _foreground && !_disposed && revision != null
        ? Map.of(_entries)
        : {};
  }

  void put(
    String revision,
    CampusDataset data,
    DateTime receivedAt,
    int fence,
  ) {
    if (_disposed ||
        !_foreground ||
        fence != generation ||
        revision != _revision ||
        !keys.contains(data.key) ||
        !fresh(data.key, receivedAt)) {
      return;
    }
    if (!const ['ready', 'empty'].contains(data.status)) {
      forget(data.key);
      return;
    }
    _entries[data.key] = (data: data, receivedAt: receivedAt);
    _expiry.remove(data.key)?.cancel();
    // Each entry has its own fixed timer. Reuse and wall-clock rollback cannot
    // extend retention, even while no campus view is listening.
    _expiry[data.key] = Timer(lifetime - now().difference(receivedAt), () {
      _entries.remove(data.key);
      _expiry.remove(data.key);
    });
  }

  void forget(String key) {
    _entries.remove(key);
    _expiry.remove(key)?.cancel();
  }

  void clear() {
    generation++;
    for (final timer in _expiry.values) {
      timer.cancel();
    }
    _expiry.clear();
    _entries.clear();
    _revision = null;
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    if (!_foreground) clear();
  }

  void dispose() {
    _disposed = true;
    clear();
  }
}

final campusMemoryCacheProvider = Provider<CampusMemoryCache>((ref) {
  ref.watch(offlineCacheEpochProvider);
  // Replacing the API repository (including a site change) drops its data too.
  ref.watch(campusRepositoryProvider);
  final cache = CampusMemoryCache();
  WidgetsBinding.instance.addObserver(cache);
  cache.didChangeAppLifecycleState(
    WidgetsBinding.instance.lifecycleState ?? AppLifecycleState.resumed,
  );
  ref.onDispose(() {
    WidgetsBinding.instance.removeObserver(cache);
    cache.dispose();
  });
  return cache;
});
