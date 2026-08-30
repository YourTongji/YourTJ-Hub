// 排课器节次时间表（纯展示用）。
//
// 同济两套节次制的上课时间（11 节新制 calendarId>=120 / 旧 12 节制）。
// 一系统不提供官方作息表，此处为通用高校作息的近似值，仅用于课表左侧
// 起止时间展示与上午/下午/晚上分组；显示错误不影响任何排课数据。

export interface SectionTime {
  /** 节次（1-based）。 */
  section: number
  /** 开始时间 "HH:MM"。 */
  start: string
  /** 结束时间 "HH:MM"。 */
  end: string
}

/** 11 节新制（2025-2026 学年起）。 */
const SECTION_TIMES_11: SectionTime[] = [
  { section: 1, start: '08:00', end: '08:45' },
  { section: 2, start: '08:50', end: '09:35' },
  { section: 3, start: '09:50', end: '10:35' },
  { section: 4, start: '10:40', end: '11:25' },
  { section: 5, start: '11:30', end: '12:15' },
  { section: 6, start: '13:30', end: '14:15' },
  { section: 7, start: '14:20', end: '15:05' },
  { section: 8, start: '15:20', end: '16:05' },
  { section: 9, start: '16:20', end: '17:05' },
  { section: 10, start: '17:10', end: '17:55' },
  { section: 11, start: '19:00', end: '19:45' },
]

/** 旧 12 节制。 */
const SECTION_TIMES_12: SectionTime[] = [
  ...SECTION_TIMES_11.slice(0, 10),
  { section: 11, start: '19:00', end: '19:45' },
  { section: 12, start: '19:50', end: '20:35' },
]

/** 按节次制取作息表（11/12 节）。 */
export function sectionTimesFor(maxRows: number): SectionTime[] {
  return maxRows === 11 ? SECTION_TIMES_11 : SECTION_TIMES_12
}

export type DayPart = 'morning' | 'afternoon' | 'evening'

/** 按开始时间推导时段分组：<12:00 上午、<18:00 下午、否则晚上。 */
export function dayPartOfStart(start: string): DayPart {
  const hour = Number(start.slice(0, 2))
  if (Number.isNaN(hour)) return 'morning'
  if (hour < 12) return 'morning'
  if (hour < 18) return 'afternoon'
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
