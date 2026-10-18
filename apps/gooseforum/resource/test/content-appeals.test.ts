// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import * as api from '../src/runtime/api'
import { i18n } from '../src/runtime/i18n'
import ContentAppeal from '../src/site/components/ContentAppeal.vue'
import ModerationAppealQueue from '../src/site/components/ModerationAppealQueue.vue'
const wrappers: VueWrapper[] = []
afterEach(() => { wrappers.splice(0).forEach(w => w.unmount()); vi.restoreAllMocks() })
const appeal = (id = 1): api.AppealItem => ({ id, kind: 'topic', targetId: 7, revisionId: 70, topicId: 7, postNo: 0, title: 'Candidate', content: 'Rejected revision', images: ['/file/img/a.png'], reason: 'Please reconsider', status: 'pending', reviewReason: '', createdAt: '2026-10-10T00:00:00Z', reviewedAt: null })
const content = { id: 7, contentType: 'topic', revisionId: 70, processStatus: 1, hasPublishedVersion: true, title: 'Public title', content: 'Candidate', images: [], createdAt: '' } as api.MyContentItem
function mountAuthor() { const w = mount(ContentAppeal, { props: { item: content }, global: { plugins: [i18n] } }); wrappers.push(w); return w }
function mountQueue() { const w = mount(ModerationAppealQueue, { global: { plugins: [i18n] }, attachTo: document.body }); wrappers.push(w); return w }
test('author submits the rejected candidate revision once and sees decision reason', async () => {
  i18n.global.locale.value = 'en'
  const history = vi.spyOn(api, 'fetchContentAppeals').mockResolvedValueOnce({ items: [] }).mockResolvedValue({ items: [{ ...appeal(), status: 'rejected', reviewReason: 'Still violates rules' }] })
  const submit = vi.spyOn(api, 'submitContentAppeal').mockResolvedValue(true)
  const w = mountAuthor()
  await w.find('button').trigger('click'); await flushPromises()
  expect(history).toHaveBeenCalledWith('topic', 7)
  await w.find('textarea').setValue('  Please reconsider  ')
  await w.find('form').trigger('submit'); await flushPromises()
  expect(submit).toHaveBeenCalledWith('topic', 7, 70, 'Please reconsider')
  expect(w.find('textarea').exists()).toBe(false)
  expect(w.text()).toContain('Still violates rules')
})
test('failed submission keeps author explanation and remains retryable', async () => {
  vi.spyOn(api, 'fetchContentAppeals').mockResolvedValue({ items: [] })
  vi.spyOn(api, 'submitContentAppeal').mockRejectedValue(new Error('Network unavailable'))
  const w = mountAuthor(); await w.find('button').trigger('click'); await flushPromises()
  await w.find('textarea').setValue('My explanation'); await w.find('form').trigger('submit'); await flushPromises()
  expect((w.find('textarea').element as HTMLTextAreaElement).value).toBe('My explanation')
  expect(w.find('[role="alert"]').text()).toBe('Network unavailable')
})
test('queue requires rejection reason and failed decision retains the draft after refresh', async () => {
  i18n.global.locale.value = 'en'
  const load = vi.spyOn(api, 'fetchModerationAppeals').mockResolvedValue({ items: [appeal()], total: 1, page: 1, pageSize: 20 })
  const action = vi.spyOn(api, 'moderationAppealAction').mockRejectedValue(new Error('Already decided'))
  const w = mountQueue(); await flushPromises()
  let reject = w.findAll('button').find(b => b.text() === 'Reject')!
  expect(reject.attributes('disabled')).toBeDefined()
  await w.find('textarea').setValue('Rules still apply')
  reject = w.findAll('button').find(b => b.text() === 'Reject')!
  await reject.trigger('click'); await flushPromises()
  expect(action).toHaveBeenCalledWith(1, false, 'Rules still apply')
  expect(load).toHaveBeenCalledTimes(2)
  expect((w.find('textarea').element as HTMLTextAreaElement).value).toBe('Rules still apply')
  expect(w.text()).toContain('Already decided')
})
test('switching status discards late queue responses', async () => {
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
