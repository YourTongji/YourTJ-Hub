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