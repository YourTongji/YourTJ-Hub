/**
 * 编辑器浮层（选区工具条、图片工具条）定位：position: fixed，默认放在锚点上方、放不下翻到
 * 下方，横向居中并夹在边界内。浮层挂在最近的对话框（或 body）里：祖先带 transform 时
 * fixed 以该祖先为包含块，这里按实测偏差修正，所以弹层编辑器里也不会错位或被裁切。
 */
export type FloatingSide = 'top' | 'bottom'

export interface FloatingRect {
  top: number
  right: number
  bottom: number
  left: number
}

export interface FloatingPlacement {
  side?: FloatingSide
  /** 浮层与锚点的间距 */
  gap?: number
  /** 浮层与边界的最小距离 */
  margin?: number
}

export function viewportRect(): FloatingRect {
  return { top: 0, left: 0, right: window.innerWidth, bottom: window.innerHeight }
}

export function intersectRect(a: FloatingRect, b: FloatingRect): FloatingRect {
  return { top: Math.max(a.top, b.top), left: Math.max(a.left, b.left), right: Math.min(a.right, b.right), bottom: Math.min(a.bottom, b.bottom) }
}

/** 锚点是否还在可见边界内（滚出编辑区后浮层应隐藏，而不是悬在别的内容上） */
export function rectVisibleIn(anchor: FloatingRect, boundary: FloatingRect): boolean {
  return anchor.bottom > boundary.top && anchor.top < boundary.bottom && anchor.right > boundary.left && anchor.left < boundary.right
}

/** 计算视口坐标（纯函数，便于测试）：返回左上角与最终所在侧 */
export function computeFloatingPosition(
  size: { width: number; height: number },
  anchor: FloatingRect,
  boundary: FloatingRect,
  { side = 'top', gap = 8, margin = 8 }: FloatingPlacement = {},
): { left: number; top: number; side: FloatingSide } {
  const centerX = (anchor.left + anchor.right) / 2
  const minLeft = boundary.left + margin
  const maxLeft = Math.max(minLeft, boundary.right - margin - size.width)
  const left = Math.min(Math.max(centerX - size.width / 2, minLeft), maxLeft)
  const above = anchor.top - gap - size.height
  const below = anchor.bottom + gap
  const fitsAbove = above >= boundary.top + margin
  const fitsBelow = below + size.height <= boundary.bottom - margin
  let finalSide: FloatingSide = side
  if (side === 'top' && !fitsAbove && fitsBelow) finalSide = 'bottom'
  if (side === 'bottom' && !fitsBelow && fitsAbove) finalSide = 'top'
  let top = finalSide === 'top' ? above : below
  // 两侧都放不下（锚点很高，例如大图）：贴在边界内侧，压在锚点上
  top = Math.min(Math.max(top, boundary.top + margin), Math.max(boundary.top + margin, boundary.bottom - margin - size.height))
  return { left, top, side: finalSide }
}

/** 应用定位并修正 transform 祖先造成的包含块偏差 */
export function placeFloating(el: HTMLElement, anchor: FloatingRect, boundary: FloatingRect, placement?: FloatingPlacement): FloatingSide {
  const { left, top, side } = computeFloatingPosition({ width: el.offsetWidth, height: el.offsetHeight }, anchor, boundary, placement)
  el.style.left = `${left}px`
  el.style.top = `${top}px`
  const rect = el.getBoundingClientRect()
  const dx = rect.left - left
  const dy = rect.top - top
  if (Math.abs(dx) > 0.5) el.style.left = `${left - dx}px`
  if (Math.abs(dy) > 0.5) el.style.top = `${top - dy}px`
  el.dataset.side = side
  return side
}

/** 浮层挂载点：编辑器在对话框里时挂在对话框内（挂到 body 会被模态焦点陷阱当成外部点击） */
export function floatingHost(from: Element | null): HTMLElement {
  return (from?.closest<HTMLElement>('[role="dialog"]') ?? document.body)
}

/** 覆盖层（图片选中框）贴合锚点矩形，同样修正 transform 祖先的偏差 */
export function placeOverlay(el: HTMLElement, rect: FloatingRect): void {
  el.style.left = `${rect.left}px`
  el.style.top = `${rect.top}px`
  el.style.width = `${rect.right - rect.left}px`
  el.style.height = `${rect.bottom - rect.top}px`
  const actual = el.getBoundingClientRect()
  const dx = actual.left - rect.left
  const dy = actual.top - rect.top
  if (Math.abs(dx) > 0.5) el.style.left = `${rect.left - dx}px`
  if (Math.abs(dy) > 0.5) el.style.top = `${rect.top - dy}px`
}
