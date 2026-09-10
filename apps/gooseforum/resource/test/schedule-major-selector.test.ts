// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import { i18n } from '../src/runtime/i18n'

const getPkCalendars = vi.fn()
const getPkGrades = vi.fn()
const getPkMajors = vi.fn()
vi.mock('../src/runtime/pk-api', () => ({
  getPkCalendars: () => getPkCalendars(),
  getPkGrades: (calendarId: number) => getPkGrades(calendarId),
  getPkMajors: (grade: number, calendarId: number) => getPkMajors(grade, calendarId),
}))

import ScheduleMajorSelector from '../src/site/components/schedule/ScheduleMajorSelector.vue'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'

/** 三个 combobox：学期=0、年级=1、专业=2（顺序稳定，不依赖 locale 文案）。 */
function comboboxes(wrapper: VueWrapper) {
  return wrapper.findAll('[role="combobox"]')
}

/**
 * 打开第 index 个下拉并返回其选项。
 * SiteSelect 迁移 reka-ui 后：Trigger 用 pointerdown 打开（SelectTrigger.js
 * handlePointerOpen），选项经 SelectPortal 渲染到 body，须从 document 查询。
 */
async function openCombobox(wrapper: VueWrapper, index: number): Promise<Element[]> {
  await comboboxes(wrapper)[index].trigger('pointerdown', { button: 0, pageX: 10, pageY: 10 })
  await flushPromises()
  return [...document.querySelectorAll('[role="option"]')]
}

/** reka-ui SelectItem 用 pointerup 触发选择（SelectItem.js handleSelectCustomEvent）。 */
async function selectOption(wrapper: VueWrapper, option: Element) {
  option.dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, button: 0 }))
  option.dispatchEvent(new PointerEvent('pointerup', { bubbles: true, button: 0 }))
  await flushPromises()
}

/** 预置 store 中的已存选择（SchedulePage 挂载时 loadSolidify 的职责，测试直接设置）。 */
function setStoredSelection(selection: { calendarId?: number; grade?: number; major?: string }) {
  const store = useScheduleStore()
  store.state.majorSelected = {
    calendarId: selection.calendarId,
    grade: selection.grade,
    major: selection.major,
  }
}

