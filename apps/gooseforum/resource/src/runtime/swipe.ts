export type SwipeDirection = 'left' | 'right' | 'none'

export interface SwipeDecision {
  direction: SwipeDirection
  /** 位移很小，应该按普通点击处理，而不是滑动或取消操作。 */
  isTap: boolean
}

const TAP_SLOP_PX = 10
const SWIPE_THRESHOLD_PX = 48
const HORIZONTAL_AXIS_RATIO = 1.25

/**
 * 根据触摸起止点判断是否为明确的横向滑动。
 *
 * 纵向滚动需要保留给页面，因此横向位移不仅要达到阈值，还必须明显大于
 * 纵向位移。将判断做成纯函数，方便在不依赖浏览器 DOM 的情况下覆盖边界。
 */
export function decideSwipe(
  startX: number,
  startY: number,
  endX: number,
  endY: number,
): SwipeDecision {
  const horizontalDistance = endX - startX
  const verticalDistance = endY - startY
  const absoluteHorizontalDistance = Math.abs(horizontalDistance)
  const absoluteVerticalDistance = Math.abs(verticalDistance)

  if (Math.hypot(horizontalDistance, verticalDistance) <= TAP_SLOP_PX) {
    return { direction: 'none', isTap: true }
  }

  const isHorizontalSwipe = absoluteHorizontalDistance >= SWIPE_THRESHOLD_PX
    && absoluteHorizontalDistance >= absoluteVerticalDistance * HORIZONTAL_AXIS_RATIO

  if (!isHorizontalSwipe) {
    return { direction: 'none', isTap: false }
  }

  return {
    direction: horizontalDistance < 0 ? 'left' : 'right',
    isTap: false,
  }
}
