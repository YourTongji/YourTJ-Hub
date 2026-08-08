import { describe, expect, test } from 'vitest'
import { computeWaveSize, RIPPLE_SELECTOR, shouldTriggerRipple } from '../src/runtime/click-ripple'

describe('shouldTriggerRipple', () => {
  test('triggers only when enabled, motion allowed, and primary button', () => {
    expect(shouldTriggerRipple({ enabled: true, reducedMotion: false, button: 0 })).toBe(true)
    expect(shouldTriggerRipple({ enabled: false, reducedMotion: false, button: 0 })).toBe(false)
    expect(shouldTriggerRipple({ enabled: true, reducedMotion: true, button: 0 })).toBe(false)
    expect(shouldTriggerRipple({ enabled: true, reducedMotion: false, button: 2 })).toBe(false)
    expect(shouldTriggerRipple({ enabled: true, reducedMotion: false, button: 1 })).toBe(false)
  })
})

describe('computeWaveSize', () => {
  test('covers the element diagonal from its center', () => {
    expect(computeWaveSize(3, 4)).toBe(10)
    expect(computeWaveSize(0, 0)).toBe(0)
    expect(computeWaveSize(100, 100)).toBeCloseTo(282.84, 1)
  })
})

describe('RIPPLE_SELECTOR', () => {
  test('covers interactive tags, roles, and gf-* classes', () => {
    const parts = [
      'button', 'a[href]', '[role="button"]', '[role="menuitem"]',
      '[role="switch"]', '[role="tab"]', '[role="checkbox"]',
      '.gf-menu-item', '.gf-icon-button', '.gf-tab', '.gf-segmented-item',
      '[data-ripple]',
    ]
    for (const part of parts) {
      expect(RIPPLE_SELECTOR).toContain(part)
    }
  })
})
