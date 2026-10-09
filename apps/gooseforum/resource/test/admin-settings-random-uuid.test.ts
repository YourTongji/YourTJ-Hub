// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AdminSettingsPage from '../src/admin/pages/AdminSettingsPage.vue'
import { i18n, setLocale } from '../src/runtime/i18n'
import { getHttpNotifySettings, saveHttpNotifySettings, testHttpNotifyEndpoint } from '@/admin/runtime/api'

vi.mock('@/admin/runtime/api')
vi.mock('@/admin/runtime/toast', () => ({
  adminToast: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
  },
}))

let wrapper: ReturnType<typeof mount> | undefined

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
  vi.resetAllMocks()
})

beforeEach(async () => {
  await setLocale('zh')
  const originalCrypto = globalThis.crypto
  vi.stubGlobal('crypto', {
    getRandomValues: originalCrypto.getRandomValues.bind(originalCrypto),
  })
})

test('HTTP 通知端点在 randomUUID 不可用时可规范化、新增、测试和保存', async () => {
  const serverSettings = {
    enabled: true,
    endpoints: [{
      name: 'Legacy endpoint',
      channelType: 'generic',
      enabled: true,
      url: 'https://example.test/legacy',
      target: '',
      secret: '',
      events: ['topic.published'],
      timeoutSeconds: 2,
      failureCount: 0,
      lastError: '',
      abnormalTerminated: false,
    }],
  }
  vi.mocked(getHttpNotifySettings).mockImplementation(async () => structuredClone(serverSettings) as never)
  vi.mocked(saveHttpNotifySettings).mockImplementation(async settings => {
    serverSettings.endpoints = structuredClone(settings.endpoints) as typeof serverSettings.endpoints
  })
  vi.mocked(testHttpNotifyEndpoint).mockResolvedValue({ success: true, message: 'ok' } as never)

  wrapper = mount(AdminSettingsPage, {
    props: {
      kind: 'http-notify',
      payload: {
        layout: {
          sidebar: { categories: [] },
          site: { name: 'YourTJHub' },
          viewer: { username: 'admin', avatarUrl: '' },
        },
      } as never,
    },
    global: { plugins: [i18n] },
  })
  await flushPromises()

  const endpointButtons = () => [...(wrapper!.element.querySelectorAll<HTMLButtonElement>('button[aria-controls^="http-endpoint-"]'))]
  const uuidV4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/
  expect(endpointButtons()).toHaveLength(1)
  expect(endpointButtons()[0].getAttribute('aria-controls')?.slice('http-endpoint-'.length)).toMatch(uuidV4)

  const addButton = [...wrapper!.element.querySelectorAll<HTMLButtonElement>('button')]
    .find(button => button.textContent?.includes('添加地址'))
  expect(addButton, 'add endpoint action should render').toBeDefined()
  addButton!.click()
  await flushPromises()

  const cards = [...wrapper!.element.querySelectorAll<HTMLElement>('[id^="http-endpoint-"]')]
  const addedCard = cards.at(-1)!
  const addedId = addedCard.id.slice('http-endpoint-'.length)
  expect(addedId).toMatch(uuidV4)

  const urlInput = [...addedCard.querySelectorAll<HTMLInputElement>('input')]
    .find(input => input.placeholder === 'http://example.com/webhook')
  expect(urlInput, 'endpoint URL input should render').toBeDefined()
  urlInput!.value = 'https://example.test/new'
  urlInput!.dispatchEvent(new Event('input', { bubbles: true }))
  await flushPromises()

  const testButton = [...addedCard.querySelectorAll<HTMLButtonElement>('button')]
    .find(button => button.textContent?.includes('测试'))
  expect(testButton, 'test action should render').toBeDefined()
  testButton!.click()
  await flushPromises()
  expect(testHttpNotifyEndpoint).toHaveBeenCalledWith(expect.objectContaining({ id: addedId, url: 'https://example.test/new' }))

  const saveButton = [...addedCard.querySelectorAll<HTMLButtonElement>('button')]
    .find(button => button.textContent?.includes('保存'))
  expect(saveButton, 'save action should render').toBeDefined()
  saveButton!.click()
  await flushPromises()
  expect(saveHttpNotifySettings).toHaveBeenCalledWith(expect.objectContaining({
    endpoints: expect.arrayContaining([expect.objectContaining({ id: addedId, url: 'https://example.test/new' })]),
  }))
})
