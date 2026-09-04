/**
 * 图片轮播/灯箱共用的横向滑动判定。
 * TopicImageGallery 与 MarkdownImageViewer 原先各自内联同一套
 * 「阈值 + 轴向比例 + 时限」判定，这里收敛为单一契约，避免两处漂移。
 */

export type HorizontalSwipe = 'left' | 'right' | 'none'

/** 灯箱翻页共用阈值：横向位移须超过该值（px）且明显大于纵向位移 */
export const LIGHTBOX_SWIPE_THRESHOLD_PX = 40

/** 轮播主视窗翻页阈值（面积更大，允许更小的滑动距离） */
export const SLIDE_SWIPE_THRESHOLD_PX = 36

const AXIS_RATIO = 1.4
const MAX_DURATION_MS = 450

/**
 * 判定一次触摸是否为明确的横向滑动。
 * 纵向滚动需要保留给页面，因此横向位移不仅要达到阈值，还必须明显大于纵向位移。
 */
export function decideHorizontalSwipe(
  startX: number,
  startY: number,
  startTime: number,
  endTouch: Touch,
  thresholdPx = LIGHTBOX_SWIPE_THRESHOLD_PX,
): HorizontalSwipe {
  const deltaX = endTouch.clientX - startX
  const deltaY = endTouch.clientY - startY
  const deltaTime = Date.now() - startTime

  if (Math.abs(deltaX) > thresholdPx && Math.abs(deltaX) > Math.abs(deltaY) * AXIS_RATIO && deltaTime < MAX_DURATION_MS) {
    return deltaX < 0 ? 'left' : 'right'
  }
  return 'none'
}
