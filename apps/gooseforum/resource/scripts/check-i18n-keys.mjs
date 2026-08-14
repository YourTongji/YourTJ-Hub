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
 * 只识别 `key: 'value'` / `key: {` 形式的字面量（忽略类型/注释行），
 * 以 2 空格缩进推断层级。
 */
function collectKeys(file) {
  const src = readFileSync(file, 'utf8')
  const keys = new Set()
  const stack = []
  const indentRe = /^([ ]*)([A-Za-z][A-Za-z0-9]*):/
  for (const raw of src.split('\n')) {
    const line = raw.trimEnd()
    if (line.trimStart().startsWith('//')) continue
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
console.log(`[i18n] OK: ${all.size} keys × ${locales.length} locales consistent`)
