/**
 * 贴纸放大预览的落点：预览挂在 body 上（不受面板/编辑器容器裁切），
 * 只放在面板外侧，绝不盖住网格和正在指向的那格。优先面板的行尾侧、
 * 再行首侧（随书写方向镜像），横向放不下（窄屏）时放到面板上方或下方，
 * 都放不下才退到指向格正上方。坐标是视口物理坐标，最终夹在视口内。
 */

export interface PlacementBox {
  left: number
  top: number
  right: number
  bottom: number
}

export interface PreviewPlacement {
  left: number
  top: number
  /** 缩放进场的原点：落在朝向指向格的那条边上 */
  origin: string
  side: 'end' | 'start' | 'above' | 'below' | 'cell'
}

const GAP = 8
const MARGIN = 8

export function placeStickerPreview(
  cell: PlacementBox,
  panel: PlacementBox,
  size: { width: number, height: number },
  viewport: { width: number, height: number },
  rtl = false,
): PreviewPlacement {
  const { width, height } = size
  const clamp = (value: number, max: number) => Math.min(Math.max(value, MARGIN), Math.max(MARGIN, max - MARGIN))
  const cellX = (cell.left + cell.right) / 2
  const cellY = (cell.top + cell.bottom) / 2
  const sideTop = clamp(cellY - height / 2, viewport.height - height)
  const stackLeft = clamp(cellX - width / 2, viewport.width - width)
  const originY = `${Math.round(cellY - sideTop)}px`
  const originX = `${Math.round(cellX - stackLeft)}px`

  const toRight = { left: panel.right + GAP, top: sideTop, origin: `0 ${originY}` }
  const toLeft = { left: panel.left - GAP - width, top: sideTop, origin: `${width}px ${originY}` }
  const fitsRight = toRight.left + width <= viewport.width - MARGIN
  const fitsLeft = toLeft.left >= MARGIN
  const [end, endFits, start, startFits] = rtl
    ? [toLeft, fitsLeft, toRight, fitsRight]
    : [toRight, fitsRight, toLeft, fitsLeft]
  if (endFits) return { ...end, side: 'end' }
  if (startFits) return { ...start, side: 'start' }

  const aboveTop = panel.top - GAP - height
  if (aboveTop >= MARGIN) return { left: stackLeft, top: aboveTop, origin: `${originX} ${height}px`, side: 'above' }
  const belowTop = panel.bottom + GAP
  if (belowTop + height <= viewport.height - MARGIN) return { left: stackLeft, top: belowTop, origin: `${originX} 0`, side: 'below' }

  return { left: stackLeft, top: clamp(cell.top - GAP - height, viewport.height - height), origin: `${originX} ${height}px`, side: 'cell' }
}
