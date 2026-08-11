import { describe, expect, test } from 'vitest'
import { computePopoverPlacement } from '../src/site/utils/popover-position'

// 通用触发元素：宽 240、位于 (100, 200)
const TRIGGER = { top: 200, bottom: 244, left: 100, width: 240 }
const PANEL = { width: 240, height: 256 }
const VIEWPORT = { width: 390, height: 844 }

describe('computePopoverPlacement', () => {
  test('默认在触发元素下方展开，间距为 gap', () => {
    expect(computePopoverPlacement({ trigger: TRIGGER, panel: PANEL, viewport: VIEWPORT, gap: 6 })).toEqual({
      top: 250, // 244 + 6
      left: 100,
      width: 240,
    })
  })

  test('下方空间不足时向上翻转', () => {
    // 触发元素贴近屏底，下方放不下 256px 面板，改为在触发元素上方展开
    const nearBottom = { top: 600, bottom: 644, left: 100, width: 240 }
    expect(computePopoverPlacement({ trigger: nearBottom, panel: PANEL, viewport: VIEWPORT, gap: 6 })).toEqual({
      top: 338, // 600 - 6(gap) - 256(panelHeight)
      left: 100,
      width: 240,
    })
  })

  test('上下空间都不足时向下钳制在视口内', () => {
    // 视口较矮：下方放不下、上方也不够，钳制在「保底边距」与「视口下缘」之间
    const shortViewport = { width: 390, height: 300 }
    const result = computePopoverPlacement({ trigger: TRIGGER, panel: PANEL, viewport: shortViewport, gap: 6 })
    expect(result).toEqual({
      top: 36, // clamp(244+6, 8, 300-256-8)
      left: 100,
      width: 240,
    })
    expect(result.top).toBeGreaterThanOrEqual(8)
    expect(result.top + PANEL.height).toBeLessThanOrEqual(shortViewport.height)
  })

  test('水平方向钳制，触发元素贴近右缘时面板不越界', () => {
    const nearRight = { top: 200, bottom: 244, left: 370, width: 40 }
    const result = computePopoverPlacement({ trigger: nearRight, panel: { width: 240, height: 256 }, viewport: VIEWPORT, gap: 6 })
    expect(result.left).toBeLessThanOrEqual(VIEWPORT.width - 8) // 不超出右缘
    expect(result.left).toBeGreaterThanOrEqual(8)
    expect(result.left + result.width).toBeLessThanOrEqual(VIEWPORT.width - 8)
  })

  test('触发元素超宽时面板收窄到视口内', () => {
    const wide = { top: 200, bottom: 244, left: -100, width: 590 } // 590 > 390 视口
    const result = computePopoverPlacement({ trigger: wide, panel: { width: 590, height: 256 }, viewport: VIEWPORT, gap: 6 })
    expect(result.width).toBeLessThanOrEqual(VIEWPORT.width - 16)
    expect(result.left).toBeGreaterThanOrEqual(8)
    expect(result.left + result.width).toBeLessThanOrEqual(VIEWPORT.width - 8)
  })
})
