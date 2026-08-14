// 四语言 locale 键差集校验（issue #225/#230）：
// 扫描 src/locales/{zh,en,ja,it}.ts 的顶层块，收集每个块下的嵌套键路径
// （如 schedule.arrangedCount），校验四语言键集合完全一致，防止键名泄漏回归。
// 用法：node scripts/check-i18n-keys.mjs（作为 pnpm check:i18n 运行）
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const locales = ['zh', 'en', 'ja', 'it']

/**
 * 粗解析 TS locale 文件为嵌套键集合。
 * 支持 `key: 'value'`、`'key': 'value'`（引号键）与行内对象
 * `weekdays: { mon: 'x', tue: 'y' }`（单行子键展开），以 2 空格缩进推断层级。
 * 忽略类型标注/注释行。
 */
function collectKeys(file) {
  const src = readFileSync(file, 'utf8')
  const keys = new Set()
  const stack = []
  const indentRe = /^([ ]*)(?:'?([A-Za-z][A-Za-z0-9]*)'?):/
  const inlineObjRe = /^[ ]*(?:'?([A-Za-z][A-Za-z0-9]*)'?):\s*\{([^}]*)\},?$/
  for (const raw of src.split('\n')) {
    const line = raw.trimEnd()
    if (line.trimStart().startsWith('//')) continue

    // 行内对象：key: { a: 'x', b: 'y' },  → 展开子键（值内可能含 : 但键模式受限）
    const inline = inlineObjRe.exec(line)
    if (inline) {
      const parentPath = [...stack.map((s) => s.name), inline[1]].join('.')
      for (const part of inline[2].split(',')) {
        const km = /^\s*(?:'?([A-Za-z][A-Za-z0-9]*)'?):\s*['"`]/.exec(part)
        if (km) keys.add(`${parentPath}.${km[1]}`)
      }
      continue
    }

    const m = indentRe.exec(line)
    if (!m) continue
    const indent = m[1].length
    const name = m[2]
    // 维护缩进栈
    while (stack.length > 0 && stack[stack.length - 1].indent >= indent) stack.pop()
    const path = [...stack.map((s) => s.name), name].join('.')
    if (line.trimEnd().endsWith('{')) {
      stack.push({ name, indent })
    } else {
      keys.add(path)
    }
  }
  return keys
}

const sets = {}
for (const lang of locales) {
  sets[lang] = collectKeys(join(root, 'src', 'locales', `${lang}.ts`))
}

const all = new Set()
for (const lang of locales) for (const k of sets[lang]) all.add(k)

let failed = false
for (const lang of locales) {
  const missing = [...all].filter((k) => !sets[lang].has(k))
  if (missing.length > 0) {
    failed = true
    console.error(`[i18n] ${lang}.ts missing ${missing.length} keys:`)
    for (const k of missing) console.error(`  - ${k}`)
  }
}
if (failed) {
  console.error('[i18n] FAIL: locale key sets diverge')
  process.exit(1)
}

// ---- 第二道：代码静态 t() 引用 vs locale 键差集校验 ----
// 扫描 src/**/*.{vue,ts} 中 t('a.b.c') / t("a.b.c") 静态键，
// 防"引用未注册键 → 生产渲染字面键名"回归（issue #225 同类问题）。
import { readdirSync, statSync } from 'node:fs'

function collectFiles(dir) {
  const out = []
  for (const name of readdirSync(dir)) {
    if (name === 'node_modules' || name === 'locales') continue
    const p = join(dir, name)
    const st = statSync(p)
    if (st.isDirectory()) out.push(...collectFiles(p))
    else if (/\.(vue|ts)$/.test(name)) out.push(p)
  }
  return out
}

const staticRefs = new Map() // key -> [files]
for (const file of collectFiles(join(root, 'src'))) {
  const text = readFileSync(file, 'utf8')
  // t('a.b.c') / t("a.b.c")；排除 t(`...`) 模板串与 t('...', 带选项的已注册键（含命名参数）
  for (const m of text.matchAll(/\bt\(['"]([A-Za-z0-9_.-]+)['"]\)/g)) {
    const key = m[1]
    if (!staticRefs.has(key)) staticRefs.set(key, [])
    staticRefs.get(key).push(file)
  }
}

let refFailed = false
for (const [key, files] of staticRefs) {
  if (!all.has(key)) {
    refFailed = true
    console.error(`[i18n] t() references unregistered key: ${key}`)
    console.error(`    used at: ${files.slice(0, 4).join(', ')}`)
  }
}
if (refFailed) {
  console.error('[i18n] FAIL: unregistered t() keys found')
  process.exit(1)
}
console.log(`[i18n] OK: ${all.size} keys × ${locales.length} locales consistent; ${staticRefs.size} static t() refs all registered`)