describe('ScheduleMajorSelector 初始化加载', () => {
  /** 追踪当前 wrapper，afterEach 统一卸载并清理 portal 残留（全局 document 查询）。 */
  let mounted: VueWrapper | null = null

  beforeEach(() => {
    vi.resetAllMocks()
    localStorage.clear()
    setStoredSelection({})
    getPkCalendars.mockResolvedValue([{ calendarId: 121, calendarName: '2025-2026学年第2学期' }])
    mounted = null
  })

  afterEach(() => {
    mounted?.unmount()
    mounted = null
    // SiteSelect 选项经 SelectPortal 渲染到 body，卸载组件后清掉 portal 残留，
    // 避免下一个测试的全局 [role="option"] 查询命中上一个测试的选项。
    document.body.innerHTML = ''
  })

  test('无已存选择时：默认学期加载年级并写回 store，选年级可加载专业', async () => {
    // 回归（PR #323 review P1）：restoreSelection 期间 isRestoring 抑制 watch，
    // 必须把最终学期写回 store；否则选年级时 calendarId 为 undefined，专业永远加载不出。
    getPkGrades.mockResolvedValue([2025, 2024])
    getPkMajors.mockResolvedValue([{ code: '00301', name: '2025(00301 数学类)' }])

    mounted = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    const wrapper = mounted
    await flushPromises()

    // 默认第一个学期（121）已加载年级，且 store 中的学期已写回。
    expect(getPkGrades).toHaveBeenCalledWith(121)
    const store = useScheduleStore()
    expect(store.state.majorSelected.calendarId).toBe(121)

    // Grade 下拉应包含年级选项（打开后可见 2025/2024）。
    const options = await openCombobox(wrapper, 1)
    const optionTexts = options.map((o) => o.textContent)
    expect(optionTexts).toContain('2025')
    expect(optionTexts).toContain('2024')

    // 用户选择年级 2025 → watch 触发 → 用已写回的 calendarId 加载专业。
    const option = options.find((o) => o.textContent === '2025')
    await selectOption(wrapper, option!)
    expect(getPkMajors).toHaveBeenCalledWith(2025, 121)
  })

  test('专业选择框在原位置输入关键词筛选并选择专业', async () => {
    setStoredSelection({ calendarId: 121, grade: 2025, major: '00301' })
    getPkGrades.mockResolvedValue([2025])
    getPkMajors.mockResolvedValue([
      { code: '00301', name: '2025(00301 数学类)' },
      { code: '00401', name: '2025(00401 物理学类)' },
    ])

    mounted = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    const wrapper = mounted
    await flushPromises()

    const input = wrapper.get<HTMLInputElement>('[data-testid="schedule-major-combobox-input"]')
    expect(input.element).toBe(comboboxes(wrapper)[2].element)
    expect(input.element.value).toContain('数学类')
    expect(document.querySelector('[data-testid="site-select-search-input"]')).toBeNull()

    await input.setValue('物理')
    await flushPromises()

    const options = [...document.querySelectorAll('[role="option"]')]
    expect(options.map((option) => option.textContent)).toEqual([expect.stringContaining('物理学类')])

    options[0].dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(useScheduleStore().state.majorSelected.major).toBe('00401')
  })
  test('localStorage 有完整选择时恢复年级+专业', async () => {
    setStoredSelection({ calendarId: 121, grade: 2025, major: '00301' })
    getPkGrades.mockResolvedValue([2025, 2024])
    getPkMajors.mockResolvedValue([{ code: '00301', name: '2025(00301 数学类)' }])

    mounted = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    const wrapper = mounted
    await flushPromises()

    expect(getPkGrades).toHaveBeenCalledWith(121)
    expect(getPkMajors).toHaveBeenCalledWith(2025, 121)
    // 已选年级显示在下拉 trigger，专业显示在原位置的可输入 combobox。
    expect(comboboxes(wrapper)[1].text()).toContain('2025')
    expect((comboboxes(wrapper)[2].element as HTMLInputElement).value).toContain('00301')
  })

  test('localStorage 学期已不存在时回退第一个学期并加载年级', async () => {
    // 旧学期（如 122）被同步清空后，恢复逻辑应回退到当前学期并加载年级，
    // 而不是按失效的 calendarId 请求空数据。
    setStoredSelection({ calendarId: 122, grade: 2025, major: '00301' })
    getPkGrades.mockResolvedValue([2025, 2024])
    getPkMajors.mockResolvedValue([])

    mounted = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    const wrapper = mounted
    await flushPromises()

    expect(getPkGrades).toHaveBeenCalledWith(121)

    // P1：回退后 store 必须使用有效学期，旧学期/年级/专业清空（防跨学期污染）。
    const store = useScheduleStore()
    expect(store.state.majorSelected.calendarId).toBe(121)
    expect(store.state.majorSelected.grade).toBeUndefined()
    expect(store.state.majorSelected.major).toBeUndefined()

    // 回退后年级下拉有数据，且用户重新选年级可用新学期加载专业。
    const options = await openCombobox(wrapper, 1)
    expect(options.map((o) => o.textContent)).toContain('2025')
  })

  test('localStorage 学期已不存在时清空旧学期已选课程', async () => {
    // P1：替换失效学期时清掉旧学期课程缓存（对齐学期变更 watch 语义）。
    const store = useScheduleStore()
    store.pushStagedCourse({
      courseCode: '122004',
      courseName: '高等数学(122004)',
      courseNameReserved: '高等数学',
      credit: 4,
      courseType: '必',
      teacher: [],
      status: 1,
      courseDetail: [],
    })
    setStoredSelection({ calendarId: 122, grade: 2025, major: '00301' })
    getPkGrades.mockResolvedValue([2025, 2024])
    getPkMajors.mockResolvedValue([])

    mounted = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    await flushPromises()

    expect(store.state.commonLists.stagedCourses).toEqual([])
  })

  test('回退清空旧学期课程后立即持久化（刷新不复活）', async () => {
    // 回归：resetSelection 曾不写 localStorage，换学期/年级/专业清空的课程
    // 只停留在内存，刷新后 loadSolidify 用旧数据覆盖内存，课程"复活"。
    const store = useScheduleStore()
    store.pushStagedCourse({
      courseCode: '122004',
      courseName: '高等数学(122004)',
      courseNameReserved: '高等数学',
      credit: 4,
      courseType: '必',
      teacher: [],
      status: 1,
      courseDetail: [],
    })
    // 旧学期课程此前已持久化（用户上次使用时 solidify 过）。
    store.solidify()
    expect(localStorage.getItem('pk.plans')).toContain('122004')
    setStoredSelection({ calendarId: 122, grade: 2025, major: '00301' })
    getPkGrades.mockResolvedValue([2025, 2024])
    getPkMajors.mockResolvedValue([])

    mounted = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    await flushPromises()

    // 内存清空的同时 localStorage 也同步清空。
    expect(store.state.commonLists.stagedCourses).toEqual([])
    expect(localStorage.getItem('pk.plans')).not.toContain('122004')

    // 模拟刷新/重进：loadSolidify 不得把旧学期课程复活。
    store.loadSolidify()
    expect(store.state.commonLists.stagedCourses).toEqual([])
  })
})
describe('ScheduleMajorSelector 快速清除与整体清空', () => {
  let mounted: VueWrapper | null = null

  beforeEach(() => {
    vi.resetAllMocks()
    localStorage.clear()
    setStoredSelection({})
    getPkCalendars.mockResolvedValue([
      { calendarId: 121, calendarName: '2025-2026学年第2学期' },
      { calendarId: 120, calendarName: '2025-2026学年第1学期' },
    ])
    getPkGrades.mockResolvedValue([2025, 2024])
    getPkMajors.mockResolvedValue([{ code: '00301', name: '2025(00301 数学类)' }])
    mounted = null
  })

  afterEach(() => {
    mounted?.unmount()
    mounted = null
    document.body.innerHTML = ''
  })

  test('点击专业清除按钮清空已选专业，并更新 store', async () => {
    setStoredSelection({ calendarId: 121, grade: 2025, major: '00301' })
    mounted = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    const wrapper = mounted
    await flushPromises()

    const store = useScheduleStore()
    expect(store.state.majorSelected.major).toBe('00301')

    const clearMajorBtn = wrapper.find(`button[aria-label="${i18n.global.t('schedule.clearMajor')}"]`)
    expect(clearMajorBtn.exists()).toBe(true)
    await clearMajorBtn.trigger('click')
    await flushPromises()

    expect(store.state.majorSelected.major).toBeUndefined()
    expect((comboboxes(wrapper)[2].element as HTMLInputElement).value).toBe('')
  })

  test('点击专业输入框内部的 X 按钮清空已选专业', async () => {
    setStoredSelection({ calendarId: 121, grade: 2025, major: '00301' })
    mounted = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    const wrapper = mounted
    await flushPromises()

    const store = useScheduleStore()
    expect(store.state.majorSelected.major).toBe('00301')

    const clearMajorBtns = wrapper.findAll(`button[aria-label="${i18n.global.t('schedule.clearMajor')}"]`)
    expect(clearMajorBtns.length).toBeGreaterThanOrEqual(2)
    await clearMajorBtns[1].trigger('click')
    await flushPromises()

    expect(store.state.majorSelected.major).toBeUndefined()
    expect((comboboxes(wrapper)[2].element as HTMLInputElement).value).toBe('')
  })

  test('点击年级清除按钮清空年级与级联专业', async () => {
    setStoredSelection({ calendarId: 121, grade: 2025, major: '00301' })
    mounted = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    const wrapper = mounted
    await flushPromises()

    const store = useScheduleStore()
    expect(store.state.majorSelected.grade).toBe(2025)
    expect(store.state.majorSelected.major).toBe('00301')

    const clearGradeBtn = wrapper.find(`button[aria-label="${i18n.global.t('schedule.clearGrade')}"]`)
    expect(clearGradeBtn.exists()).toBe(true)
    await clearGradeBtn.trigger('click')
    await flushPromises()

    expect(store.state.majorSelected.grade).toBeUndefined()
    expect(store.state.majorSelected.major).toBeUndefined()
    expect(comboboxes(wrapper)[1].text()).toContain(i18n.global.t('schedule.selectPlaceholder'))
    expect((comboboxes(wrapper)[2].element as HTMLInputElement).value).toBe('')
  })

  test('点击清空已选配置重置学期、年级、专业', async () => {
    setStoredSelection({ calendarId: 121, grade: 2025, major: '00301' })
    mounted = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    const wrapper = mounted
    await flushPromises()

    const store = useScheduleStore()
    expect(store.state.majorSelected.calendarId).toBe(121)
    expect(store.state.majorSelected.grade).toBe(2025)
    expect(store.state.majorSelected.major).toBe('00301')

    const resetAllBtn = wrapper.find(`button[title="${i18n.global.t('schedule.resetSelectionHint')}"]`)
    expect(resetAllBtn.exists()).toBe(true)
    expect(resetAllBtn.text()).toContain(i18n.global.t('schedule.resetSelection'))

    await resetAllBtn.trigger('click')
    await flushPromises()

    expect(store.state.majorSelected.calendarId).toBeUndefined()
    expect(store.state.majorSelected.grade).toBeUndefined()
    expect(store.state.majorSelected.major).toBeUndefined()
    expect(comboboxes(wrapper)[0].text()).toContain(i18n.global.t('schedule.selectPlaceholder'))
    expect(comboboxes(wrapper)[1].text()).toContain(i18n.global.t('schedule.selectPlaceholder'))
    expect((comboboxes(wrapper)[2].element as HTMLInputElement).value).toBe('')
  })

  test('专业代码查询提示徽章展示高对比度文案并支持 Hover 与 Click 打开指南 Popover', async () => {
    vi.useFakeTimers()
    mounted = mount(ScheduleMajorSelector, { global: { plugins: [i18n] }, attachTo: document.body })
    const wrapper = mounted
    await flushPromises()

    const trigger = wrapper.find('[data-testid="schedule-major-code-trigger"]')
    expect(trigger.exists()).toBe(true)
    expect(trigger.text()).toContain(i18n.global.t('schedule.majorCodeBadge'))
    expect(trigger.attributes('title')).toBe(i18n.global.t('schedule.majorCodeHelp'))
    expect(trigger.classes()).toContain('text-primary')
    expect(trigger.classes()).toContain('bg-primary/10')
    expect(trigger.classes()).toContain('border-primary/30')
    expect(trigger.classes()).toContain('cursor-help')

    // 触发 hover (mouseenter)
    await trigger.trigger('mouseenter')
    vi.advanceTimersByTime(150)
    await flushPromises()

    expect(document.body.textContent).toContain(i18n.global.t('schedule.majorCodeGuideTitle'))

    // 移出 hover (mouseleave)
    await trigger.trigger('mouseleave')
    vi.advanceTimersByTime(260)
    await flushPromises()

    // 点击固定打开
    await trigger.trigger('click')
    await flushPromises()
    expect(document.body.textContent).toContain(i18n.global.t('schedule.majorCodeGuideTitle'))

    vi.useRealTimers()
  })
})

