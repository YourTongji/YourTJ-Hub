// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import ScheduleCoursePicker from '../src/site/components/schedule/ScheduleCoursePicker.vue'

// 选课弹窗打开时会加载字典（getPkCampuses/getPkFaculties），mock 掉避免真实 fetch。
vi.mock('../src/runtime/pk-api', () => ({
  getPkCampuses: vi.fn(async () => []),
  getPkFaculties: vi.fn(async () => []),
  getPkCoursesByMajor: vi.fn(async () => []),
  getPkOptionalTypes: vi.fn(async () => []),
  getPkCoursesByNature: vi.fn(async () => []),
  searchPkCourses: vi.fn(async () => []),
  getPkCourseDetails: vi.fn(async () => []),
}))

describe('ScheduleCoursePicker Dialog 可访问性', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh'
  })

  test('aria-describedby 指向存在的描述元素（P2-6 建议项）', async () => {
    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    const dialog = document.querySelector('[role="dialog"]')
    expect(dialog).not.toBeNull()
    const describedBy = dialog?.getAttribute('aria-describedby')
    expect(describedBy).toBeTruthy()
    const desc = document.getElementById(describedBy!)
    expect(desc).not.toBeNull()
    expect(desc?.textContent).toContain('课程')

    wrapper.unmount()
  })

  test('高级检索的校区和开课学院选择框启用关键词搜索', async () => {
    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    const tabs = [...document.querySelectorAll<HTMLElement>('[role="tab"]')]
    expect(tabs).toHaveLength(3)
    tabs[2].click()
    await flushPromises()
    const selects = [...document.querySelectorAll<HTMLElement>('[role="combobox"]')]
    expect(selects).toHaveLength(2)

    selects[0].dispatchEvent(new PointerEvent('pointerdown', { bubbles: true, button: 0, pageX: 10, pageY: 10 }))
    await flushPromises()
    expect(document.querySelector('[data-testid="site-select-search-input"]')).not.toBeNull()
    wrapper.unmount()
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })
})
