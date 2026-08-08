/**
 * Pure delimiter detection for Markdown math ($...$ and $$...$$).
 *
 * The detection rules are a port of KaTeX auto-render's brace-balanced scan,
 * plus MathJax-style inline guards (delimiters must not sit next to whitespace
 * and inline math cannot span lines) to keep prices and shell variables literal.
 *
 * This module must stay free of DOM and KaTeX imports so the directive can use
 * it as a cheap loading gate without pulling the KaTeX chunk into the bundle.
 */

export interface MathSegment {
  /** TeX source without the surrounding delimiters. */
  text: string
  /** True for block math ($$...$$), false for inline ($...$). */
  display: boolean
  /** Offset of the opening delimiter in the source string. */
  start: number
  /** Offset just after the closing delimiter. */
  end: number
}

const BLOCK_DELIMITER = '$$'
const INLINE_DELIMITER = '$'

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
    if (braceLevel <= 0 && char === INLINE_DELIMITER) {
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

export function extractMathSegments(source: string): MathSegment[] {
  const segments: MathSegment[] = []
  let index = 0

  while (index < source.length) {
    const dollarIndex = source.indexOf(INLINE_DELIMITER, index)
    if (dollarIndex === -1) break

    if (isEscaped(source, dollarIndex)) {
      index = dollarIndex + 1
      continue
    }

    if (source.startsWith(BLOCK_DELIMITER, dollarIndex)) {
      const close = findClosingDelimiter(source, BLOCK_DELIMITER, dollarIndex + BLOCK_DELIMITER.length)
      if (close !== -1) {
        const text = source.slice(dollarIndex + BLOCK_DELIMITER.length, close)
        if (text.trim().length > 0) {
          segments.push({ text, display: true, start: dollarIndex, end: close + BLOCK_DELIMITER.length })
          index = close + BLOCK_DELIMITER.length
          continue
        }
      }
      index = dollarIndex + 1
      continue
    }

    if (isWhitespace(source[dollarIndex + 1])) {
      index = dollarIndex + 1
      continue
    }

    const close = findInlineClosing(source, dollarIndex + 1)
    if (close !== -1) {
      const text = source.slice(dollarIndex + 1, close)
      if (text.length > 0 && !text.includes('\n')) {
        segments.push({ text, display: false, start: dollarIndex, end: close + 1 })
        index = close + 1
        continue
      }
    }
    index = dollarIndex + 1
  }

  return segments
}
