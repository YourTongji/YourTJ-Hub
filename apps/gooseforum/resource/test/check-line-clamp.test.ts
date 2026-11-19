// line-clamp × display 同串冲突门禁（issue #1692）。
// 正例 fixture 是 #1688 修复前的两条真实旧类串（周课表课名、管理端共享 Select 选中值），
// 门禁必须先红后绿；负例覆盖变体前缀不同与条件分支互斥两类合法写法（AC-4）。
import { describe, expect, it } from 'vitest'
import {
  collectSourceFiles,
  findLineClampConflicts,
  lineClampAllowlist,
  scanLineClampConflicts,
} from '../scripts/check-line-clamp.mjs'

// #1688 修复前 apps/gooseforum/resource/src/site/components/schedule/ScheduleTimeGrid.vue 课名 h3：
// 裸 block 与 line-clamp-2 同串，display:block 按源序覆盖截断。
const scheduleOldClass =
  'block font-semibold tracking-tight text-[11px] sm:text-xs md:text-[12.5px] leading-tight md:leading-snug line-clamp-2 break-all text-[var(--card-title)]'

// #1688 修复前 apps/gooseforum/resource/src/admin/components/ui/select/SelectTrigger.vue：
// 完整变体前缀 *:data-[slot=select-value] 下 flex 覆盖 line-clamp-1（#1677 B 形态）。
const selectOldClass =
  'border-input flex w-fit items-center justify-between gap-2 rounded-md border bg-transparent px-3 py-2 text-sm whitespace-nowrap shadow-xs outline-none *:data-[slot=select-value]:line-clamp-1 *:data-[slot=select-value]:flex *:data-[slot=select-value]:items-center *:data-[slot=select-value]:gap-2 [&_svg]:pointer-events-none [&_svg]:shrink-0'

describe('findLineClampConflicts（AC-1：#1688 旧类串必须报错）', () => {
  it('课名旧串：裸 block 与 line-clamp-2 同前缀共存', () => {
    const conflicts = findLineClampConflicts(scheduleOldClass)
    expect(conflicts).toHaveLength(1)
    expect(conflicts[0].prefix).toBe('')
    expect(conflicts[0].clamps).toEqual(['line-clamp-2'])
    expect(conflicts[0].displays).toEqual(['block'])
  })

  it('Select 旧串：同变体前缀下 flex 与 line-clamp-1 共存', () => {
    const conflicts = findLineClampConflicts(selectOldClass)
    expect(conflicts).toHaveLength(1)
    expect(conflicts[0].prefix).toBe('*:data-[slot=select-value]:')
    expect(conflicts[0].clamps).toEqual(['line-clamp-1'])
    expect(conflicts[0].displays).toEqual(['flex'])
  })

  it('line-clamp-none 同样写 display，参与冲突判定', () => {
    const conflicts = findLineClampConflicts('line-clamp-none hidden')
    expect(conflicts).toHaveLength(1)
    expect(conflicts[0].clamps).toEqual(['line-clamp-none'])
    expect(conflicts[0].displays).toEqual(['hidden'])
  })

  it('报出文件相对路径与行号', () => {
    const text = `<template>\n  <h3 class="${scheduleOldClass}">x</h3>\n</template>\n`
    const findings = scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])
    expect(findings).toHaveLength(1)
    expect(findings[0].relPath).toBe('src/site/components/Fixture.vue')
    expect(findings[0].line).toBe(2)
  })

  it('important 修饰符：display 带 important 而 clamp 不带 → 确定缺陷，报错', () => {
    // v4 后缀与 v3 前缀两种写法；important 确定性覆盖，与源序无关。
    expect(findLineClampConflicts('block! line-clamp-2')).toHaveLength(1)
    expect(findLineClampConflicts('!block line-clamp-2')).toHaveLength(1)
    // 双方同带 important → 回到源序竞争，报错。
    expect(findLineClampConflicts('line-clamp-2! block!')).toHaveLength(1)
    expect(findLineClampConflicts('!line-clamp-2 !block')).toHaveLength(1)
  })

  it('命名变体与容器查询：同前缀冲突报错（group-hover/item、@md、@lg/name）', () => {
    const named = findLineClampConflicts('group-hover/item:line-clamp-2 group-hover/item:flex')
    expect(named).toHaveLength(1)
    expect(named[0].prefix).toBe('group-hover/item:')
    const container = findLineClampConflicts('@md:line-clamp-2 @md:block')
    expect(container).toHaveLength(1)
    expect(container[0].prefix).toBe('@md:')
    expect(findLineClampConflicts('@lg/name:line-clamp-2 @lg/name:flex')).toHaveLength(1)
    expect(findLineClampConflicts('peer-checked/name:line-clamp-1 peer-checked/name:inline-flex')).toHaveLength(1)
  })

  it('单引号外层绑定属性：真实冲突分支报错', () => {
    const text = `<div :class='cond ? "x" : "block line-clamp-2"'></div>`
    const findings = scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])
    expect(findings).toHaveLength(1)
    expect(findings[0].displays).toEqual(['block'])
  })

  it('文本节点撇号不影响相邻元素真实冲突的检出', () => {
    const text = `<p>don't</p>\n<p class="block line-clamp-2">x</p>\n<p>can't</p>`
    const findings = scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])
    expect(findings).toHaveLength(1)
    expect(findings[0].line).toBe(2)
  })
})

