import { describe, expect, test, vi } from 'vitest'
import { extractMathSegments } from '../src/runtime/math-segments'
import { MAX_MATH_LENGTH, renderMath, splitIntoMathParts } from '../src/runtime/math-rendering'
import { enhanceMathText } from '../src/runtime/math-render-directive'
import type { TextNodeLike } from '../src/runtime/math-render-directive'

describe('extractMathSegments', () => {
  test('detects a simple inline segment', () => {
    expect(extractMathSegments('$x$')).toEqual([
      { text: 'x', display: false, start: 0, end: 3 },
    ])
  })

  test('detects a block segment with priority over inline', () => {
    expect(extractMathSegments('$$x$$')).toEqual([
      { text: 'x', display: true, start: 0, end: 5 },
    ])
  })

  test('detects inline and block segments in mixed text in order', () => {
    expect(extractMathSegments('a $x$ and $$y$$ end')).toEqual([
      { text: 'x', display: false, start: 2, end: 5 },
      { text: 'y', display: true, start: 10, end: 15 },
    ])
  })

  test('detects multiple inline segments', () => {
    expect(extractMathSegments('$a$ and $b$')).toEqual([
      { text: 'a', display: false, start: 0, end: 3 },
      { text: 'b', display: false, start: 8, end: 11 },
    ])
  })

  test('balances braces inside inline math', () => {
    expect(extractMathSegments('$a_{x}$')).toEqual([
      { text: 'a_{x}', display: false, start: 0, end: 7 },
    ])
    expect(extractMathSegments('$a {b}$').map((s) => s.text)).toEqual(['a {b}'])
  })

  test('allows block math to span newlines', () => {
    const [segment] = extractMathSegments('$$\na\nb\n$$')
    expect(segment.display).toBe(true)
    expect(segment.text).toContain('\n')
  })

  test('ignores an escaped opening dollar sign', () => {
    expect(extractMathSegments('\\$x$')).toEqual([])
  })

  test('rejects inline math adjacent to whitespace', () => {
    expect(extractMathSegments('$ x$')).toEqual([])
    expect(extractMathSegments('$x $')).toEqual([])
    expect(extractMathSegments('x $ x$')).toEqual([])
  })

  test('rejects inline math spanning a newline', () => {
    expect(extractMathSegments('$x\ny$')).toEqual([])
  })

  test('does not treat prices or shell variables as math', () => {
    expect(extractMathSegments('a $5 discount')).toEqual([])
    expect(extractMathSegments('$PATH and $HOME are set')).toEqual([])
    expect(extractMathSegments('cost $5 and $10 each')).toEqual([])
  })

  test('resolves an ambiguous candidate to a later well-formed segment', () => {
    expect(extractMathSegments('$5 and $x$')).toEqual([
      { text: 'x', display: false, start: 7, end: 10 },
    ])
  })

  test('leaves empty or unclosed delimiters as literal text', () => {
    expect(extractMathSegments('$$$$')).toEqual([])
    expect(extractMathSegments('$')).toEqual([])
    expect(extractMathSegments('$$x')).toEqual([])
    expect(extractMathSegments('plain text without math')).toEqual([])
  })
})

describe('renderMath', () => {
  test('renders inline math with a katex span', () => {
    const html = renderMath('x^2', false)
    expect(html).not.toBeNull()
    expect(html).toContain('katex')
  })

  test('renders block math in display mode', () => {
    const html = renderMath('\\frac{a}{b}', true)
    expect(html).not.toBeNull()
    expect(html).toContain('katex-display')
  })

  test('escapes script-like source instead of injecting it', () => {
    const html = renderMath('<script>alert(1)</script>', false)
    expect(html).not.toBeNull()
    expect(html).not.toContain('<script')
    expect(html).not.toContain('onerror=')
    expect(html).not.toContain('<img')
    expect(html).toContain('&lt;script')
  })

  test('escapes event-handler attributes in malformed input', () => {
    const html = renderMath('x onerror="alert(1)"', false)
    expect(html).not.toContain('<script')
    expect(html).not.toContain('onerror="')
    expect(html).not.toContain('javascript:')
  })

  test('refuses over-long segments to avoid client-side jank', () => {
    expect(renderMath(`x`.repeat(MAX_MATH_LENGTH + 1), false)).toBeNull()
    expect(renderMath(`x`.repeat(MAX_MATH_LENGTH), false)).not.toBeNull()
  })
})

