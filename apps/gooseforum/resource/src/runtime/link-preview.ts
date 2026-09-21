import type { LinkPreview } from '@gooseforum/client'
import { markdownPreview } from '@/runtime/markdown'
import { safeUrl } from '@/runtime/safe-url'

/**
 * 每篇正文最多接管几张卡片。取 5 是为了对齐接口边界：`LinkPreviewResolveRequest`
 * 的 `maxItems` 与控制器校验都是 5，所以一篇帖子的候选恰好能一次请求拿完。
 */
export const MAX_LINK_PREVIEWS = 5

/**
 * 单次 resolve 请求的 URL 上限，与上面契约里的 `maxItems: 5` 一致。去重队列是跨
 * 正文容器共享的，所以一屏内有多个正文时仍可能凑出超过一篇的候选，必须自行分批。
 */
export const LINK_PREVIEW_REQUEST_LIMIT = 5

export interface RenderedLinkCandidate {
  paragraph: HTMLParagraphElement
  anchor: HTMLAnchorElement
  url: string
}

export interface FindCandidateOptions {
  /**
   * 重新扫描一个已经被增强过的根时必须为 true：那时候选段落仍带着上次留下的
   * `data-gf-link-preview` 标记，而被隐藏（`hidden = true`）的也只是显示，节点
   * 还在原处。带上标记一起扫，才能把「同一批节点」和「v-html 换过一批新节点」
   * 区分开，从而决定是原地保留还是重建。
   */
  includeEnhanced?: boolean
}

export function findRenderedLinkCandidates(root: HTMLElement, options: FindCandidateOptions = {}): RenderedLinkCandidate[] {
  const candidates: RenderedLinkCandidate[] = []
  for (const paragraph of root.querySelectorAll<HTMLParagraphElement>('p')) {
    if (candidates.length >= MAX_LINK_PREVIEWS) break
    if (!options.includeEnhanced && paragraph.dataset.gfLinkPreview !== undefined) continue
    if (paragraph.closest('blockquote, li, td, th')) continue
    const elements = Array.from(paragraph.children)
    if (elements.length !== 1 || !(elements[0] instanceof HTMLAnchorElement)) continue
    const anchor = elements[0]
    const href = safeUrl(anchor.getAttribute('href'), 'site-link')
    const label = anchor.textContent?.trim()
    const labelUrl = safeUrl(label, 'external')
    if (!href || !labelUrl || paragraph.textContent?.trim() !== label) continue
    if (new URL(href, document.baseURI).href !== new URL(labelUrl, document.baseURI).href) continue
    if (anchor.querySelector('img')) continue
    candidates.push({ paragraph, anchor, url: href })
  }
  return candidates
}

export function scanMarkdownLinkCandidates(markdown: string): string[] {
  const source = markdown ?? ''
  if (!source.trim()) return []
  const lines = source.replace(/\r\n?/g, '\n').split('\n')
  const tokens = markdownPreview.parse(source, {})
  const urls: string[] = []

  for (let index = 0; index < tokens.length && urls.length < MAX_LINK_PREVIEWS; index += 1) {
    const token = tokens[index]
    const inline = tokens[index + 1]
    if (token.type !== 'paragraph_open' || inline?.type !== 'inline' || !token.map) continue
    const raw = lines.slice(token.map[0], token.map[1]).join('\n').trim()
    const url = safeUrl(raw, 'external')
    if (!url) continue
    const children = inline.children ?? []
    const text = children.filter(child => child.type === 'text').map(child => child.content).join('')
    const links = children.filter(child => child.type === 'link_open')
    if (links.length !== 1 || text !== raw) continue
    urls.push(url)
  }

  return urls
}

export function usableLinkPreview(preview: LinkPreview | undefined): preview is LinkPreview {
  // 校园网卡片的标题可能为空（部署配置没给名字），兜底文案由客户端按语言补，
  // 所以这里对有 campus 标记的卡片只要求有地址。
  if (preview?.status !== 'ready' || !preview.url) return false
  return Boolean(preview.title) || Boolean(preview.campus)
}