describe('findLineClampConflicts（AC-4：合法写法零误报）', () => {
  it('变体前缀不同不报：sm:line-clamp-2 + 裸 block', () => {
    expect(findLineClampConflicts('sm:line-clamp-2 block text-sm')).toEqual([])
  })

  it('变体前缀逐字比较：仅差一个变体段也不报', () => {
    expect(
      findLineClampConflicts('*:data-[slot=select-value]:line-clamp-1 data-[slot=select-value]:flex'),
    ).toEqual([])
  })

  it('#1688 修复后的课名与 Select 新串全绿', () => {
    expect(findLineClampConflicts(scheduleOldClass.replace(/^block /, ''))).toEqual([])
    expect(
      findLineClampConflicts(selectOldClass.replace('*:data-[slot=select-value]:line-clamp-1 ', '')),
    ).toEqual([])
  })

  it('条件分支互斥类串逐串扫描不误报', () => {
    // ModerationAppealQueue.vue 形态：展开分支不带 clamp，收起分支不带 display 类。
    const text = `:class="expanded ? 'break-words whitespace-pre-wrap' : 'line-clamp-4 break-words'"`
    expect(scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])).toEqual([])
    // ModerationPage.vue 形态：三元一支带 clamp、另一支带 display。
    const ternary = `:class="isChat ? 'block whitespace-pre-wrap' : 'line-clamp-2 break-all'"`
    expect(scanLineClampConflicts([{ relPath: 'src/site/pages/Fixture.vue', text: ternary }])).toEqual([])
  })

  it('注释文本中的类名不参与判定', () => {
    const text = [
      '// hidden/block 与 line-clamp-* 都写 display，同元素按源序互相覆盖（机制说明）',
      '<!-- block line-clamp-2 是历史缺陷形态，可见性放外层 span -->',
      '/* flex line-clamp-1 */',
    ].join('\n')
    expect(scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])).toEqual([])
  })

  it('模板插值与文本插值内的引号不产生跨元素假串（UserCard.vue 形态）', () => {
    // :style 对象的模板插值里含单引号/花括号，其后另一元素的 class 串不得被拼接判定。
    const text = [
      `<div :style="{ left: \`\${position.left}px\`, ...coverStyle }">`,
      `  <span>{{ t('userCard.someKey') }}</span>`,
      `  <p class="line-clamp-2 text-sm">x</p>`,
      `  <div class="flex items-center">y</div>`,
      `</div>`,
    ].join('\n')
    expect(scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])).toEqual([])
  })

  it('display 工具类精确匹配，不误伤同前缀的其他工具类', () => {
    expect(findLineClampConflicts('line-clamp-2 inline-block')).toHaveLength(1)
    expect(findLineClampConflicts('line-clamp-2 inline-flex')).toHaveLength(1)
    expect(findLineClampConflicts('line-clamp-2 text-sm inline-decoration')).toEqual([])
  })

  it('important 修饰符：clamp 带 important 而 display 不带 → clamp 必胜，合法', () => {
    expect(findLineClampConflicts('line-clamp-2! block')).toEqual([])
    expect(findLineClampConflicts('!line-clamp-2 block')).toEqual([])
    expect(findLineClampConflicts('!line-clamp-2 block!')).toHaveLength(1) // display 也带 → 源序竞争
  })

  it('命名变体前缀不同不报：group-hover/item 与裸 group-hover 是不同前缀', () => {
    expect(findLineClampConflicts('group-hover/item:line-clamp-2 group-hover:flex')).toEqual([])
  })

  it('文本节点撇号不产生跨元素假串（don\'t … can\'t 形态）', () => {
    // 两个撇号若被当作字符串定界符，会把中间两个元素的合法类串拼成假串。
    const text = `<p>don't</p>\n<p class="line-clamp-2">x</p>\n<div class="flex">y</div>\n<p>can't</p>`
    expect(scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])).toEqual([])
    // 文本里直接提到类名也不判定。
    const prose = `<p>don't use block line-clamp-2 together can't</p>`
    expect(scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text: prose }])).toEqual([])
  })

  it('单引号外层绑定的互斥分支不误报', () => {
    const text = `<div :class='cond ? "block" : "line-clamp-2"'></div>`
    expect(scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])).toEqual([])
  })

  it('非 class 属性值不判定：title/aria-label/data-* 提到类名不误报', () => {
    const text = [
      `<div title="use block line-clamp-2 together" aria-label="block line-clamp-2" data-hint="flex line-clamp-1">`,
      `  <p class="line-clamp-2">x</p>`,
      `</div>`,
    ].join('\n')
    expect(scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])).toEqual([])
  })

  it(':style 等非 class 绑定表达式不判定', () => {
    const text = `<div :style="{ content: 'block line-clamp-2' }"><p class="line-clamp-2">x</p></div>`
    expect(scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])).toEqual([])
  })
})

