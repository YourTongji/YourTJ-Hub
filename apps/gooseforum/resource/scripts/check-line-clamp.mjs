// line-clamp × display 同串冲突门禁（issue #1692）：
// 同一元素上 line-clamp-* 与其他写 display 的工具类共存时，构建 CSS 同 specificity
// 按源序覆盖，行数截断会静默失效（缺陷链 #1639/#1640 → #1677/#1688）。
// 逐串扫描 src/** 的 .vue/.ts 引号内 class token 串（含 :class 字面量分支），
// 同一串内 line-clamp-N/none 与裸 display 类共存、且完整变体前缀逐字相同才报错。
// 边界（v1，见 docs/development/coding-conventions.md）：不合并同元素多属性
// （静态 class 与 :class 分写）、不追踪跨组件变体传导、不扫 gohtml 模板。
// 用法：node scripts/check-line-clamp.mjs（作为 pnpm check:line-clamp / pnpm check 运行）
import { readdirSync, readFileSync, statSync } from 'node:fs'
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

export function toPosix(p) {
  return p.split(sep).join('/')
}

export function collectSourceFiles() {
  const out = []
  const walk = (dir) => {
    for (const name of readdirSync(dir)) {
      if (name === 'node_modules') continue
      const p = join(dir, name)
      if (statSync(p).isDirectory()) walk(p)
      else if (/\.(vue|ts)$/.test(name)) {
        out.push({ relPath: toPosix(relative(root, p)), text: readFileSync(p, 'utf8') })
      }
    }
  }
  walk(join(root, 'src'))
  return out
}

// 屏蔽注释（HTML/块/行），注释里的机制说明不参与判定。等长空白保留行号。
// 单遍状态机：引号串内的 //（如 'https://…'）不是注释；只在引号外识别注释起点。
// 模板字符串的 ${...} 表达式区域按花括号深度跳过（内部引号不影响配平）。
function maskComments(text) {
  const chars = text.split('')
  const blank = (from, to) => {
    for (let k = from; k < to && k < chars.length; k++) {
      if (chars[k] !== '\n') chars[k] = ' '
    }
  }
  let i = 0
  const n = text.length
  while (i < n) {
    const c = text[i]
    if (c === "'" || c === '"' || c === '`') {
      // 跳过整个字符串字面量（同层闭引号；模板串跳过 ${...}）。
      let j = i + 1
      while (j < n) {
        const d = text[j]
        if (d === '\\') { j += 2; continue }
        if (c === '`' && d === '$' && text[j + 1] === '{') {
          let depth = 1
          j += 2
          let quote = null
          while (j < n && depth > 0) {
            const e = text[j]
            if (e === '\\') { j += 2; continue }
            if (quote) {
              if (e === quote) quote = null
            } else if (e === "'" || e === '"' || e === '`') quote = e
            else if (e === '{') depth++
            else if (e === '}') depth--
            j++
          }
          continue
        }
        if (d === c) break
        j++
      }
      i = j + 1
      continue
    }
    if (c === '<' && text.startsWith('<!--', i)) {
      const close = text.indexOf('-->', i + 4)
      const end = close === -1 ? n : close + 3
      blank(i, end)
      i = end
      continue
    }
    if (c === '/' && text[i + 1] === '*') {
      const close = text.indexOf('*/', i + 2)
      const end = close === -1 ? n : close + 2
      blank(i, end)
      i = end
      continue
    }
    if (c === '/' && text[i + 1] === '/') {
      let end = text.indexOf('\n', i)
      if (end === -1) end = n
      blank(i, end)
      i = end
      continue
    }
    i++
  }
  return chars.join('')
}

// 从开引号下标 open 找同层闭引号；模板字符串按花括号深度跳过 ${...}。
// 返回闭引号下标（未找到返回 end）。
function findStringEnd(text, open, end) {
  const q = text[open]
  let j = open + 1
  while (j < end) {
    const c = text[j]
    if (c === '\\') { j += 2; continue }
    if (q === '`' && c === '$' && text[j + 1] === '{') {
      let depth = 1
      j += 2
      let quote = null
      while (j < end && depth > 0) {
        const d = text[j]
        if (d === '\\') { j += 2; continue }
        if (quote) {
          if (d === quote) quote = null
        } else if (d === "'" || d === '"' || d === '`') quote = d
        else if (d === '{') depth++
        else if (d === '}') depth--
        j++
      }
      continue
    }
    if (c === q) return j
    j++
  }
  return end
}

