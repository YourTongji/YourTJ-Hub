// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import AgentContentCorrectionDialog from '../src/admin/pages/management/AgentContentCorrectionDialog.vue'
import { editBotContent, getBotContentSource } from '../src/admin/runtime/api'
import { i18n, setLocale } from '../src/runtime/i18n'
import type { AdminAgent } from '../src/admin/types'

vi.mock('../src/admin/runtime/api', () => ({ editBotContent: vi.fn(), getBotContentSource: vi.fn() }))
let wrapper: VueWrapper | undefined
const priorLocale = i18n.global.locale.value
afterEach(() => {
  wrapper?.unmount()
  document.body.innerHTML = ''
  vi.resetAllMocks()
  i18n.global.locale.value = priorLocale
})
async function setup() {
  await setLocale('en')
  vi.mocked(getBotContentSource).mockResolvedValue({ topicId: 10, postId: 11, title: 'Original', content: 'Original body', categoryId: [1], revisionNo: 3, botUserId: 9, botName: 'Bot' })
  wrapper = mount(AgentContentCorrectionDialog, { props: { agent: { agentId: 9, nickname: 'Bot' } as AdminAgent, categories: [{ id: 2, label: 'Campus' }, { id: 3, label: 'Study' }, { id: 4, label: 'Life' }] }, global: { plugins: [i18n] }, attachTo: document.body })
  await flushPromises()
  return new DOMWrapper(document.body)
}
async function click(body: DOMWrapper, label: string) {
  const button = body.findAll('button').find(item => item.text() === label)
  if (!button) throw new Error(`Missing button ${label}`)
  await button.trigger('click')
  await flushPromises()
}
async function load(body: DOMWrapper) {
  await body.find('input').setValue('10')
  await body.find('form').trigger('submit')
  await flushPromises()
}
test('saves loaded topic revision and retains draft on conflict', async () => {
  const body = await setup()
  await load(body)
  await body.find('textarea').setValue('Corrected body')
  vi.mocked(editBotContent).mockRejectedValue(new Error('Version conflict; reload source'))
  await body.findAll('form')[1].trigger('submit')
  await flushPromises()
  expect(editBotContent).toHaveBeenCalledWith('topic', { topicId: 10, title: 'Original', content: 'Corrected body', categoryId: [1], expectedRevisionNo: 3 })
  expect((body.find('textarea').element as HTMLTextAreaElement).value).toBe('Corrected body')
  expect(body.find('[role="alert"]').text()).toContain('Version conflict')
})
test('rejects another bot source and never enables editing it', async () => {
  const body = await setup()
  vi.mocked(getBotContentSource).mockResolvedValueOnce({ topicId: 10, postId: 11, title: '', content: 'Other', categoryId: [1], revisionNo: 1, botUserId: 99, botName: 'Other' })
  await load(body)
  expect(body.find('textarea').exists()).toBe(false)
  expect(body.find('[role="alert"]').text()).toContain('another bot')
})
test('uses reply endpoint and distinguishes moderation from saved content', async () => {
  const body = await setup()
  await click(body, 'Reply')
  await load(body)
  vi.mocked(editBotContent).mockResolvedValue({ code: 0, messageCode: 'content.moderation.checking' })
  await body.findAll('form')[1].trigger('submit')
  await flushPromises()
  expect(getBotContentSource).toHaveBeenCalledWith({ postId: 10 })
  expect(editBotContent).toHaveBeenCalledWith('post', { postId: 11, content: 'Original body', expectedRevisionNo: 3 })
  expect(body.find('[role="status"]').text()).not.toContain('Correction saved')
  expect(body.find('textarea').exists()).toBe(false)
})
test('changing lookup kind or ID cannot redirect a loaded draft save', async () => {
  const body = await setup()
  await load(body)
  await click(body, 'Reply')
  await body.find('input').setValue('99')
  vi.mocked(editBotContent).mockResolvedValue({ code: 0 })
  await body.findAll('form')[1].trigger('submit')
  await flushPromises()
  expect(editBotContent).toHaveBeenCalledWith('topic', expect.objectContaining({ topicId: 10, expectedRevisionNo: 3 }))
})

test('preserves unavailable categories and caps selection at three', async () => {
  const body = await setup()
  await load(body)
  expect(body.text()).toContain('Category #1')
  const checkbox = (label: string) => body.findAll('label').find(item => item.text().includes(label))!.find('[role="checkbox"]')
  expect(checkbox('Category #1').attributes('aria-checked')).toBe('true')
  await checkbox('Campus').trigger('click')
  await checkbox('Study').trigger('click')
  expect(checkbox('Life').attributes('disabled')).toBeDefined()
  await checkbox('Category #1').trigger('click')
  expect(body.text()).toContain('Category #1')
  await checkbox('Category #1').trigger('click')
  vi.mocked(editBotContent).mockResolvedValue({ code: 0 })
  await body.findAll('input')[1].setValue('')
  await body.findAll('form')[1].trigger('submit')
  await flushPromises()
  expect(editBotContent).toHaveBeenCalledWith('topic', expect.objectContaining({ title: '', categoryId: [2, 3, 1] }))
})
