import { extractMathSegments } from './math-segments'

export interface MathPlaceholder {
  token: string
  original: string
}

const MATH_PLACEHOLDER_PREFIX = '@@YOURTJ_MATH_'

export function protectMathSegments(source: string): { source: string; placeholders: MathPlaceholder[] } {
  const segments = extractMathSegments(source)
  if (segments.length === 0) return { source, placeholders: [] }

  const placeholders: MathPlaceholder[] = []
  const used = new Set<string>()
  let output = ''
  let last = 0

  for (const [index, segment] of segments.entries()) {
    const token = uniqueMathToken(source, index, used)
    output += source.slice(last, segment.start)
    output += token
    placeholders.push({ token, original: source.slice(segment.start, segment.end) })
    last = segment.end
  }
  output += source.slice(last)

  return { source: output, placeholders }
}

export function restoreMathSegments(rendered: string, placeholders: MathPlaceholder[]): string {
  for (const placeholder of placeholders) {
    rendered = rendered.split(placeholder.token).join(placeholder.original)
  }
  return rendered
}

function uniqueMathToken(source: string, index: number, used: Set<string>): string {
  let token = `${MATH_PLACEHOLDER_PREFIX}${index}@@`
  for (let suffix = 1; source.includes(token) || used.has(token); suffix++) {
    token = `${MATH_PLACEHOLDER_PREFIX}${index}_${suffix}@@`
  }
  used.add(token)
  return token
}
