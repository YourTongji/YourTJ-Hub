// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import ScheduleCoursePicker from '../src/site/components/schedule/ScheduleCoursePicker.vue'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'
import type { PkCourse } from '../src/site/types/pk'

const searchPkCourses = vi.hoisted(() => vi.fn(async (): Promise<PkCourse[]> => []))

// 选课弹窗打开时会加载字典（getPkCampuses/getPkFaculties），mock 掉避免真实 fetch。
vi.mock('../src/runtime/pk-api', () => ({
  getPkCampuses: vi.fn(async () => []),
  getPkFaculties: vi.fn(async () => []),
  getPkCoursesByMajor: vi.fn(async () => []),
  getPkOptionalTypes: vi.fn(async () => []),
  getPkCoursesByNature: vi.fn(async () => []),
  searchPkCourses,
  getPkCourseDetails: vi.fn(async () => []),
}))

function makeCourse(courseCode: string): PkCourse {
  return {
    courseCode,
    courseName: `课程${courseCode}`,
    courseNameReserved: `课程${courseCode}`,
    credit: 3,
    courseType: '查',
    teacher: [],
    status: 0,
    courseDetail: [],
  }
}

describe('ScheduleCoursePicker Dialog 可访问性', () => {
  beforeEach(() => {
    i18n.global.locale.value = 'zh'
    searchPkCourses.mockReset()
    searchPkCourses.mockResolvedValue([])
    const store = useScheduleStore()
    store.clearStagedAndSelectedCourses()
    store.setCompulsoryCourses([])
    store.setMajorInfo({ calendarId: 121, grade: 2025, major: '00301' })
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

  test('高级检索将计划内课程置顶，其他结果保持原顺序', async () => {
    const store = useScheduleStore()
    const wrapper = mount(ScheduleCoursePicker, {
      props: { open: true },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    store.setCompulsoryCourses([makeCourse('122005')])
    searchPkCourses.mockResolvedValueOnce([makeCourse('122004'), makeCourse('122005'), makeCourse('122006')])

    const tabs = [...document.querySelectorAll<HTMLElement>('[role="tab"]')]
    tabs[2].click()
    await flushPromises()
    document.querySelector<HTMLFormElement>('[role="dialog"] form')?.dispatchEvent(
      new Event('submit', { bubbles: true, cancelable: true }),
    )
    await flushPromises()

    const names = [...document.querySelectorAll<HTMLElement>('[role="dialog"] ul li')]
      .map((row) => row.querySelector('span span')?.textContent?.trim())
    expect(names).toEqual(['课程122005', '课程122004', '课程122006'])

    wrapper.unmount()
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })
})