describe('splitIntoMathParts', () => {
  test('returns a single text part without math', () => {
    expect(splitIntoMathParts('plain text')).toEqual([{ type: 'text', value: 'plain text' }])
  })

  test('interleaves text and math parts', () => {
    const parts = splitIntoMathParts('a $x$ b', () => '<span class="katex"></span>')
    expect(parts).toEqual([
      { type: 'text', value: 'a ' },
      { type: 'math', html: '<span class="katex"></span>', display: false },
      { type: 'text', value: ' b' },
    ])
  })

  test('keeps the raw delimited text when rendering fails', () => {
    const parts = splitIntoMathParts('a $x$ b', () => null)
    expect(parts).toEqual([
      { type: 'text', value: 'a ' },
      { type: 'text', value: '$x$' },
      { type: 'text', value: ' b' },
    ])
  })

  test('keeps an over-long math segment as literal text', () => {
    const longMath = `$${`x`.repeat(MAX_MATH_LENGTH + 1)}$`
    const parts = splitIntoMathParts(longMath)
    expect(parts).toEqual([{ type: 'text', value: longMath }])
  })
})

describe('enhanceMathText', () => {
  function fakeNode(data: string, replaceWith = vi.fn()): TextNodeLike {
    return { data, replaceWith }
  }

  test('does not load the renderer without math markers', async () => {
    const nodes = [fakeNode('plain text')]
    const loader = vi.fn(async () => ({
      renderMath,
      splitIntoMathParts,
      buildFragment: (parts: unknown) => parts,
    }))

    const changed = await enhanceMathText({} as ParentNode, loader, () => nodes)

    expect(changed).toBe(false)
    expect(loader).not.toHaveBeenCalled()
    expect(nodes[0].replaceWith).not.toHaveBeenCalled()
  })

  test('does not load the renderer for dollar signs that are not math', async () => {
    const nodes = [fakeNode('a $5 discount and a $HOME path')]
    const loader = vi.fn(async () => ({
      renderMath,
      splitIntoMathParts,
      buildFragment: (parts: unknown) => parts,
    }))

    const changed = await enhanceMathText({} as ParentNode, loader, () => nodes)

    expect(changed).toBe(false)
    expect(loader).not.toHaveBeenCalled()
    expect(nodes[0].replaceWith).not.toHaveBeenCalled()
  })

  test('preserves content when the renderer cannot load', async () => {
    const node = fakeNode('$x$')
    const loader = vi.fn(async () => {
      throw new Error('chunk unavailable')
    })

    const changed = await enhanceMathText({} as ParentNode, loader, () => [node])

    expect(changed).toBe(false)
    expect(node.replaceWith).not.toHaveBeenCalled()
  })

  test('replaces math-bearing text nodes and returns true', async () => {
    const node = fakeNode('$x$')
    const buildFragment = vi.fn((parts: unknown) => parts)
    const loader = vi.fn(async () => ({ renderMath, splitIntoMathParts, buildFragment }))

    const changed = await enhanceMathText({} as ParentNode, loader, () => [node])

    expect(changed).toBe(true)
    expect(node.replaceWith).toHaveBeenCalledTimes(1)
    expect(buildFragment).toHaveBeenCalled()
  })

  test('skips nodes without math even when siblings contain it', async () => {
    const plain = fakeNode('plain')
    const math = fakeNode('$x$')
    const loader = vi.fn(async () => ({
      renderMath,
      splitIntoMathParts,
      buildFragment: (parts: unknown) => parts,
    }))

    const changed = await enhanceMathText({} as ParentNode, loader, () => [plain, math])

    expect(changed).toBe(true)
    expect(plain.replaceWith).not.toHaveBeenCalled()
    expect(math.replaceWith).toHaveBeenCalledTimes(1)
  })

  test('loads the renderer only once across re-enhancement passes', async () => {
    const node = fakeNode('$x$')
    const replaced = new Set<TextNodeLike>()
    const nodes: TextNodeLike[] = [node]
    node.replaceWith = vi.fn((fragment: unknown) => {
      replaced.add(node)
    })
    const loader = vi.fn(async () => ({
      renderMath,
      splitIntoMathParts,
      buildFragment: (parts: unknown) => parts,
    }))

    await enhanceMathText({} as ParentNode, loader, () => nodes.filter((n) => !replaced.has(n)))
    await enhanceMathText({} as ParentNode, loader, () => nodes.filter((n) => !replaced.has(n)))

    expect(loader).toHaveBeenCalledTimes(1)
  })
})

describe('inline math across element boundaries', () => {
  test('leaves split inline math literal until pre-render protection exists', async () => {
    const fakeNode = (data: string) => ({ data, replaceWith: vi.fn() })
    const nodes = [fakeNode('$a'), fakeNode('b'), fakeNode('c$')]
    const loader = vi.fn(async () => ({
      renderMath,
      splitIntoMathParts,
      buildFragment: (parts: unknown) => parts,
    }))

    const changed = await enhanceMathText({} as ParentNode, loader, () => nodes)

    expect(changed).toBe(false)
    expect(loader).not.toHaveBeenCalled()
    expect(nodes.every((node) => node.replaceWith.mock.calls.length === 0)).toBe(true)
  })
})
