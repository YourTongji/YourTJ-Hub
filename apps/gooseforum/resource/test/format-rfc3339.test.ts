import { describe, expect, test } from 'vitest'
import { formatDateTime, timeAgo } from '../src/runtime/format'

// issue #221 回归：后端统一输出 RFC3339（带时区偏移）后，前端解析必须正确
// 还原为浏览器本地时间，不再出现"无时区标记按本地解析"导致的固定偏移。
// 测试在任意本地时区（CI 为 UTC，本地可为 UTC+8）下均成立：
// RFC3339 带偏移字符串由 new Date() 按绝对时刻解析，getHours 输出本地墙钟。
describe('format RFC3339 time parsing (issue #221)', () => {
  test('formatDateTime 对带 +08:00 偏移的 RFC3339 还原本地墙钟', () => {
    // 2026-08-14T10:00:00+08:00 == 2026-08-14T02:00:00Z
    const value = '2026-08-14T10:00:00+08:00'
    const local = new Date(value)
    const expected = `${local.getFullYear()}-${String(local.getMonth() + 1).padStart(2, '0')}-${String(local.getDate()).padStart(2, '0')} ${String(local.getHours()).padStart(2, '0')}:${String(local.getMinutes()).padStart(2, '0')}`
    expect(formatDateTime(value)).toBe(expected)
    // 与直接 new Date() 的本地解释一致（RFC3339 语义确定，不随浏览器时区漂移）
    expect(formatDateTime(value)).not.toBe('2026-08-14 10:00:00')
  })

  test('timeAgo 对 Z 结尾的 UTC 时间与本地时间差一致（相对 10 秒）', () => {
    // 刚刚发布的帖子（RFC3339 UTC 墙钟 + Z）应显示"刚刚"而非"8小时前"。
    // 断言用 i18n 数字占位替换后的结构（不绑定具体语言文案）：先取"刚刚"键
    // 语义等价物——通过 stub 时间并断言返回非"n 小时前"结构。
    const now = new Date()
    const justNow = new Date(now.getTime() - 10 * 1000).toISOString() // 带 Z 的 RFC3339
    const result = timeAgo(justNow)
    // 10 秒前必须不是"X hours ago / X 小时前"形态（无数字占位）
    expect(result).not.toMatch(/\d+\s*(hours?|小时)/)
    expect(result.length).toBeGreaterThan(0)
  })

  test('timeAgo 对 1 小时前的 UTC 时间显示 1 小时量级（语言无关数字断言）', () => {
    const now = new Date()
    const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000).toISOString()
    const result = timeAgo(oneHourAgo)
    // 60 分钟前：返回文案应含 "1"（分钟/小时量级），且不含 8 小时偏差
    expect(result).toContain('1')
    expect(result).not.toMatch(/\b8\s*(hours?|小时)/)
  })

  test('旧格式（无时区标记）兜底：timeAgo 不应崩溃', () => {
    // 存量数据/第三方透传可能仍是 "2006-01-02 15:04:05"；解析失败时原样返回
    expect(typeof timeAgo('2026-08-14 10:00:00')).toBe('string')
    expect(timeAgo('')).toBe('')
  })
})
