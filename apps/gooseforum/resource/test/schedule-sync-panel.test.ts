// @vitest-environment happy-dom
import { afterEach, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import ScheduleSyncPanel from '../src/site/components/schedule/ScheduleSyncPanel.vue'
import { scheduleSync } from '../src/site/composables/useScheduleSync'
import en from '../src/locales/en'
afterEach(() => {
  scheduleSync.stop()
  vi.restoreAllMocks()
})
it('requires a choice only for conflicting fields and leaves a clear recovery action', async () => {
  const plan = {
    id: 'p',
    name: 'Plan',
    createdAt: 1,
    stagedCourses: [],
    selectedCourses: [],
    customEvents: [],
  }
  scheduleSync.conflicts.value = [
    {
      id: 'p',
      base: plan,
      local: plan,
      remote: { plan, revision: 2, updatedAt: '' },
      fields: [{ path: ['name'], local: 'Local title', remote: 'Remote title' }],
    },
  ]
  const resolve = vi.spyOn(scheduleSync, 'resolveConflict').mockResolvedValue()
  const wrapper = mount(ScheduleSyncPanel, {
    global: { plugins: [createI18n({ legacy: false, locale: 'en', messages: { en } })] },
  })
  expect(wrapper.get('button').attributes('disabled')).toBeDefined()
  expect(wrapper.text()).toContain('Local title')
  expect(wrapper.text()).toContain('Remote title')
  await wrapper.get('input[value="remote"]').setValue()
  await wrapper.get('button').trigger('click')
  expect(resolve).toHaveBeenCalledWith('p', { '["name"]': 'remote' })
  scheduleSync.conflicts.value = []
  scheduleSync.drafts.value = { p: plan }
  await wrapper.vm.$nextTick()
  expect(wrapper.text()).toContain('Recovery drafts')
  expect(wrapper.text()).toContain('Restore as new plan')
  expect(wrapper.findAll('input')).toHaveLength(0)
  wrapper.unmount()
})
