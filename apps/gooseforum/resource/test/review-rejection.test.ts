// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import * as api from '../src/runtime/api'
import { twemojiVuePlugin } from '../src/runtime/twemoji'
import ModerationReviewQueue from '../src/site/components/ModerationReviewQueue.vue'
import RejectionReasonFields from '../src/site/components/RejectionReasonFields.vue'
import RejectionReasonDialog from '../src/admin/components/RejectionReasonDialog.vue'
import ReviewQueueManagementPage from '../src/admin/pages/management/ReviewQueueManagementPage.vue'
import ModerationActionPage from '../src/site/pages/ModerationActionPage.vue'
import * as adminApi from '../src/admin/runtime/api'
import { adminText } from '../src/admin/runtime/i18n-text'
let wrapper: VueWrapper | undefined
afterEach(() => { wrapper?.unmount(); wrapper = undefined; vi.restoreAllMocks(); window.history.replaceState({}, '', '/') })
test('moderator must explain rejection and can retry without losing the explanation', async () => {
  i18n.global.locale.value = 'en'
  vi.spyOn(api, 'fetchModerationReviewQueue').mockResolvedValue({ items: [{ id: 7, revisionId: 19, title: 'Candidate', excerpt: 'Draft', userId: 9, username: 'author', processStatus: 2, createdAt: '', images: [] }], total: 1, page: 1, pageSize: 20 })
  const action = vi.spyOn(api, 'moderationReviewAction').mockRejectedValueOnce(new Error('Try again')).mockResolvedValue('success')
  wrapper = mount(ModerationReviewQueue, { global: { plugins: [i18n, twemojiVuePlugin] }, attachTo: document.body })
  await flushPromises()
  await wrapper.findAll('button').find(b => b.text() === i18n.global.t('moderation.review.reject'))!.trigger('click')
  expect(wrapper.find('select').exists()).toBe(true)
  expect(action).not.toHaveBeenCalled()
  await wrapper.find('select').setValue('other')
  await wrapper.findAll('button').find(b => b.text() === i18n.global.t('moderation.review.confirmReject'))!.trigger('click')
  expect(action).not.toHaveBeenCalled()
  expect(wrapper.find('[role="alert"]').exists()).toBe(true)
  await wrapper.find('textarea').setValue('  Please remove the private address.  ')
  await wrapper.findAll('button').find(b => b.text() === i18n.global.t('moderation.review.confirmReject'))!.trigger('click')
  await flushPromises()
  expect(action).toHaveBeenLastCalledWith('topic', 7, false, 19, 'Please remove the private address.')
  expect((wrapper.find('textarea').element as HTMLTextAreaElement).value).toContain('private address')
  await wrapper.findAll('button').find(b => b.text() === i18n.global.t('moderation.review.confirmReject'))!.trigger('click')
  await flushPromises()
  expect(action).toHaveBeenCalledTimes(2)
  expect(wrapper.text()).not.toContain('Candidate')
})

test('shared rejection dialog validates Unicode totals and disables pending controls', async () => {
  i18n.global.locale.value = 'en'
  wrapper = mount(RejectionReasonDialog, { props: { open: true }, global: { plugins: [i18n] }, attachTo: document.body })
  await flushPromises()
  const fields = wrapper.getComponent(RejectionReasonFields)
  await fields.find('select').setValue('other')
  await fields.find('textarea').setValue('😀'.repeat(513))
  await new DOMWrapper(document.querySelector('[data-test="confirm-rejection"]')!).trigger('click')
  expect(wrapper.emitted('confirm')).toBeUndefined()
  expect(document.querySelector('[role="alert"]')?.textContent).toContain('512')
  await fields.find('textarea').setValue('😀'.repeat(512))
  await new DOMWrapper(document.querySelector('[data-test="confirm-rejection"]')!).trigger('click')
  expect(wrapper.emitted('confirm')?.[0]?.[0]).toBe('😀'.repeat(512))
  await wrapper.setProps({ busy: true })
  expect(fields.find('select').attributes('disabled')).toBeDefined()
  expect(fields.find('textarea').attributes('disabled')).toBeDefined()
  document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
  await flushPromises()
  expect(wrapper.emitted('update:open')).toBeUndefined()
  await wrapper.setProps({ busy: false })
  const cancel = [...document.querySelectorAll('[role="dialog"] button')].find(b => b.textContent?.trim() === i18n.global.t('common.cancel'))!
  await new DOMWrapper(cancel).trigger('click')
  expect(wrapper.emitted('update:open')?.at(-1)).toEqual([false])
})

