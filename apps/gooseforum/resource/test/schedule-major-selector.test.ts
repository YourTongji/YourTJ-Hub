// @vitest-environment happy-dom
import { beforeEach, describe, expect, test, vi } from 'vitest'
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

/** 三个 SiteSelect：学期=0、年级=1、专业=2（顺序稳定，不依赖 locale 文案）。 */
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
  beforeEach(() => {
    vi.resetAllMocks()
    localStorage.clear()
    setStoredSelection({})
    getPkCalendars.mockResolvedValue([{ calendarId: 121, calendarName: '2025-2026学年第2学期' }])
  })

  test('无已存选择时：默认学期加载年级并写回 store，选年级可加载专业', async () => {
    // 回归（PR #323 review P1）：restoreSelection 期间 isRestoring 抑制 watch，
    // 必须把最终学期写回 store；否则选年级时 calendarId 为 undefined，专业永远加载不出。
    getPkGrades.mockResolvedValue([2025, 2024])
    getPkMajors.mockResolvedValue([{ code: '00301', name: '2025(00301 数学类)' }])

    const wrapper = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
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

  test('localStorage 有完整选择时恢复年级+专业', async () => {
    setStoredSelection({ calendarId: 121, grade: 2025, major: '00301' })
    getPkGrades.mockResolvedValue([2025, 2024])
    getPkMajors.mockResolvedValue([{ code: '00301', name: '2025(00301 数学类)' }])

    const wrapper = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    await flushPromises()

    expect(getPkGrades).toHaveBeenCalledWith(121)
    expect(getPkMajors).toHaveBeenCalledWith(2025, 121)
    // 已选年级与专业应显示在触发按钮上。
    expect(comboboxes(wrapper)[1].text()).toContain('2025')
    expect(comboboxes(wrapper)[2].text()).toContain('00301')
  })

  test('localStorage 学期已不存在时回退第一个学期并加载年级', async () => {
    // 旧学期（如 122）被同步清空后，恢复逻辑应回退到当前学期并加载年级，
    // 而不是按失效的 calendarId 请求空数据。
    setStoredSelection({ calendarId: 122, grade: 2025, major: '00301' })
    getPkGrades.mockResolvedValue([2025, 2024])
    getPkMajors.mockResolvedValue([])

    const wrapper = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
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
    store.state.commonLists.stagedCourses = [
      {
        courseCode: '122004',
        courseName: '高等数学(122004)',
        courseNameReserved: '高等数学',
        credit: 4,
        courseType: '必',
        teacher: [],
        status: 1,
        courseDetail: [],
      },
    ]
    setStoredSelection({ calendarId: 122, grade: 2025, major: '00301' })
    getPkGrades.mockResolvedValue([2025, 2024])
    getPkMajors.mockResolvedValue([])

    mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    await flushPromises()

    expect(store.state.commonLists.stagedCourses).toEqual([])
  })
})
