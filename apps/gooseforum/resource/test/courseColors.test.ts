import { describe, expect, test } from 'vitest'
import {
  courseColorSlotCount,
  courseColorSlotFor,
  courseColorSlots,
  courseSlotVar,
  courseContentVar,
} from '../src/site/utils/courseColors'

// ---- oklch → sRGB → WCAG 对比度 工具（与 courseColors.ts 色板配套）----

function oklchToSrgb(oklchStr: string): [number, number, number] {
  const match = /oklch\(\s*([\d.]+)%\s+([\d.]+)\s+([\d.]+)\s*\)/.exec(oklchStr)
  if (!match) throw new Error(`bad oklch: ${oklchStr}`)
  const L = Number(match[1]) / 100
  const C = Number(match[2])
  const h = (Number(match[3]) * Math.PI) / 180

  const a = C * Math.cos(h)
  const b = C * Math.sin(h)
  const l_ = L + 0.3963377774 * a + 0.2158037573 * b
  const m_ = L - 0.1055613458 * a - 0.0638541728 * b
  const s_ = L - 0.0894841775 * a - 1.291485548 * b
  const l = l_ ** 3
  const m = m_ ** 3
  const s = s_ ** 3

  const r = 4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s
  const g = -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s
  const bl = -0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s

  const lin = (v: number): number => {
    const c = v > 0.0031308 ? 1.055 * Math.pow(v, 1 / 2.4) - 0.055 : 12.92 * v
    return Math.max(0, Math.min(1, c))
  }
  return [lin(r), lin(g), lin(bl)]
}

function relativeLuminance(rgb: [number, number, number]): number {
  const [r, g, b] = rgb
  const chan = (c: number): number => (c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4))
  return 0.2126 * chan(r) + 0.7152 * chan(g) + 0.0722 * chan(b)
}

function contrastRatio(bg: string, fg: string): number {
  const lb = relativeLuminance(oklchToSrgb(bg))
  const lf = relativeLuminance(oklchToSrgb(fg))
  const [hi, lo] = lb > lf ? [lb, lf] : [lf, lb]
  return (hi + 0.05) / (lo + 0.05)
}

describe('courseColorSlots 对比度（issue #226 验收：≥4.5:1）', () => {
  test('gf-light：8 槽位深底 + 近白字全部 ≥4.5:1', () => {
    for (const slot of courseColorSlots) {
      const ratio = contrastRatio(slot.lightBg, slot.lightContent)
      expect(ratio, `slot ${slot.index} light 对比度`).toBeGreaterThanOrEqual(4.5)
    }
  })

  test('gf-dark：8 槽位亮底 + 近黑字全部 ≥4.5:1', () => {
    for (const slot of courseColorSlots) {
      const ratio = contrastRatio(slot.darkBg, slot.darkContent)
      expect(ratio, `slot ${slot.index} dark 对比度`).toBeGreaterThanOrEqual(4.5)
    }
  })

  test('槽位数与序号连续（1..N）', () => {
    expect(courseColorSlotCount).toBe(8)
    courseColorSlots.forEach((slot, i) => expect(slot.index).toBe(i + 1))
  })
})

describe('courseColorSlotFor 同课同色与分布（issue #226）', () => {
  test('同一课程标识跨调用返回同一槽位', () => {
    const seed = '122004.01'
    const first = courseColorSlotFor(seed)
    for (let i = 0; i < 20; i++) expect(courseColorSlotFor(seed)).toBe(first)
  })

  test('不同课号分布到至少 4 个槽位（避免大量撞色）', () => {
    const seeds = ['100001', '100002', '100003', '100004', '100005', '122004', '220001', '310101', 'A001', 'B002']
    const used = new Set(seeds.map((s) => courseColorSlotFor(s)))
    expect(used.size).toBeGreaterThanOrEqual(4)
  })

  test('返回 1..8 区间', () => {
    for (let i = 0; i < 50; i++) {
      const slot = courseColorSlotFor(`seed-${i}`)
      expect(slot).toBeGreaterThanOrEqual(1)
      expect(slot).toBeLessThanOrEqual(8)
    }
  })
})

describe('courseSlotVar / courseContentVar 与 tokens.css 双写防漂移', () => {
  test('变量名符合 tokens.css 定义', () => {
    expect(courseSlotVar(1)).toBe('--gf-color-course-1')
    expect(courseContentVar(1)).toBe('--gf-color-course-1-content')
  })

  test('tokens.css 实际定义了全部 8 槽双主题（light+dark）', () => {
    const fs = require('node:fs')
    const path = require('node:path')
    const css = fs.readFileSync(
      path.join(__dirname, '..', 'src', 'styles', 'tokens.css'),
      'utf8',
    )
    for (const slot of courseColorSlots) {
      expect(css, `light ${slot.lightBg}`).toContain(`--gf-color-course-${slot.index}: ${slot.lightBg};`)
      expect(css, `light content`).toContain(`--gf-color-course-${slot.index}-content: ${slot.lightContent};`)
      expect(css, `dark ${slot.darkBg}`).toContain(`--gf-color-course-${slot.index}: ${slot.darkBg};`)
      expect(css, `dark content`).toContain(`--gf-color-course-${slot.index}-content: ${slot.darkContent};`)
    }
  })
})
