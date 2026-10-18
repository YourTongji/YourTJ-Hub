// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import * as api from '../src/runtime/api'
import { i18n } from '../src/runtime/i18n'
import ModerationActionPage from '../src/site/pages/ModerationActionPage.vue'
import type { ModerationApprovalActionView } from '@gooseforum/client'

const wrappers: VueWrapper[] = []
afterEach(() => { wrappers.splice(0).forEach(w => w.unmount()); vi.restoreAllMocks(); window.history.replaceState(null, '', '/') })
const view = (extra: Partial<ModerationApprovalActionView> = {}): ModerationApprovalActionView => ({
  state: 'ready', subject: 'appeal', action: 'reject', targetType: 'topic', targetId: 7, title: 'Course notes',
  excerpt: 'Rejected revision', originalReason: 'Looks like a duplicate', appealReason: 'Different angle, sources added',
  workbenchUrl: '/moderation?tab=appeals', expiresAt: '2099-01-02T00:00:00Z', ...extra,
})
async function mountPage(preview: ModerationApprovalActionView) {
  window.history.replaceState(null, '', '/moderation/action?token=v1.abc.def')
  vi.spyOn(api, 'previewModerationApprovalAction').mockResolvedValue(preview)
  const w = mount(ModerationActionPage, { props: { layout: {} as never, props: {} }, global: { plugins: [i18n] }, attachTo: document.body })
  wrappers.push(w)
  await flushPromises()
  return w
}

test('appeal rejection shows both sides and requires a reason the author will see', async () => {
  i18n.global.locale.value = 'en'
  const execute = vi.spyOn(api, 'executeModerationApprovalAction').mockResolvedValue(view({ state: 'done' }))
  const w = await mountPage(view())
  expect(w.text()).toContain('Reject appeal')
  expect(w.text()).toContain('Looks like a duplicate')
  expect(w.text()).toContain('Different angle, sources added')
  expect(w.get('label[for="moderation-action-reason"]').text()).toBe('Reason for rejecting')

  const confirm = w.findAll('button').find(b => b.text() === 'Reject appeal')!
  await confirm.trigger('click'); await flushPromises()
  expect(execute).not.toHaveBeenCalled()
  expect(w.get('#moderation-action-reason').attributes('aria-invalid')).toBe('true')
  expect(w.get('#moderation-action-reason-help').text()).toContain('Add a reason before rejecting.')

  await w.get('#moderation-action-reason').setValue('  Duplicates an existing topic  ')
  await confirm.trigger('click'); await flushPromises()
  expect(execute).toHaveBeenCalledWith('v1.abc.def', 'Duplicates an existing topic')
  expect(w.get('[role="status"]').text()).toBe(i18n.global.t('moderationAction.states.done'))
})

test('appeal approval needs no reason; stale appeals and other subjects keep their own copy', async () => {
  i18n.global.locale.value = 'en'
  const execute = vi.spyOn(api, 'executeModerationApprovalAction').mockResolvedValue(view({ action: 'approve', state: 'done' }))
  const approve = await mountPage(view({ action: 'approve' }))
  expect(approve.get('label[for="moderation-action-reason"]').text()).toBe('Note to the author (optional)')
  await approve.findAll('button').find(b => b.text() === 'Approve appeal and publish')!.trigger('click'); await flushPromises()
  expect(execute).toHaveBeenCalledWith('v1.abc.def', '')

  const stale = await mountPage(view({ state: 'changed' }))
  expect(stale.get('[role="status"]').text()).toContain('This appeal needs no action.')
  expect(stale.find('#moderation-action-reason').exists()).toBe(false)

  const review = await mountPage({ state: 'ready', subject: 'review.topic', action: 'reject', targetType: 'topic', title: 'Pending', workbenchUrl: '/moderation?tab=review' })
  expect(review.find('#moderation-action-reason').exists()).toBe(false)
  expect(review.findAll('button').some(b => b.text() === i18n.global.t('moderationAction.actions.reject'))).toBe(true)
})
