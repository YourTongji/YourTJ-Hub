import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Explicit device-local opt-in. Restoring storage never overrides a newer choice.
class AnalyticsConsent extends Notifier<bool> {
  static const preferenceKey = 'visitor_analytics_opt_in';
  int _revision = 0;
  bool _disposed = false;
  Future<void> _writes = Future.value();

  @override
  bool build() {
    ref.onDispose(() => _disposed = true);
    _restore();
    return false;
  }

  Future<void> _restore() async {
    final revision = _revision;
    try {
      final prefs = await SharedPreferences.getInstance();
      if (!_disposed && revision == _revision) {
        state = prefs.getBool(preferenceKey) ?? false;
      }
    } catch (_) {
      // Without a saved choice, collection remains off.
    }
  }

  Future<bool> setEnabled(bool enabled) async {
    final revision = ++_revision;
    if (!enabled) state = false; // Stop immediately, even if storage fails.
    bool saved = false;
    _writes = _writes.then((_) async {
      try {
        final prefs = await SharedPreferences.getInstance();
        saved = await prefs.setBool(preferenceKey, enabled);
      } catch (_) {
        saved = false;
      }
      if (!_disposed && revision == _revision && saved) state = enabled;
    });
    await _writes;
    return saved;
  }
}

final analyticsConsentProvider = NotifierProvider<AnalyticsConsent, bool>(
  AnalyticsConsent.new,
);
