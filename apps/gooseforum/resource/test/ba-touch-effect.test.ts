import { describe, expect, test } from 'vitest'
import { shouldEnableBaTouchEffect, shouldKeepTrailAlive } from '../src/runtime/ba-touch-effect'

describe('shouldEnableBaTouchEffect', () => {
  test('enables only when click animation is on and reduced motion is off', () => {
    expect(shouldEnableBaTouchEffect({ clickAnimationEnabled: true, reducedMotion: false })).toBe(true)
    expect(shouldEnableBaTouchEffect({ clickAnimationEnabled: false, reducedMotion: false })).toBe(false)
    expect(shouldEnableBaTouchEffect({ clickAnimationEnabled: true, reducedMotion: true })).toBe(false)
    expect(shouldEnableBaTouchEffect({ clickAnimationEnabled: false, reducedMotion: true })).toBe(false)
  })
})

describe('shouldKeepTrailAlive', () => {
  test('mouse and pen trails require a pressed button', () => {
    expect(shouldKeepTrailAlive({ pointerType: 'mouse', buttons: 1 })).toBe(true)
    expect(shouldKeepTrailAlive({ pointerType: 'mouse', buttons: 0 })).toBe(false)
    expect(shouldKeepTrailAlive({ pointerType: 'pen', buttons: 1 })).toBe(true)
    expect(shouldKeepTrailAlive({ pointerType: 'pen', buttons: 0 })).toBe(false)
  })

  test('touch trails stay alive without button state', () => {
    expect(shouldKeepTrailAlive({ pointerType: 'touch', buttons: 0 })).toBe(true)
  })
})
