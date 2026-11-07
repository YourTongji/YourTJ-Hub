// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { NotificationPayload, NotificationsPageProps } from '@gooseforum/client'

const api = vi.hoisted(() => ({ fetchNotifications: vi.fn(), markAllNotificationsRead: vi.fn(), markNotificationRead: vi.fn() }))
vi.mock('../src/runtime/api', () => api)
vi.mock('../src/runtime/browser-notification', () => ({ maybeNotifyUnread: vi.fn() }))
vi.mock('../src/runtime/document-title', () => ({ setUnreadMessagesDocumentTitle: vi.fn() }))

let status: ReturnType<typeof import('../src/runtime/unread-status').useUnreadStatus>
let wrapper: VueWrapper | undefined
const initial = { notifications: true, messages: true, notificationCount: 3, messageCount: 2 }
const item = { id: 1, eventType: 'comment', isRead: false, createdAt: '', title: '', content: '', actor: { id: 1, username: 'Alice' }, topic: { id: 2, title: 'Campus', url: '/p/2' }, payload: {} } as NotificationPayload
const payload = (count: number) => ({ items: [item], nextCursor: 0, hasNext: false, unreadCount: count })
const response = (count: number) => ({ ok: true, json: async () => ({ code: 0, result: { ...initial, notificationCount: count } }) })
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}
async function mountPage(count = 3) {
  const { i18n, setLocale } = await import('../src/runtime/i18n')
  await setLocale('en')
  const { default: Page } = await import('../src/site/pages/NotificationsPage.vue')
  wrapper = mount(Page, { props: { layout: {} as any, props: { notifications: [item], total: 1, unreadCount: count, pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' } } as NotificationsPageProps }, global: { plugins: [i18n] } })
  return wrapper
}
async function click(text: string) {
  await wrapper!.findAll('button').find(button => button.text() === text)!.trigger('click')
}
function expectCount(count: number, owner = 1024) {
  expect(status.notificationCount.value).toBe(count)
  expect(JSON.parse(localStorage.getItem(`goose:unread-status:${owner}`)!).data.notificationCount).toBe(count)
}
beforeEach(async () => {
  vi.resetModules()
  vi.resetAllMocks()
  localStorage.clear()
  status = (await import('../src/runtime/unread-status')).useUnreadStatus()
  status.startPolling(initial, 1024)
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  status.stopPolling(true)
  vi.unstubAllGlobals()
})

test.each([false, true])('an older list cannot replace a newer poll (poll finishes first: %s)', async pollFirst => {
  const list = deferred<ReturnType<typeof payload>>()
  const poll = deferred<ReturnType<typeof response>>()
  api.fetchNotifications.mockReturnValueOnce(list.promise)
  vi.stubGlobal('fetch', vi.fn(() => poll.promise))
  await mountPage()
  await click('Mentions')
  const refresh = status.refresh(true)
  if (pollFirst) { poll.resolve(response(8)); await refresh }
  list.resolve(payload(3))
  await flushPromises()
  if (!pollFirst) { poll.resolve(response(8)); await refresh }
  expectCount(8)
})

test('a list finishing after mark-all cannot resurrect read notifications', async () => {
  const list = deferred<ReturnType<typeof payload>>()
  api.fetchNotifications.mockReturnValueOnce(list.promise)
  api.markAllNotificationsRead.mockResolvedValueOnce(true)
  await mountPage()
  await click('Mentions')
  await click('Mark all read')
  await flushPromises()
  expectCount(0)
  list.resolve(payload(3))
  await flushPromises()
  expectCount(0)
  expect(wrapper!.text()).not.toContain('3 unread')
})

test.each(['list', 'mark-all', 'mark-one-success', 'mark-one-failure'] as const)('an old account %s completion cannot alter the next account or its cache', async operation => {
  const pending = deferred<any>()
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(6)))
  await mountPage()
  if (operation === 'list') {
    api.fetchNotifications.mockReturnValueOnce(pending.promise)
    await click('Mentions')
  } else if (operation === 'mark-all') {
    api.markAllNotificationsRead.mockReturnValueOnce(pending.promise)
    await click('Mark all read')
  } else {
    api.markNotificationRead.mockReturnValueOnce(pending.promise)
    await wrapper!.get('article button').trigger('click')
  }
  status.startPolling({ ...initial, notificationCount: 6 }, 2048)
  if (operation === 'mark-one-failure') pending.reject(new Error('offline'))
  else pending.resolve(operation === 'list' ? payload(3) : true)
  await flushPromises()
  expectCount(6, 2048)
  expect(fetch).not.toHaveBeenCalled()
})

test('only confirmed single reads fence an in-flight poll', async () => {
  const mark = deferred<boolean>()
  const poll = deferred<ReturnType<typeof response>>()
  api.markNotificationRead.mockReturnValueOnce(mark.promise)
  vi.stubGlobal('fetch', vi.fn(() => poll.promise))
  await mountPage()
  const refresh = status.refresh(true)
  await wrapper!.get('article button').trigger('click')
  expectCount(3)
  mark.resolve(true)
  await flushPromises()
  expectCount(2)
  poll.resolve(response(3))
  await refresh
  expectCount(2)
})

test('a single-read completion reconciles a newer snapshot instead of restoring the page count', async () => {
  const mark = deferred<boolean>()
  api.markNotificationRead.mockReturnValueOnce(mark.promise)
  vi.stubGlobal('fetch', vi.fn().mockResolvedValueOnce(response(9)).mockResolvedValueOnce(response(8)))
  await mountPage()
  await wrapper!.get('article button').trigger('click')
  await status.refresh(true)
  expectCount(9)
  mark.resolve(true)
  await flushPromises()
  expectCount(8)
})

test('logging out and back into the same account invalidates an old list request', async () => {
  const list = deferred<ReturnType<typeof payload>>()
  api.fetchNotifications.mockReturnValueOnce(list.promise)
  await mountPage()
  await click('Mentions')
  status.stopPolling(true)
  status.startPolling({ ...initial, notificationCount: 7 }, 1024)
  list.resolve(payload(3))
  await flushPromises()
  expectCount(7)
})

test('a zero-count page payload is not a confirmed read that fences a poll', async () => {
  const poll = deferred<ReturnType<typeof response>>()
  vi.stubGlobal('fetch', vi.fn(() => poll.promise))
  const refresh = status.refresh(true)
  await mountPage(0)
  poll.resolve(response(8))
  await refresh
  expectCount(8)
})

test('a newer list snapshot wins over an older poll without dropping its message count', async () => {
  const poll = deferred<object>()
  vi.stubGlobal('fetch', vi.fn(() => poll.promise))
  api.fetchNotifications.mockResolvedValueOnce(payload(8))
  await mountPage()
  const refresh = status.refresh(true)
  await click('Mentions')
  await flushPromises()
  poll.resolve({ ok: true, json: async () => ({ code: 0, result: { ...initial, messageCount: 20 } }) })
  await refresh
  expectCount(8)
  expect(status.messageCount.value).toBe(20)
})

test('a failed single read rolls back its row without blocking a newer poll count', async () => {
  const mark = deferred<boolean>()
  const poll = deferred<ReturnType<typeof response>>()
  api.markNotificationRead.mockReturnValueOnce(mark.promise)
  vi.stubGlobal('fetch', vi.fn(() => poll.promise))
  await mountPage()
  const refresh = status.refresh(true)
  await wrapper!.get('article button').trigger('click')
  mark.reject(new Error('offline'))
  await flushPromises()
  expect(wrapper!.findAll('article button')).toHaveLength(1)
  poll.resolve(response(8))
  await refresh
  expectCount(8)
})
