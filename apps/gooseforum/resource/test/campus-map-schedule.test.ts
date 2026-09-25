// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { expect, it, vi } from 'vitest'
import zh from '../src/locales/zh'
import CampusMapSchedulePanel from '../src/site/components/CampusMapSchedulePanel.vue'

const api = vi.hoisted(() => ({ calendars: vi.fn(), byTime: vi.fn(), details: vi.fn(), latest: vi.fn() }))
vi.mock('@/runtime/pk-api', () => ({
  getPkCalendars: api.calendars,
  getPkCoursesByTime: api.byTime,
  getPkCourseDetails: api.details,
  getPkLatestUpdate: api.latest,
}))

it('discards an in-flight query when its period changes', async () => {
  api.calendars.mockResolvedValue([{ calendarId: 122, calendarName: 'Current term' }])
  api.latest.mockResolvedValue({ latestSyncAt: null })
  let finish!: (value: unknown) => void
  api.byTime.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
  api.details.mockResolvedValue({ TJCS101: [{ campus: '嘉定校区', teachingClassId: 1,
    arrangementInfo: [{ arrangementText: '周一第3-4节', occupyDay: 1, occupyTime: [3, 4], occupyWeek: [1], occupyRoom: '安楼A101' }],
  }] })
  const wrapper = mount(CampusMapSchedulePanel, {
    props: { resolveLocation: vi.fn() },
    global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh } })] },
  })
  await flushPromises()
  await wrapper.get('.atlas-schedule__submit').trigger('click')
  await wrapper.findAll('select').at(-1)!.setValue('2')
  finish({ courses: [{ courseCode: 'TJCS101', courseName: 'Stale course' }] })
  await flushPromises()
  expect(wrapper.text()).not.toContain('Stale course')
  expect(wrapper.get('.atlas-schedule__submit').attributes('disabled')).toBeUndefined()
  wrapper.unmount()
})

it('filters synced arrangements and emits the uniquely resolved map target', async () => {
  api.calendars.mockResolvedValue([{ calendarId: 122, calendarName: '2026-2027学年第一学期' }])
  api.latest.mockResolvedValue({ latestSyncAt: '2026-09-23' })
  api.byTime.mockResolvedValue({ courses: [{ courseCode: 'TJCS101', courseName: '程序设计' }] })
  api.details.mockResolvedValue({
    TJCS101: [{
      campus: '嘉定校区',
      teachingClassId: 1,
      arrangementInfo: [
        { arrangementText: '周一第1-2节', occupyDay: 1, occupyTime: [1, 2], occupyWeek: [1], occupyRoom: '安楼A101', teacherAndCode: '张老师' },
        { arrangementText: '周一第3-4节', occupyDay: 1, occupyTime: [3, 4], occupyWeek: [1], occupyRoom: '博楼B201', teacherAndCode: '张老师' },
      ],
    }],
  })
  const resolveLocation = vi.fn().mockResolvedValue({ campusId: 'jiading', featureId: 'a' })
  const wrapper = mount(CampusMapSchedulePanel, {
    props: { resolveLocation },
    global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh } })] },
  })
  await flushPromises()
  await wrapper.get('.atlas-schedule__submit').trigger('click')
  await flushPromises()

  expect(api.byTime).toHaveBeenCalledWith(122, 1, 1, true)
  expect(wrapper.findAll('.atlas-schedule__list button')).toHaveLength(1)
  expect(wrapper.text()).toContain('安楼A101')
  await wrapper.get('.atlas-schedule__list button').trigger('click')
  await flushPromises()
  expect(resolveLocation).toHaveBeenCalledWith('嘉定校区', '安楼A101')
  expect(wrapper.emitted('select')?.at(-1)).toEqual([{ campusId: 'jiading', featureId: 'a' }])
  wrapper.unmount()
})

it('limits the result list to the selected building', async () => {
  api.calendars.mockResolvedValue([{ calendarId: 122, calendarName: '2026-2027学年第一学期' }])
  api.latest.mockResolvedValue({ latestSyncAt: '2026-09-23' })
  api.byTime.mockResolvedValue({ courses: [{ courseCode: 'TJCS101', courseName: '程序设计' }] })
  api.details.mockResolvedValue({
    TJCS101: [{ campus: '嘉定校区', teachingClassId: 1, arrangementInfo: [
      { arrangementText: '周一第1-2节', occupyDay: 1, occupyTime: [1, 2], occupyWeek: [1], occupyRoom: '安楼A101' },
      { arrangementText: '周一第1-2节', occupyDay: 1, occupyTime: [1, 2], occupyWeek: [1], occupyRoom: '博楼B201' },
    ] }],
  })
  const wrapper = mount(CampusMapSchedulePanel, {
    props: {
      resolveLocation: vi.fn(),
      building: { campusId: 'jiading', featureId: 'an', name: '安楼' },
      matchLocation: (_campus: string, room: string) => room.startsWith('安楼')
        ? { campusId: 'jiading', featureId: 'an' }
        : { campusId: 'jiading', featureId: 'bo' },
    },
    global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh } })] },
  })
  await flushPromises()
  await wrapper.get('.atlas-schedule__submit').trigger('click')
  await flushPromises()

  expect(wrapper.findAll('.atlas-schedule__list button')).toHaveLength(1)
  expect(wrapper.text()).toContain('安楼A101')
  expect(wrapper.text()).not.toContain('博楼B201')
  expect(wrapper.text()).toContain('课程模块最近同步：2026-09-23')
  wrapper.unmount()
})

