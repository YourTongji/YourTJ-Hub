/**
 * 图片相对宽度（MADR 0131）：Markdown 图片地址末尾的 `#w=<预设>` 片段表示按正文宽度的百分比
 * 显示。片段不随请求发送，旧内容、其他 Markdown 渲染器与服务端附件解析都会忽略它；
 * Web 编辑区、预览与发布后正文用 CSS 属性选择器（prose.css）按片段排版，App 解析同一片段。
 * 共享 fixture：packages/api-contract/fixtures/markdown-image-width.json。
 */
export const IMAGE_WIDTH_PRESETS = [25, 33, 50, 66, 75, 100] as const
export type ImageWidth = (typeof IMAGE_WIDTH_PRESETS)[number]

/** 渲染下限（CSS px）：窄屏上小图不至于缩成看不清的缩略图 */
export const IMAGE_MIN_RENDERED_WIDTH = 160

const WIDTH_FRAGMENT_RE = /#w=(\d{2,3})$/

function isPreset(value: number): value is ImageWidth {
  return (IMAGE_WIDTH_PRESETS as readonly number[]).includes(value)
}

/** 地址里的宽度预设；没有或不是预设值时为 null */
export function imageWidthOf(src: string): ImageWidth | null {
  const match = WIDTH_FRAGMENT_RE.exec(src)
  const value = match ? Number(match[1]) : NaN
  return isPreset(value) ? value : null
}

/** 能否写入宽度：已带其他片段（如 SVG 精灵 `#icon`）的地址不改动 */
export function canSetImageWidth(src: string): boolean {
  const hash = src.indexOf('#')
  return hash < 0 || /^#w=\d*$/.test(src.slice(hash))
}

/** 写入或清除宽度（null = 按图片原始尺寸自适应） */
export function withImageWidth(src: string, width: ImageWidth | null): string {
  if (!canSetImageWidth(src)) return src
  const hash = src.indexOf('#')
  const base = hash < 0 ? src : src.slice(0, hash)
  return width ? `${base}#w=${width}` : base
}

/** 拖拽得到的宽度比例吸附到最近的预设（≥0.9 视为铺满） */
export function snapImageWidth(ratio: number): ImageWidth {
  if (ratio >= 0.9) return 100
  let best: ImageWidth = IMAGE_WIDTH_PRESETS[0]
  for (const preset of IMAGE_WIDTH_PRESETS) {
    if (Math.abs(preset / 100 - ratio) < Math.abs(best / 100 - ratio)) best = preset
  }
  return best
}
