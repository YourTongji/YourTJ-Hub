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

// 提取 class 语境的候选串。单遍扫描、同层配对：
// - `:attr="expr"` / `v-bind:attr="expr"` 是绑定表达式：外层双引号只圈定表达式边界，
//   内部单引号/反引号字面量才是候选串（三元互斥分支各自成串，AC-4）。
//   表达式整体消费，内部引号绝不与后续 HTML 伪配对（UserCard.vue :style 形态）。
// - `class="..."` 静态属性与其他双引号串按字面量整串收集。
// - 单引号串是 JS 字面量，内部引号字符属于内容（如 [&_svg:not([class*='text-'])]），不下钻。
// - 文本插值 {{...}} 是 JS 表达式，整体跳过（其中 t('a.b') 的引号不参与配对）。
function extractStringLiterals(text) {
  const out = []
  const n = text.length
  // 回看窗口可能从长属性名中间截断（如多行 :style 对象），故不锚定冒号起点：
  // 只要紧邻开引号前是「标识符尾巴 = 」即按绑定表达式处理。
  // class= 不含冒号也不会误入（其前是字母而非冒号）——用冒号存在性区分。
  const bindingAttr = /:[A-Za-z0-9_.\-[\]]*\s*=\s*$/
  let i = 0
  while (i < n) {
    const c = text[i]
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
    if (c === '"') {
      const close = findStringEnd(text, i, n)
      const before = text.slice(Math.max(0, i - 120), i)
      if (bindingAttr.test(before)) {
        // 绑定表达式：只收集内部嵌套字面量，外层不判定。
        let j = i + 1
        while (j < close) {
          const d = text[j]
          if (d === "'" || d === '`') {
            const innerClose = findStringEnd(text, j, close)
            out.push({ value: text.slice(j + 1, innerClose), offset: j })
            j = innerClose + 1
            continue
          }
          j++
        }
      } else {
        out.push({ value: text.slice(i + 1, close), offset: i })
      }
      i = close + 1
      continue
    }
    if (c === "'" || c === '`') {
      const close = findStringEnd(text, i, n)
      out.push({ value: text.slice(i + 1, close), offset: i })
      i = close + 1
      continue
    }
    i++
  }
  return out
}

// ---- 变体前缀切分 ----
// 变体链逐段消费：裸段（sm、hover、*、dark…）可带 arbitrary 括号尾巴
// （data-[slot=select-value]、[&_svg:not([class*='text-'])]），每段后跟 ':'。
// 返回 [完整变体前缀, 工具类名]；#1677 B 形态要求前缀逐字相同才判冲突。
const bareVariantSegment = /^[A-Za-z0-9_*-]+$/
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

// 对单个 class token 串判定冲突：按完整变体前缀分组，组内同时出现
// line-clamp-* 与裸 display 类即冲突。返回 [{ prefix, clamps, displays }]。
export function findLineClampConflicts(classString) {
  const groups = new Map()
  for (const token of classString.split(/\s+/)) {
    if (!token || token.includes('${')) continue // 模板插值不是静态类名
    const [prefix, utility] = splitVariant(token)
    const isClamp = clampUtility.test(utility)
    const isDisplay = displayUtilities.has(utility)
    if (!isClamp && !isDisplay) continue
    if (!groups.has(prefix)) groups.set(prefix, { clamps: [], displays: [] })
    const g = groups.get(prefix)
    if (isClamp && !g.clamps.includes(utility)) g.clamps.push(utility)
    if (isDisplay && !g.displays.includes(utility)) g.displays.push(utility)
  }
  const conflicts = []
  for (const [prefix, g] of groups) {
    if (g.clamps.length > 0 && g.displays.length > 0) {
      conflicts.push({ prefix, clamps: g.clamps, displays: g.displays })
    }
  }
  return conflicts
}

// files: [{ relPath, text }]；返回 [{ relPath, line, prefix, clamps, displays }]。
export function scanLineClampConflicts(files, allowlist = lineClampAllowlist) {
  const findings = []
  for (const { relPath, text } of files) {
    if (allowlist.has(relPath)) continue
    const masked = maskComments(text)
    for (const { value, offset } of extractStringLiterals(masked)) {
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