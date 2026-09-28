import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:forum_app/src/messages/chat_timeline.dart';

ChatMessagePayload _message(int id, String createdAt, {bool isSelf = false}) {
  return ChatMessagePayload(
    id: id,
    senderId: isSelf ? 1 : 2,
    content: '消息 $id',
    msgType: 1,
    isRead: 1,
    createdAt: createdAt,
    isSelf: isSelf,
  );
}

/// 设备本地墙钟对应的绝对时刻（带 Z 的 RFC3339），与设备时区无关。
String _local(
  int year,
  int month,
  int day,
  int hour,
  int minute, [
  int second = 0,
]) =>
    DateTime(year, month, day, hour, minute, second).toUtc().toIso8601String();

int _separatorCount(List<ChatTimelineItem> items) =>
    items.where((ChatTimelineItem item) => item.showDaySeparator).length;

void main() {
  group('buildChatTimeline', () {
    test('空列表返回空结果', () {
      expect(buildChatTimeline(const <ChatMessagePayload>[]), isEmpty);
    });

    test('列表首条消息插入日期分隔并显示时间', () {
      final items = buildChatTimeline(<ChatMessagePayload>[
        _message(1, _local(2026, 9, 27, 9, 30)),
      ]);

      expect(items, hasLength(1));
      expect(items.single.showDaySeparator, isTrue);
      expect(items.single.showTimestamp, isTrue);
      expect(items.single.day, DateTime(2026, 9, 27, 9, 30));
    });

    test('同一天间隔在阈值内不重复分隔也不显示时间', () {
      final items = buildChatTimeline(<ChatMessagePayload>[
        _message(1, _local(2026, 9, 27, 9, 30)),
        _message(2, _local(2026, 9, 27, 9, 32)),
        _message(3, _local(2026, 9, 27, 9, 34)),
      ]);

      expect(_separatorCount(items), 1);
      expect(items.map((ChatTimelineItem item) => item.showTimestamp), <bool>[
        true,
        false,
        false,
      ]);
    });

    test('同一天间隔超过 5 分钟重新显示时间但不插入日期分隔', () {
      final items = buildChatTimeline(<ChatMessagePayload>[
        _message(1, _local(2026, 9, 27, 9, 30)),
        _message(2, _local(2026, 9, 27, 9, 36)),
      ]);

      expect(_separatorCount(items), 1);
      expect(items.map((ChatTimelineItem item) => item.showTimestamp), <bool>[
        true,
        true,
      ]);
    });

    test('恰好 5 分钟视为同一分组，超过 1 秒才重新显示', () {
      final items = buildChatTimeline(<ChatMessagePayload>[
        _message(1, _local(2026, 9, 27, 9, 30)),
        _message(2, _local(2026, 9, 27, 9, 35)),
        _message(3, _local(2026, 9, 27, 9, 40, 1)),
      ]);

      expect(items.map((ChatTimelineItem item) => item.showTimestamp), <bool>[
        true,
        false,
        true,
      ]);
    });

    test('跨午夜插入新的日期分隔并显示时间', () {
      final items = buildChatTimeline(<ChatMessagePayload>[
        _message(1, _local(2026, 9, 27, 23, 58)),
        _message(2, _local(2026, 9, 28, 0, 2)),
      ]);

      expect(_separatorCount(items), 2);
      expect(items[1].day, DateTime(2026, 9, 28, 0, 2));
      expect(items.map((ChatTimelineItem item) => item.showTimestamp), <bool>[
        true,
        true,
      ]);
    });

    test('自己的消息与对方的消息使用同一分组规则', () {
      final items = buildChatTimeline(<ChatMessagePayload>[
        _message(1, _local(2026, 9, 27, 9, 30)),
        _message(2, _local(2026, 9, 27, 9, 40), isSelf: true),
        _message(3, _local(2026, 9, 27, 9, 42)),
        _message(4, _local(2026, 9, 27, 9, 44), isSelf: true),
      ]);

      expect(_separatorCount(items), 1);
      expect(items.map((ChatTimelineItem item) => item.showTimestamp), <bool>[
        true,
        true,
        false,
        false,
      ]);
    });

    test('前置更早的一页后重新分组：分隔不重复，边界消息不再显示时间', () {
      final List<ChatMessagePayload> page = <ChatMessagePayload>[
        _message(3, _local(2026, 9, 27, 10, 0)),
        _message(4, _local(2026, 9, 27, 10, 3)),
      ];
      final List<ChatTimelineItem> before = buildChatTimeline(page);
      expect(before.map((ChatTimelineItem item) => item.showTimestamp), <bool>[
        true,
        false,
      ]);
      expect(_separatorCount(before), 1);

      final List<ChatTimelineItem> after = buildChatTimeline(
        <ChatMessagePayload>[
          _message(1, _local(2026, 9, 27, 9, 58)),
          _message(2, _local(2026, 9, 27, 9, 59)),
          ...page,
        ],
      );

      expect(_separatorCount(after), 1);
      expect(after.map((ChatTimelineItem item) => item.showTimestamp), <bool>[
        true,
        false,
        false,
        false,
      ]);
    });

    test('前置跨日的一页后旧的首条消息保留时间但不重复日期分隔', () {
      final List<ChatTimelineItem> items =
          buildChatTimeline(<ChatMessagePayload>[
            _message(1, _local(2026, 9, 26, 22, 0)),
            _message(2, _local(2026, 9, 27, 10, 0)),
            _message(3, _local(2026, 9, 27, 10, 30)),
          ]);

      expect(_separatorCount(items), 2);
      expect(items.map((ChatTimelineItem item) => item.showTimestamp), <bool>[
        true,
        true,
        true,
      ]);
    });

    test('尾部追加新消息只影响最后一条的分组', () {
      final List<ChatMessagePayload> messages = <ChatMessagePayload>[
        _message(1, _local(2026, 9, 27, 10, 0)),
        _message(2, _local(2026, 9, 27, 10, 2)),
      ];
      messages.add(_message(3, _local(2026, 9, 27, 10, 3)));
      final List<ChatTimelineItem> close = buildChatTimeline(messages);
      expect(close.last.showTimestamp, isFalse);

      final List<ChatTimelineItem> late = buildChatTimeline(
        <ChatMessagePayload>[
          ...messages.sublist(0, 2),
          _message(4, _local(2026, 9, 28, 8, 0)),
        ],
      );
      expect(late.last.showDaySeparator, isTrue);
      expect(late.last.showTimestamp, isTrue);
    });

    test('无法解析的时间戳不猜日期：不插入分隔并保留原时间显示', () {
      final List<ChatTimelineItem> items = buildChatTimeline(
        <ChatMessagePayload>[
          _message(1, 'garbage'),
          _message(2, _local(2026, 9, 27, 10, 0)),
        ],
      );

      expect(items.first.day, isNull);
      expect(items.first.showDaySeparator, isFalse);
      expect(items.first.showTimestamp, isTrue);
      // 无法比较前后间隔时按新分组处理，避免把时间戳藏到无法证明相邻的消息上。
      expect(items[1].showDaySeparator, isTrue);
      expect(items[1].showTimestamp, isTrue);
    });

    test('无时区历史时间戳与等价的 UTC 表示分成同一天/同一组', () {
      final List<ChatTimelineItem> bare = buildChatTimeline(
        <ChatMessagePayload>[
          _message(1, '2026-09-27 09:30:00'),
          _message(2, '2026-09-27 09:40:00'),
        ],
      );
      final List<ChatTimelineItem> zoned = buildChatTimeline(
        <ChatMessagePayload>[
          _message(1, '2026-09-27T09:30:00Z'),
          _message(2, '2026-09-27T09:40:00Z'),
        ],
      );

      expect(
        bare.map((ChatTimelineItem item) => item.day),
        zoned.map((ChatTimelineItem item) => item.day),
      );
      expect(
        bare.map((ChatTimelineItem item) => item.showTimestamp),
        zoned.map((ChatTimelineItem item) => item.showTimestamp),
      );
      expect(
        bare.map((ChatTimelineItem item) => item.showDaySeparator),
        zoned.map((ChatTimelineItem item) => item.showDaySeparator),
      );
    });

    test('乱序的相邻时间戳按新分组处理', () {
      final List<ChatTimelineItem> items = buildChatTimeline(
        <ChatMessagePayload>[
          _message(1, _local(2026, 9, 27, 10, 0)),
          _message(2, _local(2026, 9, 27, 9, 0)),
        ],
      );

      expect(items[1].showTimestamp, isTrue);
    });
  });
}
