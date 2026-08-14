// 课程块配色对比度校验（issue #226）
//
// 校验方式：先按「浏览器渲染」语义将各槽位 oklch 转 sRGB（8-bit），
// 再以 sRGB 计算 WCAG 相对亮度与对比度 —— 与最终渲染结果一致。
import { describe, expect, it } from 'vitest'
import {
  courseColorSlotCount,
  courseColorSlotFor,
  courseColorSlots,
  courseContentDark,
  courseContentLight,
} from '../src/site/utils/courseColors'

/** oklch(L% C H) → 8-bit sRGB hex。 */
function oklchToSrgbHex(oklch: string): string {
  const match = /^oklch\(([\d.]+)%\s+([\d.]+)\s+([\d.]+)\)$/.exec(oklch)
  if (!match) throw new Error(`unparseable oklch: ${oklch}`)
  const L = Number(match[1]) / 100
  const C = Number(match[2])
  const H = (Number(match[3]) * Math.PI) / 180
  const a = C * Math.cos(H)
  const b = C * Math.sin(H)
  const l = L + 0.3963377774 * a + 0.2158037573 * b
  const m = L - 0.1055613458 * a - 0.0638541728 * b
  const s = L - 0.0894841775 * a - 1.291485548 * b
  const transfer = (x: number) => {
    const t = Math.max(0, Math.min(1, x)) ** 3
    if (t > 0.0031308) return 1.055 * t ** (1 / 2.4) - 0.055
    return 12.92 * t
  }
  return [transfer(l), transfer(m), transfer(s)]
    .map((v) => Math.round(Math.max(0, Math.min(1, v)) * 255).toString(16).padStart(2, '0'))
    .join('')
}

function sRgbLum(hex: string): number {
  const f = (c: number) => (c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4))
  return (
    0.2126 * f(parseInt(hex.slice(0, 2), 16) / 255) +
    0.7152 * f(parseInt(hex.slice(2, 4), 16) / 255) +
    0.0722 * f(parseInt(hex.slice(4, 6), 16) / 255)
  )
}

function contrast(fgHex: string, bgHex: string): number {
  const lf = sRgbLum(fgHex)
  const lb = sRgbLum(bgHex)
  return (Math.max(lf, lb) + 0.05) / (Math.min(lf, lb) + 0.05)
}

describe('courseColors 槽位映射', () => {
  it('槽位数量固定为 8，且序号从 1 开始连续', () => {
    expect(courseColorSlotCount).toBe(8)
    courseColorSlots.forEach((slot, index) => {
      expect(slot.index).toBe(index + 1)
    })
  })

  it('同一课程标识跨调用映射到同一槽位（同课同色）', () => {
    for (const seed of ['122005.01', '122004.01', 'HIGH-MATH-01', '课程A']) {
      const first = courseColorSlotFor(seed)
      for (let i = 0; i < 5; i++) {
        expect(courseColorSlotFor(seed)).toBe(first)
      }
    }
  })

  it('不同课程标识分布覆盖多个槽位（非单色）', () => {
    const seeds = ['122001.01', '122002.01', '122003.01', '122004.01', '122005.01', '122006.01', '122007.01', '122008.01', '122009.01', '122010.01']
    const slots = new Set(seeds.map((seed) => courseColorSlotFor(seed)))
    expect(slots.size).toBeGreaterThanOrEqual(4)
  })

  it('回退/缺省输入也能映射（fallback seed）', () => {
    expect(courseColorSlotFor('course')).toBeGreaterThanOrEqual(1)
    expect(courseColorSlotFor('course')).toBeLessThanOrEqual(8)
  })
})

describe('courseColors 对比度（WCAG AA 4.5:1）', () => {
  const lightHex = oklchToSrgbHex(courseContentLight)
  const darkHex = oklchToSrgbHex(courseContentDark)

  it('深底槽位（darkText=false）用近白字，对比度 ≥4.5:1', () => {
    for (const slot of courseColorSlots) {
      if (slot.darkText) continue
      const ratio = contrast(lightHex, oklchToSrgbHex(slot.bg))
      expect(ratio, `slot ${slot.index} (#${oklchToSrgbHex(slot.bg)}): ${ratio.toFixed(2)}:1`).toBeGreaterThanOrEqual(4.5)
    }
  })

  it('浅底槽位（darkText=true）用近黑字，对比度 ≥4.5:1', () => {
    for (const slot of courseColorSlots) {
      if (!slot.darkText) continue
      const ratio = contrast(darkHex, oklchToSrgbHex(slot.bg))
      expect(ratio, `slot ${slot.index} (#${oklchToSrgbHex(slot.bg)}): ${ratio.toFixed(2)}:1`).toBeGreaterThanOrEqual(4.5)
    }
  })
})
