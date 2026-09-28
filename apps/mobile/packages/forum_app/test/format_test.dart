import 'package:flutter_test/flutter_test.dart';

import 'package:forum_app/l10n/app_localizations_zh.dart';
import 'package:forum_app/src/format.dart';

void main() {
  final AppLocalizationsZh zh = AppLocalizationsZh();

  group('timeAgo', () {
    test('60 秒内返回 "刚刚"', () {
      final now = DateTime(2026, 8, 7, 12, 0, 0);
      final iso = DateTime(2026, 8, 7, 11, 59, 30).toUtc().toIso8601String();
      expect(timeAgo(iso, now: now, l10n: zh), '刚刚');
    });

    test('60 分钟内返回分钟数', () {
      final now = DateTime(2026, 8, 7, 12, 0, 0);
      final iso = DateTime(2026, 8, 7, 11, 30, 0).toUtc().toIso8601String();
      expect(timeAgo(iso, now: now, l10n: zh), '30 分钟前');
    });

    test('24 小时内返回小时数', () {
      final now = DateTime(2026, 8, 7, 12, 0, 0);
      final iso = DateTime(2026, 8, 7, 6, 0, 0).toUtc().toIso8601String();
      expect(timeAgo(iso, now: now, l10n: zh), '6 小时前');
    });

    test('7 天内返回天数', () {
      final now = DateTime(2026, 8, 7, 12, 0, 0);
      final iso = DateTime(2026, 8, 4, 12, 0, 0).toUtc().toIso8601String();
      expect(timeAgo(iso, now: now, l10n: zh), '3 天前');
    });

    test('>= 7 天回退到绝对日期 YYYY-MM-DD(web format.ts 语义)', () {
      final now = DateTime(2026, 8, 7, 12, 0, 0);
      // 恰好 7 天前 → 绝对日期。
      final iso7d = DateTime(2026, 7, 31, 12, 0, 0).toUtc().toIso8601String();
      expect(timeAgo(iso7d, now: now, l10n: zh), '2026-07-31');
      // 跨年 → 绝对日期。
      final isoCrossYear = DateTime(
        2025,
        12,
        25,
        12,
        0,
        0,
      ).toUtc().toIso8601String();
      expect(timeAgo(isoCrossYear, now: now, l10n: zh), '2025-12-25');
    });

    test('无效时间戳返回原字符串', () {
      expect(timeAgo('not-a-date', l10n: zh), 'not-a-date');
    });
  });

  group('formatDate', () {
    test('返回 YYYY-MM-DD', () {
      expect(
        formatDate(DateTime(2026, 8, 7, 10).toUtc().toIso8601String()),
        '2026-08-07',
      );
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
      expect(
        formatDateTime(DateTime(2026, 8, 7, 10, 5).toUtc().toIso8601String()),
        '2026-08-07 10:05',
      );
    });

    test('无效输入返回原值', () {
      expect(formatDateTime('garbage'), 'garbage');
    });
  });

  group('formatChatTime', () {
    test('今天 → 仅 HH:mm', () {
      final now = DateTime(2026, 8, 9, 23, 58);
      final iso = DateTime(2026, 8, 9, 23, 53).toUtc().toIso8601String();
      expect(formatChatTime(iso, l10n: zh, now: now), '23:53');
    });

    test('同年 → M月D日 HH:mm(zh dateMonthDayTime)', () {
      final now = DateTime(2026, 8, 9, 23, 58);
      final sameYear = DateTime(2026, 1, 15, 9, 30);
      final iso = sameYear.toUtc().toIso8601String();
      expect(formatChatTime(iso, l10n: zh, now: now), '1月15日 09:30');
    });

    test('跨年 → YYYY年M月D日 HH:mm(zh dateYearMonthDayTime)', () {
      final now = DateTime(2026, 8, 9, 23, 58);
      final crossYear = DateTime(2025, 12, 31, 23, 59);
      final iso = crossYear.toUtc().toIso8601String();
      expect(formatChatTime(iso, l10n: zh, now: now), '2025年12月31日 23:59');
    });

    test('无效输入返回原值', () {
      expect(formatChatTime('garbage', l10n: zh), 'garbage');
    });
  });

  group('timestamp timezone conversion', () {
    test(
      'UTC and nonzero offsets render the same local instant across midnight',
      () {
        final local = DateTime(2026, 1, 1, 0, 15);
        final utc = local.toUtc();
        final offset =
            '${utc.subtract(const Duration(hours: 7)).toIso8601String().replaceFirst('Z', '')}-07:00';
        for (final value in [utc.toIso8601String(), offset]) {
          expect(formatChatTime(value, now: local, l10n: zh), '00:15');
          expect(formatDate(value), '2026-01-01');
          expect(formatDateTime(value), '2026-01-01 00:15');
        }
      },
    );
    test('date-only school calendar values keep their date', () {
      expect(formatDate('2026-01-01'), '2026-01-01');
      expect(formatDateTime('2026-01-01'), '2026-01-01');
    });
  });

  group('formatChatClock', () {
    test('仅返回本地 HH:mm', () {
      final String iso = DateTime(2026, 9, 27, 23, 5).toUtc().toIso8601String();
      expect(formatChatClock(iso), '23:05');
    });

    test('带偏移的 RFC3339 还原设备本地墙钟', () {
      final DateTime local = DateTime(2026, 1, 1, 0, 15);
      final String utc = local.toUtc().toIso8601String();
      final String offset =
          '${local.toUtc().subtract(const Duration(hours: 7)).toIso8601String().replaceFirst('Z', '')}-07:00';
      expect(formatChatClock(utc), '00:15');
      expect(formatChatClock(offset), '00:15');
    });

    test('无效输入返回原值', () {
      expect(formatChatClock('garbage'), 'garbage');
    });
  });

  group('formatChatDayLabel', () {
    final DateTime now = DateTime(2026, 9, 28, 10, 0);

    test('今天返回 dateToday', () {
      expect(
        formatChatDayLabel(DateTime(2026, 9, 28, 0, 5), l10n: zh, now: now),
        '今天',
      );
    });

    test('昨天返回 dateYesterday', () {
      expect(
        formatChatDayLabel(DateTime(2026, 9, 27, 23, 59), l10n: zh, now: now),
        '昨天',
      );
    });

    test('同年更早返回 dateMonthDay', () {
      expect(
        formatChatDayLabel(DateTime(2026, 1, 15, 9, 30), l10n: zh, now: now),
        '1月15日',
      );
    });

    test('跨年返回 dateYearMonthDay', () {
      expect(
        formatChatDayLabel(DateTime(2025, 12, 31, 23, 59), l10n: zh, now: now),
        '2025年12月31日',
      );
    });

    test('昨天按本地日历日计算，跨月/跨年边界正确', () {
      expect(
        formatChatDayLabel(
          DateTime(2025, 12, 31, 12),
          l10n: zh,
          now: DateTime(2026, 1, 1, 8),
        ),
        '昨天',
      );
      expect(
        formatChatDayLabel(
          DateTime(2026, 2, 28, 23),
          l10n: zh,
          now: DateTime(2026, 3, 1, 1),
        ),
        '昨天',
      );
    });
  });

  group('无时区历史时间戳按 UTC 解释（对齐 Web parseDate,issue #221）', () {
    test('无时区日期时间解析为同一 UTC 时刻', () {
      expect(
        parseChatTimestamp('2026-09-27 09:30:00')?.millisecondsSinceEpoch,
        DateTime.utc(2026, 9, 27, 9, 30).millisecondsSinceEpoch,
      );
      expect(
        parseChatTimestamp('2026-09-27T09:30:00')?.millisecondsSinceEpoch,
        DateTime.utc(2026, 9, 27, 9, 30).millisecondsSinceEpoch,
      );
    });

    test('无时区值与带偏移表示的同一时刻显示一致', () {
      final DateTime utc = DateTime.utc(2026, 9, 27, 9, 30);
      final String offset =
          '${utc.subtract(const Duration(hours: 7)).toIso8601String().replaceFirst('Z', '')}-07:00';
      expect(
        parseChatTimestamp('2026-09-27 09:30:00'),
        parseChatTimestamp(offset),
      );
      expect(
        formatChatClock('2026-09-27 09:30:00'),
        formatChatClock(utc.toIso8601String()),
      );
    });

    test('纯日期日历值保持本地日历日,不随时区偏移', () {
      expect(parseChatTimestamp('2026-09-27'), DateTime(2026, 9, 27));
      expect(formatDate('2026-09-27'), '2026-09-27');
      expect(formatDateTime('2026-09-27'), '2026-09-27');
      expect(formatChatClock('2026-09-27'), '2026-09-27');
      expect(
        formatChatDayLabel(
          DateTime(2026, 9, 27),
          l10n: zh,
          now: DateTime(2026, 9, 28),
        ),
        '昨天',
      );
    });

    test('timeAgo 对无时区历史值使用同一 UTC 语义', () {
      final DateTime now = DateTime.utc(2026, 9, 27, 12, 0).toLocal();
      expect(timeAgo('2026-09-27 09:30:00', now: now, l10n: zh), '2 小时前');
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
