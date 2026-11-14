/**
 * 连续空行（#1584 / #1630 DQ-1，docs/decisions/0133-visible-blank-lines.md）。
 *
 * Markdown 原文是唯一来源：编辑器每多按一次 Enter 留下的空段落写成多一行空行。
 * 渲染时，顶层块之间一段空行里第一行之后的每个空行显示为一个空行占位，
 * 每处最多 MAX_RENDERED_BLANK_LINES 个；代码、数学、列表与缩进续行内部不处理。
 *
 * 做法是在这些位置插入一个只含标记的段落，交给原有解析器后再换成占位，
 * 因此不改变任何块结构。Go（markdown2html/blank_lines.go）与 Dart
 * （core/markdown/blank_lines.dart）实现同一规则，共用
 * packages/api-contract/fixtures/markdown-blank-lines.json。
 */

import { extractMathSegments } from './math-segments'

export const MAX_RENDERED_BLANK_LINES = 3

/** 发布 HTML 里的空行占位；Go 端输出同一结构，样式见 styles/prose.css。 */
export const BLANK_LINE_HTML = '<p class="gf-blank-line" aria-hidden="true"></p>'

const MARKER_PREFIX = 'GFBLANKLINE'
const FENCE_RE = /^(`{3,}|~{3,})/
const LIST_ITEM_RE = /^(?:([*+-])|\d{1,9}([.)]))(?:[ \t]|$)/
// CommonMark 的空行只含空格与制表符；全角空格等算正文。
const BLANK_RE = /^[ \t]*$/

/** 返回原文里不存在的标记，保证换回占位时不会误伤正文。 */
export function blankLineMarker(source: string): string {
  let marker = MARKER_PREFIX
  for (let suffix = 0; source.includes(marker); suffix++) marker = `${MARKER_PREFIX}${suffix}`
  return marker
}

/**
 * 在可见空行处插入标记段落；没有可见空行时原样返回 source。
 * max 为每处上限，编辑器重新载入时传 Infinity 以保留作者的全部空段。
 */
export function markBlankLines(source: string, marker: string, max = MAX_RENDERED_BLANK_LINES): string {
  if (max < 1 || !/\n[ \t]*\r?\n[ \t]*\r?\n/.test(source)) return source

  const normalized = source.replace(/\r\n?/g, '\n')
  // 公式范围与 protectMathSegments 相同：标记不会落进随后整体换成占位的公式里。
  const math = extractMathSegments(normalized)
  let nextMath = 0
  const out: string[] = []
  let pending: string[] = []
  let fence: string | null = null
  let list = ''
  let seenBlock = false
  let changed = false
  let offset = 0

  for (const line of normalized.split('\n')) {
    const start = offset
    offset += line.length + 1
    if (fence !== null) {
      out.push(line)
      if (closesFence(line, fence)) fence = null
      continue
    }
    while (nextMath < math.length && math[nextMath].end <= start) nextMath++
    if (nextMath < math.length && math[nextMath].start < start) {
      out.push(...pending, line)
      pending = []
      continue
    }
    if (BLANK_RE.test(line)) {
      pending.push(line)
      continue
    }

    const indent = indentWidth(line)
    const body = line.replace(/^[ \t]+/, '')
    // 换了项目符号或有序分隔符就是另一个列表。
    const match = indent < 4 ? LIST_ITEM_RE.exec(body) : null
    const item = match ? (match[1] ?? `1${match[2]}`) : ''
    const extra = Math.min(pending.length - 1, max)
    // 只在不缩进的新块之前插入：缩进行可能是列表续行或缩进代码，同类列表项之间
    // 的空行属于同一个列表。不确定时宁可不显示，也不改动结构。
    if (seenBlock && extra > 0 && indent === 0 && !(item !== '' && item === list)) {
      out.push('')
      for (let count = 0; count < extra; count++) out.push(marker, '')
      changed = true
    } else {
      out.push(...pending)
    }
    if (item !== '') list = item
    else if (pending.length > 0 && indent === 0) list = ''
    pending = []
    out.push(line)
    seenBlock = true

    if (indent < 4) {
      const open = FENCE_RE.exec(body)
      if (open && !(open[1][0] === '`' && body.slice(open[1].length).includes('`'))) fence = open[1]
    }
  }
  out.push(...pending)
  return changed ? out.join('\n') : source
}

function closesFence(line: string, fence: string): boolean {
  if (indentWidth(line) >= 4) return false
  const body = line.replace(/^[ \t]+|[ \t]+$/g, '')
  if (body.length < fence.length) return false
  for (const char of body) if (char !== fence[0]) return false
  return true
}

function indentWidth(line: string): number {
  let width = 0
  for (const char of line) {
    if (char === ' ') width++
    else if (char === '\t') width += 4 - (width % 4)
    else break
  }
  return width
}
