// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import { getPkCampuses, getPkCoursesByMajor, getPkFaculties, getPkOptionalTypes } from '../src/runtime/pk-api'
import { useScheduleStore } from '../src/site/composables/useScheduleStore'
import ScheduleCoursePicker from '../src/site/components/schedule/ScheduleCoursePicker.vue'
import ScheduleExportDialog from '../src/site/components/schedule/ScheduleExportDialog.vue'

const animation = vi.hoisted(() => ({
  resize: vi.fn(), destroy: vi.fn(), play: vi.fn((done: () => void) => done()),
}))
vi.mock('../src/site/utils/imgFxCanvas', () => ({
  ImgFxController: class {
    resize = animation.resize
    destroy = animation.destroy
    play = animation.play
  },
}))

vi.mock('../src/runtime/pk-api', () => ({
  getPkCampuses: vi.fn(async () => []),
  getPkFaculties: vi.fn(async () => []),
  getPkCoursesByMajor: vi.fn(async () => []),
  getPkOptionalTypes: vi.fn(async () => []),
  getPkCoursesByNature: vi.fn(async () => []),
  searchPkCourses: vi.fn(async () => []),
  getPkCourseDetails: vi.fn(async () => []),
}))

beforeEach(() => {
  vi.clearAllMocks()
  i18n.global.locale.value = 'zh'
  useScheduleStore().setMajorInfo({ calendarId: 121, grade: 2025, major: '00301' })
})
afterEach(() => { document.body.innerHTML = '' })
enableAutoUnmount(afterEach)

test('mounting the lazy course picker open loads plan courses and search dictionaries', async () => {
  const wrapper = mount(ScheduleCoursePicker, {
    props: { open: true },
    global: { plugins: [i18n] },
    attachTo: document.body,
  })
  await flushPromises()
  expect(getPkCoursesByMajor).toHaveBeenCalledWith(2025, '00301', 121)
  expect(getPkOptionalTypes).toHaveBeenCalledWith(121)
  expect(getPkCampuses).toHaveBeenCalledOnce()
  expect(getPkFaculties).toHaveBeenCalledOnce()

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true })
  await flushPromises()
  expect(getPkCoursesByMajor).toHaveBeenCalledOnce() // Reuse the unchanged major's loaded courses.
  expect(getPkCampuses).toHaveBeenCalledTimes(2)
})

test('mounting the lazy export dialog open enables copy and download, including a remount', async () => {
  for (let opening = 0; opening < 2; opening++) {
    const wrapper = mount(ScheduleExportDialog, {
      props: {
        open: true,
        cellCourses: Array.from({ length: 12 }, () => Array.from({ length: 7 }, () => [])),
        cellSpans: Array.from({ length: 12 }, () => Array(7).fill(1)),
        occupiedGrid: Array.from({ length: 12 }, () => Array(7).fill(false)),
        activeCalendar: null,
        activePlanName: '测试方案',
        weekViewLabel: '全部周次',
        exportSummaryText: '',
      },
      global: { plugins: [i18n] },
      attachTo: document.body,
    })
    await flushPromises()
    expect(animation.play).toHaveBeenCalledTimes(opening + 1)
    for (const key of ['schedule.exportCopy', 'schedule.exportDownload']) {
      const button = [...document.querySelectorAll('button')].find(button =>
        button.textContent?.includes(i18n.global.t(key)),
      )
      expect(button).toBeDefined()
      expect(button!.disabled).toBe(false)
    }
    wrapper.unmount()
    expect(animation.destroy).toHaveBeenCalledTimes(opening + 1)
  }
})
