// 课程块配色（issue #226/#234 吸收）：按槽位稳定映射 + 对比度达标
//
// 策略：
// - 颜色由课程稳定标识 hash 到 8 个固定色阶槽位（桌面/移动端一致，同课同色）。
// - 槽位色经 CSS 自定义属性（--gf-color-course-1..8 / -content）解析，
//   tokens.css 中 gf-light / gf-dark 双主题各定义一份（深底近白字 / 亮底近黑字），
//   每槽位背景与文字对比度 ≥4.5:1（WCAG AA，sRGB 实测见 test/courseColors.test.ts）。
// - 本文件是色板的 TS 镜像：供组件取值与单测断言（对比度 / 同课同色 / 分布），
//   并防 tokens.css 与常量双写漂移（测试会交叉校验）。

export interface CourseColorSlot {
  /** 1-based 槽位号 */
  index: number
  /** gf-light 底色（oklch 字面量，与 tokens.css --gf-color-course-N 一致） */
  lightBg: string
  /** gf-light 文字色（与 --gf-color-course-N-content 一致） */
  lightContent: string
  /** gf-dark 底色 */
  darkBg: string
  /** gf-dark 文字色 */
  darkContent: string
}

/** 8 个色阶槽位：浅色主题深底近白字；深色主题亮底近黑字。 */
export const courseColorSlots: readonly CourseColorSlot[] = [
  { index: 1, lightBg: 'oklch(45% 0.19 268)', lightContent: 'oklch(98% 0.003 247)', darkBg: 'oklch(72% 0.15 262)', darkContent: 'oklch(16% 0.02 265)' },
  { index: 2, lightBg: 'oklch(42% 0.12 200)', lightContent: 'oklch(98% 0.003 247)', darkBg: 'oklch(72% 0.1 200)', darkContent: 'oklch(16% 0.02 265)' },
  { index: 3, lightBg: 'oklch(44% 0.13 155)', lightContent: 'oklch(98% 0.003 247)', darkBg: 'oklch(70% 0.12 155)', darkContent: 'oklch(16% 0.02 265)' },
  { index: 4, lightBg: 'oklch(45% 0.15 50)', lightContent: 'oklch(98% 0.003 247)', darkBg: 'oklch(74% 0.13 80)', darkContent: 'oklch(16% 0.02 265)' },
  { index: 5, lightBg: 'oklch(46% 0.2 20)', lightContent: 'oklch(98% 0.003 247)', darkBg: 'oklch(70% 0.16 25)', darkContent: 'oklch(16% 0.02 265)' },
  { index: 6, lightBg: 'oklch(40% 0.14 280)', lightContent: 'oklch(98% 0.003 247)', darkBg: 'oklch(70% 0.12 285)', darkContent: 'oklch(16% 0.02 265)' },
  { index: 7, lightBg: 'oklch(43% 0.17 230)', lightContent: 'oklch(98% 0.003 247)', darkBg: 'oklch(72% 0.13 230)', darkContent: 'oklch(16% 0.02 265)' },
  { index: 8, lightBg: 'oklch(47% 0.12 330)', lightContent: 'oklch(98% 0.003 247)', darkBg: 'oklch(70% 0.13 330)', darkContent: 'oklch(16% 0.02 265)' },
]

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

export function courseContentVar(index: number): string {
  return `--gf-color-course-${index}-content`
}
