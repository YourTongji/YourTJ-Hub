import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// 全局主题模式(跟随系统 / 手动浅色 / 手动深色)。
///
/// 默认跟随系统;设置页可手动切换,并通过 shared_preferences 持久化,
/// 应用启动时恢复上次选择。
class ThemeModeNotifier extends Notifier<ThemeMode> {
  static const String _prefsKey = 'theme_mode';

  @override
  ThemeMode build() {
    _restore();
    return ThemeMode.system;
  }

  /// 启动时从本地恢复上次的手动选择(异步,失败静默保持跟随系统)。
  Future<void> _restore() async {
    try {
      final SharedPreferences prefs = await SharedPreferences.getInstance();
      final String? saved = prefs.getString(_prefsKey);
      if (saved == null) return;
      state = ThemeMode.values.firstWhere(
        (m) => m.name == saved,
        orElse: () => ThemeMode.system,
      );
    } catch (_) {
      // 无本地存储(如测试环境)时静默保持默认。
    }
  }

  Future<void> _persist(ThemeMode mode) async {
    try {
      final SharedPreferences prefs = await SharedPreferences.getInstance();
      await prefs.setString(_prefsKey, mode.name);
    } catch (_) {
      // 持久化失败不影响本次会话内的切换。
    }
  }

  void setMode(ThemeMode mode) {
    state = mode;
    _persist(mode);
  }

  void toggleDark(bool dark) {
    state = dark ? ThemeMode.dark : ThemeMode.light;
    _persist(state);
  }
}

final themeModeProvider =
    NotifierProvider<ThemeModeNotifier, ThemeMode>(ThemeModeNotifier.new);
