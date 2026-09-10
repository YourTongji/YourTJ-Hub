import 'package:flutter/material.dart';

import 'gf_colors.dart';

/// 站点主题运行时覆盖层（Route A 主题同步）。
///
/// 内置 `GfColors.light/dark`（tokens.json 镜像）仍是唯一事实源且保持编译期
/// 常量（`tokens_test.dart` 等值断言不动）；本类只把服务端下发的站点主题
/// token 逐键合并到内置色板上生成运行时 [GfColors]：
/// - 键名按 tokens.css 形态（`color-base-100` 等，去 `color-` 前缀后匹配
///   [GfColors.asMap]）；
/// - 逐键 hex 校验（`#RRGGBB`/`#AARRGGBB`），非法或缺省键回退内置值；
/// - `enabled=false` 或解析异常时整体回退内置主题。
@immutable
class GfRuntimeTheme {
  const GfRuntimeTheme._(this.light, this.dark);

  /// 服务端 light 模式色板（回退合并后的完整 22 键色板）。
  final GfColors light;

  /// 服务端 dark 模式色板。
  final GfColors dark;

  /// 从站点主题下发载荷构建覆盖层；[themesByMode] 为 `mode → tokens`。
  /// 两个模式都缺 tokens 时返回 null（调用方整体回退内置主题）。
  static GfRuntimeTheme? fromTokens(
    Map<String, Map<String, String>> themesByMode,
  ) {
    final Map<String, String>? lightTokens = _nonEmpty(themesByMode['light']);
    final Map<String, String>? darkTokens = _nonEmpty(themesByMode['dark']);
    if (lightTokens == null && darkTokens == null) return null;
    try {
      return GfRuntimeTheme._(
        _merge(GfColors.light, lightTokens),
        _merge(GfColors.dark, darkTokens),
      );
    } catch (_) {
      // 任一模式解析异常：整体回退内置主题，不部分生效。
      return null;
    }
  }

  static Map<String, String>? _nonEmpty(Map<String, String>? tokens) =>
      tokens == null || tokens.isEmpty ? null : tokens;

  /// 把服务端 tokens 逐键覆盖到 [base] 上；非法键忽略（回退内置）。
  static GfColors _merge(GfColors base, Map<String, String>? tokens) {
    if (tokens == null || tokens.isEmpty) return base;
    final Map<String, Color> merged = Map<String, Color>.of(base.asMap);
    tokens.forEach((String rawKey, String value) {
      final String key = rawKey.startsWith('color-')
          ? rawKey.substring('color-'.length)
          : rawKey;
      if (!merged.containsKey(key)) return; // 非色彩键（radius/size 等）忽略
      final Color? parsed = _parseHex(value);
      if (parsed != null) merged[key] = parsed;
    });
    return GfColors(
      base100: merged['base-100']!,
      base200: merged['base-200']!,
      base300: merged['base-300']!,
      baseContent: merged['base-content']!,
      iconMuted: merged['icon-muted']!,
      line: merged['line']!,
      primary: merged['primary']!,
      primaryContent: merged['primary-content']!,
      secondary: merged['secondary']!,
      secondaryContent: merged['secondary-content']!,
      accent: merged['accent']!,
      accentContent: merged['accent-content']!,
      neutral: merged['neutral']!,
      neutralContent: merged['neutral-content']!,
      info: merged['info']!,
      infoContent: merged['info-content']!,
      success: merged['success']!,
      successContent: merged['success-content']!,
      warning: merged['warning']!,
      warningContent: merged['warning-content']!,
      error: merged['error']!,
      errorContent: merged['error-content']!,
    );
  }

  /// 解析 `#RRGGBB` / `#AARRGGBB`（含无 `#` 前缀容错）；非法返回 null。
  static Color? _parseHex(String value) {
    var hex = value.trim();
    if (hex.isEmpty) return null;
    if (hex.startsWith('#')) hex = hex.substring(1);
    if (hex.length == 6) hex = 'FF$hex';
    if (hex.length != 8) return null;
    final int? argb = int.tryParse(hex, radix: 16);
    if (argb == null) return null;
    return Color(argb);
  }
}