describe('class 类属性与 .ts 覆盖面', () => {
  it('*-class 属性（img-class、enter-active-class）参与判定', () => {
    const img = `<UserAvatar img-class="block line-clamp-2" />`
    expect(scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text: img }])).toHaveLength(1)
    const transition = `<Transition enter-active-class="flex line-clamp-1"><p>x</p></Transition>`
    expect(scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text: transition }])).toHaveLength(1)
  })

  it('.ts 纯代码文件：普通字符串字面量参与判定', () => {
    const ts = `export const cls = 'block line-clamp-2'\nexport const ok = 'line-clamp-2 text-sm'\n`
    const findings = scanLineClampConflicts([{ relPath: 'src/site/utils/fixture.ts', text: ts }])
    expect(findings).toHaveLength(1)
    expect(findings[0].line).toBe(1)
  })

  it('.vue 的 script 块内类串参与判定', () => {
    const vue = `<script setup lang="ts">\nconst cls = 'block line-clamp-2'\n</script>\n<template><p :class="cls">x</p></template>`
    const findings = scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text: vue }])
    expect(findings).toHaveLength(1)
    expect(findings[0].line).toBe(2)
  })
})

describe('scanLineClampConflicts（AC-2：当前 src 全量零冲突）', () => {
  it('豁免清单初始为空', () => {
    expect(lineClampAllowlist.size).toBe(0)
  })

  it('对当前 src/**/*.{vue,ts} 全量扫描零报错', () => {
    const files = collectSourceFiles()
    expect(files.length).toBeGreaterThan(100)
    expect(scanLineClampConflicts(files)).toEqual([])
  })
})

// ---- PR #1693 维护者审查（WALKERKILLER，head 609eb602b）八条发现的复现回归 ----

describe('正则字面量不吞文件（审查 finding 1）', () => {
  it('含引号的正则之后的类串仍被扫描', () => {
    const ts = `const t = (x) => x.replace(/'/g, '')\nexport const C = 'flex line-clamp-2'\n`
    const findings = scanLineClampConflicts([{ relPath: 'src/runtime/fixture.ts', text: ts }])
    expect(findings).toHaveLength(1)
    expect(findings[0].line).toBe(2)
  })

  it('现网六种正则形态之后的类串均被扫描（markdown-import/appearance-settings/mention/wiki/blank-lines/pkExport）', () => {
    const regexes = [
      `/^(['"])(.*)\\1$/`,
      `/["']/g`,
      `/[/\\\\:*?"<>|]/`,
      `/^[=+\\-@\\t\\r]/`,
      `/\\n{3,}/g`,
      `/@(\\w+)/g`,
    ]
    for (const re of regexes) {
      const ts = `const re = ${re}\nexport const C = 'flex line-clamp-2'\n`
      const findings = scanLineClampConflicts([{ relPath: 'src/runtime/fixture.ts', text: ts }])
      expect(findings, `regex ${re}`).toHaveLength(1)
    }
  })

  it('模板插值内含引号的正则不破坏配平（appearance-settings/pkExport 真实形态）', () => {
    const ts = [
      'function q(name: string): string {',
      '  return `"${name.replace(/["\']/g, \'\')}"`',
      '}',
      `export const C = 'flex line-clamp-2'`,
    ].join('\n')
    const findings = scanLineClampConflicts([{ relPath: 'src/runtime/fixture.ts', text: ts }])
    expect(findings).toHaveLength(1)
    expect(findings[0].line).toBe(4)
  })

  it('除号不误判为正则：a / b 之后的类串仍被扫描', () => {
    const ts = `const half = total / 2\nexport const C = 'flex line-clamp-2'\n`
    const findings = scanLineClampConflicts([{ relPath: 'src/runtime/fixture.ts', text: ts }])
    expect(findings).toHaveLength(1)
  })

  it('深层嵌套模板线性完成（30 层不挂死，插值区只扫一遍）', () => {
    const ts = `const a = ${'`x${'.repeat(30)}1${'}`'.repeat(30)}\nexport const C = 'flex line-clamp-2'\n`
    const start = Date.now()
    const findings = scanLineClampConflicts([{ relPath: 'src/runtime/fixture.ts', text: ts }])
    expect(Date.now() - start).toBeLessThan(2000)
    expect(findings).toHaveLength(1)
    expect(findings[0].line).toBe(2)
  })
})

