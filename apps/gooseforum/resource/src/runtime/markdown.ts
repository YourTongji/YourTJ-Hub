import MarkdownIt from 'markdown-it'
import anchor from 'markdown-it-anchor'
import taskLists from 'markdown-it-task-lists'
import { BLANK_LINE_HTML, blankLineMarker, markBlankLines } from './blank-lines'
import { protectMathSegments, restoreMathSegments } from './math-protection'

export const markdownPreview = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  breaks: false,
})
  .use(anchor, {
    slugify: (value: string) => value.trim().toLowerCase().replace(/\s+/g, '-'),
  })
  .use(taskLists, { enabled: true })

markdownPreview.renderer.rules.s_open = () => '<del>'
markdownPreview.renderer.rules.s_close = () => '</del>'

const STICKER_TOKEN_RE = /\[:sticker:([^\s:[\]]{1,64}):]/g

// Resolvable sticker tokens in prose text become the same img[data-gf-sticker]
// the server renders; code, links and math (already placeholders) stay literal.
markdownPreview.core.ruler.push('gf_sticker', (state) => {
  const stickers = (state.env as { stickers?: ReadonlyMap<string, string> }).stickers
  if (!stickers?.size) return
  const escape = markdownPreview.utils.escapeHtml
  for (const block of state.tokens) {
    if (block.type !== 'inline' || !block.children) continue
    const children: typeof block.children = []
    let linkDepth = 0
    for (const child of block.children) {
      if (child.type === 'link_open') linkDepth++
      if (child.type === 'link_close') linkDepth--
      if (child.type !== 'text' || linkDepth > 0 || !child.content.includes('[:sticker:')) {
        children.push(child)
        continue
      }
      let cursor = 0
      for (const match of child.content.matchAll(STICKER_TOKEN_RE)) {
        const url = stickers.get(match[1])
        if (!url) continue
        if (match.index > cursor) {
          const text = new state.Token('text', '', 0)
          text.content = child.content.slice(cursor, match.index)
          children.push(text)
        }
        const image = new state.Token('html_inline', '', 0)
        image.content = `<img src="${escape(url)}" alt="${escape(match[1])}" data-gf-sticker="${escape(match[1])}">`
        children.push(image)
        cursor = match.index + match[0].length
      }
      if (cursor === 0) {
        children.push(child)
      } else if (cursor < child.content.length) {
        const text = new state.Token('text', '', 0)
        text.content = child.content.slice(cursor)
        children.push(text)
      }
    }
    block.children = children
  }
})

export function renderMarkdownPreview(source: string, options: { stickers?: ReadonlyMap<string, string> } = {}) {
  const input = source || ''
  // Extra blank lines between blocks render as the same spacers as the server.
  const marker = blankLineMarker(input)
  const marked = markBlankLines(input, marker)
  const protectedMath = protectMathSegments(marked)
  const html = markdownPreview.render(protectedMath.source, { stickers: options.stickers })
  const restored = restoreMathSegments(html, protectedMath.placeholders)
  return marked === input ? restored : restored.replaceAll(`<p>${marker}</p>`, BLANK_LINE_HTML)
}
