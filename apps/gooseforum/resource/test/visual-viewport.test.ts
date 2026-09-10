import { describe, expect, test } from 'vitest'
import { visualViewportScrollDelta } from '../src/runtime/visual-viewport'

describe('visualViewportScrollDelta', () => {
  test('moves content up when the keyboard covers the target', () => {
    expect(visualViewportScrollDelta(
      { top: 660, bottom: 684 },
      { offsetTop: 0, height: 500 },
      72,
      24,
    )).toBe(208)
  })

  test('moves content down when the target is behind the sticky header', () => {
    expect(visualViewportScrollDelta(
      { top: 40, bottom: 64 },
      { offsetTop: 0, height: 500 },
      72,
      24,
    )).toBe(-32)
  })

  test('accounts for a shifted visual viewport', () => {
    expect(visualViewportScrollDelta(
      { top: 430, bottom: 454 },
      { offsetTop: 80, height: 400 },
      72,
      24,
    )).toBe(0)
  })

  test('does not scroll when the target is already visible', () => {
    expect(visualViewportScrollDelta(
      { top: 180, bottom: 204 },
      { offsetTop: 0, height: 500 },
      72,
      24,
    )).toBe(0)
  })
})
