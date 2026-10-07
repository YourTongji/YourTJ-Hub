// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import {
  DOMWrapper,
  flushPromises,
  mount,
  type VueWrapper,
} from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import AnonymousManagementPage from '../src/admin/pages/management/AnonymousManagementPage.vue'
import {
  listAnonymousIdentities,
  governAnonymousIdentity,
} from '../src/admin/runtime/api'
import zh from '../src/locales/zh'
vi.mock('../src/admin/runtime/api', () => ({
  listAnonymousIdentities: vi.fn(),
  governAnonymousIdentity: vi.fn(),
}))
const row = {
  kind: 'persona' as const,
  publicUid: 'a'.repeat(32),
  name: '躲进云里的猫',
  avatarUrl: '/avatar.svg',
  profileUrl: '/a/test',
  selectedAt: '2026-10-07T08:00:00Z',
  disabled: false,
  governanceDisabled: false,
  owner: {
    userId: 123,
    username: 'private-owner',
    closed: false,
    frozen: false,
  },
}
let wrapper: VueWrapper
afterEach(() => {
  wrapper?.unmount()
  document.body.innerHTML = ''
  vi.resetAllMocks()
})
function start() {
  wrapper = mount(AnonymousManagementPage, {
    attachTo: document.body,
    global: {
      plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh } })],
    },
  })
  return wrapper
}
async function reveal() {
  await wrapper.get('#anonymous-view-reason').setValue('处理举报')
  await wrapper.get('form').trigger('submit')
  await flushPromises()
}
test('private mappings require a reason; denied refresh clears previously revealed rows', async () => {
  vi.mocked(listAnonymousIdentities).mockResolvedValue({
    items: [row],
    total: 1,
    page: 1,
    pageSize: 10,
  })
  start()
  expect(listAnonymousIdentities).not.toHaveBeenCalled()
  await reveal()
  expect(listAnonymousIdentities).toHaveBeenCalledWith({
    page: 1,
    pageSize: 10,
    status: 'all',
    search: '',
    reason: '处理举报',
  })
  expect(wrapper.text()).toContain('private-owner')
  expect(wrapper.find('a[href="/u/123"]').exists()).toBe(true)
  vi.mocked(listAnonymousIdentities).mockRejectedValue(
    new Error('permission revoked'),
  )
  await wrapper
    .findAll('button')
    .find((b) => b.text() === '刷新')!
    .trigger('click')
  await flushPromises()
  expect(wrapper.text()).not.toContain('private-owner')
  expect(wrapper.text()).toContain('permission revoked')
})
test('hiding mappings discards an in-flight response', async () => {
  let resolve!: (value: any) => void
  vi.mocked(listAnonymousIdentities).mockImplementation(
    () =>
      new Promise((r) => {
        resolve = r
      }),
  )
  start()
  await reveal()
  await wrapper
    .findAll('button')
    .find((b) => b.text() === '隐藏身份关系')!
    .trigger('click')
  resolve({ items: [row], total: 1, page: 1, pageSize: 10 })
  await flushPromises()
  expect(wrapper.text()).not.toContain('private-owner')
})

test('session expiry removes revealed rows and private action dialogs', async () => {
  vi.mocked(listAnonymousIdentities).mockResolvedValue({
    items: [row],
    total: 1,
    page: 1,
    pageSize: 10,
  })
  start()
  await reveal()
  await wrapper
    .findAll('button')
    .find((b) => b.text() === '封禁')!
    .trigger('click')
  await flushPromises()
  window.dispatchEvent(new Event('goose:session-cleared'))
  await flushPromises()
  expect(wrapper.text()).not.toContain('private-owner')
  expect(new DOMWrapper(document.body).find('[role="dialog"]').exists()).toBe(
    false,
  )
})
test('governance uses the UID and action reason and reloads the audited list', async () => {
  vi.mocked(listAnonymousIdentities).mockResolvedValue({
    items: [row],
    total: 1,
    page: 1,
    pageSize: 10,
  })
  vi.mocked(governAnonymousIdentity).mockResolvedValue(true)
  start()
  await reveal()
  await wrapper
    .findAll('button')
    .find((b) => b.text() === '封禁')!
    .trigger('click')
  await flushPromises()
  const body = new DOMWrapper(document.body)
  await body.get('[role="dialog"] form').trigger('submit')
  expect(governAnonymousIdentity).not.toHaveBeenCalled()
  await body.get('#anonymous-action-reason').setValue('持续违规')
  await body.get('[role="dialog"] form').trigger('submit')
  await flushPromises()
  expect(governAnonymousIdentity).toHaveBeenCalledWith({
    publicUid: row.publicUid,
    disabled: true,
    reason: '持续违规',
  })
  expect(listAnonymousIdentities).toHaveBeenCalledTimes(2)
})
