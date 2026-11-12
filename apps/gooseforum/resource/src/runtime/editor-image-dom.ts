/**
 * 编辑区图片的 DOM 读写（MADR 0131 图片工具条）：所见即所得（wysiwyg）里图片就是 `<img>`；
 * 即时渲染（ir）里图片是带 Markdown 标记的 `.vditor-ir__node[data-type="img"]`，地址与替代
 * 文字存在标记文本中（序列化读的是标记），所以两处都要改。源码模式（sv）不提供图片工具条。
 * 改完由调用方让 Vditor 记撤销栈并触发 input（见 VditorOfficial 的 commitEditorDom）。
 */
export type EditorMode = 'wysiwyg' | 'ir' | 'sv'

const IR_IMAGE_NODE = '.vditor-ir__node[data-type="img"]'

/** 可编辑的正文图片：排除 emoji、贴纸、代码块/图表预览里的渲染图与引用式图片 */
export function editableImage(target: EventTarget | null, editable: HTMLElement): HTMLImageElement | null {
  if (!(target instanceof HTMLImageElement) || !editable.contains(target)) return null
  if (target.matches('.emoji, .twemoji, [data-gf-sticker], .gf-sticker-inline, .gf-sticker-inline *')) return null
  if (target.getAttribute('data-type') === 'link-ref') return null
  // wysiwyg 的编辑根本身是 <pre class="vditor-reset">，只看根以内的祖先
  const hit = target.parentElement?.closest('.vditor-wysiwyg__preview, .vditor-ir__preview, .vditor-wysiwyg__block, pre, .gf-sticker-inline')
  if (hit && hit !== editable && editable.contains(hit)) return null
  return target
}

/** 选区是否落在代码块、公式等渲染块里（这些地方不显示格式条；行内代码仍显示，便于取消）；同样排除编辑根自身 */
export function insideRenderedBlock(node: Node | null, editable: HTMLElement): boolean {
  const element = node instanceof Element ? node : node?.parentElement
  const hit = element?.closest('pre, .vditor-wysiwyg__block, .vditor-ir__node[data-type="code-block"], .vditor-ir__node[data-type="math-block"], .katex')
  return Boolean(hit && hit !== editable && editable.contains(hit))
}

export function editorImages(editable: HTMLElement): HTMLImageElement[] {
  return Array.from(editable.querySelectorAll('img')).filter((img) => editableImage(img, editable) === img)
}

function irNode(img: HTMLImageElement): HTMLElement | null {
  return img.closest<HTMLElement>(IR_IMAGE_NODE)
}

/** 图片在 Markdown 里的地址：ir 模式以标记文本为准（img.src 可能已被浏览器补全为绝对地址） */
export function editorImageSource(img: HTMLImageElement, mode: EditorMode): string {
  if (mode === 'ir') {
    const marker = irNode(img)?.querySelector('.vditor-ir__marker--link')
    if (marker?.textContent) return marker.textContent
  }
  return img.getAttribute('src') ?? ''
}

export function setEditorImageSource(img: HTMLImageElement, src: string, mode: EditorMode): void {
  img.setAttribute('src', src)
  if (mode !== 'ir') return
  const marker = irNode(img)?.querySelector('.vditor-ir__marker--link')
  if (marker) marker.textContent = src
}

export function setEditorImageAlt(img: HTMLImageElement, alt: string, mode: EditorMode): void {
  img.setAttribute('alt', alt)
  if (mode !== 'ir') return
  // ir 标记依次为 `!` `[` 替代文字 `]` `(` 地址 `)`；替代文字为空时没有中间那一段
  const brackets = irNode(img)?.querySelectorAll<HTMLElement>(':scope > .vditor-ir__marker--bracket')
  if (!brackets?.length) return
  if (brackets.length >= 3) {
    brackets[1].textContent = alt
  } else if (alt) {
    const label = document.createElement('span')
    label.className = 'vditor-ir__marker vditor-ir__marker--bracket'
    label.textContent = alt
    brackets[0].after(label)
  }
}

function isBlankBlock(element: Element): boolean {
  return !element.querySelector('img, video, iframe') && (element.textContent ?? '').replace(/[​\s]/g, '') === ''
}

/** 删除图片；所在段落因此变空时一并删除（保留编辑区至少一个段落由 Vditor 自行处理） */
export function removeEditorImage(img: HTMLImageElement, mode: EditorMode): void {
  const node = mode === 'ir' ? irNode(img) ?? img : img
  const block = node.parentElement?.closest('p, li, blockquote') ?? null
  node.remove()
  if (block && block.tagName === 'P' && isBlankBlock(block) && block.parentElement?.children.length && block.parentElement.children.length > 1) {
    block.remove()
  }
}
