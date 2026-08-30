// 排课器时间工具：周次掩码、交集判断、arrangeInfoText 文本解析。
//
// 冲突检测以「同一天 + 同一节次 + 周次交集非空」为判据，因此核心是
// 把周次数组转成位掩码做 O(1) 交集；文本解析器作为后端只返回
// arrangeInfoText（原始文本）时的降级手段。

/** 周次位掩码：bit i 置位 = 第 i+1 周上课。周次范围 [1,16]。 */
export type WeekMask = number

/** 学年周数上限（一系统 16 周/学期）。 */
export const MAX_WEEK = 16

/** 从展开的周次数组构建位掩码。非法周次（非整数 / 越界）忽略。 */
export function buildWeekMask(weeks: readonly number[]): WeekMask {
  let mask = 0
  for (const week of weeks) {
    if (Number.isInteger(week) && week >= 1 && week <= MAX_WEEK) {
      mask |= 1 << (week - 1)
    }
  }
  return mask
}

/** 两个周次数组是否有交集。 */
export function weeksOverlap(a: readonly number[], b: readonly number[]): boolean {
  return a.some((week) => b.includes(week))
}

/** 两个周次掩码是否有交集。 */
export function weekMasksOverlap(a: WeekMask, b: WeekMask): boolean {
  return (a & b) !== 0
}

/** 排课文本解析结果（一段上课安排）。 */
export interface PkArrangementParse {
  weekStart: number
  weekEnd: number
  /** 单周 / 双周 / 全周 */
  weekParity: 'odd' | 'even' | null
  /** 非连续周显式列出（如 "1,3,5周"）；区间型为空数组 */
  specificWeeks: number[]
  /** 星期 1-7 */
  day: number
  /** 节次起点 1-12 */
  sectionStart: number
  /** 节次终点 1-12 */
  sectionEnd: number
  /** 地点原文（无则空串） */
  location: string
}

const DAY_MAP: Record<string, number> = {
  一: 1, 二: 2, 三: 3, 四: 4, 五: 5, 六: 6, 日: 7, 天: 7,
}

function parseDay(text: string): number | null {
  // 兼容「周一」与「星期一」
  const match = /(?:星期|周)([一二三四五六日天])/.exec(text)
  return match ? DAY_MAP[match[1]] ?? null : null
}

function parseWeeks(text: string): Pick<PkArrangementParse, 'weekStart' | 'weekEnd' | 'weekParity' | 'specificWeeks'> | null {
  // 奇偶后缀：1-8周(单周) / 1-8周(单) / 单周 / 双周
  let parity: 'odd' | 'even' | null = null
  if (/(单周|\(单\)|（单）)/.test(text)) parity = 'odd'
  else if (/(双周|\(双\)|（双）)/.test(text)) parity = 'even'

  // 周次片段：兼容 [1-8周]、1-8周、1、3、5周、1-3,5周（「周」必需，避免误吞节次区间）
  const weekToken = /\[?((?:\d{1,2}\s*[-~－—]\s*\d{1,2}|\d{1,2})(?:[、,，]\s*\d{1,2})*)\s*周\]?/.exec(text)
  if (!weekToken) return null

  const content = weekToken[1]
  const weeks = new Set<number>()
  // 区间型（1-8）展开
  const rangeMatch = /(\d{1,2})\s*[-~－—]\s*(\d{1,2})/.exec(content)
  if (rangeMatch) {
    const start = clampWeek(Number(rangeMatch[1]))
    const end = clampWeek(Number(rangeMatch[2]))
    for (let w = start; w <= end; w++) weeks.add(w)
  }
  // 枚举数字（含区间端点与 1-3,5 的尾部枚举）
  for (const m of content.matchAll(/\d{1,2}/g)) {
    weeks.add(clampWeek(Number(m[0])))
  }
  if (weeks.size === 0) return null

  let weekList = [...weeks].sort((a, b) => a - b)
  if (parity === 'odd') weekList = weekList.filter((w) => w % 2 === 1)
  if (parity === 'even') weekList = weekList.filter((w) => w % 2 === 0)
  if (weekList.length === 0) return null

  const isContiguous = weekList[weekList.length - 1] - weekList[0] + 1 === weekList.length
  return {
    weekStart: Math.min(...weekList),
    weekEnd: Math.max(...weekList),
    weekParity: parity,
    specificWeeks: isContiguous ? [] : weekList,
  }
}

function parseSections(text: string): { sectionStart: number; sectionEnd: number } | null {
  // 区间：3-4节 / 第3-4节 / 3～4节（「节」必需，避免误吞周次区间）
  const range = /(?:第)?(\d{1,2})\s*[-~－—]\s*(\d{1,2})\s*节/.exec(text)
  if (range) {
    const start = clampSection(Number(range[1]))
    const end = clampSection(Number(range[2]))
    return { sectionStart: Math.min(start, end), sectionEnd: Math.max(start, end) }
  }
  // 单节：第3节 / 3节
  const single = /(?:第)?(\d{1,2})\s*节/.exec(text)
  if (single) {
    const section = clampSection(Number(single[1]))
    return { sectionStart: section, sectionEnd: section }
  }
  return null
}

function clampWeek(value: number): number {
  if (!Number.isFinite(value)) return 1
  return Math.min(MAX_WEEK, Math.max(1, value))
}

function clampSection(value: number): number {
  if (!Number.isFinite(value)) return 1
  return Math.min(12, Math.max(1, value))
}

/**
 * 解析一段（或多段）arrangeInfoText 为结构化安排。
 *
 * 典型输入：`"1-8周 周一 3-4节 同济楼A201；9-16周 周三 5-6节 线上"`
 * 分段符：`；` `;` `|` 换行。解析失败的段被忽略（不抛错）。
 */
