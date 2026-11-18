// line-clamp × display 同串冲突门禁（issue #1692）：
// 同一元素上 line-clamp-* 与其他写 display 的工具类共存时，构建 CSS 同 specificity
// 按源序覆盖（含跨断点：display 变体链是 clamp 变体链超集时，断点/状态激活即覆盖），
// 行数截断静默失效（缺陷链 #1639/#1640 → #1677/#1688）。
// 单遍语境扫描 src/** 的 .vue/.ts：模板区只收集 class 类属性串（含 :class 字面量分支），
// script/style 块与 .ts 按代码语境收集全部字符串/模板字面量；注释、文本节点、
// {{ }} 插值、正则字面量不参与判定。判定语义与边界见 docs/development/coding-conventions.md。
// 用法：node scripts/check-line-clamp.mjs（作为 pnpm check:line-clamp / pnpm check 运行）
import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join, relative, sep } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

// 写 display 属性的裸工具类（Tailwind v4 display utilities）。
const displayUtilities = new Set([
  'block',
  'inline',
  'inline-block',
  'inline-flex',
  'flex',
  'grid',
  'inline-grid',
  'flow-root',
  'table',
  'inline-table',
  'table-caption',
  'table-cell',
  'table-column',
  'table-column-group',
  'table-footer-group',
  'table-header-group',
  'table-row',
  'table-row-group',
  'contents',
  'list-item',
  'hidden',
])

// 豁免清单沿用 audit-site-colors.mjs 的 intentionalUtilityAllowlist 模式：
// 文件级豁免（相对 resource 根的 POSIX 路径），每项须在 PR 中说明理由；不放宽规则。
export const lineClampAllowlist = new Set([])

// 单文件大小上限：防御异常大文件拖垮 CI/pre-push（正常源码远小于此）。
const maxFileSize = 2 * 1024 * 1024

export function toPosix(p) {
  return p.split(sep).join('/')
}

export function collectSourceFiles() {
  const out = []
  const walk = (dir) => {
    // withFileTypes + 跳过符号链接：不跟随链接读仓库外文件或设备文件（CI 可信入口加固）。
    for (const dirent of readdirSync(dir, { withFileTypes: true })) {
      if (dirent.name === 'node_modules' || dirent.isSymbolicLink()) continue
      const p = join(dir, dirent.name)
      if (dirent.isDirectory()) walk(p)
      else if (dirent.isFile() && /\.(vue|ts)$/.test(dirent.name)) {
        const text = readFileSync(p, 'utf8')
        if (text.length > maxFileSize) {
          console.error(`[line-clamp] FAIL: ${toPosix(relative(root, p))} exceeds ${maxFileSize} bytes`)
          process.exit(1)
        }
        out.push({ relPath: toPosix(relative(root, p)), text })
      }
    }
  }
  walk(join(root, 'src'))
  return out
}

// ---- 语境扫描：单遍状态机，同时产出候选串与结构错误 ----
// 三种语境：
// - code（.ts 全文 / .vue 的 script、style 块）：字符串/模板字面量全收集；
//   正则字面量整体跳过（其中引号不是定界符）；// 与 /* */ 是注释。
// - tag（HTML 标签内）：只收集 class 类属性（class / *-class，排除 data-*/aria-*）；
//   :class 绑定外层引号只圈定边界，下钻内部异种引号字面量（互斥分支各自成串）；
//   <!-- --> 在标签态入口识别并整体跳过。引号只在「属性名 = 」前窗成立时才消费，
//   否则视为杂散字符跳过（文本裸 < 误入标签态时不吞文件）。
// - text（模板文本节点）：整体跳到下一个 '<'；撇号是自然语言不是定界符；
//   {{ }} 插值整体跳过；<!-- --> 整体跳过。
// 未闭合的字符串/模板字面量是结构错误（fail loud），不静默吞掉文件剩余部分。
const classAttrName = /(?:^|[\s<])((?:[A-Za-z0-9_]+-)*class)\s*=\s*$/
const bindingClassAttrName = /(?:^|[\s<])(?::|v-bind:)((?:[A-Za-z0-9_]+-)*class)\s*=\s*$/
const anyAttrEq = /(?:^|[\s<])(?::|v-bind:)?[A-Za-z0-9_.\-[\]]+\s*=\s*$/
const isClassAttrName = (name) => !/^(?:data|aria)-/.test(name)

