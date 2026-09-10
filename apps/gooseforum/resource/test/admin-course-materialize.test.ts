// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import CourseMaterializePanel from '../src/admin/components/CourseMaterializePanel.vue'
import { materializePkCalendar } from '../src/admin/runtime/api'
import type { PkMaterializeResult, PkSyncStatusItem } from '../src/admin/types'

vi.mock('../src/admin/runtime/api', () => ({ materializePkCalendar: vi.fn() }))
vi.mock('../src/admin/runtime/i18n-text', () => ({
  adminText: (key: string, params?: unknown) => key + (params ? JSON.stringify(params) : ''),
}))
const calendar: PkSyncStatusItem = { calendarId: 122, calendarName: '2026-2027-1', status: 'completed', rowsWritten: 12, totalPages: 1, lastCommittedPage: 1, errorMsg: '' }
const report: PkMaterializeResult = { calendarId: 122, coursesInserted: 1, coursesUpdated: 2, instructorsInserted: 1, aliasesInserted: 2, aliasesSkipped: 0, offeringsInserted: 1, offeringsUpdated: 3 }
let wrapper: VueWrapper | undefined
afterEach(() => { wrapper?.unmount(); vi.resetAllMocks() })

describe('local course materialization', () => {
  test('needs a local calendar, not an upstream cookie', async () => {
    wrapper = mount(CourseMaterializePanel, { props: { calendars: [] } })
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    await wrapper.setProps({ calendars: [calendar] })
    expect(wrapper.get('button').attributes('disabled')).toBeUndefined()
    await wrapper.setProps({ calendars: [{ ...calendar, status: 'running' }] })
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
  })

  test('prevents duplicate submissions and reports committed counts', async () => {
    let finish!: (value: PkMaterializeResult) => void
    vi.mocked(materializePkCalendar).mockReturnValue(new Promise(resolve => { finish = resolve }))
    wrapper = mount(CourseMaterializePanel, { props: { calendars: [calendar] } })
    await wrapper.get('button').trigger('click')
    expect(materializePkCalendar).toHaveBeenCalledWith('122')
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    expect(wrapper.get('select').attributes('disabled')).toBeDefined()
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    finish(report)
    await flushPromises()
    expect(wrapper.get('[role="status"]').text()).toContain('"added":1,"updated":3')
    expect(wrapper.get('button').attributes('disabled')).toBeUndefined()
  })

  test('retains failure feedback and permits a retry', async () => {
    vi.mocked(materializePkCalendar).mockRejectedValueOnce(new Error('partial snapshot'))
    wrapper = mount(CourseMaterializePanel, { props: { calendars: [calendar] } })
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toBe('partial snapshot')
    expect(wrapper.get('button').attributes('disabled')).toBeUndefined()
    vi.mocked(materializePkCalendar).mockResolvedValue(report)
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
  })
})
