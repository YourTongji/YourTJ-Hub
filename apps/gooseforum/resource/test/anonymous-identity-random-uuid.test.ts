// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AnonymousIdentityDialog from '../src/site/components/AnonymousIdentityDialog.vue'
import {
  confirmName,
  disableIdentity,
  generateNames,
  getIdentityState,
  setProfileContent,
  type IdentityState,
} from '../src/runtime/anonymous-identity'
import { i18n, setLocale } from '../src/runtime/i18n'

vi.mock('../src/runtime/anonymous-identity', () => ({
  confirmName: vi.fn(),
  disableIdentity: vi.fn(),
  generateNames: vi.fn(),
  getIdentityState: vi.fn(),
  setProfileContent: vi.fn(),
}))

let wrapper: ReturnType<typeof mount> | undefined

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
  vi.resetAllMocks()
})

test('draw sends a UUID v4 and reuses it after an ambiguous failure in an insecure context', async () => {
  await setLocale('zh')
  const originalCrypto = globalThis.crypto
  vi.stubGlobal('crypto', {
    getRandomValues: originalCrypto.getRandomValues.bind(originalCrypto),
  })

  const batch = {
    id: 'batch-retry',
    day: '2026-10-07',
    words: ['星辰'],
    expiresAt: '2026-10-07T16:00:00Z',
    createdAt: '2026-10-07T00:00:00Z',
  }
  const state: IdentityState = {
    persona: null,
    nameSelectedAt: null,
    nameChangeAvailableAt: null,
    disabled: false,
    governanceDisabled: false,
    showContent: true,
    day: batch.day,
    remaining: 10,
    resetsAt: batch.expiresAt,
    batches: [],
    lexiconVersion: 'test',
  }
  vi.mocked(getIdentityState).mockImplementation(async () => structuredClone(state))
  vi.mocked(generateNames)
    .mockRejectedValueOnce(new Error('lost response'))
    .mockImplementationOnce(async () => {
      state.batches = [batch]
      state.remaining = 9
      return batch
    })

  wrapper = mount(AnonymousIdentityDialog, {
    props: { open: true },
    attachTo: document.body,
    global: { plugins: [i18n] },
  })
  await flushPromises()

  const drawButton = () => [...(document.querySelector('[role="dialog"]')?.querySelectorAll('button') ?? [])]
    .find((button) => button.textContent?.includes('花名'))
  expect(drawButton(), 'draw button should render').toBeDefined()
  drawButton()!.click()
  await flushPromises()
  expect(document.querySelector('[role="alert"]')?.textContent).toContain('lost response')

  drawButton()!.click()
  await flushPromises()

  const keys = vi.mocked(generateNames).mock.calls.map((call) => call[1])
  expect(keys).toHaveLength(2)
  expect(keys[0]).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
  expect(keys[1]).toBe(keys[0])
})
