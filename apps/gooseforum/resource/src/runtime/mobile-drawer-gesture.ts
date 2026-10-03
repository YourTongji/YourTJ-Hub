export const mobileDrawerViewportQuery = '(max-width: 1023px)'
export const mobileDrawerCoarsePointerQuery = '(pointer: coarse)'
export const mobileDrawerGestureHintKey = 'goose:mobile-drawer-gesture-hint:v1'

const edgeGuard = 24
const intentDistance = 12
const triggerDistance = 72
const flingDistance = 36
const flingVelocity = 0.45
const axisRatio = 1.35
const maxDuration = 600

export interface DrawerSwipeState {
  pointerId: number
  startX: number
  startY: number
  startedAt: number
}

export interface DrawerSwipePoint {
  clientX: number
  clientY: number
  timeStamp: number
}

export type DrawerSwipeDirection = 'left' | 'right'
export type DrawerSwipeDecision = 'pending' | 'tracking' | 'cancel' | 'trigger'

let hintSeenInMemory = false

export function drawerSwipeIgnoredTarget(target: EventTarget | null): boolean {
  if (typeof Element === 'undefined' || !(target instanceof Element)) return false
  if (
    target.closest(
      'input, textarea, select, video, audio, canvas, [contenteditable]:not([contenteditable="false"]), [role="slider"], [draggable="true"], [data-drawer-swipe-ignore]',
    )
  ) {
    return true
  }

  for (let element: Element | null = target; element; element = element.parentElement) {
    if (!(element instanceof HTMLElement)) continue
    const style = window.getComputedStyle(element)
    if (
      (style.overflowX === 'auto' || style.overflowX === 'scroll') &&
      element.scrollWidth > element.clientWidth + 1
    ) {
      return true
    }
  }
  return false
}

export function canStartDrawerSwipe(
  clientX: number,
  target: EventTarget | null,
  options: { protectViewportEdges?: boolean } = {},
): boolean {
  if (drawerSwipeIgnoredTarget(target)) return false
  if (options.protectViewportEdges === false) return true
  return clientX > edgeGuard && clientX < window.innerWidth - edgeGuard
}

export function drawerSwipeDecision(
  state: DrawerSwipeState,
  event: DrawerSwipePoint,
  direction: DrawerSwipeDirection,
): DrawerSwipeDecision {
  const dx = event.clientX - state.startX
  const dy = event.clientY - state.startY
  const elapsed = Math.max(1, event.timeStamp - state.startedAt)
  const horizontal = Math.abs(dx)
  const vertical = Math.abs(dy)
  const signedDistance = direction === 'right' ? dx : -dx

  if (horizontal < intentDistance && vertical < intentDistance) return 'pending'
  if (event.timeStamp - state.startedAt > maxDuration) return 'cancel'
  if (signedDistance <= 0 || horizontal < vertical * axisRatio) return 'cancel'
  if (signedDistance >= triggerDistance) return 'trigger'
  if (signedDistance >= flingDistance && signedDistance / elapsed >= flingVelocity) return 'trigger'
  return 'tracking'
}

export function shouldOfferDrawerGestureHint(): boolean {
  if (hintSeenInMemory) return false
  try {
    if (window.localStorage.getItem(mobileDrawerGestureHintKey) === '1') {
      hintSeenInMemory = true
      return false
    }
  } catch {
    // Storage may be unavailable in private/restricted contexts; memory still prevents SPA repeats.
  }
  return true
}

export function markDrawerGestureHintSeen(): void {
  hintSeenInMemory = true
  try {
    window.localStorage.setItem(mobileDrawerGestureHintKey, '1')
  } catch {
    // The gesture remains usable when persistence is unavailable.
  }
}

export function hasTouchLikePointer(): boolean {
  return window.matchMedia?.(mobileDrawerCoarsePointerQuery).matches || navigator.maxTouchPoints > 0
}