// 提取 class 语境的候选串。.vue 走模板感知扫描，.ts 走纯代码扫描：
// - 标签内（< 与 > 之间）只收集 class 类属性（class / *-class，如 img-class、
//   enter-active-class）：`:class="expr"` 绑定外层引号只圈定边界，内部异种引号/
//   反引号字面量才是候选串（三元互斥分支各自成串，AC-4）；静态 class 整串收集。
//   title/aria-label/data-* 等自然语言属性值不判定（误报面收窄）。
// - 标签外（模板文本节点）：撇号是自然语言（don't / can't），不是字符串定界符，
//   整体跳过到下一个 '<'，防止两个撇号伪配对出跨元素假串（误报方向）。
// - script/style 块与 .ts 文件是 JS 语境：引号正常配对、全部字面量收集，
//   不受标签态影响（.ts 无模板文本，纯代码扫描保证覆盖面）。
// - 文本插值 {{...}} 是 JS 表达式，整体跳过（其中 t('a.b') 的引号不参与配对）。
// 已知边界：文本节点中裸露的 '<'（如 "a < b"）会提前进入标签态，属粗扫描可接受
// 漂移；Vue 模板要求转义裸 '<'，实际源码不出现。
// class / *-class（img-class、enter-active-class、leave-to-class 等多段前缀均覆盖）。
const classAttr = /(?:^|[\s<])(?:[A-Za-z0-9_]+-)*class\s*=\s*$/
const bindingClassAttr = /(?:^|[\s<])(?::|v-bind:)(?:[A-Za-z0-9_]+-)*class\s*=\s*$/
function extractStringLiterals(text, { codeOnly = false } = {}) {
  const out = []
  const n = text.length
  const collectQuoted = (open, close, isBinding) => {
    if (isBinding) {
      let j = open + 1
      while (j < close) {
        const d = text[j]
        if (d === '\\') { j += 2; continue }
        if ((d === "'" || d === '"' || d === '`') && d !== text[open]) {
          const innerClose = findStringEnd(text, j, close)
          out.push({ value: text.slice(j + 1, innerClose), offset: j })
          j = innerClose + 1
          continue
        }
        j++
      }
    } else {
      out.push({ value: text.slice(open + 1, close), offset: open })
    }
  }
  // script/style 块内是代码语境：找到块尾，整块按引号正常扫描。
  const codeBlockEnd = (start) => {
    const m = /^<(script|style)\b/i.exec(text.slice(start, start + 8))
    if (!m) return -1
    const closeRe = new RegExp(`</${m[1]}\\s*>`, 'i')
    const rest = text.slice(start)
    const cm = closeRe.exec(rest)
    return cm ? start + cm.index : n
  }
  let i = 0
  let inTag = false
  while (i < n) {
    const c = text[i]
    if (!codeOnly && !inTag && c === '<') {
      const blockEnd = codeBlockEnd(i)
      if (blockEnd !== -1) {
        // script/style：块内按代码语境扫描（引号配对），块尾后回到文本态。
        i = scanCode(text, i + 1, blockEnd, out, collectQuoted)
        continue
      }
      inTag = true
      i++
      continue
    }
    if (!codeOnly && inTag && c === '>') {
      inTag = false
      i++
      continue
    }
    if (!codeOnly && !inTag) {
      // 文本节点：跳过 {{...}} 与裸引号，直到下一个 '<'。
      if (c === '{' && text[i + 1] === '{') {
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
        i = j
        continue
      }
      i++
      continue
    }
    // 标签内：只收集 class 类属性值；代码语境（script/style/.ts）收集全部字面量。
    if (c === '"' || c === "'" || c === '`') {
      const close = findStringEnd(text, i, n)
      if (codeOnly) {
        collectQuoted(i, close, false)
      } else {
        const before = text.slice(Math.max(0, i - 120), i)
        if (c !== '`' && bindingClassAttr.test(before)) collectQuoted(i, close, true)
        else if (classAttr.test(before)) collectQuoted(i, close, false)
        // 其余属性值（title/aria-label/:style 等）整体跳过，不判定。
      }
      i = close + 1
      continue
    }
    i++
  }
  return out
}

