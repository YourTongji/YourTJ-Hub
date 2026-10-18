// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import * as api from '../src/runtime/api'
import { i18n } from '../src/runtime/i18n'
import ContentAppeal from '../src/site/components/ContentAppeal.vue'
import ManagedContent from '../src/site/components/ManagedContent.vue'
import ModerationAppealQueue from '../src/site/components/ModerationAppealQueue.vue'
const wrappers: VueWrapper[] = []
afterEach(() => { wrappers.splice(0).forEach(w => w.unmount()); vi.restoreAllMocks() })
const appeal = (id = 1): api.AppealItem => ({ id, kind: 'topic', targetId: 7, revisionId: 70, topicId: 7, postNo: 0, title: 'Candidate', content: 'Rejected revision', images: ['/file/img/a.png'], reason: 'Please reconsider', originalReviewReason: 'Looks like spam', status: 'pending', reviewReason: '', createdAt: '2026-10-10T00:00:00Z', reviewedAt: null })
const content = (extra: Partial<api.MyContentItem> = {}) => ({ id: 7, contentType: 'topic', revisionId: 70, processStatus: 1, hasPublishedVersion: false, reviewReason: 'Looks like spam', title: 'Public title', content: 'Candidate', images: [], createdAt: '', canAppeal: true, appeal: null, ...extra }) as api.MyContentItem
function mountAuthor(item = content(), open = true) { const w = mount(ContentAppeal, { props: { item, open }, global: { plugins: [i18n] } }); wrappers.push(w); return w }
function mountQueue() { const w = mount(ModerationAppealQueue, { global: { plugins: [i18n] }, attachTo: document.body }); wrappers.push(w); return w }
const en = () => { i18n.global.locale.value = 'en' }

test('author submits the latest rejected revision and immediately sees it under review', async () => {
  en()
  const submit = vi.spyOn(api, 'submitContentAppeal').mockResolvedValue(true)
  const w = mountAuthor()
  expect(w.get('[data-test="appeal-form"]').text()).toContain('Each version can be appealed once.')
  await w.find('textarea').setValue('  Please reconsider  ')
  expect(w.text()).toContain('21/512')
  await w.find('form').trigger('submit'); await flushPromises()
  expect(submit).toHaveBeenCalledWith('topic', 7, 70, 'Please reconsider')
  expect(w.emitted('update:open')?.at(-1)).toEqual([false])
  expect(w.emitted('saved')).toHaveLength(1)
  expect(w.get('[data-test="appeal-state"]').text()).toContain('Appeal under review')
  expect(w.get('[role="status"]').text()).toBe('Appeal submitted. We’ll notify you of the decision.')
})

test('network failure keeps the reason and explains how to recover; server failures show their reason', async () => {
  en()
  const submit = vi.spyOn(api, 'submitContentAppeal').mockRejectedValueOnce(new TypeError('Failed to fetch'))
  const w = mountAuthor()
  await w.find('textarea').setValue('My explanation'); await w.find('form').trigger('submit'); await flushPromises()
  expect((w.find('textarea').element as HTMLTextAreaElement).value).toBe('My explanation')
  expect(w.get('[role="alert"]').text()).toContain('your reason is still here')
  submit.mockRejectedValueOnce(new api.ApiResponseError('You’ve already appealed this version.', 'content.appeal.duplicate'))
  await w.find('form').trigger('submit'); await flushPromises()
  expect(w.get('[role="alert"]').text()).toBe('You’ve already appealed this version.')
  expect(w.emitted('saved')).toHaveLength(1)
})

test('decided appeals explain the outcome and never reopen the form', () => {
  en()
  const w = mountAuthor(content({ canAppeal: false, appeal: { status: 'rejected', reason: 'Context', reviewReason: 'Still breaks the rules', createdAt: '2026-10-10T00:00:00Z', reviewedAt: '2026-10-11T00:00:00Z' } }))
  const state = w.get('[data-test="appeal-state"]').text()
  expect(state).toContain('Appeal not approved')
  expect(state).toContain('Still breaks the rules')
  expect(state).toContain('can’t be appealed again')
  expect(w.find('[data-test="appeal-form"]').exists()).toBe(false)
  const stale = mountAuthor(content({ canAppeal: false, appeal: { status: 'stale', reason: 'Context', reviewReason: '', createdAt: '2026-10-10T00:00:00Z', reviewedAt: null } }))
  expect(stale.text()).toContain('Appeal closed')
})

test('content management offers appeal only when allowed and warns before editing a pending appeal', async () => {
  en()
  const appealable = mount(ManagedContent, { props: { item: content() }, global: { plugins: [i18n] } }); wrappers.push(appealable)
  expect(appealable.find('[data-test="appeal-toggle"]').exists()).toBe(true)
  expect(appealable.text()).toContain('Looks like spam')
  const blocked = mount(ManagedContent, { props: { item: content({ canAppeal: false }) }, global: { plugins: [i18n] } }); wrappers.push(blocked)
  expect(blocked.find('[data-test="appeal-toggle"]').exists()).toBe(false)
  const pending = mount(ManagedContent, { props: { item: content({ contentType: 'post', canAppeal: false, appeal: { status: 'pending', reason: 'Context', reviewReason: '', createdAt: '2026-10-10T00:00:00Z', reviewedAt: null } }) }, global: { plugins: [i18n] } }); wrappers.push(pending)
  expect(pending.find('[data-test="appeal-toggle"]').exists()).toBe(false)
  await pending.findAll('button').find(b => b.text() === 'Edit and resubmit')!.trigger('click')
  expect(pending.get('[role="alert"]').text()).toContain('will close the appeal under review')
})

