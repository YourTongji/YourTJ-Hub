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

  test('无已存选择时，默认第一个学期也应加载年级', async () => {
    // 回归：restoreSelection 只在 localStorage 有完整选择时加载年级，
    // 首次访问（无选择）Term 默认选中但 Grade 永远为空。
    getPkGrades.mockResolvedValue([2025, 2024])
    getPkMajors.mockResolvedValue([])

    const wrapper = mount(ScheduleMajorSelector, { global: { plugins: [i18n] } })
    await flushPromises()

    expect(getPkGrades).toHaveBeenCalledWith(121)

    // Grade 下拉应包含年级选项（打开后可见 2025/2024）。
    await comboboxes(wrapper)[1].trigger('click')
    await flushPromises()
    const options = wrapper.findAll('[role="option"]').map((o) => o.text())
    expect(options).toContain('2025')
    expect(options).toContain('2024')
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
    await comboboxes(wrapper)[1].trigger('click')
    await flushPromises()
    expect(wrapper.findAll('[role="option"]').map((o) => o.text())).toContain('2025')
  })
})
