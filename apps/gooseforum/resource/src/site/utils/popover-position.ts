export interface PopoverTrigger {
  top: number
  bottom: number
  left: number
  width: number
}

export interface PopoverPanel {
  width: number
  height: number
}

export interface PopoverViewport {
  width: number
  height: number
}

export interface PopoverPlacement {
  top: number
  left: number
  width: number
}

export interface PopoverPlacementOptions {
  trigger: PopoverTrigger
  panel: PopoverPanel
  viewport: PopoverViewport
  /** 触发元素与面板之间的间距（px），默认 6 */
  gap?: number
  /** 面板与视口边缘的最小边距（px），默认 8 */
  padding?: number
  /** 面板最大宽度（px）；触发元素超宽时收窄，默认不限 */
  maxWidth?: number
}

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max)
}

/**
 * 弹层定位纯函数：默认在触发元素下方展开；下方空间不足时向上翻转；
 * 上下都不足时钳制在视口内；水平方向保证面板完整可见，超宽时收窄。
 * 返回视口坐标（top/left/width），供 position: fixed 使用。
 */
export function computePopoverPlacement(options: PopoverPlacementOptions): PopoverPlacement {
  const { trigger, panel, viewport } = options
  const gap = options.gap ?? 6
  const padding = options.padding ?? 8
  const maxWidth = options.maxWidth ?? Number.POSITIVE_INFINITY

  // 水平：面板不宽于触发元素 / maxWidth / 视口（扣除两侧边距）；超宽时收窄
  const width = clamp(Math.min(trigger.width, maxWidth, viewport.width - padding * 2), 0, viewport.width - padding * 2)
  let left = trigger.left
  left = clamp(left, padding, viewport.width - width - padding)

  // 垂直：优先向下展开，下方放不下则向上翻转，再不行则向下钳制保底
  const below = trigger.bottom + gap
  const above = trigger.top - gap - panel.height
  let top: number
  if (below + panel.height <= viewport.height - padding) {
    top = below
  } else if (above >= padding) {
    top = above
  } else {
    top = clamp(below, padding, viewport.height - panel.height - padding)
  }

  return { top, left, width }
}
