import 'package:flutter/material.dart';

import '../l10n/app_localizations.dart';
import '../l10n/app_localizations_zh.dart';

/// 将后端返回的 hex 颜色(如 `#2563eb`)解析为 [Color]。
/// 解析失败时回退 [fallback]。
Color colorFromHex(String hex, {Color fallback = const Color(0xFF62748E)}) {
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
///
/// 与 web 一致:刚/分/时/天(<7 天);≥7 天回退到绝对日期
/// `formatDate`(web format.ts:29-42)。
String timeAgo(String isoTime, {DateTime? now, AppLocalizations? l10n}) {
  final AppLocalizations loc = l10n ?? _fallbackL10n;
  final DateTime? parsed = DateTime.tryParse(isoTime.replaceFirst(' ', 'T'));
  if (parsed == null) return isoTime;
  // 只依赖绝对时刻差,不输出本地时区字段。
  final DateTime current = now ?? DateTime.now();
  final Duration diff = current.difference(parsed);
  if (diff.inSeconds < 60) return loc.timeAgoJustNow;
  if (diff.inMinutes < 60) return loc.timeAgoMinutes(diff.inMinutes);
  if (diff.inHours < 24) return loc.timeAgoHours(diff.inHours);
  if (diff.inDays < 7) return loc.timeAgoDays(diff.inDays);
  return formatDate(isoTime);
}

/// 从 ISO 字符串提取 `YYYY-MM-DD` 字段(不依赖进程时区)。
///
/// 业务语义:后端返回的时间字符串即展示时间(web 端浏览器本地时区近似
/// 等价于后端时区),因此直接取字符串字段而非 `toLocal()` 转换,保证
/// 任意运行环境(含 CI UTC)下输出一致。
String? _dateField(String value) {
  final String normalized = value.replaceFirst(' ', 'T');
  final int sep = normalized.indexOf('T');
  final String datePart = sep == -1 ? normalized : normalized.substring(0, sep);
  if (!RegExp(r'^\d{4}-\d{2}-\d{2}$').hasMatch(datePart)) return null;
  return datePart;
}

/// 从 ISO 字符串提取 `HH:mm` 字段(不依赖进程时区)。
String? _timeField(String value) {
  final String normalized = value.replaceFirst(' ', 'T');
  final int sep = normalized.indexOf('T');
  if (sep == -1) return null;
  final String timePart = normalized.substring(sep + 1);
  if (timePart.length < 5) return null;
  final String hhmm = timePart.substring(0, 5);
  if (!RegExp(r'^\d{2}:\d{2}$').hasMatch(hhmm)) return null;
  return hhmm;
}

/// 绝对日期 `YYYY-MM-DD`(对齐 web `format.ts` formatDate)。
/// 无效输入返回原字符串(web 对空返回空串)。
String formatDate(String value) {
  if (value.isEmpty) return '';
  return _dateField(value) ?? value;
}

/// 日期时间 `YYYY-MM-DD HH:mm`(对齐 web `format.ts` formatDateTime)。
String formatDateTime(String value) {
  if (value.isEmpty) return '';
  final String? date = _dateField(value);
  final String? time = _timeField(value);
  if (date == null || time == null) return value;
  return '$date $time';
}

/// 聊天时间(对齐 web `format.ts` formatChatTime):
/// 今天 → `HH:mm`;同年 → `M月D日 HH:mm`(zh)/`M/D HH:mm`(en);
/// 跨年 → `YYYY年M月D日 HH:mm`(zh)/`YYYY/M/D HH:mm`(en)。
///
/// 日期字段直接取自输入字符串(不依赖进程时区);"今天/同年"按输入
/// 字符串的日期与设备本地日期比较。
String formatChatTime(String value, {AppLocalizations? l10n, DateTime? now}) {
  final AppLocalizations loc = l10n ?? _fallbackL10n;
  final String? date = _dateField(value);
  final String? time = _timeField(value);
  if (date == null || time == null) return value;
  final int year = int.parse(date.substring(0, 4));
  final int month = int.parse(date.substring(5, 7));
  final int day = int.parse(date.substring(8, 10));
  final DateTime current = now ?? DateTime.now();
  final bool sameDay =
      year == current.year && month == current.month && day == current.day;
  if (sameDay) return time;
  if (year == current.year) return loc.dateMonthDayTime(month, day, time);
  return loc.dateYearMonthDayTime(year, month, day, time);
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
