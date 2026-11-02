// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { i18n, setLocale } from '../src/runtime/i18n'
import { useUnreadStatus } from '../src/runtime/unread-status'
import UnreadIndicator from '../src/site/components/UnreadIndicator.vue'
import UnreadDisplaySettings from '../src/site/components/UnreadDisplaySettings.vue'

const unread = useUnreadStatus()
let wrapper: ReturnType<typeof mount> | undefined
beforeEach(async () => {
  await setLocale('en')
  localStorage.clear()
  unread.stopPolling(true)
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  unread.stopPolling(true)
  vi.unstubAllGlobals()
})

test('numeric indicator caps visually but keeps its exact accessible count', async () => {
  wrapper = mount(UnreadIndicator, { props: { unread: true, count: 150, badgeStyle: 'number' }, global: { plugins: [i18n] } })
  expect(wrapper.text()).toBe('99+')
  expect(wrapper.attributes('aria-label')).toBe('Unread: 150')
  await wrapper.setProps({ badgeStyle: 'dot' })
  expect(wrapper.text()).toBe('')
  expect(wrapper.attributes('aria-label')).toBe('Unread')
  expect(wrapper.html()).not.toMatch(/150|99/)
  await wrapper.setProps({ unread: false })
  expect(wrapper.find('[role="img"]').exists()).toBe(false)
})

test('an unknown count keeps the boolean dot without inventing a one', () => {
  wrapper = mount(UnreadIndicator, { props: { unread: true, badgeStyle: 'number' }, global: { plugins: [i18n] } })
  expect(wrapper.text()).toBe('')
  expect(wrapper.attributes('aria-label')).toBe('Unread')
})

test('the labelled switch disables during saving, then rolls back or commits without count requests', async () => {
  unread.startPolling({ notifications: true, messages: false, notificationCount: 7, badgeStyle: 'number' }, 1024)
  let finish!: (value: unknown) => void
  const fetch = vi.fn(() => new Promise(resolve => { finish = resolve }))
  vi.stubGlobal('fetch', fetch)
  wrapper = mount(UnreadDisplaySettings, { global: { plugins: [i18n] } })
  const control = wrapper.get('[role="switch"]')
  expect(control.attributes('aria-label')).toBe('Show unread counts')
  expect(control.attributes('aria-describedby')).toBe('unread-style-hint')
  await control.trigger('click')
  expect(control.attributes('disabled')).toBeDefined()
  expect(wrapper.find('[role="status"]').exists()).toBe(true)
  finish({ ok: false, json: async () => ({ code: 1, messageCode: 'common.operation.failed' }) })
  await flushPromises()
  expect(control.attributes('aria-checked')).toBe('true')
  expect(wrapper.find('[role="alert"]').exists()).toBe(true)
  expect(unread.notificationCount.value).toBe(7)
  await control.trigger('click')
  finish({ ok: true, json: async () => ({ code: 0, result: { badgeStyle: 'dot' } }) })
  await flushPromises()
  expect(control.attributes('aria-checked')).toBe('false')
  expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  expect(fetch).toHaveBeenCalledTimes(2)
  for (const [path, init] of fetch.mock.calls as unknown as [string, RequestInit][]) {
    expect(path).toBe('/api/forum/notification-preferences')
    expect(init.method).toBe('PUT')
    expect(JSON.parse(init.body as string)).toEqual({ badgeStyle: 'dot' })
  }
})
