import { describe, expect, test } from 'vitest'
import { protectMathSegments, restoreMathSegments } from '../src/runtime/math-protection'
import { renderMarkdownPreview } from '../src/runtime/markdown'

describe('math protection', () => {
  test('round-trips protected math back into the rendered HTML', () => {
    const original = 'inline $x$ and $$y$$'
    const protectedMath = protectMathSegments(original)

    expect(protectedMath.placeholders).toHaveLength(2)
    expect(protectedMath.source).not.toContain('$')
    expect(restoreMathSegments(protectedMath.source, protectedMath.placeholders)).toBe(original)
  })

  test('escapes HTML-sensitive characters when restoring math', () => {
    const original = `$f'(x)$ and $a < b & c$`
    const protectedMath = protectMathSegments(original)

    expect(restoreMathSegments(protectedMath.source, protectedMath.placeholders)).toBe('$f&#39;(x)$ and $a &lt; b &amp; c$')
  })

  test('keeps math primes literal instead of applying typographer', () => {
    const html = renderMarkdownPreview("$f'(x)$")

    expect(html).toContain('$f&#39;(x)$')
    expect(html).not.toContain('’')
  })

  test('keeps bare asterisks inside inline math from becoming emphasis', () => {
    const html = renderMarkdownPreview('$a*b*c$')

    expect(html).toContain('$a*b*c$')
    expect(html).not.toContain('<em>')
  })

  test('keeps block math with blank lines as one renderable segment', () => {
    const html = renderMarkdownPreview('$$\n\nE = mc^2\n\n$$')

    expect(html).toContain('<p>$$')
    expect(html).toContain('$$</p>')
    expect(html).not.toContain('<p>$$</p>')
  })

  test('restores math inside inline code without leaking placeholders', () => {
    const html = renderMarkdownPreview('`$x$`')

    expect(html).toContain('<code>$x$</code>')
    expect(html).not.toContain('@@YOURTJ_MATH_')
  })

  test('escapes raw HTML inside math so preview cannot inject markup', () => {
    const html = renderMarkdownPreview('$<script>alert(1)</script>$')

    expect(html).not.toContain('<script')
    expect(html).toContain('&lt;script')
  })
})