test('queue shows both sides, explains a missing rejection note in place, and removes decided cards', async () => {
  en()
  vi.spyOn(api, 'fetchModerationAppeals').mockResolvedValue({ items: [appeal()], total: 1, page: 1, pageSize: 20 })
  const action = vi.spyOn(api, 'moderationAppealAction').mockResolvedValue(true)
  const w = mountQueue(); await flushPromises()
  expect(w.text()).toContain('Looks like spam')
  expect(w.text()).toContain('Please reconsider')
  await w.findAll('button').find(b => b.text() === 'Reject appeal')!.trigger('click'); await flushPromises()
  expect(action).not.toHaveBeenCalled()
  expect(w.get('[role="alert"]').text()).toContain('Add a reason before rejecting')
  expect(document.activeElement).toBe(w.find('textarea').element)
  await w.find('textarea').setValue('Rules still apply')
  await w.findAll('button').find(b => b.text() === 'Reject appeal')!.trigger('click'); await flushPromises()
  expect(action).toHaveBeenCalledWith(1, false, 'Rules still apply')
  expect(w.find('[data-test="appeal-card-1"]').exists()).toBe(false)
  expect(w.emitted('count')?.at(-1)).toEqual([0])
  expect(w.text()).toContain('Rejected. The author will see your note.')
})

test('stale or already-decided appeals leave the queue with the server explanation', async () => {
  en()
  vi.spyOn(api, 'fetchModerationAppeals').mockResolvedValue({ items: [appeal()], total: 1, page: 1, pageSize: 20 })
  vi.spyOn(api, 'moderationAppealAction').mockRejectedValue(new api.ApiResponseError('The author edited or deleted this post, so the appeal is closed.', 'moderation.appeal.stale'))
  const w = mountQueue(); await flushPromises()
  await w.findAll('button').find(b => b.text() === 'Approve and publish')!.trigger('click'); await flushPromises()
  expect(w.find('[data-test="appeal-card-1"]').exists()).toBe(false)
  expect(w.text()).toContain('so the appeal is closed')
})

test('other decision failures keep the moderator note on the card', async () => {
  en()
  vi.spyOn(api, 'fetchModerationAppeals').mockResolvedValue({ items: [appeal()], total: 1, page: 1, pageSize: 20 })
  vi.spyOn(api, 'moderationAppealAction').mockRejectedValue(new TypeError('Failed to fetch'))
  const w = mountQueue(); await flushPromises()
  await w.find('textarea').setValue('Rules still apply')
  await w.findAll('button').find(b => b.text() === 'Reject appeal')!.trigger('click'); await flushPromises()
  expect(w.find('[data-test="appeal-card-1"]').exists()).toBe(true)
  expect((w.find('textarea').element as HTMLTextAreaElement).value).toBe('Rules still apply')
  expect(w.text()).toContain('Refresh the list and try again')
})

test('switching status discards late queue responses', async () => {
  en()
  let finish!: (result: { items: api.AppealItem[]; total: number; page: number; pageSize: number }) => void
  vi.spyOn(api, 'fetchModerationAppeals').mockImplementationOnce(() => new Promise(resolve => { finish = resolve })).mockResolvedValue({ items: [{ ...appeal(2), title: 'Approved candidate', status: 'approved' }], total: 1, page: 1, pageSize: 20 })
  const w = mountQueue()
  await w.findAll('button').find(b => b.text() === 'Approved')!.trigger('click'); await flushPromises()
  finish({ items: [{ ...appeal(), title: 'Late pending candidate' }], total: 1, page: 1, pageSize: 20 }); await flushPromises()
  expect(w.text()).toContain('Approved candidate'); expect(w.text()).not.toContain('Late pending candidate')
})

test('image preview closes inline and preserves moderator explanation', async () => {
  vi.spyOn(api, 'fetchModerationAppeals').mockResolvedValue({ items: [appeal()], total: 1, page: 1, pageSize: 20 })
  const w = mountQueue(); await flushPromises()
  await w.find('textarea').setValue('Review draft')
  const button = w.findAll('button').find(b => b.find('img').exists())!
  await button.trigger('click'); await flushPromises()
  expect(document.querySelector('[role="dialog"] img')?.getAttribute('src')).toBe('/file/img/a.png')
  window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' })); await flushPromises()
  expect(document.querySelector('[role="dialog"]')).toBeNull()
  expect((w.find('textarea').element as HTMLTextAreaElement).value).toBe('Review draft')
})
