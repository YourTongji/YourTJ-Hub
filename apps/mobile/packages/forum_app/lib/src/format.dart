import 'package:flutter/material.dart';

import '../l10n/app_localizations.dart';
import '../l10n/app_localizations_zh.dart';

/// 将后端返回的 hex 颜色(如 `#2563eb`)解析为 [Color]。
/// 解析失败时回退 [fallback]。
Color colorFromHex(String hex, {Color fallback = Colors.blueGrey}) {
  final String cleaned = hex.replaceFirst('#', '').trim();
  if (cleaned.isEmpty) return fallback;
  if (cleaned.length == 3) {
    final int? value = int.tryParse(cleaned, radix: 16);
    if (value == null) return fallback;
    final int r = (value >> 8) & 0xF;
    final int g = (value >> 4) & 0xF;
    final int b = value & 0xF;
    return Color(
      0xFF000000 | (r << 20) | (r << 16) | (g << 12) | (g << 8) | (b << 4) | b,
    );
  }
  if (cleaned.length == 6) {
    return Color(int.tryParse('FF$cleaned', radix: 16) ?? fallback.toARGB32());
  }
  return fallback;
}

/// 相对时间(对齐 web `format.ts` timeAgo 语义,文案经 i18n)。
String timeAgo(String isoTime, {DateTime? now, AppLocalizations? l10n}) {
  final AppLocalizations loc = l10n ?? _fallbackL10n;
  final DateTime time = DateTime.tryParse(isoTime)?.toLocal() ?? DateTime.now();
  final DateTime current = now ?? DateTime.now();
  final Duration diff = current.difference(time);
  if (diff.inSeconds < 60) return loc.timeAgoJustNow;
  if (diff.inMinutes < 60) return loc.timeAgoMinutes(diff.inMinutes);
  if (diff.inHours < 24) return loc.timeAgoHours(diff.inHours);
  if (diff.inDays < 7) return loc.timeAgoDays(diff.inDays);
  if (diff.inDays < 30) return loc.timeAgoWeeks(diff.inDays ~/ 7);
  if (diff.inDays < 365) return loc.timeAgoMonths(diff.inDays ~/ 30);
  return loc.timeAgoYears(diff.inDays ~/ 365);
}

// 无 context 场景(如纯工具调用)回退中文;页面内请传入 l10n。
final AppLocalizations _fallbackL10n = AppLocalizationsZh();

/// 数字缩写(对齐 web `format.ts` formatNumber:1k/1m)。
String formatNumber(int value) {
  if (value < 1000) return '$value';
  if (value < 1000000) {
    final double v = value / 1000;
    return '${v.toStringAsFixed(v.truncateToDouble() == v ? 0 : 1)}k';
  }
  final double v = value / 1000000;
  return '${v.toStringAsFixed(v.truncateToDouble() == v ? 0 : 1)}m';
}
