// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import AgentWebhookManagementDialog from '../src/admin/pages/management/AgentWebhookManagementDialog.vue'
import { i18n, setLocale } from '../src/runtime/i18n'
import {
  getAgentInteractionIntents,
  getAgentList,
  getAgentWebhookDeliveries,
  redeliverAgentWebhookDelivery,
  replayAgentInteractionIntent,
  rotateAgentWebhookSecret,
  saveAgentWebhookConfig,
  testAgentWebhook,
} from '../src/admin/runtime/api'
import type { AdminAgent, AdminAgentInteractionIntent, AdminAgentWebhookDelivery } from '../src/admin/types'

vi.mock('../src/admin/runtime/api', () => ({
  getAgentInteractionIntents: vi.fn(),
  getAgentList: vi.fn(),
  getAgentWebhookDeliveries: vi.fn(),
  redeliverAgentWebhookDelivery: vi.fn(),
  replayAgentInteractionIntent: vi.fn(),
  rotateAgentWebhookSecret: vi.fn(),
  saveAgentWebhookConfig: vi.fn(),
  testAgentWebhook: vi.fn(),
}))

const agent: AdminAgent = {
  agentId: 8,
  username: 'helper-bot',
  nickname: 'Helper',
  avatarUrl: '/bot.png',
  email: '',
  tokenPrefix: 'agt_test',
  webhookEndpoint: 'https://example.test/hook',
  configVersion: 5,
  eventsEnabled: true,
  eventTypes: ['agent.mentioned'],
  webhookEnabled: true,
  endpointGeneration: 3,
  subscriptionGeneration: 2,
  secretConfigured: true,
  secretVersion: 1,
  latestAcceptedAt: '2026-10-04T10:00:00Z',
  pendingCount: 2,
  pauseReason: '',
  summaryUnavailable: false,
  enabled: 1,
  createdBy: 1,
  lastUsedAt: null,
  createdAt: 0,
  updatedAt: 0,
}

const delivery: AdminAgentWebhookDelivery = {
  id: 14,
  instanceId: 'instance-a',
  eventId: 'evt_14',
  agentId: 8,
  endpointGeneration: 3,
  schemaVersion: 1,
  status: 'accepted',
  reason: null,
  taskId: 55,
  round: 1,
  attemptCount: 1,
  totalAttempts: 1,
  deadline: null,
  expiresAt: '2026-10-11T10:00:00Z',
  nextRunAt: null,
  createdBy: null,
  lastRedeliveredBy: null,
  acceptedAt: '2026-10-04T10:00:00Z',
  createdAt: '2026-10-04T09:59:00Z',
  updatedAt: '2026-10-04T10:00:00Z',
  attempts: [{
    id: 'attempt-1',
    deliveryId: 14,
    round: 1,
    number: 1,
    httpStatus: 204,
    errorClass: null,
    durationMs: 28,
    authorizedAt: '2026-10-04T09:59:30Z',
    completedAt: '2026-10-04T10:00:00Z',
  }],
}

const deadIntent: AdminAgentInteractionIntent = {
  id: 'intent-9',
  agentId: 8,
  sourceOccurrenceId: 'occ-9',
  postId: 31,
  revision: 2,
  status: 'dead',
  createdAt: '2026-10-04T09:00:00Z',
  expiresAt: '2026-10-11T09:00:00Z',
  retryCount: 3,
  lastError: 'temporary database error',
  taskId: 88,
}

let wrapper: VueWrapper | undefined
let priorLocale = 'zh'
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  vi.resetAllMocks()
  i18n.global.locale.value = priorLocale
})

async function mountDialog({ initialDelivery = delivery, initialIntent = deadIntent }: {
  initialDelivery?: AdminAgentWebhookDelivery
  initialIntent?: AdminAgentInteractionIntent
} = {}) {
  priorLocale = i18n.global.locale.value
  await setLocale('en')
  vi.mocked(getAgentWebhookDeliveries).mockResolvedValue({ list: [initialDelivery], total: 1, page: 1, pageSize: 10 })
  vi.mocked(getAgentInteractionIntents).mockResolvedValue({ list: [initialIntent], total: 1, page: 1, pageSize: 10 })
  wrapper = mount(AgentWebhookManagementDialog, {
    props: { open: true, agent },
    global: { plugins: [i18n] },
    attachTo: document.body,
  })
  return { page: wrapper, body: new DOMWrapper(document.body) }
}