it('converts a selected date to the course module week and weekday', async () => {
  api.calendars.mockResolvedValue([{ calendarId: 122, calendarName: '2026-2027学年第一学期', startDate: '2026-09-07', endDate: '2027-01-31' }])
  api.latest.mockResolvedValue({ latestSyncAt: '2026-09-23' })
  api.byTime.mockResolvedValue({ courses: [] })
  const wrapper = mount(CampusMapSchedulePanel, {
    props: { resolveLocation: vi.fn() },
    global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh } })] },
  })
  await flushPromises()
  await wrapper.findAll('select')[1]!.setValue('true')
  await wrapper.get('input[type="date"]').setValue('2026-09-14')
  await wrapper.get('.atlas-schedule__submit').trigger('click')
  await flushPromises()

  expect(api.byTime).toHaveBeenCalledWith(122, 1, 1, true)
  expect(wrapper.text()).toContain('课程日期：2026-09-14')
  wrapper.unmount()
})

it('keeps period-12 arrangements for historical 12-section calendars', async () => {
  api.calendars.mockResolvedValue([{ calendarId: 119, calendarName: '2024-2025学年第二学期' }])
  api.latest.mockResolvedValue({ latestSyncAt: '2026-09-23' })
  api.byTime.mockResolvedValue({ courses: [{ courseCode: 'TJCS301', courseName: '艺术鉴赏' }] })
  api.details.mockResolvedValue({
    TJCS301: [{ campus: '嘉定校区', teachingClassId: 2, arrangementInfo: [
      { arrangementText: '周一第10-12节', occupyDay: 1, occupyTime: [10, 11, 12], occupyWeek: [1], occupyRoom: 'D404' },
    ] }],
  })
  const wrapper = mount(CampusMapSchedulePanel, {
    props: { resolveLocation: vi.fn() },
    global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh } })] },
  })
  await flushPromises()
  await wrapper.findAll('select')[3]!.setValue('6')
  await wrapper.get('.atlas-schedule__submit').trigger('click')
  await flushPromises()

  expect(api.byTime).toHaveBeenCalledWith(119, 1, 6, true)
  expect(wrapper.findAll('.atlas-schedule__list button')).toHaveLength(1)
  expect(wrapper.text()).toContain('周一第10-12节')
  expect(wrapper.findAll('select')[3]!.findAll('option')[5]!.text()).toContain('第 10–12 节')
  wrapper.unmount()
})

it('drops period-12-only arrangements for current 11-section calendars', async () => {
  api.calendars.mockResolvedValue([{ calendarId: 122, calendarName: '2026-2027学年第一学期' }])
  api.latest.mockResolvedValue({ latestSyncAt: '2026-09-23' })
  api.byTime.mockResolvedValue({ courses: [{ courseCode: 'TJCS101', courseName: '程序设计' }] })
  api.details.mockResolvedValue({
    TJCS101: [{ campus: '嘉定校区', teachingClassId: 1, arrangementInfo: [
      { arrangementText: '周一第10-11节', occupyDay: 1, occupyTime: [10, 11], occupyWeek: [1], occupyRoom: '安楼A101' },
      { arrangementText: '周一第12节', occupyDay: 1, occupyTime: [12], occupyWeek: [1], occupyRoom: '安楼A102' },
    ] }],
  })
  const wrapper = mount(CampusMapSchedulePanel, {
    props: { resolveLocation: vi.fn() },
    global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh } })] },
  })
  await flushPromises()
  await wrapper.findAll('select')[3]!.setValue('6')
  await wrapper.get('.atlas-schedule__submit').trigger('click')
  await flushPromises()

  expect(api.byTime).toHaveBeenCalledWith(122, 1, 6, true)
  expect(wrapper.findAll('.atlas-schedule__list button')).toHaveLength(1)
  expect(wrapper.text()).toContain('安楼A101')
  expect(wrapper.text()).not.toContain('安楼A102')
  wrapper.unmount()
})
