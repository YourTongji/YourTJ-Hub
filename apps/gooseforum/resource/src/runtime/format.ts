import { i18n } from './i18n'

export function formatNumber(value: number): string {
  if (value >= 1000000) return `${trim(value / 1000000)}m`
  if (value >= 1000) return `${trim(value / 1000)}k`
  return String(value)
}

export function formatDateTime(value: string): string {
  if (!value) return ''
  // 纯日期值没有时刻可显示，保留原值（与移动端 `_timeField` 的兜底一致）。
  if (!hasTimePart(value)) return value
  const date = parseDate(value)
  if (Number.isNaN(date.getTime())) return value
  return `${formatDate(value)} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function formatChatTime(value: string): string {
  if (!value) return ''
  if (!hasTimePart(value)) return value
  const date = parseDate(value)
  if (Number.isNaN(date.getTime())) return value
  const now = new Date()
  const time = `${pad(date.getHours())}:${pad(date.getMinutes())}`
  if (isSameDay(date, now)) return time
  if (date.getFullYear() === now.getFullYear()) {
    return i18n.global.t('date.monthDayTime', { month: date.getMonth() + 1, day: date.getDate(), time })
  }
  return i18n.global.t('date.yearMonthDayTime', { year: date.getFullYear(), month: date.getMonth() + 1, day: date.getDate(), time })
}

export function timeAgo(value: string): string {
  if (!value) return ''
  const timestamp = parseDate(value).getTime()
  if (Number.isNaN(timestamp)) return value
  const seconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000))
  if (seconds < 60) return i18n.global.t('time.justNow')
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return i18n.global.t('time.minuteAgo', { count: minutes })
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return i18n.global.t('time.hourAgo', { count: hours })
  const days = Math.floor(hours / 24)
  if (days < 7) return i18n.global.t('time.dayAgo', { count: days })
  return formatDate(value)
}

export function formatDate(value: string): string {
  if (!value) return ''
  const date = parseDate(value)
  if (Number.isNaN(date.getTime())) return value.split(' ')[0] || value
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

/**
 * 私信日期分隔标签：今天/昨天/同年 `M月D日`/跨年 `YYYY年M月D日`。
 * [day] 为浏览器本地时刻，与 [formatChatTime] 使用同一时区基准。
 */
export function formatChatDayLabel(day: Date, now: Date = new Date()): string {
  if (isSameDay(day, now)) return i18n.global.t('date.today')
  const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1)
  if (isSameDay(day, yesterday)) return i18n.global.t('date.yesterday')
  if (day.getFullYear() === now.getFullYear()) {
    return i18n.global.t('date.monthDay', { month: day.getMonth() + 1, day: day.getDate() })
  }
  return i18n.global.t('date.yearMonthDay', {
    year: day.getFullYear(),
    month: day.getMonth() + 1,
    day: day.getDate(),
  })
}

/**
 * 聊天气泡内的时刻（仅浏览器本地 `HH:mm`）。
 *
 * 私信按日期分隔与时间分组渲染（见 `runtime/chat-timeline.ts`），日期已由分隔
 * 标签表达，气泡内只保留时刻；无法解析时返回原值。
 */
export function formatChatClock(value: string): string {
  if (!value) return ''
  // 纯日期值不编造时刻，保留原值（移动端 `formatChatClock` 同规则）。
  if (!hasTimePart(value)) return value
  const date = parseDate(value)
  if (Number.isNaN(date.getTime())) return value
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

/** 值是否带时刻部分（纯日期日历值没有）。 */
function hasTimePart(value: string): boolean {
  return value.replace(' ', 'T').includes('T')
}

/**
 * 解析服务端时间字符串。
 *
 * - 带偏移/Z 的 RFC3339：按绝对时刻解析。
 * - 无时区标记的**日期时间**（旧服务端 `time.DateTime` 输出）：按 UTC 墙钟解析
 *   （issue #221；服务器运行在 UTC）。补 'Z' 避免被按浏览器本地时区误解。
 * - 纯**日期**的日历值（校历、日期型字段）：按本地日历日解析，不随时区偏移，
 *   与移动端 `parseChatTimestamp` 的日期分支一致。
 */
export function parseDate(value: string): Date {
  const normalized = value.includes('T') ? value : value.replace(' ', 'T')
  if (!normalized.includes('T')) return new Date(`${normalized}T00:00:00`)
  return /[zZ]|[+-]\d{2}:?\d{2}$/.test(normalized)
    ? new Date(normalized)
    : new Date(`${normalized}Z`)
}

function isSameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear()
    && a.getMonth() === b.getMonth()
    && a.getDate() === b.getDate()
}

function pad(value: number): string {
  return String(value).padStart(2, '0')
}

function trim(value: number): string {
  return value.toFixed(value >= 10 ? 0 : 1).replace(/\.0$/, '')
}
