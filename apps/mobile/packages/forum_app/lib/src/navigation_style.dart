import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

/// 窄窗口底部导航的外观:悬浮 Dock(默认)或经典底栏。
///
/// 持久化沿用 [ThemeModeNotifier] 的范式:revision 守卫防止恢复与用户选择
/// 竞争,写入串行化,失败静默(不影响本次会话内的切换)。
class NavigationStyleNotifier extends Notifier<GfNavigationStyle> {
  static const String prefsKey = 'navigation_style';
  int _revision = 0;
  bool _disposed = false;
  Future<void> _writes = Future.value();

  @override
  GfNavigationStyle build() {
    ref.onDispose(() => _disposed = true);
    _restore();
    return GfNavigationStyle.dock;
  }

  Future<void> _restore() async {
    final revision = _revision;
    try {
      final SharedPreferences prefs = await SharedPreferences.getInstance();
      if (_disposed || revision != _revision) return;
      final String? saved = prefs.getString(prefsKey);
      if (saved == null) return;
      state = GfNavigationStyle.values.firstWhere(
        (style) => style.name == saved,
        orElse: () => GfNavigationStyle.dock,
      );
    } catch (_) {
      // 无本地存储(如测试环境)时静默保持默认。
    }
  }

  void setStyle(GfNavigationStyle style) {
    _revision++;
    state = style;
    _writes = _writes.then((_) async {
      try {
        final SharedPreferences prefs = await SharedPreferences.getInstance();
        await prefs.setString(prefsKey, style.name);
      } catch (_) {
        // 持久化失败不影响本次会话内的切换。
      }
    });
  }

  /// Drain older writes before local reset so they cannot restore an old style.
  Future<void> resetToDefault() {
    _revision++;
    state = GfNavigationStyle.dock;
    final next = _writes.then((_) async {
      final prefs = await SharedPreferences.getInstance();
      if (!await prefs.remove(prefsKey)) {
        throw StateError('Navigation style reset failed');
      }
    });
    _writes = next.catchError((_) {});
    return next;
  }
}

final navigationStyleProvider =
    NotifierProvider<NavigationStyleNotifier, GfNavigationStyle>(
      NavigationStyleNotifier.new,
    );

/// Gives every route, dialog and sheet the chosen navigation style, so the
/// shell and the pages that clear it measure the same bar.
class AppNavigationStyleScope extends ConsumerWidget {
  const AppNavigationStyleScope({super.key, required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) => GfNavigationStyleScope(
    style: ref.watch(navigationStyleProvider),
    child: child,
  );
}
