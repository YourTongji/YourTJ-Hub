// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, expect, test, vi } from 'vitest'
import type { CampusEvent } from '@gooseforum/client'
import { i18n } from '../src/runtime/i18n'
import CampusTimetable from '../src/site/components/CampusTimetable.vue'
import { getPkSectionTimes } from '../src/runtime/pk-api'

vi.mock('../src/runtime/pk-api', () => ({ getPkSectionTimes: vi.fn() }))
afterEach(() => vi.restoreAllMocks())
const lesson = (name: string, weeks: number[], start = 3, end = 4): CampusEvent => ({ name, weeks, start, end, day: 1, teacher: '张老师', room: '南楼 101', campus: '四平路', credits: '2' })

test('official courses reuse section grid and configured times without editing or persisting a planner draft', async () => {
  vi.mocked(getPkSectionTimes).mockResolvedValue({ sectionTimes: [{ section: 3, start: '10:05', end: '10:50' }], maxRowsDefault: 11 })
  const persist = vi.spyOn(Storage.prototype, 'setItem')
  const wrapper = mount(CampusTimetable, { props: { courses: [lesson('单周课程', [1, 3]), lesson('双周课程', [2, 4])], week: 1, today: 1 }, global: { plugins: [i18n] } })
  try {
    await flushPromises()
    expect(wrapper.findAll('tbody tr')).toHaveLength(11)
    expect(wrapper.find('.schedule-course-card').text()).toContain('单周课程')
    expect(wrapper.text()).not.toContain('双周课程')
    expect(wrapper.find('td[rowspan="2"] .schedule-course-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('10:05-10:50')
    expect(wrapper.find('[role="button"]').exists()).toBe(false)
    expect(wrapper.find('td[aria-label]').exists()).toBe(false)
    expect(wrapper.find('input').exists()).toBe(false)
    expect(wrapper.find('.font-mono.hidden.md\\:block').exists()).toBe(false)
    await wrapper.setProps({ week: 2 })
    expect(wrapper.text()).toContain('双周课程')
    expect(wrapper.text()).not.toContain('单周课程')
    expect(persist).not.toHaveBeenCalled()
  } finally { wrapper.unmount() }
})

test('official records without class IDs remain distinct and invalid ranges cannot distort the grid', async () => {
  vi.mocked(getPkSectionTimes).mockRejectedValue(new Error('offline'))
  const courses = [lesson('同名课程', [1]), { ...lesson('同名课程', [1]), teacher: '李老师' }, lesson('异常安排', [1], 4, 3)]
  const wrapper = mount(CampusTimetable, { props: { courses, week: 1, today: 1 }, global: { plugins: [i18n] } })
  try {
    await flushPromises()
    expect(wrapper.findAll('.schedule-course-card')).toHaveLength(2)
    expect(wrapper.text()).not.toContain('异常安排')
    expect(wrapper.text()).toContain('10:00-10:45')
    expect(wrapper.find('.schedule-course-card').attributes('style')).toContain('color-mix')
  } finally { wrapper.unmount() }
})
