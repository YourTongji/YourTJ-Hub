import { describe, expect, test, vi } from 'vitest'
import { enhanceCodeBlocks, extractCodeLanguage } from '../src/runtime/code-highlight-directive'
import { highlightCode } from '../src/runtime/code-highlighting'
import { renderMarkdownPreview } from '../src/runtime/markdown'

type FakeCodeElement = HTMLElement & {
  addedClasses: Set<string>
}

function fakeCodeElement(className: string, textContent: string, innerHTML = textContent): FakeCodeElement {
  const addedClasses = new Set<string>()
  return {
    className,
    textContent,
    innerHTML,
    dataset: {},
    addedClasses,
    classList: {
      add: (...classes: string[]) => classes.forEach(className => addedClasses.add(className)),
    },
  } as unknown as FakeCodeElement
}

function fakeRoot(elements: HTMLElement[]): ParentNode {
  return {
    querySelectorAll: vi.fn(() => elements),
  } as unknown as ParentNode
}

describe('highlightCode', () => {
  test('highlights recognized languages and aliases', () => {
    expect(highlightCode('package main\nfunc main() {}', 'go')).toContain('hljs-keyword')
    expect(highlightCode('const total: number = 1', 'ts')).toContain('hljs-keyword')
  })

  test('does not auto-detect missing or unknown languages', () => {
    expect(highlightCode('const value = 1', '')).toBeNull()
    expect(highlightCode('const value = 1', 'madeup')).toBeNull()
  })

  test('escapes script-like source and tolerates malformed code', () => {
    const script = highlightCode('const value = "<script>alert(1)</script>"', 'javascript')
    expect(script).toContain('&lt;script&gt;')
    expect(script).not.toContain('<script>')
    expect(highlightCode('func main( {', 'go')).not.toBeNull()
  })
})

describe('Markdown preview integration', () => {
  test('emits enhancement hooks while preserving escaped fallback HTML', () => {
    const known = renderMarkdownPreview('```go\npackage main\n```')
    const unknown = renderMarkdownPreview('```madeup\n<script>alert(1)</script>\n```')

    expect(known).toContain('class="language-go"')
    expect(known).not.toContain('class="hljs-')
    expect(unknown).toContain('class="language-madeup"')
    expect(unknown).toContain('&lt;script&gt;alert(1)&lt;/script&gt;')
    expect(unknown).not.toContain('<script>')
    expect(unknown).not.toContain('class="hljs-')
  })
})

describe('extractCodeLanguage', () => {
  test('extracts a language class among unrelated classes', () => {
    expect(extractCodeLanguage({ className: 'hljs language-TypeScript compact' })).toBe('TypeScript')
  })

  test('rejects empty and missing language classes', () => {
    expect(extractCodeLanguage({ className: 'language- compact' })).toBeNull()
    expect(extractCodeLanguage({ className: 'compact' })).toBeNull()
  })
})

describe('enhanceCodeBlocks', () => {
  test('does not load the highlighter without labelled code blocks', async () => {
    const unlabelled = fakeCodeElement('', 'const plain = true')
    const loader = vi.fn(async () => ({ highlightCode }))

    await enhanceCodeBlocks(fakeRoot([unlabelled]), loader)

    expect(loader).not.toHaveBeenCalled()
    expect(unlabelled.innerHTML).toBe('const plain = true')
  })

  test('highlights recognized blocks once', async () => {
    const code = fakeCodeElement('language-go', 'package main')
    const loader = vi.fn(async () => ({ highlightCode }))
    const root = fakeRoot([code])

    await enhanceCodeBlocks(root, loader)
    const firstHTML = code.innerHTML
    await enhanceCodeBlocks(root, loader)

    expect(firstHTML).toContain('hljs-keyword')
    expect(code.innerHTML).toBe(firstHTML)
    expect(code.addedClasses).toContain('hljs')
    expect(code.dataset.codeHighlight).toBe('done')
    expect(loader).toHaveBeenCalledTimes(1)
  })

  test('leaves unknown and unlabelled blocks plain', async () => {
    const unknown = fakeCodeElement('language-madeup', '<widget>safe</widget>', '&lt;widget&gt;safe&lt;/widget&gt;')
    const unlabelled = fakeCodeElement('', 'const plain = true')
    const loader = vi.fn(async () => ({ highlightCode }))

    await enhanceCodeBlocks(fakeRoot([unknown, unlabelled]), loader)

    expect(unknown.innerHTML).toBe('&lt;widget&gt;safe&lt;/widget&gt;')
    expect(unknown.dataset.codeHighlight).toBe('unsupported')
    expect(unlabelled.innerHTML).toBe('const plain = true')
    expect(unlabelled.dataset.codeHighlight).toBeUndefined()
  })

  test('preserves original markup when the highlighter cannot load', async () => {
    const code = fakeCodeElement('language-go', 'package main', 'package main')

    await enhanceCodeBlocks(fakeRoot([code]), async () => {
      throw new Error('chunk unavailable')
    })

    expect(code.innerHTML).toBe('package main')
    expect(code.dataset.codeHighlight).toBeUndefined()
    expect(code.addedClasses).not.toContain('hljs')
  })
})
