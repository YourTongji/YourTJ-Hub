import 'package:core/core.dart';

import '../format.dart';

/// 相邻消息间隔超过该阈值时,在气泡下重新显示时间戳(常见 IM 分组粒度)。
const Duration chatTimestampGroupGap = Duration(minutes: 5);

/// 私信消息的时间分块结果:消息本身 + 它是否开启新的日期分隔/时间分组。
class ChatTimelineItem {
  const ChatTimelineItem({
    required this.message,
    required this.day,
    required this.showDaySeparator,
    required this.showTimestamp,
  });

  final ChatMessagePayload message;

  /// 消息所属设备本地日历日;时间戳无法解析时为 null。
  final DateTime? day;

  /// 是否在该消息前插入日期分隔标签(每条消息是其所在本地日历日的第一条)。
  final bool showDaySeparator;

  /// 是否在该消息气泡下显示时刻(所在日/分组的第一条,或与上一条间隔超过阈值)。
  final bool showTimestamp;
}

/// 将按时间升序的消息列表按设备本地日历日与 [gap] 分块。
///
/// 每条消息都只依赖自身与相邻的前一条消息,因此不需要增量状态:历史分页
/// 前置更早一页、或尾部追加新消息后,重新调用一次即可得到正确结果——
/// 分页边界处不再"开启分组"的消息会自动隐藏时间戳,且不会出现重复的日期分隔。
///
/// 时间戳无法解析时按新分组处理(不猜日期、不插入分隔、保留原值显示),
/// 避免把时间藏在无法证明相邻的消息上。
List<ChatTimelineItem> buildChatTimeline(
  List<ChatMessagePayload> messages, {
  Duration gap = chatTimestampGroupGap,
}) {
  final List<ChatTimelineItem> items = <ChatTimelineItem>[];
  for (int index = 0; index < messages.length; index++) {
    final ChatMessagePayload message = messages[index];
    final DateTime? current = parseChatTimestamp(message.createdAt);
    if (current == null) {
      items.add(
        ChatTimelineItem(
          message: message,
          day: null,
          showDaySeparator: false,
          showTimestamp: true,
        ),
      );
      continue;
    }
    final DateTime? previous = index == 0
        ? null
        : parseChatTimestamp(messages[index - 1].createdAt);
    final bool first = index == 0;
    final bool startsDay =
        first || previous == null || !_sameLocalDay(previous, current);
    final bool startsGroup = startsDay || _exceedsGap(previous, current, gap);
    items.add(
      ChatTimelineItem(
        message: message,
        day: current,
        showDaySeparator: startsDay,
        showTimestamp: startsGroup,
      ),
    );
  }
  return items;
}

bool _sameLocalDay(DateTime a, DateTime b) =>
    a.year == b.year && a.month == b.month && a.day == b.day;

bool _exceedsGap(DateTime previous, DateTime current, Duration gap) {
  final Duration delta = current.difference(previous);
  // 时间倒序(乱序数据)同样视为新分组。
  return delta.isNegative || delta > gap;
}
