// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { UserCardPayload } from '@gooseforum/client'
import AgentProfileEditor from '../src/site/components/AgentProfileEditor.vue'
import { updateAgent } from '../src/admin/runtime/api'
import { i18n, setLocale } from '../src/runtime/i18n'
vi.mock('../src/admin/runtime/api', () => ({ updateAgent: vi.fn() }))
let wrapper: VueWrapper | undefined
const priorLocale = i18n.global.locale.value
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = ''; vi.resetAllMocks(); i18n.global.locale.value = priorLocale })
async function setup() {
  await setLocale('en')
  wrapper = mount(AgentProfileEditor, { props: { user: { userId: 9, username: 'bot', nickname: 'Bot', avatarUrl: '/static/pic/1.webp', profileCoverUrl: '/cover.webp', bio: 'Bio', signature: '', website: '', websiteName: '' } as UserCardPayload }, global: { plugins: [i18n] }, attachTo: document.body })
  await wrapper.find('button').trigger('click')
  await flushPromises()
  return new DOMWrapper(document.body)
}
test('saves only the target bot profile, and keeps a failed draft for retry', async () => {
  const body = await setup()
  await body.find('[name="nickname"]').setValue('New bot name')
  await body.find('[name="bio"]').setValue('Updated bio')
  vi.mocked(updateAgent).mockRejectedValueOnce(new Error('Permission denied'))
  await body.find('form').trigger('submit')
  await flushPromises()
  expect(updateAgent).toHaveBeenCalledWith(expect.objectContaining({ agentId: 9, nickname: 'New bot name', bio: 'Updated bio', avatarUrl: '/static/pic/1.webp', profileCoverUrl: '/cover.webp' }))
  expect(body.find('[role="alert"]').text()).toContain('Permission denied')
  expect((body.find('[name="nickname"]').element as HTMLInputElement).value).toBe('New bot name')
  expect(wrapper!.emitted('saved')).toBeUndefined()
  vi.mocked(updateAgent).mockResolvedValueOnce({ agentId: 9 } as never)
  await body.find('form').trigger('submit')
  await flushPromises()
  expect(wrapper!.emitted('saved')).toHaveLength(1)
})
test('preset selection and cover clearing update the bot profile', async () => {
  const body = await setup()
  await body.findAll('[aria-pressed]')[1].trigger('click')
  await body.findAll('button').find(item => item.text() === 'Clear')!.trigger('click')
  vi.mocked(updateAgent).mockResolvedValueOnce({ agentId: 9 } as never)
  await body.find('form').trigger('submit')
  await flushPromises()
  expect(updateAgent).toHaveBeenCalledWith(expect.objectContaining({ agentId: 9, avatarUrl: '/static/pic/2.webp', profileCoverUrl: '' }))
})
