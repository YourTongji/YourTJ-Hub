import 'dart:ui';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

const appLanguageNames = {
  'zh': '简体中文',
  'en': 'English',
  'ja': '日本語',
  'de': 'Deutsch',
};

Locale? normalizeAppLocale(String? value) {
  final code = value?.trim().toLowerCase().split(RegExp('[-_,;]')).first;
  return appLanguageNames.containsKey(code) ? Locale(code!) : null;
}

Locale resolveAppLocale(Locale? choice, [List<Locale>? system]) {
  if (choice != null) return choice;
  for (final locale in system ?? PlatformDispatcher.instance.locales) {
    final normalized = normalizeAppLocale(locale.languageCode);
    if (normalized != null) return normalized;
  }
  return const Locale('zh');
}

/// A device preference, independent of public profile language and sessions.
/// Null follows the system; a later user choice wins over asynchronous restore.
class AppLocaleNotifier extends Notifier<Locale?> {
  static const preferenceKey = 'app_locale';
  int _revision = 0;
  bool _disposed = false;
  Future<void> _writes = Future.value();
  @override
  Locale? build() {
    ref.onDispose(() => _disposed = true);
    _restore();
    return null;
  }

  Future<void> _restore() async {
    final revision = _revision;
    try {
      final prefs = await SharedPreferences.getInstance();
      if (!_disposed && revision == _revision) {
        state = normalizeAppLocale(prefs.getString(preferenceKey));
      }
    } catch (_) {
      /* System locale remains available without storage. */
    }
  }

  void setLocale(Locale? locale) {
    _revision++;
    state = normalizeAppLocale(locale?.languageCode);
    final code = state?.languageCode;
    _writes = _writes.then((_) async {
      try {
        final prefs = await SharedPreferences.getInstance();
        if (code == null) {
          await prefs.remove(preferenceKey);
        } else {
          await prefs.setString(preferenceKey, code);
        }
      } catch (_) {
        /* The current session still uses the chosen language. */
      }
    });
  }
}

final appLocaleProvider = NotifierProvider<AppLocaleNotifier, Locale?>(
  AppLocaleNotifier.new,
);
