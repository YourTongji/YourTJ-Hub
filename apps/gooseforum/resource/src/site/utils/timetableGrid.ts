// 课表网格共享几何与文案 helper：
// ScheduleTimeTable（交互网格）与 ScheduleExportDialog（PNG 海报）必须视觉一致，
// 此前行高分配算法与文本格式化在两个组件内各复制一份且常量已漂移；统一收敛到本模块，
// 以参数化度量（RowMetrics）保留两端的视觉差异，避免后续布局调整静默分叉。
import { dayPartBoundaries, type DayPart, type SectionTime } from '@/site/utils/sectionTimes'
import { detectWeekParity, formatWeeksText } from '@/site/utils/pkArrange'
import type { PkCourseOnTable } from '@/site/types/pk'

type Translate = (key: string, params?: Record<string, unknown>) => string

export interface RowMetrics {
  /** 单行基准高度（px）。 */
  baseH: number
  /** 单元格上下 padding 合计（px）。 */
  padV: number
  /** 多门课叠放时单张紧凑卡片的估算高度（px）。 */
  multiCardH: number
}

/** 交互网格（ScheduleTimeTable）的行高度量。 */
export function interactiveRowMetrics(isMobile: boolean): RowMetrics {
  // 每张叠放（紧凑模式）卡片的现实最小高度估算，含边框、paddings 和单行课名+教室+周次
  // 移动端：p-1(4px*2)=8 + 课名12 + 教室10 + 周次10 + gap(2*4)=8 ≈ 58px
  // 桌面端：p-1.5(6px*2)=12 + 课名14 + 教室11 + 周次11 + gap(2*4)=8 ≈ 72px
  return isMobile ? { baseH: 52, padV: 4, multiCardH: 58 } : { baseH: 58, padV: 8, multiCardH: 72 }
}

/** 导出海报（ScheduleExportDialog）的行高度量：画幅固定 1140px，行高更舒展。 */
export const posterRowMetrics: RowMetrics = { baseH: 76, padV: 8, multiCardH: 90 }

/** 网格几何输入：三份网格派生数据与行高度量。 */
export interface GridLayout {
  cellCourses: PkCourseOnTable[][][]
  cellSpans: number[][]
  occupiedGrid: boolean[][]
}

/**
 * 动态计算每行的基准与扩展高度：
 * 统筹整网格所有单元格的最小空间需求，当同行存在单双周多门课纵向堆叠时，
 * 该节次行会自动增高；同行单门课自动均分撑满扩展后的行高，消除下半截留白。
 *
 * 排序策略：相同 span 内，多门课格子优先处理，确保行高先被真实内容需求撑高，
 * 再处理单门课——单门课无需拉升行高，但可以感知到已被多门课拉升的行高并正确计算 minHeight。
 */
export function computeRowHeights(grid: GridLayout, metrics: RowMetrics): number[] {
  const { baseH, padV, multiCardH } = metrics
  const rowCount = grid.cellCourses.length
  if (rowCount === 0) return []
  const rowHeights = new Array(rowCount).fill(baseH)

  interface CellInfo {
    span: number
    rIndex: number
    dayIndex: number
    count: number
  }

  const cells: CellInfo[] = []
  for (let r = 0; r < rowCount; r++) {
    const row = grid.cellCourses[r]
    if (!row) continue
    for (let d = 0; d < 7; d++) {
      if (!grid.occupiedGrid?.[r]?.[d]) {
        const span = grid.cellSpans?.[r]?.[d] || 1
        const count = row[d]?.length || 0
        cells.push({ span, rIndex: r, dayIndex: d, count })
      }
    }
  }

  // 主排序：span 升序（小跨度先确定基准），次排序：count 降序（同 span 内多门课先撑高行高）
  cells.sort((a, b) => a.span - b.span || b.count - a.count)

  for (const { span, rIndex, count } of cells) {
    if (count === 0) continue
    let reqInner = 0
    if (count === 1) {
      // 单门课只需填满当前行高，不主动拉伸（行高由多门课决定）
      reqInner = Math.max(span * baseH - padV, baseH - padV)
    } else {
      reqInner = count * multiCardH + (count - 1) * 4
    }
    const reqTotal = reqInner + padV
    let curTotal = 0
    for (let i = 0; i < span; i++) {
      curTotal += rowHeights[rIndex + i] || baseH
    }
    if (reqTotal > curTotal) {
      const diff = reqTotal - curTotal
      const perRow = Math.ceil(diff / span)
      for (let i = 0; i < span; i++) {
        const idx = rIndex + i
        if (idx < rowCount) {
          rowHeights[idx] += perRow
        }
      }
    }
  }

  return rowHeights
}

