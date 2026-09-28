import { afterEach, describe, expect, test } from 'vitest'
import {
  buildChatTimeline,
  CHAT_TIMESTAMP_GROUP_GAP_MS,
} from '../src/runtime/chat-timeline'
import {
  formatChatClock,
  formatChatDayLabel,
  formatChatTime,
  formatDate,
  formatDateTime,
  parseDate,
} from '../src/runtime/format'
import { i18n } from '../src/runtime/i18n'

interface TestMessage {
  id: number
  createdAt: string
  /** 分组规则与发送方无关：自己/对方的消息走同一条路径。 */
  isSelf: boolean
}

function message(id: number, createdAt: string, isSelf = false): TestMessage {
  return { id, createdAt, isSelf }
}

/** 设备本地墙钟 → 带时区的 RFC3339，与后端输出形态一致。 */
function local(
  year: number,
  month: number,
  day: number,
  hour: number,
  minute: number,
  second = 0,
): string {
  return new Date(year, month - 1, day, hour, minute, second).toISOString()
}

const separators = (items: ReturnType<typeof buildChatTimeline>) =>
  items.filter((item) => item.showDaySeparator).length

describe('buildChatTimeline', () => {
  test('空列表返回空结果', () => {
    expect(buildChatTimeline([])).toEqual([])
  })

  test('列表首条消息插入日期分隔并显示时间', () => {
    const items = buildChatTimeline([message(1, local(2026, 9, 27, 9, 30))])

    expect(items).toHaveLength(1)
    expect(items[0]!.showDaySeparator).toBe(true)
    expect(items[0]!.showTimestamp).toBe(true)
    expect(items[0]!.day?.getHours()).toBe(9)
  })

  test('同一天间隔在阈值内不重复分隔也不显示时间', () => {
    const items = buildChatTimeline([
      message(1, local(2026, 9, 27, 9, 30)),
      message(2, local(2026, 9, 27, 9, 32)),
      message(3, local(2026, 9, 27, 9, 34)),
    ])

    expect(separators(items)).toBe(1)
    expect(items.map((item) => item.showTimestamp)).toEqual([true, false, false])
  })

  test('同一天间隔超过 5 分钟重新显示时间但不插入日期分隔', () => {
    const items = buildChatTimeline([
      message(1, local(2026, 9, 27, 9, 30)),
      message(2, local(2026, 9, 27, 9, 36)),
    ])

    expect(separators(items)).toBe(1)
    expect(items.map((item) => item.showTimestamp)).toEqual([true, true])
  })

  test('恰好 5 分钟视为同一分组，超过 1 秒才重新显示', () => {
    const items = buildChatTimeline([
      message(1, local(2026, 9, 27, 9, 30)),
      message(2, local(2026, 9, 27, 9, 35)),
      message(3, local(2026, 9, 27, 9, 40, 1)),
    ])

    expect(items.map((item) => item.showTimestamp)).toEqual([true, false, true])
    expect(CHAT_TIMESTAMP_GROUP_GAP_MS).toBe(5 * 60 * 1000)
  })

  test('跨午夜插入新的日期分隔并显示时间', () => {
    const items = buildChatTimeline([
      message(1, local(2026, 9, 27, 23, 58)),
      message(2, local(2026, 9, 28, 0, 2)),
    ])

    expect(separators(items)).toBe(2)
    expect(items[1]!.day?.getDate()).toBe(28)
    expect(items.map((item) => item.showTimestamp)).toEqual([true, true])
  })

  test('自己的消息与对方的消息使用同一分组规则', () => {
    const items = buildChatTimeline([
      message(1, local(2026, 9, 27, 9, 30)),
      message(2, local(2026, 9, 27, 9, 40), true),
      message(3, local(2026, 9, 27, 9, 42)),
      message(4, local(2026, 9, 27, 9, 44), true),
    ])

    expect(separators(items)).toBe(1)
    expect(items.map((item) => item.showTimestamp)).toEqual([
      true,
      true,
      false,
      false,
    ])
  })

  test('前置更早的一页后重新分组：分隔不重复，边界消息不再显示时间', () => {
    const page = [message(3, local(2026, 9, 27, 10, 0)), message(4, local(2026, 9, 27, 10, 3))]
    const before = buildChatTimeline(page)
    expect(before.map((item) => item.showTimestamp)).toEqual([true, false])
    expect(separators(before)).toBe(1)

    const after = buildChatTimeline([
      message(1, local(2026, 9, 27, 9, 58)),
      message(2, local(2026, 9, 27, 9, 59)),
      ...page,
    ])

    expect(separators(after)).toBe(1)
    expect(after.map((item) => item.showTimestamp)).toEqual([
      true,
      false,
      false,
      false,
    ])
  })

  test('前置跨日的一页后旧的首条消息保留时间但不重复日期分隔', () => {
    const items = buildChatTimeline([
      message(1, local(2026, 9, 26, 22, 0)),
      message(2, local(2026, 9, 27, 10, 0)),
      message(3, local(2026, 9, 27, 10, 30)),
    ])

    expect(separators(items)).toBe(2)
    expect(items.map((item) => item.showTimestamp)).toEqual([true, true, true])
  })

  test('无法解析的时间戳不猜日期：不插入分隔并保留原时间显示', () => {
    const items = buildChatTimeline([
      message(1, 'garbage'),
      message(2, local(2026, 9, 27, 10, 0)),
    ])

    expect(items[0]!.day).toBeNull()
    expect(items[0]!.showDaySeparator).toBe(false)
    expect(items[0]!.showTimestamp).toBe(true)
    expect(items[1]!.showDaySeparator).toBe(true)
    expect(items[1]!.showTimestamp).toBe(true)
  })

  test('乱序的相邻时间戳按新分组处理', () => {
    const items = buildChatTimeline([
      message(1, local(2026, 9, 27, 10, 0)),
      message(2, local(2026, 9, 27, 9, 0)),
    ])

    expect(items[1]!.showTimestamp).toBe(true)
  })
})

