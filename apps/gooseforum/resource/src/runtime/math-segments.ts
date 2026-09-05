/**
 * Pure delimiter detection for Markdown math.
 *
 * Supports `$...$`, `$$...$$`, `\(...\)`, `\[...\]`, and common
 * `\begin{env}...\end{env}` environments. The detection rules use a
 * brace-balanced scan and keep `$` delimiters away from prices and shell
 * variables.
 *
 * This module must stay free of DOM and KaTeX imports so the directive can use
 * it as a cheap loading gate without pulling the KaTeX chunk into the bundle.
 */

export interface MathSegment {
  /** TeX source without the surrounding delimiters. */
  text: string
  /** True for block math, false for inline math. */
  display: boolean
  /** Offset of the opening delimiter in the source string. */
  start: number
  /** Offset just after the closing delimiter. */
  end: number
}

interface MathDelimiter {
  open: string
  close: string
  display: boolean
  singleDollar: boolean
}

const MATH_ENVIRONMENTS = [
  'equation',
  'equation*',
  'align',
  'align*',
  'aligned',
  'gather',
  'gather*',
  'multline',
  'multline*',
  'split',
  'cases',
  'matrix',
  'pmatrix',
  'bmatrix',
  'vmatrix',
  'Vmatrix',
] as const

const MATH_DELIMITERS: MathDelimiter[] = [
  { open: '$$', close: '$$', display: true, singleDollar: false },
  { open: '\\[', close: '\\]', display: true, singleDollar: false },
  ...MATH_ENVIRONMENTS.map((environment) => ({
    open: `\\begin{${environment}}`,
    close: `\\end{${environment}}`,
    display: true,
    singleDollar: false,
  })),
  { open: '$', close: '$', display: false, singleDollar: true },
  { open: '\\(', close: '\\)', display: false, singleDollar: false },
]

function isEscaped(source: string, index: number): boolean {
  let backslashes = 0
  for (let i = index - 1; i >= 0 && source[i] === '\\'; i--) backslashes++
  return backslashes % 2 === 1
}

function isWhitespace(char: string | undefined): boolean {
  return char !== undefined && /\s/.test(char)
}

/**
 * Scans for a closing `delimiter`, tracking `{}` nesting and skipping
 * backslash-escaped characters (port of KaTeX auto-render's findEndOfMath).
 */
function findClosingDelimiter(source: string, delimiter: string, startIndex: number): number {
  const delimLength = delimiter.length
  let braceLevel = 0
  let index = startIndex
  while (index < source.length) {
    if (braceLevel <= 0 && source.startsWith(delimiter, index)) return index
    const char = source[index]
    if (char === '\\') {
      index++
    } else if (char === '{') {
      braceLevel++
    } else if (char === '}') {
      braceLevel--
    }
    index++
  }
  return -1
}

/**
 * Scans for a closing inline `$`. A closing dollar must not be adjacent to
 * whitespace; an interior dollar that cannot close the segment means the
 * construct is not well-formed inline math and the whole candidate is rejected.
 */
function findInlineClosing(source: string, startIndex: number): number {
  let braceLevel = 0
  let index = startIndex
  while (index < source.length) {
    const char = source[index]
    if (braceLevel <= 0 && char === '$') {
      if (index > startIndex && !isWhitespace(source[index - 1])) return index
      return -1
    }
    if (char === '\\') {
      index++
    } else if (char === '{') {
      braceLevel++
    } else if (char === '}') {
      braceLevel--
    }
    index++
  }
  return -1
}

function findNextDelimiter(source: string, start: number): { index: number; delimiter: MathDelimiter } | null {
  let best: { index: number; delimiter: MathDelimiter } | null = null
  for (const delimiter of MATH_DELIMITERS) {
    let index = source.indexOf(delimiter.open, start)
    while (index !== -1 && isEscaped(source, index)) {
      index = source.indexOf(delimiter.open, index + 1)
    }
    if (index !== -1 && (best === null || index < best.index)) {
      best = { index, delimiter }
    }
  }
  return best
}

export function extractMathSegments(source: string): MathSegment[] {
  const segments: MathSegment[] = []
  let index = 0

  while (index < source.length) {
    const found = findNextDelimiter(source, index)
    if (!found) break

    const { index: openIndex, delimiter } = found
    const openEnd = openIndex + delimiter.open.length

    if (delimiter.singleDollar) {
      if (isWhitespace(source[openIndex + 1])) {
        index = openIndex + 1
        continue
      }

      const close = findInlineClosing(source, openEnd)
      if (close !== -1) {
        const text = source.slice(openEnd, close)
        if (text.length > 0 && !text.includes('\n')) {
          segments.push({ text, display: delimiter.display, start: openIndex, end: close + 1 })
          index = close + 1
          continue
        }
      }
      index = openIndex + 1
      continue
    }

    const close = findClosingDelimiter(source, delimiter.close, openEnd)
    if (close !== -1) {
      const text = source.slice(openEnd, close)
      if (text.trim().length > 0) {
        segments.push({ text, display: delimiter.display, start: openIndex, end: close + delimiter.close.length })
        index = close + delimiter.close.length
        continue
      }
    }
    index = openIndex + delimiter.open.length
  }

  return segments
}
