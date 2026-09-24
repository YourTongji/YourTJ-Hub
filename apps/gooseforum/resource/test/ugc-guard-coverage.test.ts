import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { parse } from 'vue/compiler-sfc'

// These are rendered user-controlled bodies. Adding a reader must preserve the
// shared guard even when that surface renders no preview-card candidates.
const surfaces = [
  ['pages/CourseDetailPage.vue', 'review.contentHtml'],
  ['pages/CourseDetailPage.vue', 'sharePreview.markdownHtml'],
  ['pages/WikiPage.vue', 'page.props.page.content'],
  ['components/CoursePreviewPane.vue', 'review.contentHtml'],
  ['components/schedule/ScheduleDetailList.vue', 'rev.contentHtml'],
]
describe('UGC reader navigation boundary', () => {
  for (const [file, expression] of surfaces) it(`${file} guards ${expression}`, () => {
    const source = readFileSync(new URL(`../src/site/${file}`, import.meta.url), 'utf8')
    const ast = parse(source).descriptor.template!.ast
    let found = false
    const visit = (node: any) => {
      // An inert export-only subtree cannot be focused or navigated by users.
      if (node.type === 1 && node.props.some((p: any) => p.type === 6 && p.name === 'inert')) return
      if (node.type === 1 && node.props.some((p: any) => p.type === 7 && p.name === 'html' && p.exp?.content === expression)) {
        found = true
        expect(node.props.some((p: any) => p.type === 7 && p.name === 'content-enhancements')).toBe(true)
      }
      node.children?.forEach(visit)
    }
    visit(ast); expect(found).toBe(true)
  })
})