describe('chat timestamp labels', () => {
  const locale = i18n.global.locale as unknown as { value: string }
  const originalLocale = locale.value

  afterEach(() => {
    locale.value = originalLocale
  })

  test('formatChatClock 只输出浏览器本地 HH:mm', () => {
    expect(formatChatClock(local(2026, 9, 27, 23, 5))).toBe('23:05')
    expect(formatChatClock('garbage')).toBe('garbage')
  })

  test('无时区历史日期时间按 UTC 墙钟解析（issue #221，与移动端一致）', () => {
    const expected = Date.UTC(2026, 8, 27, 9, 30)
    expect(parseDate('2026-09-27 09:30:00').getTime()).toBe(expected)
    expect(parseDate('2026-09-27T09:30:00').getTime()).toBe(expected)
    expect(formatChatClock('2026-09-27 09:30:00')).toBe(
      formatChatClock(new Date(expected).toISOString()),
    )
  })

  test('无时区日期时间与等价的 UTC 表示分成同一天/同一组', () => {
    const bare = buildChatTimeline([
      { id: 1, createdAt: '2026-09-27 09:30:00', isSelf: false },
      { id: 2, createdAt: '2026-09-27 09:40:00', isSelf: false },
    ])
    const zoned = buildChatTimeline([
      { id: 1, createdAt: '2026-09-27T09:30:00Z', isSelf: false },
      { id: 2, createdAt: '2026-09-27T09:40:00Z', isSelf: false },
    ])
    expect(bare.map((item) => item.day?.getTime())).toEqual(zoned.map((item) => item.day?.getTime()))
    expect(bare.map((item) => item.showTimestamp)).toEqual(zoned.map((item) => item.showTimestamp))
    expect(bare.map((item) => item.showDaySeparator)).toEqual(
      zoned.map((item) => item.showDaySeparator),
    )
  })

  test('纯日期日历值保持日期且不编造时刻（与移动端一致）', () => {
    expect(parseDate('2026-09-27').getTime()).toBe(new Date(2026, 8, 27).getTime())
    expect(formatDate('2026-09-27')).toBe('2026-09-27')
    expect(formatDateTime('2026-09-27')).toBe('2026-09-27')
    expect(formatChatTime('2026-09-27')).toBe('2026-09-27')
    expect(formatChatClock('2026-09-27')).toBe('2026-09-27')
  })

  test('formatChatDayLabel 区分今天/昨天/同年/跨年', () => {
    locale.value = 'zh'
    const now = new Date(2026, 8, 28, 10, 0)
    expect(formatChatDayLabel(new Date(2026, 8, 28, 0, 5), now)).toBe('今天')
    expect(formatChatDayLabel(new Date(2026, 8, 27, 23, 59), now)).toBe('昨天')
    expect(formatChatDayLabel(new Date(2026, 0, 15, 9, 30), now)).toBe('1月15日')
    expect(formatChatDayLabel(new Date(2025, 11, 31, 23, 59), now)).toBe('2025年12月31日')
    // 跨月/跨年的“昨天”按本地日历日计算
    expect(formatChatDayLabel(new Date(2025, 11, 31, 12), new Date(2026, 0, 1, 8))).toBe('昨天')
  })
})