test('admin queue sends the exact version and retains the explanation after failure', async () => {
  i18n.global.locale.value = 'en'
  vi.spyOn(adminApi, 'getReviewQueue').mockResolvedValueOnce({ items: [{ id: 7, revisionId: 19, title: 'Candidate', excerpt: 'Draft', userId: 9, username: 'author', processStatus: 2, createdAt: '', images: [] }], total: 1, page: 1, pageSize: 10 }).mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 10 })
  const action = vi.spyOn(adminApi, 'reviewAction').mockRejectedValueOnce(new Error('Try again')).mockResolvedValue('success')
  wrapper = mount(ReviewQueueManagementPage, { props: { payload: {} as never }, global: { plugins: [i18n, twemojiVuePlugin] }, attachTo: document.body })
  await flushPromises()
  await wrapper.findAll('button').find(b => b.text() === adminText('k00gk'))!.trigger('click')
  await flushPromises()
  const fields = wrapper.getComponent(RejectionReasonFields)
  await fields.find('select').setValue('other')
  await fields.find('textarea').setValue('Please remove the private address.')
  await new DOMWrapper(document.querySelector('[data-test="confirm-rejection"]')!).trigger('click')
  await flushPromises()
  expect(action).toHaveBeenLastCalledWith('topic', 7, false, 19, 'Please remove the private address.')
  expect((fields.find('textarea').element as HTMLTextAreaElement).value).toContain('private address')
  expect(document.querySelector('[role="alert"]')?.textContent).toBe('Try again')
  await new DOMWrapper(document.querySelector('[data-test="confirm-rejection"]')!).trigger('click')
  await flushPromises()
  expect(action).toHaveBeenCalledTimes(2)
  expect(document.querySelector('[role="dialog"]')).toBeNull()
})

test('webhook confirmation requires an explanation and keeps it after a failed execution', async () => {
  i18n.global.locale.value = 'en'
  window.history.replaceState({}, '', '/moderation/action?token=synthetic-action')
  const ready = { state: 'ready' as const, subject: 'review.topic' as const, action: 'reject' as const, targetType: 'topic' as const, targetId: 7, title: 'Candidate', workbenchUrl: '/moderation' }
  vi.spyOn(api, 'previewModerationApprovalAction').mockResolvedValue(ready)
  const execute = vi.spyOn(api, 'executeModerationApprovalAction').mockRejectedValueOnce(new Error('Try again')).mockResolvedValue({ ...ready, state: 'done' })
  wrapper = mount(ModerationActionPage, { props: { layout: {} as never, props: {} as never }, global: { plugins: [i18n, twemojiVuePlugin] }, attachTo: document.body })
  await flushPromises()
  const confirm = () => wrapper!.findAll('button').find(b => b.text() === i18n.global.t('moderationAction.actions.reject'))!
  await confirm().trigger('click')
  expect(execute).not.toHaveBeenCalled()
  await wrapper.find('select').setValue('privacy')
  await wrapper.find('textarea').setValue('Please remove the address.')
  await confirm().trigger('click')
  await flushPromises()
  expect(execute).toHaveBeenLastCalledWith('synthetic-action', `${i18n.global.t('reviewRejection.reasons.privacy')}\nPlease remove the address.`)
  expect((wrapper.find('textarea').element as HTMLTextAreaElement).value).toContain('address')
  await confirm().trigger('click')
  await flushPromises()
  expect(execute).toHaveBeenCalledTimes(2)
  expect(wrapper.find('select').exists()).toBe(false)
})

for (const state of ['ready','changed'] as const) test(`business failed execution retains the draft and checks fresh authority before retry (${state})`, async () => {
 i18n.global.locale.value = 'en'
 window.history.replaceState({}, '', '/moderation/action?token=synthetic-action')
 const ready = { state:'ready' as const, subject:'review.topic' as const, action:'reject' as const, targetType:'topic' as const, targetId:7, title:'Candidate', workbenchUrl:'/moderation' }
 const preview=vi.spyOn(api,'previewModerationApprovalAction').mockResolvedValueOnce(ready).mockResolvedValue({...ready,state})
 const execute=vi.spyOn(api,'executeModerationApprovalAction').mockResolvedValueOnce({...ready,state:'failed'}).mockResolvedValue({...ready,state:'done'})
 wrapper=mount(ModerationActionPage,{props:{layout:{} as never,props:{} as never},global:{plugins:[i18n,twemojiVuePlugin]}})
 await flushPromises()
 await wrapper.get('select').setValue('privacy')
 await wrapper.get('textarea').setValue('Keep this precise explanation.')
 await wrapper.findAll('button').find(b=>b.text()===i18n.global.t('moderationAction.actions.reject'))!.trigger('click')
 await flushPromises()
 expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('Keep this precise explanation.')
 expect(wrapper.get('textarea').attributes('disabled')).toBeDefined()
 await wrapper.get('[data-testid="moderation-recheck"]').trigger('click')
 await flushPromises()
 expect(preview).toHaveBeenCalledTimes(2)
 expect(execute).toHaveBeenCalledTimes(1)
 if(state==='ready'){
  expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe('Keep this precise explanation.')
  await wrapper.findAll('button').find(b=>b.text()===i18n.global.t('moderationAction.actions.reject'))!.trigger('click')
  await flushPromises()
  expect(execute).toHaveBeenLastCalledWith('synthetic-action',`${i18n.global.t('reviewRejection.reasons.privacy')}\nKeep this precise explanation.`)
 }else{expect(wrapper.find('textarea').exists()).toBe(false)}
})
