// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AgentReplySetting from '../src/site/components/AgentReplySetting.vue'
import { i18n } from '../src/runtime/i18n'
import { updateTopicAgentReplies } from '../src/runtime/api'
vi.mock('../src/runtime/api', () => ({ updateTopicAgentReplies: vi.fn() }))
afterEach(() => vi.resetAllMocks())
const render = (canManage = true, disabled = false) => mount(AgentReplySetting, {
  props: { topicId: 42, canManage, disabled }, global: { plugins: [i18n] },
})
describe('topic robot reply control', () => {
  test('readers see the active restriction without an author control', () => {
    const allowed = render(false)
    expect(allowed.find('input').exists()).toBe(false)
    expect(allowed.text()).toBe('')
    allowed.unmount()
    const restricted = render(false, true)
    expect(restricted.find('input').exists()).toBe(false)
    expect(restricted.text()).toContain(i18n.global.t('agentReplies.disabled'))
    restricted.unmount()
  })
  test('only confirms a change after the server succeeds; failure retains the checkbox', async () => {
    vi.mocked(updateTopicAgentReplies).mockRejectedValueOnce(new Error('save failed'))
    const wrapper = render()
    await wrapper.get('input').setValue(true)
    await flushPromises()
    expect(updateTopicAgentReplies).toHaveBeenCalledWith(42, true)
    expect(wrapper.emitted('changed')).toBeUndefined()
    expect((wrapper.get('input').element as HTMLInputElement).checked).toBe(false)
    expect(wrapper.get('[role=alert]').text()).toBe('save failed')
    vi.mocked(updateTopicAgentReplies).mockResolvedValueOnce(true)
    await wrapper.get('input').setValue(true)
    await flushPromises()
    expect(wrapper.emitted('changed')).toEqual([[true]])
    await wrapper.setProps({ disabled: true })
    expect((wrapper.get('input').element as HTMLInputElement).checked).toBe(true)
    wrapper.unmount()
  })
  test('ignores a completed save after navigation to a different topic', async () => {
    let finish!: (value: boolean) => void
    vi.mocked(updateTopicAgentReplies).mockReturnValueOnce(new Promise(resolve => { finish = resolve }))
    const wrapper = render()
    await wrapper.get('input').setValue(true)
    expect(wrapper.get('input').attributes('disabled')).toBeDefined()
    await wrapper.setProps({ topicId: 99 })
    finish(true)
    await flushPromises()
    expect(wrapper.emitted('changed')).toBeUndefined()
    wrapper.unmount()
  })
})