export function parseArrangeInfoText(text: string): PkArrangementParse[] {
  const result: PkArrangementParse[] = []
  const raw = String(text ?? '').trim()
  if (!raw) return result

  const segments = raw.split(/[；;|\n]/).map((s) => s.trim()).filter(Boolean)
  for (const segment of segments) {
    const day = parseDay(segment)
    if (day === null) continue // 无星期信息无法上表

    const sections = parseSections(segment)
    if (!sections) continue

    const weeks = parseWeeks(segment)
    if (!weeks) continue

    const location = segment
      .replace(/(?:星期|周)([一二三四五六日天])/g, '')
      .replace(/[（(](单|双)周?[)）]/g, '')
      .replace(/(?:第)?\d{1,2}\s*[-~－—]\s*\d{1,2}\s*周/g, '')
      .replace(/(?:第)?\d{1,2}(?:[、,，]\d{1,2})+\s*周/g, '')
      .replace(/(?:第)?\d{1,2}\s*周/g, '')
      .replace(/(?:第)?\d{1,2}\s*[-~－—]\s*\d{1,2}\s*节?/g, '')
      .replace(/(?:第)?\d{1,2}\s*节/g, '')
      .replace(/[\[\]]/g, '')
      .trim()

    result.push({ ...weeks, day, ...sections, location })
  }
  return result
}

/**
 * 课表行数：新 11 节制（calendarId >= 120 为 2025-2026 学年第 1 学期及以后）vs 旧 12 节制。
 * 数据不足（非数字）时按 12 节处理。
 */
export function maxRowsForCalendar(calendarId: number | undefined): number {
  if (typeof calendarId === 'number' && calendarId >= 120) return 11
  return 12
}

/** UI 行 → 节段（时段）映射：1-2→1、3-4→2、5-6→3、7-8→4、9-10→5、11→6；旧制 9→5、10-12→6。 */
export function getRowSection(row: number, calendarId: number | undefined): number {
  if (maxRowsForCalendar(calendarId) === 11) {
    return Math.min(6, Math.ceil(row / 2))
  }
  if (row <= 8) return Math.ceil(row / 2)
  if (row === 9) return 5
  return 6
}

/** 同列课程的节次区间簇（相交/包含的课程归入同一格渲染）。 */
export interface PkDayCluster<T> {
  /** 簇内最早节次（1-based，格子锚定行）。 */
  start: number
  /** 簇内最晚节次（rowspan 覆盖到该行）。 */
  end: number
  items: T[]
}

/**
 * 把同一天的课程按节次区间聚类：区间相交（含包含、部分重叠）的课归为同格。
 * 容忍式冲突下部分重叠的课（如 1-2 节与 2-3 节）必须同格可见，
 * 不能让一块的 rowspan 吞掉另一块；不相交的课各自独立成格。
 */
export function clusterBySections<T extends { occupyTime: number[] }>(courses: T[]): PkDayCluster<T>[] {
  const sorted = [...courses].sort((a, b) => a.occupyTime[0] - b.occupyTime[0])
  const clusters: PkDayCluster<T>[] = []
  for (const course of sorted) {
    const start = course.occupyTime[0]
    const end = course.occupyTime[course.occupyTime.length - 1]
    const last = clusters[clusters.length - 1]
    if (last && start <= last.end) {
      last.end = Math.max(last.end, end)
      last.items.push(course)
    } else {
      clusters.push({ start, end, items: [course] })
    }
  }
  return clusters
}

/** 由学期起始日期计算「今天」是第几周（1-based）；学期外/日期非法返回 null。
 * 周一为一周之始：起始日非周一时，第一周延伸至首个周日。
 * endDate（可选）为学期最后一天：当天仍属学期内，之后返回 null；
 * 超出 MAX_WEEK 支持范围同样返回 null（不夹取，避免学期结束后停留在最后一周）。 */
export function currentWeekForDate(startDate: string, today: Date = new Date(), endDate?: string): number | null {
  const start = new Date(`${startDate}T00:00:00`)
  if (Number.isNaN(start.getTime())) return null
  const todayUtc = Date.UTC(today.getFullYear(), today.getMonth(), today.getDate())
  const diffDays = Math.floor(
    (todayUtc - Date.UTC(start.getFullYear(), start.getMonth(), start.getDate())) / 86_400_000,
  )
  if (diffDays < 0) return null
  const week = Math.floor(diffDays / 7) + 1
  if (week > MAX_WEEK) return null
  if (endDate) {
    const end = new Date(`${endDate}T00:00:00`)
    if (!Number.isNaN(end.getTime()) && todayUtc > Date.UTC(end.getFullYear(), end.getMonth(), end.getDate())) {
      return null
    }
  }
  return week
}

/** 周次数组 → 紧凑显示文本（如 [1,2,3]→"1-3"、[1,3,5]→"1,3,5"、[2]→"2"）。 */
export function formatWeeksText(weeks: readonly number[] | undefined): string {
  if (!weeks || weeks.length === 0) return ''
  const sorted = [...new Set(weeks)].sort((a, b) => a - b)
  const parts: string[] = []
  let runStart = sorted[0]
  let prev = sorted[0]
  for (let i = 1; i <= sorted.length; i++) {
    const current = sorted[i]
    if (current === prev + 1) {
      prev = current
      continue
    }
    parts.push(runStart === prev ? `${runStart}` : `${runStart}-${prev}`)
    runStart = current
    prev = current
  }
  return parts.join(',')
}
