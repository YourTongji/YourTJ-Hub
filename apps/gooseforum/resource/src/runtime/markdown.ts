import MarkdownIt from 'markdown-it'
import anchor from 'markdown-it-anchor'
import taskLists from 'markdown-it-task-lists'
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

export function renderMarkdownPreview(source: string) {
  const input = source || ''
  const protectedMath = protectMathSegments(input)
  const html = markdownPreview.render(protectedMath.source)
  return restoreMathSegments(html, protectedMath.placeholders)
}
