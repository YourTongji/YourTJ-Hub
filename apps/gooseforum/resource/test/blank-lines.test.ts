import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'
import { BLANK_LINE_HTML, blankLineMarker, markBlankLines } from '../src/runtime/blank-lines'
import { renderMarkdownPreview } from '../src/runtime/markdown'

type BlankLineCase = { name: string; markdown: string; marked: string | null; unlimited?: boolean }

const fixture = JSON.parse(
  readFileSync(resolve(process.cwd(), '../../../packages/api-contract/fixtures/markdown-blank-lines.json'), 'utf8'),
) as { marker: string; maxRendered: number; cases: BlankLineCase[] }

const spacerCount = (html: string) => html.split(BLANK_LINE_HTML).length - 1

describe('visible blank lines', () => {
  for (const item of fixture.cases) {
    test(`shared fixture: ${item.name}`, () => {
      const max = item.unlimited ? Infinity : fixture.maxRendered
      expect(markBlankLines(item.markdown, fixture.marker, max)).toBe(item.marked ?? item.markdown)
    })
  }

  test('marker never occurs in the source', () => {
    expect(blankLineMarker('甲')).toBe('GFBLANKLINE')
    expect(blankLineMarker('GFBLANKLINE GFBLANKLINE0')).toBe('GFBLANKLINE1')
  })

  test('preview renders one spacer per extra blank line', () => {
    expect(spacerCount(renderMarkdownPreview('甲\n\n乙'))).toBe(0)
    expect(spacerCount(renderMarkdownPreview('甲\n\n\n乙'))).toBe(1)
    expect(spacerCount(renderMarkdownPreview('甲\n\n\n\n\n\n\n\n乙'))).toBe(3)
    expect(renderMarkdownPreview('甲\n\n\n乙')).toBe(`<p>甲</p>\n${BLANK_LINE_HTML}\n<p>乙</p>\n`)
  })

  test('preview keeps blank lines inside code and math literal', () => {
    const code = renderMarkdownPreview('```\na\n\n\nb\n```')
    expect(code).toContain('a\n\n\nb')
    expect(spacerCount(code)).toBe(0)
    const math = renderMarkdownPreview('$$\na\n\n\nb\n$$\n\n\n乙')
    expect(math).not.toContain('GFBLANKLINE')
    expect(spacerCount(math)).toBe(1)
  })

  test('a paragraph that looks like the marker stays text', () => {
    const html = renderMarkdownPreview('GFBLANKLINE\n\n\n乙')
    expect(html).toContain('<p>GFBLANKLINE</p>')
    expect(spacerCount(html)).toBe(1)
  })

  test('editor reload marks every empty paragraph', () => {
    // Vditor writes k empty paragraphs as k+1 blank lines; reloading marks k of them
    // and serialising the emptied markers produces the same run again.
    for (const empty of [1, 2, 5]) {
      const source = `甲\n${'\n'.repeat(empty + 1)}乙\n`
      const marker = blankLineMarker(source)
      const marked = markBlankLines(source, marker, Infinity)
      expect(marked.split('\n').filter((line) => line === marker)).toHaveLength(empty)
      expect(marked.split('\n').filter((line) => line !== marker).join('\n')).toBe(source)
    }
  })
})
