import { describe, expect, test } from 'vitest'
import { createMarkdownTable, fencedCodeBlock, formatMarkdownLines, prefixMarkdownBlock, replaceMarkdownSelectionWithBlock, wrapInlineMath } from '../src/runtime/markdown-editing'

describe('Markdown editing helpers', () => {
  test('separates inserted blocks from surrounding text', () => {
    const result = replaceMarkdownSelectionWithBlock('beforeafter', 6, 6, createMarkdownTable(2, 2))
    expect(result.value).toMatch(/^before\n\n\|/)
    expect(result.value).toMatch(/\|\n\nafter$/)
  })

  test('reuses existing block boundary newlines', () => {
    const result = replaceMarkdownSelectionWithBlock('before\n\nafter', 8, 8, '---')
    expect(result.value).toBe('before\n\n---\n\nafter')
  })

  test('builds prefixed and fenced blocks', () => {
    expect(prefixMarkdownBlock('one\ntwo', '> ')).toBe('> one\n> two')
    expect(fencedCodeBlock('const value = 1')).toBe('```\nconst value = 1\n```')
  })

  test('wraps inline math with single dollar delimiters', () => {
    expect(wrapInlineMath('x')).toBe('$x$')
    expect(wrapInlineMath('math')).toBe('$math$')
  })

  test('creates tables with the requested dimensions', () => {
    const table = createMarkdownTable(3, 2)
    expect(table.split('\n')).toHaveLength(4)
    expect(table.split('\n').every(line => (line.match(/\|/g) || []).length === 3)).toBe(true)
  })

  test('changes selected line headings without consuming the next line', () => {
    const source = 'first\nsecond\nthird'
    const result = formatMarkdownLines(source, 0, 6, 'heading_2')
    expect(result.value).toBe('## first\nsecond\nthird')
  })
})
