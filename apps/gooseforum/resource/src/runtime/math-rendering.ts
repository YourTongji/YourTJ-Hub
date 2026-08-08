import katex from 'katex'
import 'katex/dist/katex.min.css'
import { extractMathSegments } from './math-segments'

/**
 * Lazily-loaded KaTeX renderer for Markdown math. This module is imported only
 * after a math marker is detected, so KaTeX (JS, CSS and fonts) stays out of
 * the base forum bundle.
 */

export type RenderedPart =
  | { type: 'text'; value: string }
  | { type: 'math'; html: string; display: boolean }

/** Upper bound for a single math segment; longer content is kept as literal text. */
export const MAX_MATH_LENGTH = 5000

/** Cap on KaTeX-generated rule/matrix sizes (em) to prevent layout bombs. */
const MAX_RULE_SIZE = 10

export function renderMath(source: string, displayMode: boolean): string | null {
  if (source.length > MAX_MATH_LENGTH) return null
  try {
    return katex.renderToString(source, { displayMode, throwOnError: false, maxSize: MAX_RULE_SIZE })
  } catch {
    return null
  }
}

/**
 * Splits a source string into plain-text and math parts. KaTeX output is HTML
 * that must be escaped (renderToString escapes its input by design); a render
 * failure falls back to the raw delimited text so nothing is ever lost.
 */
export function splitIntoMathParts(
  source: string,
  render: (source: string, displayMode: boolean) => string | null = renderMath,
): RenderedPart[] {
  const segments = extractMathSegments(source)
  if (segments.length === 0) return [{ type: 'text', value: source }]

  const parts: RenderedPart[] = []
  let index = 0
  for (const segment of segments) {
    if (segment.start > index) {
      parts.push({ type: 'text', value: source.slice(index, segment.start) })
    }
    const html = render(segment.text, segment.display)
    if (html === null) {
      parts.push({ type: 'text', value: source.slice(segment.start, segment.end) })
    } else {
      parts.push({ type: 'math', html, display: segment.display })
    }
    index = segment.end
  }
  if (index < source.length) {
    parts.push({ type: 'text', value: source.slice(index) })
  }
  return parts
}

/** Builds a document fragment from rendered parts (browser only). */
export function buildFragment(parts: RenderedPart[]): DocumentFragment {
  const fragment = document.createDocumentFragment()
  for (const part of parts) {
    if (part.type === 'text') {
      fragment.appendChild(document.createTextNode(part.value))
    } else {
      const holder = document.createElement('span')
      holder.innerHTML = part.html
      fragment.appendChild(holder.firstElementChild ?? holder)
    }
  }
  return fragment
}