// 正则字面量识别：'/' 处于表达式起始位置（前一个有效字符是运算符/开括号/逗号/
// 冒号/return 等）时才是正则，否则是除号。消费到未转义 '/'（跳过字符类 [...]）。
const regexPreceder = /[=([{,;:!&|?+\-*%<>~^]|\breturn$|\bcase$|\btypeof$|\bvoid$|\bdelete$|\bin$|\bof$|\bnew$|\byield$|\bawait$/
function looksLikeRegex(text, i) {
  let j = i - 1
  while (j >= 0 && /\s/.test(text[j])) j--
  if (j < 0) return true
  // 闭括号后跟 '/'：实参/索引结束后的除号极少出现在源码类串语境，
  // 而 `replace(/…/)` 这类调用首参正则常见——按正则处理（误判方向是多跳一段，
  // 不会把引号当定界符，结构错误由 fail-loud 兜底）。
  if (text[j] === ')' || text[j] === ']') return true
  // 窗口须从词边界起截取：固定长度窗口可能截在标识符中间，
  // 让 \breturn$ 之类误匹配（如 …trimmed.replace( 的截断尾巴）。
  let start = Math.max(0, j - 12)
  while (start > 0 && /[A-Za-z0-9_$]/.test(text[start - 1])) start--
  const before = text.slice(start, j + 1)
  return regexPreceder.test(before)
}
function skipRegex(text, start, end) {
  let j = start + 1
  let inClass = false
  while (j < end) {
    const c = text[j]
    if (c === '\\') { j += 2; continue }
    if (c === '[') inClass = true
    else if (c === ']') inClass = false
    else if (c === '/' && !inClass) return j + 1
    else if (c === '\n') return start + 1 // 正则不跨行：不是正则，按除号处理
    j++
  }
  return start + 1
}

// 从开引号找同层闭引号。模板字符串的 ${...} 插值区是代码语境：
// 用 scanCodeRange 按代码规则消费（正则/字符串/注释正确跳过，正则内的
// 花括号不干扰配平），并记录区间供递归收集。普通字符串直接找同种闭引号。
// 返回 { close, unterminated, interpolations: [{start,end}] }。
// scanCodeRange 由调用方注入（scanCode 声明在后，避免循环依赖）。
function findStringEnd(text, open, end, scanCodeRange) {
  const q = text[open]
  const interpolations = []
  let j = open + 1
  while (j < end) {
    const c = text[j]
    if (c === '\\') { j += 2; continue }
    if (q === '`' && c === '$' && text[j + 1] === '{') {
      const exprStart = j + 2
      const r = scanCodeRange(exprStart, end, true, false)
      if (!r.closed) return { close: end, unterminated: true, interpolations }
      interpolations.push({ start: exprStart, end: r.stopAt })
      j = r.stopAt + 1 // 跳过闭合 '}'
      continue
    }
    if (c === q) return { close: j, unterminated: false, interpolations }
    j++
  }
  return { close: end, unterminated: true, interpolations }
}

// 扫描产出：literals [{ value, offset }]（候选 class 串）、errors [{ offset, kind }]。
export function extractStringLiterals(text, { codeOnly = false } = {}) {
  const out = []
  const errors = []
  const n = text.length

  const pushLiteral = (open, close) => {
    out.push({ value: text.slice(open + 1, close), offset: open })
  }
  // 绑定表达式：下钻内部异种引号/反引号字面量，外层不判定。
  const drillBinding = (open, close) => {
    let j = open + 1
    while (j < close) {
      const d = text[j]
      if (d === '\\') { j += 2; continue }
      if ((d === "'" || d === '"' || d === '`') && d !== text[open]) {
        const r = findStringEnd(text, j, close, scanCodeRange)
        if (r.unterminated) errors.push({ offset: j, kind: 'unterminated-string' })
        pushLiteral(j, r.close)
        for (const interp of r.interpolations) scanCode(interp.start, interp.end)
        j = r.close + 1
        continue
      }
      j++
    }
  }
  const skipHtmlComment = (i) => {
    const close = text.indexOf('-->', i + 4)
    return close === -1 ? n : close + 3
  }
  const skipInterpolation = (i) => {
    let depth = 1
    let j = i + 2
    let quote = null
    while (j < n && depth > 0) {
      const d = text[j]
      if (d === '\\') { j += 2; continue }
      if (quote) {
        if (d === quote) quote = null
      } else if (d === "'" || d === '"' || d === '`') quote = d
      else if (d === '{') depth++
      else if (d === '}') depth--
      j++
    }
    return j
  }
  // code 语境扫描（.ts 全文 / script、style 块 / 模板插值区）。
  // stopAtBrace=true 时用于模板 ${...} 插值配平：追踪花括号深度，
  // 深度归零的 '}' 即插值结束（正则/字符串/注释内的花括号已被各自跳过）。
  // 返回 { stopAt, closed }；普通扫描返回 { stopAt: end, closed: true }。
  const scanCodeRange = (start, end, stopAtBrace = false, collect = true) => {
    let i = start
    let depth = 0
    while (i < end) {
      const c = text[i]
      if (c === '/' && text[i + 1] === '/') {
        const nl = text.indexOf('\n', i)
        i = nl === -1 || nl > end ? end : nl
        continue
      }
      if (c === '/' && text[i + 1] === '*') {
        const close = text.indexOf('*/', i + 2)
        i = close === -1 || close + 2 > end ? end : close + 2
        continue
      }
      if (c === '/' && looksLikeRegex(text, i)) {
        i = skipRegex(text, i, end)
        continue
      }
      if (c === "'" || c === '"' || c === '`') {
        const r = findStringEnd(text, i, end, scanCodeRange)
        if (r.unterminated) errors.push({ offset: i, kind: 'unterminated-string' })
        if (collect) pushLiteral(i, r.close)
        // 模板插值区是代码语境：其中的正则/字符串按代码规则递归扫描
        // （如 `"${s.replace(/"/g, '""')}"` 的正则含引号，不是定界符）。
        // 配平扫描不重复收集（外层已收集整条模板字面量）。
        for (const interp of r.interpolations) scanCodeRange(interp.start, interp.end, false, collect)
        i = r.close + 1
        continue
      }
      if (stopAtBrace) {
        if (c === '{') depth++
        else if (c === '}') {
          if (depth === 0) return { stopAt: i, closed: true }
          depth--
        }
      }
      i++
    }
    return { stopAt: end, closed: !stopAtBrace }
  }
  const scanCode = (start, end) => scanCodeRange(start, end).stopAt

  let i = 0
  let mode = codeOnly ? 'code' : 'text'
  while (i < n) {
    if (mode === 'code') {
      // codeOnly（.ts）扫到文件尾；script/style 块由块尾正则界定。
      i = scanCode(i, n)
      break
    }
    const c = text[i]
    if (mode === 'text') {
      if (c === '<') {
        if (text.startsWith('<!--', i)) { i = skipHtmlComment(i); continue }
        const block = /^<(script|style)\b/i.exec(text.slice(i, i + 8))
        if (block) {
          const closeRe = new RegExp(`</${block[1]}\\s*>`, 'i')
          const cm = closeRe.exec(text.slice(i))
          const blockEnd = cm ? i + cm.index : n
          // 块内从开标签 '>' 之后开始扫代码。
          const openTagEnd = text.indexOf('>', i)
          const codeStart = openTagEnd === -1 || openTagEnd > blockEnd ? blockEnd : openTagEnd + 1
          scanCode(codeStart, blockEnd)
          i = cm ? blockEnd + cm[0].length : n
          continue
        }
        mode = 'tag'
        i++
        continue
      }
      if (c === '{' && text[i + 1] === '{') { i = skipInterpolation(i); continue }
      i++
      continue
    }
    // mode === 'tag'
    if (c === '>') { mode = 'text'; i++; continue }
    if (c === '<' && text.startsWith('<!--', i)) { i = skipHtmlComment(i); continue }
    if (c === '"' || c === "'" || c === '`') {
      const before = text.slice(Math.max(0, i - 120), i)
      const bindM = c !== '`' ? bindingClassAttrName.exec(before) : null
      const staticM = c !== '`' ? classAttrName.exec(before) : null
      if (bindM && isClassAttrName(bindM[1])) {
        const r = findStringEnd(text, i, n, scanCodeRange)
        if (r.unterminated) errors.push({ offset: i, kind: 'unterminated-string' })
        drillBinding(i, r.close)
        i = r.close + 1
        continue
      }
      if (staticM && isClassAttrName(staticM[1])) {
        const r = findStringEnd(text, i, n, scanCodeRange)
        if (r.unterminated) errors.push({ offset: i, kind: 'unterminated-string' })
        pushLiteral(i, r.close)
        i = r.close + 1
        continue
      }
      if (anyAttrEq.test(before)) {
        // 非 class 属性值（title/aria-label/:style 等）：整体消费不判定。
        const r = findStringEnd(text, i, n, scanCodeRange)
        i = r.close + 1
        continue
      }
      // 前窗不是「属性名 = 」：杂散引号（如文本裸 < 误入标签态后的撇号），
      // 跳过该字符继续，不消费到 EOF。
      i++
      continue
    }
    i++
  }
  return { literals: out, errors }
}

// ---- 变体前缀切分 ----
// 变体链逐段消费：裸段（sm、hover、*、dark…）可带 arbitrary 括号尾巴
// （data-[slot=select-value]、[&_svg:not([class*='text-'])]）与 / 修饰（…/open），
// 每段后跟 ':'。返回 [完整变体前缀, 工具类名]。
// 裸变体段含命名变体（group-hover/item、peer-checked/name）与容器查询（@md、@lg/name）。
const bareVariantSegment = /^[A-Za-z0-9_*\-/@]+$/
function splitVariant(token) {
  let i = 0
  let prefixEnd = 0
  while (i < token.length) {
    let segEnd = i
    while (segEnd < token.length && token[segEnd] !== ':' && token[segEnd] !== '[') segEnd++
    const bare = token.slice(i, segEnd)
    if (bare && !bareVariantSegment.test(bare)) break
    if (token[segEnd] === '[') {
      let depth = 0
      let quote = null
      while (segEnd < token.length) {
        const c = token[segEnd]
        if (quote) {
          if (c === quote) quote = null
        } else if (c === "'" || c === '"') quote = c
        else if (c === '[') depth++
        else if (c === ']') {
          depth--
          if (depth === 0) { segEnd++; break }
        }
        segEnd++
      }
      if (depth !== 0) break // 未配平的 arbitrary 段：放弃切分
      // arbitrary 段后的 / 修饰（data-[state=open]/open:）。
      if (token[segEnd] === '/') {
        segEnd++
        while (segEnd < token.length && /[A-Za-z0-9_-]/.test(token[segEnd])) segEnd++
      }
    } else if (!bare) break
    if (token[segEnd] !== ':') break // 段后无 ':'：余下是工具类名
    i = segEnd + 1
    prefixEnd = i
  }
  return [token.slice(0, prefixEnd), token.slice(prefixEnd)]
}

const clampUtility = /^line-clamp-(?:\d+|none|\[[^\]]+\])$/

// important 修饰符（v3 前缀 !block / v4 后缀 block!）：带 important 的声明确定性
// 覆盖不带的，与源序无关。clamp 带 important 而 display 不带 → clamp 必胜，合法；
// display 带 important 而 clamp 不带 → clamp 必败，确定缺陷；双方同带 → 源序竞争。
function stripImportant(utility) {
  if (utility.startsWith('!')) return { name: utility.slice(1), important: true }
  if (utility.endsWith('!')) return { name: utility.slice(0, -1), important: true }
  return { name: utility, important: false }
}

// 变体链段集合（前缀按 ':' 切段，arbitrary 段内冒号已在 splitVariant 处理）。
function variantSegments(prefix) {
  if (!prefix) return []
  const segs = []
  let depth = 0
  let quote = null
  let start = 0
  for (let i = 0; i < prefix.length; i++) {
    const c = prefix[i]
    if (quote) {
      if (c === quote) quote = null
      continue
    }
    if (c === "'" || c === '"') quote = c
    else if (c === '[' || c === '(') depth++
    else if (c === ']' || c === ')') depth--
    else if (c === ':' && depth === 0) {
      segs.push(prefix.slice(start, i))
      start = i + 1
    }
  }
  return segs
}

// 冲突判定（issue #1692 + PR #1693 维护者审查扩展）：
// 1. 同前缀：clamp 与 display 完整变体前缀逐字相同 → 源序竞争，报（important 可豁免）。
// 2. 超集：display 变体链段集合 ⊇ clamp 变体链段集合（含 clamp 无变体 + display 带
//    断点/状态变体）→ display 在 clamp 生效的全部语境额外激活，@media/状态规则源序
//    靠后覆盖 clamp，报（important 可豁免）。反方向（clamp 链更长）不报：clamp 在
//    更窄语境源序靠后获胜，合法。
// 返回 [{ prefix, clamps, displays, tokenOffsets }]；tokenOffsets 用于行号定位。
export function findLineClampConflicts(classString) {
  const entries = []
  let searchFrom = 0
  for (const token of classString.split(/\s+/)) {
    if (!token) continue
    const tokenOffset = classString.indexOf(token, searchFrom)
    searchFrom = tokenOffset + token.length
    if (token.includes('${')) continue // 模板插值不是静态类名
    const [prefix, utility] = splitVariant(token)
    const { name, important } = stripImportant(utility)
    const isClamp = clampUtility.test(name)
    const isDisplay = displayUtilities.has(name)
    if (!isClamp && !isDisplay) continue
    entries.push({ token, tokenOffset, prefix, segs: variantSegments(prefix), utility, important, isClamp, isDisplay })
  }
  const conflicts = []
  const reported = new Set()
  for (const c of entries.filter((e) => e.isClamp)) {
    for (const d of entries.filter((e) => e.isDisplay)) {
      const samePrefix = c.prefix === d.prefix
      const superset =
        !samePrefix && c.segs.every((s) => d.segs.includes(s)) && d.segs.length > c.segs.length
      if (!samePrefix && !superset) continue
      // clamp 带 important 而 display 不带 → clamp 确定性获胜，合法。
      if (c.important && !d.important) continue
      const key = `${c.token}\u0000${d.token}`
      if (reported.has(key)) continue
      reported.add(key)
      conflicts.push({
        prefix: samePrefix ? c.prefix : `${c.prefix || '(no variant)'} ⊂ ${d.prefix}`,
        clamps: [c.utility],
        displays: [d.utility],
        tokenOffset: d.tokenOffset,
      })
    }
  }
  return conflicts
}

// files: [{ relPath, text }]；返回 [{ relPath, line, prefix, clamps, displays }]。
// 结构错误（未闭合字面量）以 kind: 'scan-error' 条目返回，主入口 fail loud。
export function scanLineClampConflicts(files, allowlist = lineClampAllowlist) {
  const findings = []
  for (const { relPath, text } of files) {
    if (allowlist.has(relPath)) continue
    const codeOnly = relPath.endsWith('.ts')
    const { literals, errors } = extractStringLiterals(text, { codeOnly })
    const lineAt = (offset) => text.slice(0, offset).split('\n').length
    for (const e of errors) {
      findings.push({ relPath, line: lineAt(e.offset), kind: 'scan-error', detail: e.kind })
    }
    for (const { value, offset } of literals) {
      if (!value.includes('line-clamp-')) continue
      for (const conflict of findLineClampConflicts(value)) {
        // 行号定位到冲突 token（多行 class 串不指偏）。
        const line = lineAt(offset + 1 + conflict.tokenOffset)
        findings.push({ relPath, line, prefix: conflict.prefix, clamps: conflict.clamps, displays: conflict.displays })
      }
    }
  }
  return findings
}

function main() {
  const files = collectSourceFiles()
  const findings = scanLineClampConflicts(files)
  const scanErrors = findings.filter((f) => f.kind === 'scan-error')
  const conflicts = findings.filter((f) => f.kind !== 'scan-error')
  if (scanErrors.length > 0) {
    for (const f of scanErrors) {
      console.error(`[line-clamp] ${f.relPath}:${f.line} scan error: ${f.detail} (unterminated string/template literal)`)
    }
    console.error('[line-clamp] FAIL: scanner could not parse these files; fix the literal or add a reviewed allowlist entry')
    process.exit(1)
  }
  if (conflicts.length > 0) {
    for (const f of conflicts) {
      console.error(`[line-clamp] ${f.relPath}:${f.line} prefix ${f.prefix || '(no variant)'}`)
      console.error(`    ${f.clamps.join(', ')} conflicts with display: ${f.displays.join(', ')}`)
    }
    console.error(
      '[line-clamp] FAIL: line-clamp-* and display utilities share or extend a variant prefix in one class string; ' +
        'move display/visibility to a parent/child element (see docs/development/coding-conventions.md)',
    )
    process.exit(1)
  }
  console.log(`[line-clamp] OK: ${files.length} files scanned, no same-prefix/superset line-clamp/display conflicts`)
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main()
