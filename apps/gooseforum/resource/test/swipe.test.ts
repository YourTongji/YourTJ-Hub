import { describe, expect, test } from 'vitest'
import { decideSwipe } from '../src/runtime/swipe'

describe('decideSwipe', () => {
  test('treats a small movement as a tap', () => {
    expect(decideSwipe(100, 100, 106, 104)).toEqual({
      direction: 'none',
      isTap: true,
    })
  })

  test('detects a left swipe', () => {
    expect(decideSwipe(160, 200, 96, 202)).toEqual({
      direction: 'left',
      isTap: false,
    })
  })

  test('detects a right swipe', () => {
    expect(decideSwipe(96, 200, 160, 202)).toEqual({
      direction: 'right',
      isTap: false,
    })
  })

  test('ignores movement below the swipe threshold', () => {
    expect(decideSwipe(100, 100, 147, 100)).toEqual({
      direction: 'none',
      isTap: false,
    })
  })

  test('ignores mostly vertical movement so page scrolling remains available', () => {
    expect(decideSwipe(100, 100, 160, 160)).toEqual({
      direction: 'none',
      isTap: false,
    })
  })

  test('ignores diagonal movement without a clear horizontal axis', () => {
    expect(decideSwipe(100, 100, 160, 150)).toEqual({
      direction: 'none',
      isTap: false,
    })
  })
})
