import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// 全局主题模式(跟随系统 / 手动浅色 / 手动深色)。
///
/// 默认跟随系统;设置页可手动切换,并通过 shared_preferences 持久化,
/// 应用启动时恢复上次选择。
class ThemeModeNotifier extends Notifier<ThemeMode> {
  static const String _prefsKey = 'theme_mode';
  static const MethodChannel _startupChannel = MethodChannel('yourtj/startup');
  int _revision = 0;
  bool _disposed = false;
  Future<void> _writes = Future.value();

  @override
  ThemeMode build() {
    ref.onDispose(() => _disposed = true);
    _restore();
    return ThemeMode.system;
  }

  /// 启动时从本地恢复上次的手动选择(异步,失败静默保持跟随系统)。
  Future<void> _restore() async {
    final revision = _revision;
    try {
      final SharedPreferences prefs = await SharedPreferences.getInstance();
      if (_disposed || revision != _revision) return;
      final String? saved = prefs.getString(_prefsKey);
      if (saved == null) return;
      final mode = ThemeMode.values.firstWhere(
        (mode) => mode.name == saved,
        orElse: () => ThemeMode.system,
      );
      state = mode;
      if (mode != ThemeMode.system) unawaited(_syncNativeMode(mode));
    } catch (_) {
      // 无本地存储(如测试环境)时静默保持默认。
    }
  }

  Future<void> _persist(ThemeMode mode) async {
    try {
      final SharedPreferences prefs = await SharedPreferences.getInstance();
      await prefs.setString(_prefsKey, mode.name);
      await _syncNativeMode(mode);
    } catch (_) {
      // 持久化失败不影响本次会话内的切换。
    }
  }

  Future<void> _syncNativeMode(ThemeMode mode) async {
    if (kIsWeb || defaultTargetPlatform != TargetPlatform.android) return;
    try {
      await _startupChannel.invokeMethod<bool>('setThemeMode', {
        'mode': mode.name,
      });
    } on PlatformException {
      // Keep the Flutter theme usable if the device rejects native syncing.
    } on MissingPluginException {
      // The native bridge is unavailable in widget tests and non-app engines.
    }
  }

  void setMode(ThemeMode mode) {
    _revision++;
    state = mode;
    // Platform writes may finish out of order; persist each selection only after
    // the previous write completes. A failed write does not block later choices.
    _writes = _writes.then((_) => _persist(mode));
  }

  void toggleDark(bool dark) {
    setMode(dark ? ThemeMode.dark : ThemeMode.light);
  }
}

final themeModeProvider = NotifierProvider<ThemeModeNotifier, ThemeMode>(
  ThemeModeNotifier.new,
);