/** 跨 span 单元格的内部可用高度（扣除上下 padding）；offset 为该格锚定行在 rowHeights 中的下标。 */
export function cellInnerHeightFor(
  span: number,
  rowHeights: number[],
  metrics: RowMetrics,
  offset = 0,
): number {
  let totalH = 0
  for (let i = 0; i < span; i++) {
    totalH += rowHeights[offset + i] || metrics.baseH
  }
  return Math.max(metrics.baseH - metrics.padV, totalH - metrics.padV)
}

/** 单元格内课程卡片的最小高度：单门课撑满，多门叠放均分（扣除卡片间 gap-1 = 4px）。 */
export function cardMinHeightFor(
  span: number,
  rowHeights: number[],
  courseCount: number,
  metrics: RowMetrics,
  offset = 0,
): number {
  const innerH = cellInnerHeightFor(span, rowHeights, metrics, offset)
  const count = Math.max(1, courseCount)
  // 单门课（含跨 2+ 节的舒展模式）：直接返回格子全高，彻底撑满，消除下半截空白
  if (count === 1) {
    return Math.max(metrics.baseH - metrics.padV, innerH)
  }
  const available = innerH - (count - 1) * 4
  return Math.max(metrics.baseH - metrics.padV, Math.floor(available / count))
}

/** 教师名（teacherAndCode "张三(T001)" → "张三"）。 */
export function teacherName(course: PkCourseOnTable): string {
  return String(course.teacherAndCode || '').replace(/\([^)]*\)$/g, '').trim()
}

/** 紧凑展示教师名：超过 maxVisible 位时显示「前 maxVisible-1 位 等」。 */
export function compactTeacherName(raw: string, maxVisible: number): string {
  if (!raw) return ''
  const teachers = raw.split(/[,，、]/).map((s) => s.trim()).filter(Boolean)
  if (teachers.length <= maxVisible) return teachers.join('、')
  return `${teachers.slice(0, maxVisible - 1).join('、')} 等`
}

/** 单双周标识文本（无奇偶性返回 null；多课/紧凑展示用）。 */
export function weekParityLabel(weeks: readonly number[] | undefined, t: Translate): string | null {
  const parity = detectWeekParity(weeks)
  if (parity === 'odd') return t('schedule.parityOdd')
  if (parity === 'even') return t('schedule.parityEven')
  return null
}

/**
 * 周次格式精简（如单周 1-15 提炼为 "1-15周(单周)"，避免粗暴罗列 "1,3,5,7,9,11,13,15周"）。
 */
export function formatDisplayWeeks(weeks: readonly number[] | undefined, t: Translate): string {
  if (!weeks || weeks.length === 0) return ''
  const parity = detectWeekParity(weeks)
  const sorted = [...new Set(weeks)].sort((a, b) => a - b)
  if (parity === 'odd' && sorted.length >= 3) {
    const isRegularStep = sorted.every((w, i) => i === 0 || w === sorted[i - 1] + 2)
    if (isRegularStep) {
      return `${sorted[0]}-${sorted[sorted.length - 1]}周(${t('schedule.parityOdd')})`
    }
  }
  if (parity === 'even' && sorted.length >= 3) {
    const isRegularStep = sorted.every((w, i) => i === 0 || w === sorted[i - 1] + 2)
    if (isRegularStep) {
      return `${sorted[0]}-${sorted[sorted.length - 1]}周(${t('schedule.parityEven')})`
    }
  }
  return t('schedule.weeksN', { range: formatWeeksText(weeks) })
}

/** 该行（1-based）是否为某时段分组首行：返回上午/下午/晚上分组标签，否则 null。 */
export function dayPartLabelForRow(row: number, sectionTimes: SectionTime[], t: Translate): string | null {
  const starts = dayPartBoundaries(sectionTimes)
  for (const [part, start] of Object.entries(starts)) {
    if (start === row) {
      return t(`schedule.dayPart.${part as DayPart}`)
    }
  }
  return null
}
