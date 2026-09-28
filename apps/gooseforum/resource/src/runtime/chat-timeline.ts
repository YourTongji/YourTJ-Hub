import { parseDate } from './format'

/** 相邻消息间隔超过该阈值时，在气泡下重新显示时间戳（与移动端一致）。 */
export const CHAT_TIMESTAMP_GROUP_GAP_MS = 5 * 60 * 1000

/** 时间分块只需要消息的身份与创建时间，其余字段原样透传。 */
export interface ChatTimelineMessage {
  id: number
  createdAt: string
}

export interface ChatTimelineEntry<T extends ChatTimelineMessage = ChatTimelineMessage> {
  message: T
  /** 消息所属浏览器本地日历日；时间戳无法解析时为 null。 */
  day: Date | null
  /** 是否在该消息前插入日期分隔标签（每条消息是其所在本地日历日的第一条）。 */
  showDaySeparator: boolean
  /** 是否在该消息气泡下显示时刻（所在日/分组的第一条，或与上一条间隔超过阈值）。 */
  showTimestamp: boolean
}

/**
 * 将按时间升序的消息列表按浏览器本地日历日与 `gapMs` 分块。
 *
 * 每条消息只依赖自身与相邻的前一条消息，因此不需要增量状态：历史分页前置更早
 * 一页、或尾部追加新消息后，重新调用一次即可得到正确结果——分页边界处不再
 * “开启分组”的消息会自动隐藏时间戳，且不会出现重复的日期分隔。
 *
 * 规则与自己/对方消息无关；时间戳无法解析时按新分组处理（不猜日期、不插入分隔、
 * 保留原值显示），避免把时间藏在无法证明相邻的消息上。
 */
export function buildChatTimeline<T extends ChatTimelineMessage>(
  messages: readonly T[],
  gapMs: number = CHAT_TIMESTAMP_GROUP_GAP_MS,
): ChatTimelineEntry<T>[] {
  const items: ChatTimelineEntry<T>[] = []
  for (let index = 0; index < messages.length; index += 1) {
    const message = messages[index]
    const current = parseDate(message.createdAt)
    const currentTime = current.getTime()
    if (Number.isNaN(currentTime)) {
      items.push({
        message,
        day: null,
        showDaySeparator: false,
        showTimestamp: true,
      })
      continue
    }
    const parsedPrevious = index === 0 ? null : parseDate(messages[index - 1].createdAt)
    const previous =
      parsedPrevious !== null && !Number.isNaN(parsedPrevious.getTime()) ? parsedPrevious : null
    const startsDay = previous === null || !isSameLocalDay(previous, current)
    const delta = previous === null ? 0 : currentTime - previous.getTime()
    // 时间倒序（乱序数据）同样视为新分组。
    const startsGroup = startsDay || delta < 0 || delta > gapMs
    items.push({
      message,
      day: current,
      showDaySeparator: startsDay,
      showTimestamp: startsGroup,
    })
  }
  return items
}

function isSameLocalDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  )
}
