// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import SchedulePage from '../src/site/pages/SchedulePage.vue'
import { i18n, setLocale } from '../src/runtime/i18n'
import { hydrateFlashMessages, useFlashMessages } from '../src/runtime/flash-message'
import { scheduleSync } from '../src/site/composables/useScheduleSync'
import type { LayoutPayload } from '@gooseforum/client'
vi.mock('../src/runtime/pk-api', () => ({ getPkLatestUpdate: async () => ({}), syncPkCourseInfo: vi.fn() }))
vi.mock('../src/site/composables/useScheduleSync', async original => ({ ...await original<typeof import('../src/site/composables/useScheduleSync')>(), startScheduleSync: vi.fn(), stopScheduleSync: vi.fn() }))
afterEach(() => { vi.restoreAllMocks(); const flash = useFlashMessages(); flash.messages.value.forEach(item => flash.dismiss(item.id)); sessionStorage.clear() })
test.each([false, true])('saving gives immediate feedback on the current document (success=%s)', async success => {
 await setLocale('zh')
 hydrateFlashMessages()
 vi.spyOn(scheduleSync, 'needsOwnerConfirmation').mockReturnValue(false)
 vi.spyOn(scheduleSync, 'saveNow').mockResolvedValue(success)
 vi.spyOn(scheduleSync, 'syncOnPageEnter').mockResolvedValue(undefined)
 const wrapper = shallowMount(SchedulePage, { props: { layout: { viewer: { id:42, isAuthenticated:true } } as LayoutPayload, props: {} }, global: { plugins:[i18n], stubs: { PageHeader: { template:'<div><slot name="actions" /></div>' } } } })
 try {
  await flushPromises()
  await wrapper.get('button.gf-button-primary').trigger('click')
  await flushPromises()
  expect(useFlashMessages().messages.value.at(-1)?.type).toBe(success ? 'success':'error')
  expect(sessionStorage.getItem('goose:flash-messages')).toBeNull()
 } finally { wrapper.unmount() }
})

test('guest save immediately explains login without attempting a cloud write', async () => {
 const save = vi.spyOn(scheduleSync, 'saveNow')
 const wrapper = shallowMount(SchedulePage, { props: { layout: {} as LayoutPayload, props: {} }, global: { plugins:[i18n], stubs: { PageHeader: { template:'<div><slot name="actions" /></div>' } } } })
 try {
  await wrapper.get('button.gf-button-primary').trigger('click')
  expect(useFlashMessages().messages.value.at(-1)?.type).toBe('warning')
  expect(save).not.toHaveBeenCalled()
  expect(sessionStorage.getItem('goose:flash-messages')).toBeNull()
 } finally { wrapper.unmount() }
})
