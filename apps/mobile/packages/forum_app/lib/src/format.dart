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
  // 与聊天时间同一解析语义(无时区历史值按 UTC,见 [parseChatTimestamp])。
  final DateTime? parsed = parseChatTimestamp(isoTime);
  if (parsed == null) return isoTime;
  // 只依赖绝对时刻差,不输出本地时区字段。
  final DateTime current = (now ?? DateTime.now()).toLocal();
  final Duration diff = current.difference(parsed);
  if (diff.inSeconds < 60) return loc.timeAgoJustNow;
  if (diff.inMinutes < 60) return loc.timeAgoMinutes(diff.inMinutes);
  if (diff.inHours < 24) return loc.timeAgoHours(diff.inHours);
  if (diff.inDays < 7) return loc.timeAgoDays(diff.inDays);
  return formatDate(isoTime);
}

/// 将服务端时间字符串解析为设备本地时刻;无效输入返回 null。
///
/// - 带偏移/Z 的 RFC3339:按绝对时刻解析。
/// - 无时区标记的**日期时间**(旧服务端 `time.DateTime` 输出):按 UTC 墙钟解析
///   (issue #221;服务器运行在 UTC,与 Web `parseDate` 一致),否则设备时区会
///   固定偏移,日期分隔与相对时间都会错位。
/// - 纯**日期**的日历值(校历、日期型字段):按本地日历日解析,不随时区偏移,
///   任何时区下都保持同一个日历日。
DateTime? parseChatTimestamp(String value) {
  final String normalized = value.replaceFirst(' ', 'T');
  if (!normalized.contains('T')) return DateTime.tryParse(normalized);
  if (_hasZoneDesignator(normalized)) {
    return DateTime.tryParse(normalized)?.toLocal();
  }
  return DateTime.tryParse('${normalized}Z')?.toLocal();
}

bool _hasZoneDesignator(String value) =>
    RegExp(r'[zZ]$|[+-]\d{2}:?\d{2}$').hasMatch(value);

String? _dateField(String value) {
  final parsed = parseChatTimestamp(value);
  if (parsed == null) return null;
  final year = parsed.year.toString().padLeft(4, '0');
  final month = parsed.month.toString().padLeft(2, '0');
  final day = parsed.day.toString().padLeft(2, '0');
  return '$year-$month-$day';
}

String? _timeField(String value) {
  if (!value.replaceFirst(' ', 'T').contains('T')) return null;
  final parsed = parseChatTimestamp(value);
  if (parsed == null) return null;
  return '${parsed.hour.toString().padLeft(2, '0')}:${parsed.minute.toString().padLeft(2, '0')}';
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
/// 日期分组、消息时间和“今天/同年”全部采用设备本地时区。
String formatChatTime(String value, {AppLocalizations? l10n, DateTime? now}) {
  final AppLocalizations loc = l10n ?? _fallbackL10n;
  final String? date = _dateField(value);
  final String? time = _timeField(value);
  if (date == null || time == null) return value;
  final int year = int.parse(date.substring(0, 4));
  final int month = int.parse(date.substring(5, 7));
  final int day = int.parse(date.substring(8, 10));
  final DateTime current = (now ?? DateTime.now()).toLocal();
  final bool sameDay =
      year == current.year && month == current.month && day == current.day;
  if (sameDay) return time;
  if (year == current.year) return loc.dateMonthDayTime(month, day, time);
  return loc.dateYearMonthDayTime(year, month, day, time);
}

/// 聊天气泡内的时刻(仅设备本地 `HH:mm`)。
///
/// 私信按日期分隔与时间分组渲染(见 `messages/chat_timeline.dart`),
/// 日期已由分隔标签表达,气泡内只保留时刻;纯日期值与无法解析的输入返回原值。
String formatChatClock(String value) => _timeField(value) ?? value;

/// 私信日期分隔标签:今天/昨天/同年 `M月D日`/跨年 `YYYY年M月D日`。
/// [day] 为设备本地时刻,与 [formatChatTime] 使用同一时区基准。
String formatChatDayLabel(
  DateTime day, {
  AppLocalizations? l10n,
  DateTime? now,
}) {
  final AppLocalizations loc = l10n ?? _fallbackL10n;
  final DateTime current = (now ?? DateTime.now()).toLocal();
  if (_sameLocalDay(day, current)) return loc.dateToday;
  final DateTime yesterday = DateTime(
    current.year,
    current.month,
    current.day - 1,
  );
  if (_sameLocalDay(day, yesterday)) return loc.dateYesterday;
  if (day.year == current.year) return loc.dateMonthDay(day.month, day.day);
  return loc.dateYearMonthDay(day.year, day.month, day.day);
}

bool _sameLocalDay(DateTime a, DateTime b) =>
    a.year == b.year && a.month == b.month && a.day == b.day;

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