describe('注释屏蔽与文本态（审查 finding 2/3）', () => {
  it('撇号奇数时 HTML 注释内的旧冲突串不误报', () => {
    const text = [
      `<p>Don't truncate</p>`,
      `<!-- <p class="flex line-clamp-2">old</p> -->`,
      `<p class="line-clamp-2">new</p>`,
    ].join('\n')
    expect(scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])).toEqual([])
  })

  it('文本裸 < 不吞掉后续真实冲突', () => {
    const text = [
      `<p>余额 < 10 元，don't panic</p>`,
      `<div class="flex line-clamp-2">REAL CONFLICT</div>`,
    ].join('\n')
    const findings = scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])
    expect(findings).toHaveLength(1)
    expect(findings[0].line).toBe(2)
  })

  it('未闭合字面量 fail loud（scan-error），不静默吞文件', () => {
    const ts = `export const broken = 'unterminated\nexport const C = 'flex line-clamp-2'\n`
    const findings = scanLineClampConflicts([{ relPath: 'src/runtime/fixture.ts', text: ts }])
    expect(findings.some((f) => f.kind === 'scan-error')).toBe(true)
  })
})

describe('跨断点 display 覆盖（审查 finding 4：超集规则）', () => {
  it('无变体 clamp + 断点 display → 报错（桌面端截断失效）', () => {
    expect(findLineClampConflicts('line-clamp-2 sm:block')).toHaveLength(1)
    expect(findLineClampConflicts('line-clamp-2 sm:flex')).toHaveLength(1)
    expect(findLineClampConflicts('line-clamp-2 md:grid lg:hidden')).toHaveLength(2)
  })

  it('状态变体超集同样报错：hover:flex 覆盖基础 clamp', () => {
    expect(findLineClampConflicts('line-clamp-2 hover:flex')).toHaveLength(1)
    expect(findLineClampConflicts('sm:line-clamp-2 sm:hover:block')).toHaveLength(1)
  })

  it('反方向合法：clamp 变体链更长不报（sm:line-clamp-2 + 裸 block）', () => {
    expect(findLineClampConflicts('sm:line-clamp-2 block')).toEqual([])
    expect(findLineClampConflicts('sm:hover:line-clamp-2 sm:block')).toEqual([])
  })

  it('不相关变体链不报：sm:line-clamp-2 + md:block 语境不相交', () => {
    expect(findLineClampConflicts('sm:line-clamp-2 md:block')).toEqual([])
  })

  it('important 豁免超集：clamp 带 important 确定性获胜', () => {
    expect(findLineClampConflicts('line-clamp-2! sm:block')).toEqual([])
    expect(findLineClampConflicts('line-clamp-2 sm:block!')).toHaveLength(1)
  })
})

describe('data-*/aria-* 排除与行号定位（审查 finding 5/8）', () => {
  it('data-class / :data-class 不是 class 列表，不判定', () => {
    expect(scanLineClampConflicts([{ relPath: 'F.vue', text: `<div data-class="flex line-clamp-2"></div>` }])).toEqual([])
    expect(scanLineClampConflicts([{ relPath: 'F.vue', text: `<div :data-class="'flex line-clamp-2'"></div>` }])).toEqual([])
    expect(scanLineClampConflicts([{ relPath: 'F.vue', text: `<div aria-class="flex line-clamp-2"></div>` }])).toEqual([])
  })

  it('多行 class 串行号定位到冲突 token 而非开引号', () => {
    const text = `<p class="mt-1.5\n  line-clamp-2\n  flex\n  text-xs">x</p>`
    const findings = scanLineClampConflicts([{ relPath: 'src/site/components/Fixture.vue', text }])
    expect(findings).toHaveLength(1)
    expect(findings[0].line).toBe(3) // flex 在第 3 行，不是开引号的第 1 行
  })
})

