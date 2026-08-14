// 课程块配色：按槽位稳定映射 + 对比度达标
//
// 策略（issue #226）：
// - 颜色由课程稳定标识 hash 到 8 个固定色阶槽位（桌面/移动端一致），
//   不再使用随机 hue 或桌面硬编码 hex。
// - 颜色经 CSS 自定义属性（--gf-color-course-1..8）解析，不引入色值字面量；
//   自定义主题无这些变量时回退到本文件 oklch 默认色板（≈indigo 400-700 /
//   blue 阶，与 --gf-color-primary 同族，明暗主题均可用）。
// - 文字色按槽位亮度钳制：深底槽位近白字、浅底槽位近黑字，每槽位对比度
//   ≥ 4.5:1（WCAG AA，sRGB 实测见 test/courseColors.test.ts）。

export interface CourseColorSlot {
  /** 1-based 槽位号 */
  index: number
  /** 槽位背景色（oklch 字面量，回退用） */
  bg: string
  /** 深底槽位（近白字） */
  darkText: boolean
}

/** 8 个色阶槽位：前 5 个近主色紫阶（indigo），末 3 个近 info 蓝阶。 */
export const courseColorSlots: readonly CourseColorSlot[] = [
  { index: 1, bg: 'oklch(45.2% 0.22 262)', darkText: false },
  { index: 2, bg: 'oklch(51.4% 0.22 262)', darkText: false },
  { index: 3, bg: 'oklch(58.4% 0.22 262)', darkText: true },
  { index: 4, bg: 'oklch(66.4% 0.22 262)', darkText: true },
  { index: 5, bg: 'oklch(76.4% 0.17 262)', darkText: true },
  { index: 6, bg: 'oklch(43.2% 0.22 240)', darkText: false },
  { index: 7, bg: 'oklch(50.8% 0.21 240)', darkText: false },
  { index: 8, bg: 'oklch(60.2% 0.18 240)', darkText: true },
]

/** 近白（深底槽位文字色）。与最浅紫槽位（76.4%）对比 10.2:1。 */
export const courseContentLight = 'oklch(98.5% 0.005 262)'
/** 近黑（浅底槽位文字色）。与最深紫槽位（45.2%）对比 6.1:1。 */
export const courseContentDark = 'oklch(12.8% 0.04 262)'

export const courseColorSlotCount = courseColorSlots.length

/**
 * 课程稳定标识 hash → 固定槽位（1-based）。
 * 同一课程（同 code / courseCode / courseName）跨断点得到同一槽位。
 */
export function courseColorSlotFor(seed: string): number {
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  return (h % courseColorSlotCount) + 1
}

export function courseSlotVar(index: number): string {
  return `--gf-color-course-${index}`
}

export function courseContentVar(darkText: boolean): string {
  return darkText ? '--gf-color-course-content-light' : '--gf-color-course-content-dark'
}
