// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'

import ScheduleTimeTable from '../src/site/components/schedule/ScheduleTimeTable.vue'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'
import type { PkCourseDetail, PkStagedCourse } from '../src/site/types/pk'

function makeDetail(code: string, day: number, time: number[], weeks: number[]): PkCourseDetail {
  return {
    code,
    campus: '四平路校区',
    teachers: [{ teacherCode: 'T1', teacherName: '张三' }],
    teachingLanguage: '中文',
    arrangementInfo: [
      {
        arrangementText: `周${day} ${time.join('-')}节`,
        occupyDay: day,
        occupyTime: time,
        occupyWeek: weeks,
        occupyRoom: 'A101',
        teacherAndCode: '张三(T1)',
      },
    ],
  }
}

function makeStaged(courseCode: string, name: string, details: PkCourseDetail[]): PkStagedCourse {
  return {
    courseCode,
    courseName: courseCode,
    courseNameReserved: name,
    credit: 3,
    courseType: '必',
    teacher: [],
    status: 0,
    courseDetail: details,
  }
}

function mountTable() {
  return mount(ScheduleTimeTable, {
    global: {
      plugins: [i18n],
    },
    attachTo: document.body,
  })
}

/** 同一格内的课程块：course 块是 role=button 且 aria-label=课名（空格 td 无课名 label）。 */
function cellBlocks(nameA: string, nameB?: string): HTMLElement[] {
  const blocks = [...document.querySelectorAll<HTMLElement>('[role="button"]')]
  return blocks.filter((el) => {
    const label = el.getAttribute('aria-label') ?? ''
    return Boolean((nameA && label.includes(nameA)) || (nameB && label.includes(nameB)))
  })
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ScheduleTimeTable 同格多课渲染', () => {
  beforeEach(() => {
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setWeekView({ week: null, useCurrent: false })
  })

  test('全部周次下同格两门课竖向堆叠（不再横向细条）', async () => {
    const store = useScheduleStore()
    // 同周一 3-4 节、周次重叠（真冲突，容忍式同格）。
    const detailA = makeDetail('100A.01', 1, [3, 4], [1, 8])
    const detailB = makeDetail('100B.01', 1, [3, 4], [1, 8])
    store.pushStagedCourse(makeStaged('100A', '课程甲', [detailA]))
    store.pushStagedCourse(makeStaged('100B', '课程乙', [detailB]))
    // 真实流程：点课/弹窗选班先写 clickedCourseInfo，appendToTimeTable 以此命名课表行。
    store.setClickedCourseInfo({ courseCode: '100A', courseName: '课程甲' })
    store.stageCourse(detailA)
    store.setClickedCourseInfo({ courseCode: '100B', courseName: '课程乙' })
    store.stageCourse(detailB)
    store.solidify()

    mountTable()
    await flushPromises()

    const blocks = cellBlocks('课程甲', '课程乙')
    expect(blocks).toHaveLength(2)
    // 两块同属一个格容器，容器为纵向 flex（竖排），不再 flex-row（横排细条）。
    const container = blocks[0].parentElement
    expect(container).toBeTruthy()
    expect(container!.className).toContain('flex-col')
    expect(container!.className).not.toContain('flex-row')
  })

  test('单双周同位共存：不判冲突且两门课都在该位置显示', async () => {
    const store = useScheduleStore()
    // 同周一 1-2 节：甲单周、乙双周——周次无交集，不构成冲突。
    const detailA = makeDetail('200A.01', 1, [1, 2], [1, 3, 5, 7])
    const detailB = makeDetail('200B.01', 1, [1, 2], [2, 4, 6, 8])
    store.pushStagedCourse(makeStaged('200A', '单周课', [detailA]))
    store.pushStagedCourse(makeStaged('200B', '双周课', [detailB]))
    store.setClickedCourseInfo({ courseCode: '200A', courseName: '单周课' })
    store.stageCourse(detailA)
    store.setClickedCourseInfo({ courseCode: '200B', courseName: '双周课' })
    store.stageCourse(detailB)
    store.solidify()

    mountTable()
    await flushPromises()

    // 两门课都渲染在同一格。
    const blocks = cellBlocks('单周课', '双周课')
    expect(blocks).toHaveLength(2)
    // 周次无交集：无 ⚠ 冲突角标（title=Conflict，happy-dom 下 i18n 为 en）。
    expect(document.querySelector('[title="Conflict"]')).toBeNull()
    expect(store.stats().conflictCount).toBe(0)

    // 单双周标签与教室均完整呈现
    expect(blocks[0].textContent).toContain('A101')
    expect(blocks[1].textContent).toContain('A101')
    // 消除锯齿边：卡片独立不带 border-dashed，全向对称 1px 细边框（非 AI-slop 彩色单边粗边框）
    expect(blocks[0].className).not.toContain('border-dashed')
    expect(blocks[0].className).not.toContain('[border-left-style:solid]')
    expect(blocks[0].className).toContain('border')
  })

  test('同一教学班多段周次安排智能合并为单个舒展卡片（杜绝 7 层切片与高度畸变）', async () => {
    const store = useScheduleStore()
    // 模拟教务系统 12117901 包含 7 段分批排课
    const multiDetail: PkCourseDetail = {
      code: '12117901',
      campus: '四平路校区',
      teachers: [{ teacherCode: 'T1', teacherName: '黄湘通' }, { teacherCode: 'T2', teacherName: '乔培军' }],
      teachingLanguage: '中文',
      arrangementInfo: [
        { arrangementText: '周1 第5-6节 [1-2 16] 南219', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [1, 2, 16], occupyRoom: '南219', teacherAndCode: '黄湘通(T1)' },
        { arrangementText: '周1 第5-6节 [3-4] 南219', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [3, 4], occupyRoom: '南219', teacherAndCode: '乔培军(T2)' },
        { arrangementText: '周1 第5-6节 [5-6] 南219', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [5, 6], occupyRoom: '南219', teacherAndCode: '黄湘通(T1)' },
        { arrangementText: '周1 第5-6节 [7] 南219', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [7], occupyRoom: '南219', teacherAndCode: '黄湘通(T1)' },
        { arrangementText: '周1 第5-6节 [8-10 15] 南219', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [8, 9, 10, 15], occupyRoom: '南219', teacherAndCode: '黄湘通(T1)' },
        { arrangementText: '周1 第5-6节 [12-13] 南219', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [12, 13], occupyRoom: '南219', teacherAndCode: '黄湘通(T1)' },
        { arrangementText: '周1 第5-6节 [11] 南219', occupyDay: 1, occupyTime: [5, 6], occupyWeek: [11], occupyRoom: '南219', teacherAndCode: '黄湘通(T1)' },
      ],
    }
    store.pushStagedCourse(makeStaged('121179', '现代分析测试技术', [multiDetail]))
    store.setClickedCourseInfo({ courseCode: '121179', courseName: '现代分析测试技术' })
    store.stageCourse(multiDetail)
    store.solidify()

    mountTable()
    await flushPromises()

    // 7 段安排合并为唯一的一张舒展大卡片，绝不再切分成 7 块
    const blocks = cellBlocks('现代分析测试技术', '')
    expect(blocks).toHaveLength(1)
    const cardText = blocks[0].textContent ?? ''
    // 完整课程名、班号、教师、合并周次与教室全部保留
    expect(cardText).toContain('现代分析测试技术')
    expect(cardText).toContain('12117901')
    expect(cardText).toContain('南219')
    expect(cardText).toContain('1-13,15-16')
  })

  test('有课程时课表抬头横幅展示方案名与统计信息', async () => {
    const store = useScheduleStore()
    const detail = makeDetail('300A.01', 2, [3, 4], [1, 16])
    store.pushStagedCourse(makeStaged('300A', '流体力学', [detail]))
    store.setClickedCourseInfo({ courseCode: '300A', courseName: '流体力学' })
    store.stageCourse(detail)
    store.solidify()

    mountTable()
    await flushPromises()

    const banner = document.querySelector('h2')
    expect(banner).toBeTruthy()
    // 抬头展示方案名称
    expect(document.body.textContent).toContain('Plan 1')
  })

  test('课程块采用轻盈通透色底设计与信息减省（多教师缩略、教室突出、aria-label 提供无障碍且杜绝原生 title 遮挡浮板）', async () => {
    const store = useScheduleStore()
    const detail: PkCourseDetail = {
      code: '12117901',
      campus: '四平路校区',
      teachingLanguage: '中文',
      teachers: [
        { teacherCode: 'T1', teacherName: '黄湘通' },
        { teacherCode: 'T2', teacherName: '乔培军' },
        { teacherCode: 'T3', teacherName: '江小英' },
        { teacherCode: 'T4', teacherName: '张灵敏' },
      ],
      arrangementInfo: [
        {
          arrangementText: '周1 第3-4节 [1,3,5,7,9] 南219',
          occupyDay: 1,
          occupyTime: [3, 4],
          occupyWeek: [1, 3, 5, 7, 9],
          occupyRoom: '南219',
          teacherAndCode: '黄湘通、乔培军、江小英、张灵敏',
        },
      ],
    }
    store.pushStagedCourse(makeStaged('121179', '现代分析测试技术', [detail]))
    store.setClickedCourseInfo({ courseCode: '121179', courseName: '现代分析测试技术' })
    store.stageCourse(detail)
    store.solidify()

    mountTable()
    await flushPromises()

    const blocks = cellBlocks('现代分析测试技术')
    expect(blocks).toHaveLength(1)
    const card = blocks[0]

    // 视觉与边框规范：全向细边框统一性，去除 AI slop 左侧厚色条
    expect(card.className).not.toContain('[border-left-style:solid]')
    expect(card.className).not.toContain('border-l-[3.5px]')
    expect(card.style.getPropertyValue('--card-border')).toContain('color-mix')
    expect(card.style.getPropertyValue('--card-bg')).toContain('color-mix')

    // 信息减省：4 位教师在卡片内缩略显示为 "黄湘通 等"，避免多行炸裂排版
    expect(card.textContent).toContain('黄湘通 等')
    expect(card.textContent).not.toContain('张灵敏')

    // 教室突出呈现
    expect(card.textContent).toContain('南219')

    // 周次提炼为单周展示
    expect(card.textContent).toContain('1-9')

    // 辅助无障碍：杜绝原生 title（防止浏览器黑色提示框遮挡悬浮卡片），改由 aria-label 保留完整无障碍信息
    expect(card.getAttribute('title')).toBeNull()
    const ariaLabel = card.getAttribute('aria-label') ?? ''
    expect(ariaLabel).toContain('黄湘通、乔培军、江小英、张灵敏')
    expect(ariaLabel).toContain('南219')
  })

  test('鼠标 hover 课程卡片触发详细信息浮板，且卡片与子元素均无原生 title 避免黑底 tooltip 遮盖', async () => {
    const store = useScheduleStore()
    const detail = makeDetail('700A.01', 1, [1, 2], [1, 8])
    store.pushStagedCourse(makeStaged('700A', '数据结构', [detail]))
    store.setClickedCourseInfo({ courseCode: '700A', courseName: '数据结构' })
    store.stageCourse(detail)
    store.solidify()

    mountTable()
    await flushPromises()

    const blocks = cellBlocks('数据结构')
    expect(blocks).toHaveLength(1)
    const card = blocks[0]

    // 严禁包含任何原生 title
    expect(card.getAttribute('title')).toBeNull()
    expect(card.querySelectorAll('[title]')).toHaveLength(0)

    // 模拟鼠标 hover
    card.dispatchEvent(new MouseEvent('mouseenter', { bubbles: true }))
    await new Promise((resolve) => setTimeout(resolve, 150))
    await flushPromises()

    // 浮板正确展示详细课程信息
    const popover = document.querySelector('.z-\\[2200\\]')
    expect(popover).toBeTruthy()
    expect(popover?.textContent).toContain('数据结构')
    expect(popover?.textContent).toContain('A101')

    // 模拟鼠标移出
    card.dispatchEvent(new MouseEvent('mouseleave', { bubbles: true }))
    await flushPromises()
    expect(document.querySelector('.z-\\[2200\\]')).toBeNull()
  })

  test('无冲突时点击导出图片直接唤起独立生成器画幅与品牌标识', async () => {
    const store = useScheduleStore()
    const detail = makeDetail('500A.01', 1, [1, 2], [1, 8])
    store.pushStagedCourse(makeStaged('500A', '高等数学', [detail]))
    store.setClickedCourseInfo({ courseCode: '500A', courseName: '高等数学' })
    store.stageCourse(detail)
    store.solidify()

    mountTable()
    await flushPromises()

    const exportBtn = [...document.querySelectorAll('button')].find((b) =>
      b.textContent?.includes('Export image') || b.textContent?.includes('导出图片'),
    )
    expect(exportBtn).toBeTruthy()
    exportBtn!.click()
    await flushPromises()

    // 冲突拦截弹窗不触发
    expect(document.body.textContent).not.toContain('Time conflicts detected')
    expect(document.body.textContent).not.toContain('课表存在时间冲突')

    // 独立生成器弹窗展开：具有画布与品牌标签，标题仅简洁呈现「课程表」，杜绝「同济大学课程表」等冗余文案
    const bodyText = document.body.textContent ?? ''
    expect(bodyText.includes('Timetable') || bodyText.includes('课程表')).toBe(true)
    expect(bodyText).not.toContain('同济大学课程表')
    expect(bodyText).not.toContain('同济大学 ·')
    expect(bodyText.includes('Generated by YourTJ Community') || bodyText.includes('由YourTJ社区生成')).toBe(true)

    // 导出的海报针对重要信息采用特制字重与高对比度，不弱化核心信息，且顶部横幅具备双层重心平衡结构
    const poster = document.querySelector('.z-\\[2101\\]')
    expect(poster).toBeTruthy()
    expect(poster?.textContent).toContain('A101')
    expect(poster?.textContent).toContain('高等数学')

    // 校验品牌标识 Badge 在海报内唯一呈现，并拥有独立的图形与文本
    const brandBadges = poster?.querySelectorAll('img[alt="YourTJ Logo"]')
    expect(brandBadges?.length).toBe(1)
  })

  test('有冲突时点击导出图片先唤起冲突警示，支持去调整或仍要导出', async () => {
    const store = useScheduleStore()
    const detailA = makeDetail('600A.01', 1, [3, 4], [1, 8])
    const detailB = makeDetail('600B.01', 1, [3, 4], [1, 8])
    store.pushStagedCourse(makeStaged('600A', '冲突课A', [detailA]))
    store.pushStagedCourse(makeStaged('600B', '冲突课B', [detailB]))
    store.setClickedCourseInfo({ courseCode: '600A', courseName: '冲突课A' })
    store.stageCourse(detailA)
    store.setClickedCourseInfo({ courseCode: '600B', courseName: '冲突课B' })
    store.stageCourse(detailB)
    store.solidify()

    mountTable()
    await flushPromises()

    const exportBtn = [...document.querySelectorAll('button')].find((b) =>
      b.textContent?.includes('Export image') || b.textContent?.includes('导出图片'),
    )
    expect(exportBtn).toBeTruthy()
    exportBtn!.click()
    await flushPromises()

    // 冲突提示弹窗成功拦截唤起
    const hasConflictWarning =
      document.body.textContent?.includes('Schedule Has Time Conflicts') ||
      document.body.textContent?.includes('课表存在时间冲突')
    expect(hasConflictWarning).toBe(true)

    // 此时独立生成器画布尚未显示
    expect(document.body.textContent).not.toContain('Generated by YourTJ Community')

    // 点击「仍要导出」
    const continueBtn = [...document.querySelectorAll('button')].find((b) =>
      b.textContent?.includes('Export Anyway') || b.textContent?.includes('仍要导出'),
    )
    expect(continueBtn).toBeTruthy()
    continueBtn!.click()
    await flushPromises()

    // 冲突弹窗放行，唤起独立生成器（简明课程表，无同济大学前缀）
    expect(document.body.textContent?.includes('Timetable') || document.body.textContent?.includes('课程表')).toBe(true)
    expect(document.body.textContent).not.toContain('同济大学课程表')
    expect(document.body.textContent?.includes('Generated by YourTJ Community') || document.body.textContent?.includes('由YourTJ社区生成')).toBe(true)
  })

  test('独立生成器浮层具备无边框暗场毛玻璃、img-fx Canvas 动效层与底部悬浮操作坞（无重放冗余）', async () => {
    const store = useScheduleStore()
    const detail = makeDetail('700A.01', 2, [1, 2], [1, 8])
    store.pushStagedCourse(makeStaged('700A', '软件工程', [detail]))
    store.setClickedCourseInfo({ courseCode: '700A', courseName: '软件工程' })
    store.stageCourse(detail)
    store.solidify()

    mountTable()
    await flushPromises()

    const exportBtn = [...document.querySelectorAll('button')].find((b) =>
      b.textContent?.includes('Export image') || b.textContent?.includes('导出图片'),
    )
    expect(exportBtn).toBeTruthy()
    exportBtn!.click()
    await flushPromises()

    // 1. 无边框暗场与毛玻璃背景
    const overlay = document.querySelector('.z-\\[2100\\]')
    expect(overlay).toBeTruthy()
    expect(overlay?.className).toContain('bg-slate-950/80')
    expect(overlay?.className).toContain('backdrop-blur-md')

    // 2. 居中无任何灰底/边框容器，仅悬浮海报；右上角无冗余关闭按钮
    const modalContent = document.querySelector('.z-\\[2101\\]')
    expect(modalContent).toBeTruthy()
    expect(modalContent?.className).toContain('bg-transparent')
    expect(modalContent?.className).toContain('border-none')
    expect(modalContent?.querySelector('.top-4.right-4')).toBeNull()

    // 3. img-fx Canvas 动效遮罩层
    const canvas = modalContent?.querySelector('canvas')
    expect(canvas).toBeTruthy()
    expect(canvas?.className).toContain('z-30')

    // 4. 底部独立悬浮胶囊控制坞
    const dock = modalContent?.querySelector('nav[aria-label="Export Actions"]')
    expect(dock).toBeTruthy()
    expect(dock?.className).toContain('rounded-full')
    expect(dock?.className).toContain('backdrop-blur-xl')

    // 不包含重放动效按钮
    const replayBtn = [...dock!.querySelectorAll('button')].find((b) =>
      b.textContent?.includes('Replay') || b.textContent?.includes('重放动效'),
    )
    expect(replayBtn).toBeUndefined()

    // 复制按钮在移动端隐藏 (hidden sm:flex)
    const copyBtn = [...dock!.querySelectorAll('button')].find((b) =>
      b.textContent?.includes('Copy') || b.textContent?.includes('复制图片'),
    )
    expect(copyBtn).toBeTruthy()
    expect(copyBtn?.className).toContain('hidden')
    expect(copyBtn?.className).toContain('sm:flex')

    const downloadBtn = [...dock!.querySelectorAll('button')].find((b) =>
      b.textContent?.includes('Download') || b.textContent?.includes('下载图片'),
    )
    expect(downloadBtn).toBeTruthy()
    // img-fx 渐显完成前导出/复制保持禁用，防止把马赛克动画截进导出图片
    expect((downloadBtn as HTMLButtonElement).disabled).toBe(true)
    expect((copyBtn as HTMLButtonElement).disabled).toBe(true)

    // 5. 点击海报内部不关闭，点击焦点外空白区域关闭预览
    const poster = modalContent?.querySelector('.bg-white.rounded-2xl') as HTMLElement
    expect(poster).toBeTruthy()
    poster.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    poster.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(document.querySelector('.z-\\[2101\\]')).toBeTruthy()

    modalContent?.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true }))
    modalContent?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(document.querySelector('.z-\\[2101\\]')).toBeNull()
  })

  test('包含外部选课与排课工具推荐气泡', async () => {
    const wrapper = mountTable()
    await flushPromises()

    const externalTools = wrapper.findComponent({ name: 'ScheduleExternalToolsTip' })
    expect(externalTools.exists()).toBe(true)

    const trigger = externalTools.find('[data-testid="schedule-external-tools-trigger"]')
    expect(trigger.exists()).toBe(true)

    await trigger.trigger('click')
    await flushPromises()

    const popover = document.querySelector('[data-testid="schedule-external-tools-popover"]')
    expect(popover).toBeTruthy()
    const links = popover?.querySelectorAll('a')
    expect(links?.length).toBe(2)
  })

  test('作息时间列表头自适应列宽与移动端纵向双行起止时间', async () => {
    const store = useScheduleStore()
    const detail = makeDetail('CS101.01', 1, [1, 2], [1, 16])
    store.pushStagedCourse(makeStaged('CS101', '操作系统', [detail]))
    store.setClickedCourseInfo({ courseCode: 'CS101', courseName: '操作系统' })
    store.stageCourse(detail)
    store.solidify()

    const wrapper = mountTable()
    await flushPromises()

    // 1. 表头列宽优化为 w-[50px] sm:w-[60px] md:w-[86px]
    const headerTh = wrapper.find('th.w-\\[50px\\]')
    expect(headerTh.exists()).toBe(true)
    expect(headerTh.classes()).toContain('sm:w-[60px]')
    expect(headerTh.classes()).toContain('md:w-[86px]')

    // 2. 作息时间格：桌面端单行 (hidden md:block) 与移动端双行 (md:hidden)
    const timeTds = wrapper.findAll('tbody tr td:first-child')
    expect(timeTds.length).toBeGreaterThan(0)
    const firstTd = timeTds[0]
    expect(firstTd.classes()).toContain('overflow-hidden')

    // 桌面端时间
    const desktopTime = firstTd.find('span.hidden.md\\:block')
    expect(desktopTime.exists()).toBe(true)
    expect(desktopTime.text()).toContain('08:00-08:45')

    // 移动端紧凑双行
    const mobileTime = firstTd.find('div.md\\:hidden')
    expect(mobileTime.exists()).toBe(true)
    expect(mobileTime.text()).toContain('08:00')
    expect(mobileTime.text()).toContain('08:45')
  })

  test('移动端课程卡片防过度截断（Anti-Truncation）与图标精简', async () => {
    const store = useScheduleStore()
    const detail = makeDetail('CS101.01', 1, [1, 2], [1, 16])
    detail.arrangementInfo[0].occupyRoom = '南楼101'
    detail.arrangementInfo[0].teacherAndCode = '赵宗显(T1)'
    store.pushStagedCourse(makeStaged('CS101', '操作系统', [detail]))
    store.setClickedCourseInfo({ courseCode: 'CS101', courseName: '操作系统' })
    store.stageCourse(detail)
    store.solidify()

    const wrapper = mountTable()
    await flushPromises()

    const card = wrapper.find('.schedule-course-card')
    expect(card.exists()).toBe(true)
    // 1. 移动端更轻量的 padding，释放 20% 以上横向宽度
    expect(card.classes()).toContain('p-1')
    expect(card.classes()).toContain('md:p-2')

    // 2. 课号在移动端隐藏 (hidden md:block)，免去无价值的数字截断挤占空间
    const codeEl = card.find('.font-mono')
    expect(codeEl.classes()).toContain('hidden')
    expect(codeEl.classes()).toContain('md:block')

    // 3. 教室名使用 break-all md:truncate，不再强制在移动端打点截断
    const roomSpan = card.find('span.break-all')
    expect(roomSpan.exists()).toBe(true)
    expect(roomSpan.text()).toContain('南楼101')

    // 4. 教师名完整呈现，无粗暴 truncate
    const teacherEl = card.findAll('span.break-all').find((s) => s.text().includes('赵宗显'))
    expect(teacherEl).toBeDefined()
  })

  test('跨节次课程自动设置动态 minHeight 撑满多节次格子', async () => {
    const store = useScheduleStore()
    // 跨 1-2 节 (span=2) 的课程
    const detail = makeDetail('CS102.01', 1, [1, 2], [1, 16])
    store.pushStagedCourse(makeStaged('CS102', '计算机网络', [detail]))
    store.setClickedCourseInfo({ courseCode: 'CS102', courseName: '计算机网络' })
    store.stageCourse(detail)
    store.solidify()

    const wrapper = mountTable()
    await flushPromises()

    const card = wrapper.find('.schedule-course-card')
    expect(card.exists()).toBe(true)
    const style = card.attributes('style') || ''
    // span=2 时必须计算出 min-height，确保撑满 2 节对应的高度，杜绝下半截留白误导
    expect(style).toContain('min-height')
  })

  test('同行存在单双周多门课导致行高扩展时，同行单门课卡片自动增高撑满对应扩展高度', async () => {
    const store = useScheduleStore()
    // 周一 1-2 节：单门课
    const detailMon = makeDetail('CS103.01', 1, [1, 2], [1, 16])
    store.pushStagedCourse(makeStaged('CS103', '算法分析', [detailMon]))
    store.setClickedCourseInfo({ courseCode: 'CS103', courseName: '算法分析' })
    store.stageCourse(detailMon)

    // 周三 1-2 节：两门课单双周共存（单周一门、双周一门）
    const detailWed1 = makeDetail('CS104.01', 3, [1, 2], [1, 3, 5, 7, 9, 11, 13, 15])
    const detailWed2 = makeDetail('CS105.01', 3, [1, 2], [2, 4, 6, 8, 10, 12, 14, 16])
    store.pushStagedCourse(makeStaged('CS104', '单周课', [detailWed1]))
    store.pushStagedCourse(makeStaged('CS105', '双周课', [detailWed2]))
    store.setClickedCourseInfo({ courseCode: 'CS104', courseName: '单周课' })
    store.stageCourse(detailWed1)
    store.setClickedCourseInfo({ courseCode: 'CS105', courseName: '双周课' })
    store.stageCourse(detailWed2)
    store.solidify()

    const wrapper = mountTable()
    await flushPromises()

    const cards = wrapper.findAll('.schedule-course-card')
    expect(cards.length).toBeGreaterThanOrEqual(3)

    // 找到周一算法分析卡片
    const monCard = cards.find((c) => c.text().includes('算法分析'))
    expect(monCard).toBeDefined()
    const monStyle = monCard!.attributes('style') || ''
    // 因为周三有两门单双周课堆叠（multiCardH=72/58px），行高被显著撑大
    // 周一单门课应自动感知扩展后的行高并直接返回 cellInnerHeight 作为 minHeight，彻底撑满
    const match = monStyle.match(/min-height:\s*(\d+)px/)
    expect(match).toBeTruthy()
    const minH = parseInt(match![1], 10)
    // 两门叠放课需 2*72+4=148px inner + 8px padding = 156px total；单门课应与之对齐
    // 桌面端：cellInnerHeight ≥ 148px；移动端：cellInnerHeight ≥ 2*58+4=120-8=112px
    // 取保守下界 110px，确保显著超过未扩展时的双节单课高度 (2*58-8=108px)
    expect(minH).toBeGreaterThanOrEqual(110)
  })

  test('表格单元格 td 采用 h-px，确保同行其他列被撑高时短卡片百分比高度能跟随完全撑满', async () => {
    const store = useScheduleStore()
    const detail = makeDetail('CS106.01', 1, [1, 2], [1, 16])
    store.pushStagedCourse(makeStaged('CS106', '测试课程', [detail]))
    store.setClickedCourseInfo({ courseCode: 'CS106', courseName: '测试课程' })
    store.stageCourse(detail)
    store.solidify()

    const wrapper = mountTable()
    await flushPromises()

    // 单元格 td 具备 h-px，以便在 CSS 表格渲染模型下使直接子容器的 height: 100% (h-full) 能继承计算出使用的行高
    const courseTd = wrapper.findAll('tbody tr td.h-px').find((td) => td.find('.schedule-course-card').exists())
    expect(courseTd).toBeDefined()
    // 单元格内部卡片容器具备 h-full 与 w-full
    const container = courseTd!.find('.h-full.w-full')
    expect(container.exists()).toBe(true)
  })
})