// script/style 块内的代码语境扫描：引号正常配对收集全部字面量，{{ }} 不适用。
function scanCode(text, start, end, out, collectQuoted) {
  let i = start
  while (i < end) {
    const c = text[i]
    if (c === '"' || c === "'" || c === '`') {
      const close = findStringEnd(text, i, end)
      collectQuoted(i, close, false)
      i = close + 1
      continue
    }
    i++
  }
  return end
}

// ---- 变体前缀切分 ----
// 变体链逐段消费：裸段（sm、hover、*、dark…）可带 arbitrary 括号尾巴
// （data-[slot=select-value]、[&_svg:not([class*='text-'])]），每段后跟 ':'。
// 返回 [完整变体前缀, 工具类名]；#1677 B 形态要求前缀逐字相同才判冲突。
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
// display 带 important 而 clamp 不带 → clamp 必败，确定缺陷；双方同带 → 回到源序竞争。
function stripImportant(utility) {
  if (utility.startsWith('!')) return { name: utility.slice(1), important: true }
  if (utility.endsWith('!')) return { name: utility.slice(0, -1), important: true }
  return { name: utility, important: false }
}

// 对单个 class token 串判定冲突：按完整变体前缀分组，组内同时出现
// line-clamp-* 与裸 display 类、且 important 层级不构成 clamp 必胜时冲突。
// 返回 [{ prefix, clamps, displays }]。
export function findLineClampConflicts(classString) {
  const groups = new Map()
  for (const token of classString.split(/\s+/)) {
    if (!token || token.includes('${')) continue // 模板插值不是静态类名
    const [prefix, utility] = splitVariant(token)
    const { name, important } = stripImportant(utility)
    const isClamp = clampUtility.test(name)
    const isDisplay = displayUtilities.has(name)
    if (!isClamp && !isDisplay) continue
    if (!groups.has(prefix)) groups.set(prefix, { clamps: [], displays: [] })
    const g = groups.get(prefix)
    const entry = { utility, important }
    if (isClamp && !g.clamps.some((c) => c.utility === utility)) g.clamps.push(entry)
    if (isDisplay && !g.displays.some((d) => d.utility === utility)) g.displays.push(entry)
  }
  const conflicts = []
  for (const [prefix, g] of groups) {
    if (g.clamps.length === 0 || g.displays.length === 0) continue
    // clamp 全部带 important 且 display 全部不带 → clamp 确定性获胜，不报。
    const clampAlwaysWins =
      g.clamps.every((c) => c.important) && g.displays.every((d) => !d.important)
    if (clampAlwaysWins) continue
    conflicts.push({
      prefix,
      clamps: g.clamps.map((c) => c.utility),
      displays: g.displays.map((d) => d.utility),
    })
  }
  return conflicts
}

// files: [{ relPath, text }]；返回 [{ relPath, line, prefix, clamps, displays }]。
export function scanLineClampConflicts(files, allowlist = lineClampAllowlist) {
  const findings = []
  for (const { relPath, text } of files) {
    if (allowlist.has(relPath)) continue
    const masked = maskComments(text)
    // .ts 是纯代码文件（无模板文本），全部字面量参与判定；.vue 走模板感知扫描。
    const codeOnly = relPath.endsWith('.ts')
    for (const { value, offset } of extractStringLiterals(masked, { codeOnly })) {
      if (!value.includes('line-clamp-')) continue
      for (const conflict of findLineClampConflicts(value)) {
        const line = masked.slice(0, offset).split('\n').length
        findings.push({ relPath, line, ...conflict })
      }
    }
  }
  return findings
}

function main() {
  const files = collectSourceFiles()
  const findings = scanLineClampConflicts(files)
  if (findings.length > 0) {
    for (const f of findings) {
      const prefix = f.prefix || '(no variant)'
      console.error(`[line-clamp] ${f.relPath}:${f.line} prefix ${prefix}`)
      console.error(`    ${f.clamps.join(', ')} conflicts with display: ${f.displays.join(', ')}`)
    }
    console.error(
      '[line-clamp] FAIL: line-clamp-* and display utilities share a variant prefix in one class string; ' +
        'move display/visibility to a parent/child element (see docs/development/coding-conventions.md)',
    )
    process.exit(1)
  }
  console.log(`[line-clamp] OK: ${files.length} files scanned, no same-prefix line-clamp/display conflicts`)
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main()