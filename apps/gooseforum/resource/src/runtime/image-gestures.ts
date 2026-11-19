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

/** 灯箱缩放范围：1 = 适应屏幕，最大 4 倍 */
export const LIGHTBOX_MIN_SCALE = 1
export const LIGHTBOX_MAX_SCALE = 4

/** 图片相对舞台中心的位移（px）与缩放倍数 */
export interface ZoomView {
  scale: number
  x: number
  y: number
}

export interface ZoomSize {
  width: number
  height: number
}

export const IDENTITY_ZOOM: ZoomView = { scale: 1, x: 0, y: 0 }

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

/** 限制倍数，并让放大后的图片边缘不离开舞台（图片比舞台小的方向保持居中） */
export function clampZoomView(view: ZoomView, content: ZoomSize, stage: ZoomSize): ZoomView {
  const scale = clamp(view.scale, LIGHTBOX_MIN_SCALE, LIGHTBOX_MAX_SCALE)
  if (scale === LIGHTBOX_MIN_SCALE) return IDENTITY_ZOOM
  const maxX = Math.max(0, (content.width * scale - stage.width) / 2)
  const maxY = Math.max(0, (content.height * scale - stage.height) / 2)
  return { scale, x: maxX ? clamp(view.x, -maxX, maxX) : 0, y: maxY ? clamp(view.y, -maxY, maxY) : 0 }
}

/** 以 point（相对舞台中心）为不动点缩放：指针 / 双指中点下的那块图片停在原处 */
export function zoomViewAt(view: ZoomView, nextScale: number, point: { x: number; y: number }, content: ZoomSize, stage: ZoomSize): ZoomView {
  const scale = clamp(nextScale, LIGHTBOX_MIN_SCALE, LIGHTBOX_MAX_SCALE)
  const ratio = scale / view.scale
  return clampZoomView({ scale, x: point.x - (point.x - view.x) * ratio, y: point.y - (point.y - view.y) * ratio }, content, stage)
}
