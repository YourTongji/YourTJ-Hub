import 'package:flutter_test/flutter_test.dart';

import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/format.dart';

void main() {
  final AppLocalizationsZh zh = AppLocalizationsZh();

  group('timeAgo', () {
    test('60 秒内返回 "刚刚"', () {
      final now = DateTime(2026, 8, 7, 12, 0, 0);
      final iso = DateTime(2026, 8, 7, 11, 59, 30).toIso8601String();
      expect(timeAgo(iso, now: now, l10n: zh), '刚刚');
    });

    test('60 分钟内返回分钟数', () {
      final now = DateTime(2026, 8, 7, 12, 0, 0);
      final iso = DateTime(2026, 8, 7, 11, 30, 0).toIso8601String();
      expect(timeAgo(iso, now: now, l10n: zh), '30 分钟前');
    });

    test('24 小时内返回小时数', () {
      final now = DateTime(2026, 8, 7, 12, 0, 0);
      final iso = DateTime(2026, 8, 7, 6, 0, 0).toIso8601String();
      expect(timeAgo(iso, now: now, l10n: zh), '6 小时前');
    });

    test('7 天内返回天数', () {
      final now = DateTime(2026, 8, 7, 12, 0, 0);
      final iso = DateTime(2026, 8, 4, 12, 0, 0).toIso8601String();
      expect(timeAgo(iso, now: now, l10n: zh), '3 天前');
    });

    test('>= 7 天回退到绝对日期 YYYY-MM-DD(web format.ts 语义)', () {
      final now = DateTime(2026, 8, 7, 12, 0, 0);
      // 恰好 7 天前 → 绝对日期。
      final iso7d = DateTime(2026, 7, 31, 12, 0, 0).toIso8601String();
      expect(timeAgo(iso7d, now: now, l10n: zh), '2026-07-31');
      // 跨年 → 绝对日期。
      final isoCrossYear = DateTime(2025, 12, 25, 12, 0, 0).toIso8601String();
      expect(timeAgo(isoCrossYear, now: now, l10n: zh), '2025-12-25');
    });

    test('无效时间戳返回原字符串', () {
      expect(timeAgo('not-a-date', l10n: zh), 'not-a-date');
    });
  });

  group('formatDate', () {
    test('返回 YYYY-MM-DD', () {
      expect(formatDate('2026-08-07T10:00:00+08:00'), '2026-08-07');
    });

    test('空串返回空串', () {
      expect(formatDate(''), '');
    });

    test('无效输入返回原值', () {
      expect(formatDate('garbage'), 'garbage');
    });
  });

  group('formatDateTime', () {
    test('返回 YYYY-MM-DD HH:mm', () {
      expect(formatDateTime('2026-08-07T10:05:00+08:00'), '2026-08-07 10:05');
    });

    test('无效输入返回原值', () {
      expect(formatDateTime('garbage'), 'garbage');
    });
  });

  group('formatChatTime', () {
    test('今天 → 仅 HH:mm', () {
      final now = DateTime.now();
      final iso = now.subtract(const Duration(minutes: 5)).toIso8601String();
      final result = formatChatTime(iso, l10n: zh);
      expect(
        RegExp(r'^\d{2}:\d{2}$').hasMatch(result),
        isTrue,
        reason: '今天应只显示 HH:mm,实际: $result',
      );
    });

    test('同年 → M月D日 HH:mm(zh dateMonthDayTime)', () {
      final now = DateTime.now();
      final sameYear = DateTime(now.year, 1, 15, 9, 30);
      final iso = sameYear.toIso8601String();
      final result = formatChatTime(iso, l10n: zh);
      expect(result, '1月15日 09:30');
    });

    test('跨年 → YYYY年M月D日 HH:mm(zh dateYearMonthDayTime)', () {
      final now = DateTime.now();
      final crossYear = DateTime(now.year - 1, 12, 31, 23, 59);
      final iso = crossYear.toIso8601String();
      final result = formatChatTime(iso, l10n: zh);
      expect(result, '${now.year - 1}年12月31日 23:59');
    });

    test('无效输入返回原值', () {
      expect(formatChatTime('garbage', l10n: zh), 'garbage');
    });
  });

  group('formatNumber', () {
    test('<1k 原样', () {
      expect(formatNumber(999), '999');
    });

    test('>=1k 转 x.xk', () {
      expect(formatNumber(1234), '1.2k');
      expect(formatNumber(1500), '1.5k');
    });

    test('>=1m 转 x.xm', () {
      expect(formatNumber(2300000), '2.3m');
    });
  });
}