async function clickByText(body: DOMWrapper, text: string) {
  const button = body.findAll('button').find(item => item.text().includes(text))
  if (!button) throw new Error(`Button not found: ${text}`)
  await button.trigger('click')
}

describe('Agent Webhook management dialog', () => {
  test('ignores a configuration response after another Agent opens', async () => {
    let resolveSave!: (value: AdminAgent) => void
    vi.mocked(saveAgentWebhookConfig).mockImplementationOnce(() => new Promise(resolve => { resolveSave = resolve }))
    const { page, body } = await mountDialog()
    await flushPromises()
    await clickByText(body, 'Save configuration')
    await page.setProps({ open: false, agent: null })
    const otherAgent = { ...agent, agentId: 9, username: 'other-bot', webhookEndpoint: 'https://other.example/hook' }
    await page.setProps({ open: true, agent: otherAgent })
    await flushPromises()
    resolveSave({ ...agent, configVersion: 6 })
    await flushPromises()
    const endpoint = body.findAll('input').find(input => input.attributes('type') === 'url')!
    expect((endpoint.element as HTMLInputElement).value).toBe(otherAgent.webhookEndpoint)
    vi.mocked(saveAgentWebhookConfig).mockResolvedValueOnce({ ...otherAgent, configVersion: 6 })
    await clickByText(body, 'Save configuration')
    await flushPromises()
    expect(saveAgentWebhookConfig).toHaveBeenLastCalledWith(expect.objectContaining({ agentId: 9 }))
  })

  test('does not reveal an old Agent signing secret in a new dialog session', async () => {
    let resolveRotation!: (value: { secret: string, secretVersion: number, configVersion: number }) => void
    vi.mocked(rotateAgentWebhookSecret).mockImplementationOnce(() => new Promise(resolve => { resolveRotation = resolve }))
    const { page, body } = await mountDialog()
    await flushPromises()
    await clickByText(body, 'Rotate signing secret')
    await flushPromises()
    const rotateButtons = body.findAll('button').filter(item => item.text().includes('Rotate signing secret'))
    await rotateButtons[1].trigger('click')
    await page.setProps({ open: false, agent: null })
    await page.setProps({ open: true, agent: { ...agent, agentId: 9, username: 'other-bot' } })
    await flushPromises()
    resolveRotation({ secret: 'whsec_previous_agent', secretVersion: 2, configVersion: 6 })
    await flushPromises()
    expect(body.text()).not.toContain('whsec_previous_agent')
  })

  test('shows accepted delivery separately from Agent acknowledgement and test replies', async () => {
    const { body } = await mountDialog()
    await flushPromises()

    expect(body.text()).toContain('Receiver accepted')
    expect(body.text()).toContain('HTTP 2xx means the receiver accepted the event')
    expect(body.text()).toContain('does not ask the Agent to generate or publish a reply')
    expect(body.text()).toContain('HTTP status 204')
    expect(body.text()).toContain('28 ms')
    expect(body.text()).toContain('Endpoint generation 3')
    expect(body.text()).toContain('Pending deliveries')
  })

  test('saves subscriptions with the current configVersion and preserves page intent', async () => {
    const updated = { ...agent, configVersion: 6, webhookEndpoint: 'https://new.example/hook' }
    vi.mocked(saveAgentWebhookConfig).mockResolvedValue(updated)
    const { body } = await mountDialog()
    await flushPromises()

    const inputs = body.findAll('input')
    const endpoint = inputs.find(input => input.attributes('type') === 'url')
    await endpoint!.setValue('https://new.example/hook')
    await clickByText(body, 'Save configuration')
    await flushPromises()

    expect(saveAgentWebhookConfig).toHaveBeenCalledWith({
      agentId: 8,
      configVersion: 5,
      eventsEnabled: true,
      eventTypes: ['agent.mentioned'],
      webhookEnabled: true,
      webhookEndpoint: 'https://new.example/hook',
    })
  })

  test('keeps the draft on a config conflict and reloads the latest version before retrying', async () => {
    const latest = { ...agent, configVersion: 6, webhookEndpoint: 'https://latest.example/hook' }
    const updated = { ...latest, configVersion: 7 }
    vi.mocked(saveAgentWebhookConfig)
      .mockRejectedValueOnce(Object.assign(new Error('stale version'), { messageCode: 'agent.webhook.configConflict' }))
      .mockResolvedValueOnce(updated)
    vi.mocked(getAgentList).mockResolvedValue([latest])
    const { body } = await mountDialog()
    await flushPromises()

    const endpoint = body.findAll('input').find(input => input.attributes('type') === 'url')!
    await endpoint.setValue('https://draft.example/hook')
    await clickByText(body, 'Save configuration')
    await flushPromises()

    expect(body.get('[role="alert"]').text()).toContain('configuration changed elsewhere')
    expect((endpoint.element as HTMLInputElement).value).toBe('https://draft.example/hook')

    await clickByText(body, 'Load latest settings')
    await flushPromises()
    expect(getAgentList).toHaveBeenCalledOnce()
    expect((endpoint.element as HTMLInputElement).value).toBe('https://latest.example/hook')

    await clickByText(body, 'Save configuration')
    await flushPromises()
    expect(saveAgentWebhookConfig).toHaveBeenLastCalledWith({
      agentId: 8,
      configVersion: 6,
      eventsEnabled: true,
      eventTypes: ['agent.mentioned'],
      webhookEnabled: true,
      webhookEndpoint: 'https://latest.example/hook',
    })
  })

  test('keeps interaction intents separate and replays a failed intent explicitly', async () => {
    const { body } = await mountDialog()
    await flushPromises()
    await clickByText(body, 'Interaction intents')
    await flushPromises()

    expect(body.text()).toContain('Source occurrence occ-9')
    expect(body.text()).toContain('Latest error: temporary database error')
    vi.mocked(replayAgentInteractionIntent).mockResolvedValue(undefined)
    await clickByText(body, 'Replay intent')
    await flushPromises()
    expect(replayAgentInteractionIntent).toHaveBeenCalledWith(8, 'intent-9')
    expect(getAgentInteractionIntents).toHaveBeenLastCalledWith(8, 1, 10)
    expect(redeliverAgentWebhookDelivery).not.toHaveBeenCalled()
  })

  test('sends a diagnostic webhook test and explains receiver acceptance', async () => {
    vi.mocked(testAgentWebhook).mockResolvedValue(delivery)
    const { body } = await mountDialog()
    await flushPromises()

    await clickByText(body, 'Send test')
    await flushPromises()

    expect(testAgentWebhook).toHaveBeenCalledWith(8)
    expect(body.get('[role="status"]').text()).toContain('Receiver accepted the test event')
    expect(body.text()).toContain('does not ask the Agent to generate or publish a reply')
  })

  test('redelivers a dead webhook delivery through the delivery API', async () => {
    const deadDelivery = { ...delivery, status: 'dead' }
    vi.mocked(redeliverAgentWebhookDelivery).mockResolvedValue({ ...deadDelivery, round: 2 })
    const { body } = await mountDialog({ initialDelivery: deadDelivery })
    await flushPromises()

    await clickByText(body, 'Redeliver')
    await flushPromises()

    expect(redeliverAgentWebhookDelivery).toHaveBeenCalledWith(8, 14)
    expect(getAgentWebhookDeliveries).toHaveBeenCalledTimes(2)
  })

  test('shows a rotated signing secret once and only through the one-time modal', async () => {
    vi.mocked(rotateAgentWebhookSecret).mockResolvedValue({ secret: 'whsec_once', secretVersion: 2, configVersion: 6 })
    vi.mocked(getAgentList).mockResolvedValue([{ ...agent, configVersion: 6, secretVersion: 2 }])
    const { body } = await mountDialog()
    await flushPromises()
    await clickByText(body, 'Rotate signing secret')
    await flushPromises()
    const rotateButtons = body.findAll('button').filter(item => item.text().includes('Rotate signing secret'))
    await rotateButtons[1].trigger('click')
    await flushPromises()

    expect(rotateAgentWebhookSecret).toHaveBeenCalledWith({ agentId: 8, configVersion: 5, emergency: false })
    expect(body.text()).toContain('whsec_once')
    expect(body.text()).toContain('shown once')
    const closeButtons = body.findAll('button').filter(item => item.text().includes('Close'))
    await closeButtons.at(-1)!.trigger('click')
    await flushPromises()
    expect(body.text()).not.toContain('whsec_once')
  })
})
