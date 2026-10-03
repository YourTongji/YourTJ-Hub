import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

/// 字号偏好的共同持久化逻辑(默认 100%)。
///
/// 持久化沿用 [ThemeModeNotifier] 的范式:revision 守卫防止恢复与用户选择
/// 竞争,写入串行化,失败静默(不影响本次会话内的调整)。拖动滑块只更新会话
/// 内状态并重启 [persistDelay] 防抖计时器,松手([persistScale])或停止拖动
/// 后计时器触发才落盘,避免一次滑动产生几十次 SharedPreferences 写;计时器
/// 活在 app 级 provider 里,设置页在拖动中途被销毁也不丢这次调整。
abstract class FontScaleNotifier extends Notifier<double> {
  FontScaleNotifier({required this.persistDelay});

  static const double defaultScale = 1;

  /// SharedPreferences 键。
  String get storageKey;

  /// 越界值向滑块范围收敛。
  double clampScale(double scale);

  /// 停止调整后多久落盘;测试里注入 [Duration.zero]。
  final Duration persistDelay;

  int _revision = 0;
  bool _disposed = false;
  Future<void> _writes = Future<void>.value();
  Timer? _persistTimer;

  @override
  double build() {
    ref.onDispose(() {
      _disposed = true;
      _persistTimer?.cancel();
    });
    _restore();
    return defaultScale;
  }

  /// 启动时恢复上次的阅读字号(异步,失败静默保持默认)。
  Future<void> _restore() async {
    final revision = _revision;
    try {
      final SharedPreferences prefs = await SharedPreferences.getInstance();
      if (_disposed || revision != _revision) return;
      // Tolerate a stored int (or any numeric encoding) from older builds.
      final Object? saved = prefs.get(storageKey);
      if (saved is! num) return;
      state = clampScale(saved.toDouble());
    } catch (_) {
      // 无本地存储(如测试环境)时静默保持默认。
    }
  }

  /// 恢复默认的 100%,作为离散操作立即落盘。
  void resetToDefault() {
    setScale(defaultScale);
    persistScale();
  }

  void setScale(double scale) {
    _revision++;
    state = clampScale(scale);
    _persistTimer?.cancel();
    _persistTimer = Timer(persistDelay, persistScale);
  }

  /// 立即把当前值落盘;滑块松手(onChangeEnd)时调用。
  void persistScale() {
    _persistTimer?.cancel();
    _persistTimer = null;
    _writes = _writes.then((_) => _persist(state));
  }

  Future<void> _persist(double scale) async {
    try {
      final SharedPreferences prefs = await SharedPreferences.getInstance();
      await prefs.setDouble(storageKey, scale);
    } catch (_) {
      // 持久化失败不影响本次会话内的调整。
    }
  }
}

/// 阅读字号偏好(80%–140%,默认 100%)。
///
/// 只作用于富内容(帖子正文/回复、Wiki、课程评价阅读态),在全局字号基础上
/// 再调整;与全局字号、系统 `TextScaler` 相乘而非替代 —— 最终字号由
/// `GfRichContentTypography` 计算后再交给 [GfAppTextScaler]。
class ContentFontScaleNotifier extends FontScaleNotifier {
  ContentFontScaleNotifier({
    super.persistDelay = const Duration(milliseconds: 600),
  });

  static const String prefsKey = 'content_font_scale';

  @override
  String get storageKey => prefsKey;

  @override
  double clampScale(double scale) =>
      GfRichContentTypography.clampUserScale(scale);
}

final contentFontScaleProvider =
    NotifierProvider<ContentFontScaleNotifier, double>(
      ContentFontScaleNotifier.new,
    );

/// 全局字号偏好(90%–130%,默认 100%)。
///
/// 由 app 根部的 [GfAppTextScaler] 叠加在设备默认字号与系统字体缩放
/// 之上,作用于整个 App 的文字(含富内容正文,阅读字号再在其上调整)。
class AppFontScaleNotifier extends FontScaleNotifier {
  AppFontScaleNotifier({
    super.persistDelay = const Duration(milliseconds: 600),
  });

  static const String prefsKey = 'app_font_scale';

  @override
  String get storageKey => prefsKey;

  @override
  double clampScale(double scale) =>
      GfAppTextScaler.clampUserScale(scale);
}

final appFontScaleProvider =
    NotifierProvider<AppFontScaleNotifier, double>(
      AppFontScaleNotifier.new,
    );

/// Layers the device-adapted default and the app text size preference
/// over the system text scaler for every route, dialog and sheet.
class AppTextScaleScope extends ConsumerWidget {
  const AppTextScaleScope({super.key, required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final MediaQueryData media = MediaQuery.of(context);
    final TextScaler scaler = GfAppTextScaler(
      system: media.textScaler,
      baseline: GfAppTextScaler.baselineFor(
        platform: Theme.of(context).platform,
        shortestSide: media.size.shortestSide,
      ),
      userScale: ref.watch(appFontScaleProvider),
    );
    return MediaQuery(
      data: media.copyWith(textScaler: scaler),
      child: child,
    );
  }
}
