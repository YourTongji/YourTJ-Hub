// @vitest-environment happy-dom
import { beforeEach, describe, expect, test } from 'vitest'
import {
  canStartDrawerSwipe,
  drawerSwipeDecision,
  drawerSwipeIgnoredTarget,
  markDrawerGestureHintSeen,
  mobileDrawerGestureHintKey,
  shouldOfferDrawerGestureHint,
  type DrawerSwipeState,
} from '../src/runtime/mobile-drawer-gesture'

function pointer(
  overrides: Partial<PointerEvent> & Pick<PointerEvent, 'clientX' | 'clientY' | 'timeStamp'>,
): PointerEvent {
  return {
    pointerId: 1,
    pointerType: 'touch',
    isPrimary: true,
    target: document.body,
    cancelable: true,
    ...overrides,
  } as PointerEvent
}

const start: DrawerSwipeState = {
  pointerId: 1,
  startX: 100,
  startY: 100,
  startedAt: 100,
}

describe('mobile drawer swipe recognition', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
  })

  test('right swipe triggers by distance or fast fling, but not by a slow short drag', () => {
    expect(drawerSwipeDecision(start, pointer({ clientX: 180, clientY: 106, timeStamp: 300 }), 'right')).toBe('trigger')
    expect(drawerSwipeDecision(start, pointer({ clientX: 140, clientY: 104, timeStamp: 150 }), 'right')).toBe('trigger')
    expect(drawerSwipeDecision(start, pointer({ clientX: 140, clientY: 104, timeStamp: 400 }), 'right')).toBe('tracking')
  })

  test('vertical, opposite and stale gestures are cancelled', () => {
    expect(drawerSwipeDecision(start, pointer({ clientX: 125, clientY: 160, timeStamp: 220 }), 'right')).toBe('cancel')
    expect(drawerSwipeDecision(start, pointer({ clientX: 70, clientY: 101, timeStamp: 180 }), 'right')).toBe('cancel')
    expect(drawerSwipeDecision(start, pointer({ clientX: 180, clientY: 102, timeStamp: 750 }), 'right')).toBe('cancel')
  })

  test('left swipe uses the same recognizer for drawer close', () => {
    expect(drawerSwipeDecision(start, pointer({ clientX: 20, clientY: 96, timeStamp: 260 }), 'left')).toBe('trigger')
    expect(drawerSwipeDecision(start, pointer({ clientX: 175, clientY: 98, timeStamp: 240 }), 'left')).toBe('cancel')
  })

  test('viewport edge guard keeps browser history gestures available', () => {
    expect(canStartDrawerSwipe(20, document.body)).toBe(false)
    expect(canStartDrawerSwipe(100, document.body)).toBe(true)
    expect(canStartDrawerSwipe(370, document.body)).toBe(false)
  })

  test('editing, explicit opt-out and horizontal scrollers take precedence', () => {
    const input = document.createElement('input')
    const ignored = document.createElement('div')
    ignored.dataset.drawerSwipeIgnore = ''
    const scroller = document.createElement('div')
    scroller.style.overflowX = 'auto'
    Object.defineProperty(scroller, 'clientWidth', { configurable: true, value: 100 })
    Object.defineProperty(scroller, 'scrollWidth', { configurable: true, value: 200 })
    const plain = document.createElement('div')
    document.body.append(input, ignored, scroller, plain)

    expect(drawerSwipeIgnoredTarget(input)).toBe(true)
    expect(drawerSwipeIgnoredTarget(ignored)).toBe(true)
    expect(drawerSwipeIgnoredTarget(scroller)).toBe(true)
    expect(drawerSwipeIgnoredTarget(plain)).toBe(false)

    input.remove()
    ignored.remove()
    scroller.remove()
    plain.remove()
  })

  test('gesture hint is persisted after it is offered', () => {
    window.localStorage.removeItem(mobileDrawerGestureHintKey)
    expect(shouldOfferDrawerGestureHint()).toBe(true)
    markDrawerGestureHintSeen()
    expect(window.localStorage.getItem(mobileDrawerGestureHintKey)).toBe('1')
    expect(shouldOfferDrawerGestureHint()).toBe(false)
  })
})
