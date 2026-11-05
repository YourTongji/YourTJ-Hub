// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import * as reka from 'reka-ui'
import { i18n, setLocale } from '../src/runtime/i18n'
import { hydrateFlashMessages, useFlashMessages } from '../src/runtime/flash-message'
import ScheduleRoughList from '../src/site/components/schedule/ScheduleRoughList.vue'
import ScheduleMajorSelector from '../src/site/components/schedule/ScheduleMajorSelector.vue'
import ScheduleExportDialog from '../src/site/components/schedule/ScheduleExportDialog.vue'
import { toPng } from 'html-to-image'

vi.mock('../src/runtime/pk-api', () => ({ getPkCalendars: async () => [], getPkGrades: async () => [], getPkMajors: async () => [] }))
const save = vi.hoisted(() => vi.fn())
vi.mock('../src/site/composables/useScheduleStore', async original => {
 const actual = await original<typeof import('../src/site/composables/useScheduleStore')>()
 return { ...actual, useScheduleStore: () => ({ ...actual.useScheduleStore(), saveSelectedCourses: save, solidify: vi.fn() }) }
})
vi.mock('html-to-image', () => ({ toPng: vi.fn() }))
vi.mock('../src/site/utils/imgFxCanvas', () => ({ ImgFxController: class { resize() {} play(done: () => void) { done() } destroy() {} } }))
const stubs = Object.fromEntries(Object.keys(reka).filter(name => /^(Dialog|Popover|Combobox)/.test(name)).map(name => [name, { template: '<div><slot /></div>' }]))
beforeEach(async () => {
 await setLocale('zh')
 save.mockReset()
 sessionStorage.clear()
 hydrateFlashMessages()
 vi.stubGlobal('ClipboardItem', class {})
 Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText: vi.fn().mockResolvedValue(undefined), write: vi.fn().mockResolvedValue(undefined) } })
})
afterEach(() => {
 vi.restoreAllMocks(); vi.unstubAllGlobals()
 const flash = useFlashMessages(); flash.messages.value.forEach(item => flash.dismiss(item.id))
 sessionStorage.clear()
})
function feedback(type: string) {
 expect(useFlashMessages().messages.value.at(-1)?.type).toBe(type)
 expect(sessionStorage.getItem('goose:flash-messages')).toBeNull()
}
test.each([false, true])('local timetable save reports immediately (failure=%s)', async failure => {
 save.mockImplementation(() => { if (failure) throw new Error('storage full') })
 const wrapper = mount(ScheduleRoughList, { global: { plugins: [i18n], stubs } })
 try { await wrapper.get('button[title="保存课表"]').trigger('click'); feedback(failure ? 'error' : 'success') }
 finally { wrapper.unmount() }
})
test.each([false, true])('query link clipboard feedback is immediate (failure=%s)', async failure => {
 if (failure) vi.mocked(navigator.clipboard.writeText).mockRejectedValue(new Error('denied'))
 const wrapper = mount(ScheduleMajorSelector, { global: { plugins: [i18n], stubs } })
 try {
  await flushPromises()
  await wrapper.findAll('button').find(b => b.text() === '复制')!.trigger('click')
  await flushPromises(); feedback(failure ? 'info' : 'success')
 } finally { wrapper.unmount() }
})
test.each(['copy-success', 'copy-failure', 'download-failure'])('image export feedback is immediate: %s', async scenario => {
 vi.mocked(toPng).mockResolvedValue('data:image/png;base64,AA==')
 if (scenario !== 'copy-success') vi.mocked(toPng).mockRejectedValue(new Error('render failed'))
 const wrapper = mount(ScheduleExportDialog, { props: { open: true, cellCourses: [], cellSpans: [], occupiedGrid: [], activeCalendar: null, activePlanName: '测试', weekViewLabel: '', exportSummaryText: '' }, global: { plugins: [i18n], stubs } })
 try {
  await flushPromises()
  const label = String(i18n.global.t(scenario === 'download-failure' ? 'schedule.exportDownload' : 'schedule.exportCopy'))
  await wrapper.findAll('button').find(b => b.text() === label)!.trigger('click')
  await flushPromises(); feedback(scenario === 'copy-success' ? 'success' : 'error')
 } finally { wrapper.unmount() }
})
