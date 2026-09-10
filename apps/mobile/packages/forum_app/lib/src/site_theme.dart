import 'dart:convert';

import 'package:core/core.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'providers.dart';

/// 站点主题同步状态。
class SiteThemeState {
  const SiteThemeState({
    required this.available,
    required this.following,
    this.runtime,
  });

  /// 服务端已启用站点主题（决定设置页开关是否展示）。
  final bool available;

  /// 用户「跟随站点主题」开关（默认开；关闭即回内置主题）。
  final bool following;

  /// 当前生效的运行时覆盖（following && available 时非 null）。
  final GfRuntimeTheme? runtime;

  SiteThemeState copyWith({
    bool? available,
    bool? following,
    GfRuntimeTheme? runtime,
  }) => SiteThemeState(
    available: available ?? this.available,
    following: following ?? this.following,
    runtime: runtime ?? this.runtime,
  );
}

/// 站点主题同步控制器（Route A 主题同步）。
///
/// - 启动时先从本地缓存恢复（key `site_theme_cache`），无网也即刻生效；
/// - 再拉取 `GET /api/site-theme/tokens` 刷新（失败静默保留缓存）；
/// - 「跟随站点主题」关闭时 runtime 置 null（GfApp 回退内置 tokens.json 主题）；
/// - 服务端 `enabled:false` 时 available=false（设置页隐藏开关），runtime 置 null。
class SiteThemeController extends Notifier<SiteThemeState> {
  static const String _followKey = 'follow_site_theme';
  static const String _cacheKey = 'site_theme_cache';

  @override
  SiteThemeState build() {
    _restoreAndRefresh();
    return const SiteThemeState(available: false, following: true);
  }

  Future<void> _restoreAndRefresh() async {
    bool following = true;
    Map<String, Map<String, String>>? cached;
    try {
      final SharedPreferences prefs = await SharedPreferences.getInstance();
      following = prefs.getBool(_followKey) ?? true;
      final String? raw = prefs.getString(_cacheKey);
      if (raw != null) {
        final Map<String, dynamic> decoded =
            jsonDecode(raw) as Map<String, dynamic>;
        cached = (decoded['themes'] as Map<String, dynamic>? ?? const {}).map(
          (mode, tokens) => MapEntry(
            mode,
            (tokens as Map<String, dynamic>).map(
              (k, v) => MapEntry(k, v.toString()),
            ),
          ),
        );
      }
    } catch (_) {
      // 无本地存储（测试/权限异常）时按默认值继续。
    }
    _apply(available: state.available, following: following, themes: cached);

    // 后台刷新：失败静默保留缓存态。
    try {
      final SiteThemePublicPayload? payload = await ref
          .read(themeRepositoryProvider)
          .fetchTokens();
      if (payload == null) return; // 旧后端无端点：保留缓存态
      final Map<String, Map<String, String>> themes = payload.enabled
          ? {for (final t in payload.themes) t.mode: t.tokens}
          : const {};
      if (themes.isNotEmpty) {
        _persistCache(themes);
      }
      _apply(
        available: payload.enabled && themes.isNotEmpty,
        following: following,
        themes: themes,
      );
    } catch (_) {
      // 网络失败：保留当前态（缓存或内置）。
    }
  }

  void _apply({
    required bool available,
    required bool following,
    Map<String, Map<String, String>>? themes,
  }) {
    GfRuntimeTheme? runtime;
    if (following && available && themes != null) {
      runtime = GfRuntimeTheme.fromTokens(themes) ?? runtime;
    }
    state = SiteThemeState(
      available: available,
      following: following,
      runtime: runtime,
    );
  }

  Future<void> _persistCache(Map<String, Map<String, String>> themes) async {
    try {
      final SharedPreferences prefs = await SharedPreferences.getInstance();
      await prefs.setString(_cacheKey, jsonEncode({'themes': themes}));
    } catch (_) {
      // 持久化失败不影响本次会话。
    }
  }

  /// 切换「跟随站点主题」：关闭立即回内置主题，开启时用缓存即刻生效并后台刷新。
  Future<void> setFollowing(bool value) async {
    state = state.copyWith(following: value);
    if (!value) {
      state = SiteThemeState(available: state.available, following: false);
    } else {
      await _restoreAndRefresh();
      return;
    }
    try {
      final SharedPreferences prefs = await SharedPreferences.getInstance();
      await prefs.setBool(_followKey, value);
    } catch (_) {
      // 持久化失败保留会话内选择。
    }
  }
}

final siteThemeProvider = NotifierProvider<SiteThemeController, SiteThemeState>(
  SiteThemeController.new,
);