describe('arbitrary 变体 / 修饰（oryn-mini finding）', () => {
  it('data-[state=open]/open: 同前缀冲突报错', () => {
    const conflicts = findLineClampConflicts('data-[state=open]/open:line-clamp-2 data-[state=open]/open:flex')
    expect(conflicts).toHaveLength(1)
    expect(conflicts[0].prefix).toBe('data-[state=open]/open:')
  })
})

// ---- PR #1693 维护者第二轮审查（head 0d9f092ab）两条误报的复现回归 ----

describe('结构性变体不参与超集比较（第二轮行评 1）', () => {
  it('display 落子元素的结构性变体不误报', () => {
    expect(findLineClampConflicts('line-clamp-2 [&_svg]:block')).toEqual([])
    expect(findLineClampConflicts('line-clamp-2 *:flex')).toEqual([])
    expect(findLineClampConflicts('line-clamp-2 [&>*]:flex')).toEqual([])
    expect(findLineClampConflicts('line-clamp-2 [&+*]:block')).toEqual([])
    expect(findLineClampConflicts('line-clamp-2 [&~*]:block')).toEqual([])
    expect(findLineClampConflicts('line-clamp-2 **:grid')).toEqual([])
  })

  it('[&:hover] 仍指元素自身，超集照常报', () => {
    expect(findLineClampConflicts('line-clamp-2 [&:hover]:block')).toHaveLength(1)
  })

  it('同前缀规则不动：结构性变体同前缀仍是真冲突（#1688 用例保持）', () => {
    expect(findLineClampConflicts('*:line-clamp-1 *:flex')).toHaveLength(1)
    expect(findLineClampConflicts('[&_svg]:line-clamp-2 [&_svg]:block')).toHaveLength(1)
  })

  it('clamp 带结构性变体 + 裸 display：反方向不报（clamp 链更长）', () => {
    expect(findLineClampConflicts('[&_svg]:line-clamp-2 block')).toEqual([])
  })
})

describe('闭括号后除号不误判正则（第二轮行评 2）', () => {
  it('f(x) / c 后同行含路径字符串与类串：不产 scan-error，冲突照报', () => {
    const ts = `const r = f(x) / c, p = 'a/b', cls = 'flex line-clamp-2'\n`
    const findings = scanLineClampConflicts([{ relPath: 'src/runtime/fixture.ts', text: ts }])
    expect(findings.some((f) => f.kind === 'scan-error')).toBe(false)
    expect(findings).toHaveLength(1)
  })

  it('arr[0] / c 与 (a+b) / c 后同行 // 注释：不产 scan-error', () => {
    const ts = [
      `const a = arr[0] / c // path a/b`,
      `const b = (a+b) / c // http://x`,
      `export const C = 'flex line-clamp-2'`,
    ].join('\n')
    const findings = scanLineClampConflicts([{ relPath: 'src/runtime/fixture.ts', text: ts }])
    expect(findings.some((f) => f.kind === 'scan-error')).toBe(false)
    expect(findings).toHaveLength(1)
    expect(findings[0].line).toBe(3)
  })

  it('if (x) /re/.test(y) 语句头后的正则仍被识别', () => {
    const ts = `if (x) /'/.test(y)\nexport const C = 'flex line-clamp-2'\n`
    const findings = scanLineClampConflicts([{ relPath: 'src/runtime/fixture.ts', text: ts }])
    expect(findings.some((f) => f.kind === 'scan-error')).toBe(false)
    expect(findings).toHaveLength(1)
    expect(findings[0].line).toBe(2)
  })

  it('replace(/…/) 首参正则不受影响（走开括号分支）', () => {
    const ts = `const t = (x) => x.replace(/'/g, '')\nexport const C = 'flex line-clamp-2'\n`
    const findings = scanLineClampConflicts([{ relPath: 'src/runtime/fixture.ts', text: ts }])
    expect(findings).toHaveLength(1)
    expect(findings[0].line).toBe(2)
  })
})
