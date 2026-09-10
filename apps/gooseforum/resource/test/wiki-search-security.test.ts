import { describe, expect, test } from 'vitest'
import { highlightQuery, sanitizeHighlightMarkup } from '../src/runtime/use-wiki-search'

describe('Wiki 搜索高亮安全性', () => {
  test('标题高亮会转义实体编码之外的 HTML', () => {
    expect(highlightQuery('标题 <img src=x onerror=alert(1)>', '标题')).toBe(
      '<mark>标题</mark> &lt;img src=x onerror=alert(1)&gt;',
    )
  })

  test('snippet 只保留裸 mark 标签', () => {
    expect(sanitizeHighlightMarkup('正文 <mark>关键词</mark> <img src=x onerror=alert(1)>')).toBe(
      '正文 <mark>关键词</mark> &lt;img src=x onerror=alert(1)&gt;',
    )
  })

  test('已编码的标签保持为安全文本', () => {
    expect(sanitizeHighlightMarkup('正文 &lt;img src=x onerror=alert(1)&gt;')).toBe(
      '正文 &lt;img src=x onerror=alert(1)&gt;',
    )
  })
})
