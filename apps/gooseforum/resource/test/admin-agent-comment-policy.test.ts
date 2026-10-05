// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import AgentCommentPolicyPage from '../src/admin/pages/management/AgentCommentPolicyPage.vue'
import { i18n, setLocale } from '../src/runtime/i18n'
import { getAgentCommentPolicy, getTopicsList, saveAgentCommentPolicy, setAgentCommentTopicPolicy } from '../src/admin/runtime/api'
import type { AdminPayload, AdminTopic } from '../src/admin/types'

vi.mock('../src/admin/runtime/api', () => ({
  getAgentCommentPolicy: vi.fn(),
  getTopicsList: vi.fn(),
  saveAgentCommentPolicy: vi.fn(),
  setAgentCommentTopicPolicy: vi.fn(),
}))

const payload = {
  component: 'manage-home',
  props: {},
  meta: { title: 'admin' },
  layout: {},
  url: '/admin/agent-comment-policy',
  version: 'test',
} as unknown as AdminPayload

const topic = (overrides: Partial<AdminTopic> = {}): AdminTopic => ({
  id: 1201,
  title: '期中复习资料汇总',
  description: '',
  categoryId: [3],
  userId: 1024,
  username: 'tongji_user',
  nickname: '',
  userAvatarUrl: '',
  topicStatus: 1,
  processStatus: 0,
  viewCount: 1,
  replyCount: 2,
  likeCount: 3,
  pinWeight: 0,
  agentCommentDisabled: false,
  createdAt: '2026-08-10T08:00:00Z',
  updatedAt: '2026-08-15T09:20:00Z',
  ...overrides,
})

let wrapper: VueWrapper | undefined
let priorLocale = 'zh'
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  vi.resetAllMocks()
  i18n.global.locale.value = priorLocale
})

async function mountPage() {
  priorLocale = i18n.global.locale.value
  await setLocale('en')
  vi.mocked(getAgentCommentPolicy).mockResolvedValue({ allowAgentComments: true })
  vi.mocked(getTopicsList).mockResolvedValue({ list: [topic()], page: 1, size: 10, total: 0, hasNext: false })
  vi.mocked(saveAgentCommentPolicy).mockResolvedValue(undefined)
  vi.mocked(setAgentCommentTopicPolicy).mockResolvedValue({ topicId: 1201, agentCommentDisabled: true })
  wrapper = mount(AgentCommentPolicyPage, { props: { payload }, global: { plugins: [i18n] }, attachTo: document.body })
  return { page: wrapper, body: new DOMWrapper(document.body) }
}

async function clickByText(body: DOMWrapper, text: string) {
  const button = body.findAll('button').find(item => item.text().includes(text))
  if (!button) throw new Error(`Button not found: ${text}`)
  await button.trigger('click')
}

describe('Agent comment policy page', () => {
  test('renders the site-wide switch and topic rows after loading', async () => {
    const { page, body } = await mountPage()
    await flushPromises()

    expect(body.text()).toContain('Agent comment policy')
    expect(body.text()).toContain('Allow Agent comments')
    const rows = page.findAll('tbody tr')
    expect(rows).toHaveLength(1)
    expect(rows[0].text()).toContain('期中复习资料汇总')
    expect(rows[0].text()).toContain('Allowed')
    expect(body.findAll('[role="switch"]')).toHaveLength(2)
  })

  test('saves the site-wide switch with the new value', async () => {
    const { body } = await mountPage()
    await flushPromises()

    await body.findAll('[role="switch"]')[0].trigger('click')
    await flushPromises()
    await clickByText(body, 'Save')
    await flushPromises()

    expect(saveAgentCommentPolicy).toHaveBeenCalledWith(false)
  })

  test('bans Agent comments on a topic and reflects the new state', async () => {
    const { page, body } = await mountPage()
    await flushPromises()

    await body.findAll('[role="switch"]')[1].trigger('click')
    await flushPromises()

    expect(setAgentCommentTopicPolicy).toHaveBeenCalledWith(1201, true)
    expect(page.findAll('tbody tr')[0].text()).toContain('Banned')
  })

  test('filters the topic list to banned topics', async () => {
    const { body } = await mountPage()
    await flushPromises()

    await clickByText(body, 'All topics')
    await flushPromises()

    expect(getTopicsList).toHaveBeenLastCalledWith(expect.objectContaining({ agentCommentDisabled: true }))
  })
})