describe('ScheduleMajorSelector 配置就绪主动提示与收起交互', () => {
  let mounted: VueWrapper | null = null

  beforeEach(() => {
    vi.resetAllMocks()
    localStorage.clear()
    setStoredSelection({})
    getPkCalendars.mockResolvedValue([
      { calendarId: 121, calendarName: '2025-2026学年第2学期' },
    ])
    getPkGrades.mockResolvedValue([2024])
    getPkMajors.mockResolvedValue([{ code: '080901', name: '计算机科学与技术' }])
    mounted = null
  })

  afterEach(() => {
    mounted?.unmount()
    mounted = null
    document.body.innerHTML = ''
  })

  test('collapsible 模式下未完成所有配置时：展示常规收起配置按钮，无就绪卡片', async () => {
    setStoredSelection({ calendarId: 121, grade: 2024, major: undefined })
    mounted = mount(ScheduleMajorSelector, {
      props: { collapsible: true },
      global: { plugins: [i18n] },
    })
    await flushPromises()

    // 不显示就绪高亮卡片
    expect(mounted.find('[data-testid="schedule-config-ready-card"]').exists()).toBe(false)

    // 显示普通收起按钮
    const collapseBtn = mounted.findAll('button').find((b) => b.text().trim() === i18n.global.t('schedule.collapseSettings'))
    expect(collapseBtn).toBeDefined()

    await collapseBtn!.trigger('click')
    await flushPromises()
    expect(mounted.emitted('toggle-collapse')).toHaveLength(1)
  })

  test('collapsible 模式下完成所有配置时：展示高醒目度就绪卡片，点击收起进入选课触发折叠', async () => {
    setStoredSelection({ calendarId: 121, grade: 2024, major: '080901' })
    mounted = mount(ScheduleMajorSelector, {
      props: { collapsible: true },
      global: { plugins: [i18n] },
    })
    await flushPromises()

    // 显示就绪高亮卡片且带有无障碍属性
    const readyCard = mounted.find('[data-testid="schedule-config-ready-card"]')
    expect(readyCard.exists()).toBe(true)
    expect(readyCard.attributes('role')).toBe('status')
    expect(readyCard.attributes('aria-live')).toBe('polite')
    expect(readyCard.text()).toContain(i18n.global.t('schedule.configReadyTitle'))
    expect(readyCard.text()).toContain(i18n.global.t('schedule.configReadyDesc'))
    expect(readyCard.text()).toContain(i18n.global.t('schedule.readyToPick'))

    // 包含高醒目度收起按钮
    const doneBtn = readyCard.find('button.gf-button-primary')
    expect(doneBtn.exists()).toBe(true)
    expect(doneBtn.text()).toContain(i18n.global.t('schedule.collapseSettingsDone'))

    await doneBtn.trigger('click')
    await flushPromises()
    expect(mounted.emitted('toggle-collapse')).toHaveLength(1)
  })
})
