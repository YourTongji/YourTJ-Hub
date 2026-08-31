// 排课器节次时间表（纯展示用，可被后台设置覆盖）。
//
// 默认作息以四个锚点构建（45 分钟一节，节间 5 分钟，大课间 25 分钟）：
//   第 3 节 10:00 开始、第 5 节 13:30 开始、第 7 节 15:30 开始、第 10 节 18:30 开始。
// 一系统不提供官方作息表，此为可运营覆盖的默认值，仅用于课表左侧
// 起止时间展示与上午/下午/晚上分组；显示错误不影响任何排课数据。

export interface SectionTime {
  /** 节次（1-based）。 */
  section: number
  /** 开始时间 "HH:MM"。 */
  start: string
  /** 结束时间 "HH:MM"。 */
  end: string
}

/** 完整 12 节制默认作息（锚点：3=10:00 / 5=13:30 / 7=15:30 / 10=18:30）。 */
export const DEFAULT_SECTION_TIMES_12: SectionTime[] = [
  { section: 1, start: '08:00', end: '08:45' },
  { section: 2, start: '08:50', end: '09:35' },
  { section: 3, start: '10:00', end: '10:45' },
  { section: 4, start: '10:50', end: '11:35' },
  { section: 5, start: '13:30', end: '14:15' },
  { section: 6, start: '14:20', end: '15:05' },
  { section: 7, start: '15:30', end: '16:15' },
  { section: 8, start: '16:20', end: '17:05' },
  { section: 9, start: '17:10', end: '17:55' },
  { section: 10, start: '18:30', end: '19:15' },
  { section: 11, start: '19:20', end: '20:05' },
  { section: 12, start: '20:10', end: '20:55' },
]

/** 11 节新制 = 12 节制前 11 节（calendarId>=120 行数裁剪后的展示）。 */
export const DEFAULT_SECTION_TIMES_11: SectionTime[] = DEFAULT_SECTION_TIMES_12.slice(0, 11)

/**
 * 按节次制取作息表：优先使用后台覆盖（overrides，按 section 对齐补齐缺口），
 * 缺失时回退默认表。maxRows 非 11 即按 12 节制处理。
 */
export function sectionTimesFor(maxRows: number, overrides?: SectionTime[] | null): SectionTime[] {
  const defaults = maxRows === 11 ? DEFAULT_SECTION_TIMES_11 : DEFAULT_SECTION_TIMES_12
  const target = maxRows === 11 ? 11 : 12
  if (!overrides || overrides.length === 0) return defaults
  const bySection = new Map(overrides.map((item) => [item.section, item]))
  return defaults.map((item) => bySection.get(item.section) ?? item).slice(0, target)
}

/** 解析 "HH:MM" 为分钟数；非法返回 null（分组推导容错用）。 */
export function parseHHMM(value: string): number | null {
  const match = /^(\d{1,2}):(\d{2})$/.exec(String(value ?? '').trim())
  if (!match) return null
  const hour = Number(match[1])
  const minute = Number(match[2])
  if (hour > 23 || minute > 59) return null
  return hour * 60 + minute
}

export type DayPart = 'morning' | 'afternoon' | 'evening'

/** 按开始时间推导时段分组：<12:00 上午、<18:00 下午、否则晚上。 */
export function dayPartOfStart(start: string): DayPart {
  const minutes = parseHHMM(start)
  if (minutes === null) return 'morning'
  if (minutes < 12 * 60) return 'morning'
  if (minutes < 18 * 60) return 'afternoon'
  return 'evening'
}

/** 分组切分点：每个 DayPart 首个节次（1-based）；无时间数据返回空。 */
export function dayPartBoundaries(times: SectionTime[]): Partial<Record<DayPart, number>> {
  const boundaries: Partial<Record<DayPart, number>> = {}
  for (const time of times) {
    const part = dayPartOfStart(time.start)
    if (boundaries[part] === undefined) boundaries[part] = time.section
  }
  return boundaries
}
