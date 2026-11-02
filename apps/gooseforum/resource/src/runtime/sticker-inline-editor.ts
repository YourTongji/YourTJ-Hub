/**
 * 编辑器内贴纸所见即所得（MADR 0106 补充）：wysiwyg 编辑区里完整且可解析的
 * `[:sticker:name:]` 文本包进不可编辑的内联块，显示与正文渲染同尺寸的贴纸图。
 *
 * 块内仍保留 token 原文：Lute 的 VditorDOM2Md 只读文本，getValue()/草稿/提交的
 * Markdown 与未装饰时逐字一致；Vditor 每次输入按块重渲染（SpinVditorDOM）会把块
 * 还原为纯文本，由宿主的 MutationObserver 在绘制前重新装饰。未知/停用 token 与
 * 代码、链接、公式里的 token 保持原文，与服务端渲染规则一致。
 */

export const STICKER_INLINE_CLASS = 'gf-sticker-inline'

const STICKER_TOKEN_RE = /\[:sticker:([^\s:[\]]{1,64}):]/g

/** 服务端只在正文文本节点里展开 token：代码、链接目标/文字、原始 HTML、公式一律跳过。 */
const SKIP_SELECTOR = [
  'pre',
  'code',
  'a',
  '.vditor-wysiwyg__block',
  '.vditor-wysiwyg__preview',
  '[data-type="html-inline"]',
  '[data-type="html-entity"]',
  '[data-type="math-inline"]',
  '[data-type="backslash"]',
  '[data-type="link-ref"]',
  '[data-type="footnotes-ref"]',
].join(', ')

export interface StickerInlineAsset {
  url: string
  label: string
}

function cssUrl(url: string): string {
  return `url(${JSON.stringify(url)})`
}

function wrapToken(text: Text, name: string, asset: StickerInlineAsset): HTMLSpanElement {
  const span = document.createElement('span')
  span.className = STICKER_INLINE_CLASS
  span.contentEditable = 'false'
  span.draggable = false
  span.spellcheck = false
  span.dataset.sticker = name
  span.setAttribute('role', 'img')
  span.setAttribute('aria-label', asset.label)
  span.style.backgroundImage = cssUrl(asset.url)

  // 光标落在 token 文本里（刚输入完闭合的 `]`）时移到图片之后：包裹会把活动 Range
  // 退回到父节点里 token 原来的位置（图片之前）。
  const selection = window.getSelection()
  const range = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null
  const caretInside = Boolean(range?.collapsed && range.startContainer === text && range.startOffset > 0)

  text.parentNode?.insertBefore(span, text)
  span.appendChild(text)
  if (caretInside && selection) {
    const after = document.createRange()
    after.setStartAfter(span)
    after.collapse(true)
    selection.removeAllRanges()
    selection.addRange(after)
  }
  return span
}

/**
 * 装饰 root 下所有可解析的完整 token，返回新包裹的个数。
 * 已装饰但贴纸不再可解析（或图片地址变化）的块就地还原/更新。
 */
export function decorateStickerTokens(
  root: HTMLElement,
  assetFor: (name: string) => StickerInlineAsset | undefined,
): number {
  for (const span of root.querySelectorAll<HTMLElement>(`.${STICKER_INLINE_CLASS}`)) {
    const asset = assetFor(span.dataset.sticker ?? '')
    if (asset) {
      const background = cssUrl(asset.url)
      if (span.style.backgroundImage !== background) span.style.backgroundImage = background
      continue
    }
    span.replaceWith(...span.childNodes)
  }

  const candidates: Text[] = []
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    const text = node as Text
    if (!text.data.includes('[:sticker:')) continue
    const parent = text.parentElement
    if (!parent || parent.closest(`.${STICKER_INLINE_CLASS}`)) continue
    const skipped = parent.closest(SKIP_SELECTOR)
    if (skipped && skipped !== root && root.contains(skipped)) continue
    candidates.push(text)
  }

  let wrapped = 0
  for (const candidate of candidates) {
    let text: Text | null = candidate
    while (text) {
      STICKER_TOKEN_RE.lastIndex = 0
      let match: RegExpExecArray | null = null
      let asset: StickerInlineAsset | undefined
      for (let next = STICKER_TOKEN_RE.exec(text.data); next; next = STICKER_TOKEN_RE.exec(text.data)) {
        // 与服务端一致：奇数个反斜杠转义的 token 保持原文
        let slashes = 0
        for (let i = next.index - 1; i >= 0 && text.data[i] === '\\'; i--) slashes++
        if (slashes % 2 !== 0) continue
        asset = assetFor(next[1])
        if (asset) {
          match = next
          break
        }
      }
      if (!match || !asset) break
      const token: Text = match.index > 0 ? text.splitText(match.index) : text
      const rest: Text | null = token.data.length > match[0].length ? token.splitText(match[0].length) : null
      wrapToken(token, match[1], asset)
      wrapped++
      text = rest
    }
  }
  return wrapped
}

/** 光标/选区端点所在的贴纸内联块（端点在块内文本里或正好是块本身）。 */
export function stickerInlineAt(node: Node): HTMLElement | null {
  const element = node instanceof Element ? node : node.parentElement
  return element?.closest<HTMLElement>(`.${STICKER_INLINE_CLASS}`) ?? null
}
