// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { expect, it, vi } from 'vitest'
import zh from '../src/locales/zh'
import CampusMapMinePanel from '../src/site/components/CampusMapMinePanel.vue'

const api = vi.hoisted(() => ({ status: vi.fn(), dataset: vi.fn() }))
vi.mock('@/runtime/campus-api', () => ({ campusAPI: api }))

it('retains a multi-location course and requires an explicit location choice', async () => {
  api.status.mockResolvedValue({ binding: { revision: 'test-binding', needsAuthorization: false } })
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit',
  }).formatToParts(new Date())
  const part = (type: Intl.DateTimeFormatPartTypes) => parts.find(item => item.type === type)?.value ?? ''
  const date = `${part('year')}-${part('month')}-${part('day')}`
  const event = {
    name: '测试课程', campus: '四平路校区', room: '单周北115，双周南203，实验室',
    day: 1, start: 1, end: 2, weeks: [1, 2], teacher: '', credits: '',
  }
  api.dataset.mockImplementation(async (key: string) => ({
    key, status: 'ready', metrics: [], events: key === 'today' ? [event] : [],
    ...(key === 'today' ? { teachingDay: { date } } : {}),
  }))
  const north = { campusId: 'siping', featureId: 'north' }
  const south = { campusId: 'siping', featureId: 'south' }
  const resolveLocation = vi.fn().mockResolvedValue([
    { raw: '单周北115', building: '北', room: '115', condition: '单周', target: north },
    { raw: '双周南203', building: '南', room: '203', condition: '双周', target: south },
    { raw: '实验室', building: '实验室', room: '', condition: '' },
  ])
  const wrapper = mount(CampusMapMinePanel, {
    props: { authenticated: true, resolveLocation },
    global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh } })] },
  })
  await flushPromises()
  await wrapper.get('.atlas-mine__course').trigger('click')
  await flushPromises()

  expect(wrapper.get('.atlas-mine__selected').text()).toContain(event.room)
  expect(wrapper.emitted('select')?.at(-1)).toEqual([null])
  const choices = wrapper.findAll('.atlas-location-choices button')
  expect(choices).toHaveLength(3)
  expect(choices[0]!.text()).toContain('单周')
  expect(choices[1]!.text()).toContain('双周')
  expect(choices[2]!.attributes('disabled')).toBeDefined()
  expect(choices[2]!.text()).toContain('未定位')
  await choices[1]!.trigger('click')
  expect(wrapper.emitted('select')?.at(-1)).toEqual([south])

  window.dispatchEvent(new Event('goose:session-cleared'))
  await flushPromises()
  expect(wrapper.find('.atlas-location-choices').exists()).toBe(false)
  expect(wrapper.text()).not.toContain(event.room)
  expect(wrapper.emitted('select')?.at(-1)).toEqual([null])
  wrapper.unmount()
})

it('ignores the first deferred location lookup after selecting A then B then A again', async () => {
  api.status.mockResolvedValue({ binding: { revision: 'test-binding', needsAuthorization: false } })
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit',
  }).formatToParts(new Date())
  const part = (type: Intl.DateTimeFormatPartTypes) => parts.find(item => item.type === type)?.value ?? ''
  const date = `${part('year')}-${part('month')}-${part('day')}`
  const events = [
    { name: '课程 A', campus: '四平路校区', room: '北115', day: 1, start: 1, end: 2, weeks: [1], teacher: '', credits: '' },
    { name: '课程 B', campus: '四平路校区', room: '南203', day: 1, start: 3, end: 4, weeks: [1], teacher: '', credits: '' },
  ]
  api.dataset.mockImplementation(async (key: string) => ({
    key, status: 'ready', metrics: [], events: key === 'today' ? events : [],
    ...(key === 'today' ? { teachingDay: { date } } : {}),
  }))
  const current = { campusId: 'siping', featureId: 'current-a' }
  let finishFirst!: (value: unknown) => void
  const resolveLocation = vi.fn()
    .mockImplementationOnce(() => new Promise(resolve => { finishFirst = resolve }))
    .mockResolvedValueOnce([{ raw: '课程 B 的地点', building: '南楼', room: '203', condition: '', target: { campusId: 'siping', featureId: 'b' } }])
    .mockResolvedValueOnce([{ raw: '当前课程 A 的地点', building: '北楼', room: '115', condition: '', target: current }])
  const wrapper = mount(CampusMapMinePanel, {
    props: { authenticated: true, resolveLocation },
    global: { plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh } })] },
  })
  await flushPromises()
  const courses = wrapper.findAll('.atlas-mine__course')
  await courses[0]!.trigger('click')
  await courses[1]!.trigger('click')
  await flushPromises()
  await courses[0]!.trigger('click')
  await flushPromises()
  expect(wrapper.emitted('select')?.at(-1)).toEqual([current])
  finishFirst([{ raw: '过时的课程 A 地点', building: '北楼', room: '115', condition: '', target: { campusId: 'siping', featureId: 'stale-a' } }])
  await flushPromises()
  expect(wrapper.emitted('select')?.at(-1)).toEqual([current])
  expect(wrapper.get('.atlas-location-choices').text()).toContain('当前课程 A 的地点')
  expect(wrapper.text()).not.toContain('过时的课程 A 地点')
  wrapper.unmount()
})
