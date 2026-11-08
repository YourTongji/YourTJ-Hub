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
async function mountPage(count = 3, items = [item], filter: 'all' | 'unread' = 'all') {
  const { i18n, setLocale } = await import('../src/runtime/i18n')
  await setLocale('en')
  const { default: Page } = await import('../src/site/pages/NotificationsPage.vue')
  wrapper = mount(Page, { props: { layout: { viewer: { id: 1024, isAuthenticated: true } } as any, props: { notifications: items, filter, total: items.length, unreadCount: count, pagination: { page: 1, nextPage: 0, hasNext: false, nextUrl: '' } } as NotificationsPageProps }, global: { plugins: [i18n] } })
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
  vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('No unread response configured')))
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

test('a poll response that lands after the read commits is not decremented again on confirm', async () => {
  const mark = deferred<boolean>()
  const poll = deferred<ReturnType<typeof response>>()
  api.markNotificationRead.mockReturnValueOnce(mark.promise)
  vi.stubGlobal('fetch', vi.fn().mockReturnValueOnce(poll.promise).mockResolvedValue(response(2)))
  await mountPage(3, [item])
  const refresh = status.refresh(true) // poll started before the read request
  await wrapper!.get('article button').trigger('click')
  poll.resolve(response(2)) // the server already applied the read when this response was produced
  await refresh
  expectCount(2)
  mark.resolve(true)
  await flushPromises()
  expectCount(2)
  expect(wrapper!.get('[data-unread-summary]').text()).toBe('2 unread')
})

test('a confirmed read after an intervening list snapshot reloads the overwritten rows', async () => {
  const mark = deferred<boolean>()
  const list = deferred<ReturnType<typeof payload>>()
  api.markNotificationRead.mockReturnValueOnce(mark.promise)
  api.fetchNotifications.mockReturnValueOnce(list.promise).mockResolvedValueOnce({ ...payload(2), items: [{ ...item, isRead: true }] })
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(2)))
  await mountPage(3, [item])
  await wrapper!.get('article button').trigger('click')
  await click('Mentions') // the list response still shows the pending row unread with count 3
  list.resolve(payload(3))
  await flushPromises()
  expect(wrapper!.get('[data-unread-summary]').text()).toBe('3 unread')
  mark.resolve(true)
  await flushPromises()
  expectCount(2)
  expect(wrapper!.get('[data-unread-summary]').text()).toBe('2 unread')
  expect(wrapper!.findAll('article button')).toHaveLength(0)
  expect(api.fetchNotifications).toHaveBeenCalledTimes(2)
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

test('list totals stay separate from poll badge and message counts', async () => {
  const poll = deferred<object>()
  vi.stubGlobal('fetch', vi.fn(() => poll.promise))
  api.fetchNotifications.mockResolvedValueOnce(payload(8))
  await mountPage()
  const refresh = status.refresh(true)
  await click('Mentions')
  await flushPromises()
  poll.resolve({ ok: true, json: async () => ({ code: 0, result: { ...initial, messageCount: 20 } }) })
  await refresh
  expectCount(3)
  expect(wrapper!.get('[data-unread-summary]').text()).toBe('8 unread')
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


test.each(['list', 'mark-all'] as const)('a mounted page resets pending %s state for the next account', async operation => {
  const old = deferred<any>()
  await mountPage()
  if (operation === 'list') {
    api.fetchNotifications.mockReturnValueOnce(old.promise)
    await click('Mentions')
  } else {
    api.markAllNotificationsRead.mockReturnValueOnce(old.promise)
    await click('Mark all read')
  }
  status.startPolling({ ...initial, notificationCount: 6 }, 2048)
  const next = { ...item, id: 20, topic: { ...item.topic!, title: 'Next account' } }
  await wrapper!.setProps({ layout: { viewer: { id: 2048, isAuthenticated: true } } as any, props: { ...wrapper!.props('props'), notifications: [next], unreadCount: 6 } })
  expect(wrapper!.text()).toContain('Next account')
  const button = wrapper!.findAll('button').find(button => button.text() === 'Mark all read')!
  expect(button).toBeDefined()
  expect(button.attributes('disabled')).toBeUndefined()
  old.resolve(operation === 'list' ? payload(3) : true)
  await flushPromises()
  expectCount(6, 2048)
  expect(wrapper!.text()).toContain('Next account')
})

test('a failed single read reloads its row after another single read succeeds', async () => {
  const failed = deferred<boolean>()
  const other = { ...item, id: 2 }
  api.markNotificationRead.mockReturnValueOnce(failed.promise).mockResolvedValueOnce(true)
  api.fetchNotifications.mockResolvedValueOnce({ ...payload(2), items: [item, { ...other, isRead: true }] })
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(2)))
  await mountPage(3, [item, other])
  await wrapper!.findAll('article button')[0]!.trigger('click')
  await wrapper!.findAll('article button')[0]!.trigger('click')
  await flushPromises()
  failed.reject(new Error('offline'))
  await flushPromises()
  expect(api.fetchNotifications).toHaveBeenCalledWith('all', 0, 20, 'all')
  expect(wrapper!.findAll('article')[0]!.findAll('button')).toHaveLength(1)
  expect(wrapper!.findAll('article')[1]!.findAll('button')).toHaveLength(0)
  expectCount(2)
})

test('a failed single read does not roll back another pending optimistic row', async () => {
  const failed = deferred<boolean>()
  const pending = deferred<boolean>()
  const reload = deferred<any>()
  api.markNotificationRead.mockReturnValueOnce(failed.promise).mockReturnValueOnce(pending.promise)
  api.fetchNotifications.mockReturnValueOnce(reload.promise)
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(3)))
  await mountPage(3, [item, { ...item, id: 2 }])
  await wrapper!.findAll('article button')[0]!.trigger('click')
  await wrapper!.findAll('article button')[0]!.trigger('click')
  failed.reject(new Error('offline'))
  await flushPromises()
  expect(wrapper!.findAll('article')[0]!.findAll('button')).toHaveLength(1)
  expect(wrapper!.findAll('article')[1]!.findAll('button')).toHaveLength(0)
  expect(wrapper!.get('[data-unread-summary]').text()).toBe('2 unread')
})

test('a list invalidated by a confirmed read automatically reloads its rows', async () => {
  const stale = deferred<ReturnType<typeof payload>>()
  api.fetchNotifications.mockReturnValueOnce(stale.promise).mockResolvedValueOnce({ ...payload(0), items: [{ ...item, isRead: true }] })
  api.markAllNotificationsRead.mockResolvedValueOnce(true)
  await mountPage()
  await click('Mentions')
  await click('Mark all read')
  await flushPromises()
  stale.resolve(payload(3))
  await flushPromises()
  expect(api.fetchNotifications).toHaveBeenCalledTimes(2)
  expect(wrapper!.findAll('article')).toHaveLength(1)
  expect(wrapper!.findAll('article button')).toHaveLength(0)
  expectCount(0)
})


test('an owner change without a new page payload clears stale rows before accepting the next payload', async () => {
  await mountPage()
  status.startPolling({ ...initial, notificationCount: 6 }, 2048)
  await wrapper!.setProps({ layout: { viewer: { id: 2048, isAuthenticated: true } } as any })
  expect(wrapper!.findAll('article')).toHaveLength(0)
  const next = { ...item, id: 20, topic: { ...item.topic!, title: 'Next account' } }
  await wrapper!.setProps({ props: { ...wrapper!.props('props'), notifications: [next], unreadCount: 6 } })
  expect(wrapper!.text()).toContain('Next account')
  expectCount(6, 2048)
})

test('a concurrent failed single read restores only its row in the unread filter', async () => {
  const failed = deferred<boolean>()
  const pending = deferred<boolean>()
  api.markNotificationRead.mockReturnValueOnce(failed.promise).mockReturnValueOnce(pending.promise)
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(3)))
  const other = { ...item, id: 2, topic: { ...item.topic!, title: 'Other pending' } }
  await mountPage(3, [item, other], 'unread')
  await wrapper!.findAll('article button')[0]!.trigger('click')
  await wrapper!.findAll('article button')[0]!.trigger('click')
  failed.reject(new Error('offline'))
  await flushPromises()
  expect(wrapper!.findAll('article')).toHaveLength(1)
  expect(wrapper!.text()).toContain('Campus')
  expect(wrapper!.text()).not.toContain('Other pending')
  expect(wrapper!.get('[data-unread-summary]').text()).toBe('2 unread')
})

test('a same-account layout refresh does not restore a pending optimistic row', async () => {
  const pending = deferred<boolean>()
  api.markNotificationRead.mockReturnValueOnce(pending.promise)
  await mountPage()
  await wrapper!.get('article button').trigger('click')
  await wrapper!.setProps({ layout: { viewer: { id: 1024, isAuthenticated: true } } as any })
  expect(wrapper!.findAll('article button')).toHaveLength(0)
  expect(wrapper!.get('[data-unread-summary]').text()).toBe('2 unread')
})

test.each([
  [0, 1, 2], [0, 2, 1], [1, 0, 2], [1, 2, 0], [2, 0, 1], [2, 1, 0],
])('concurrent failed reads restore unread row order (%s, %s, %s)', async (...failureOrder) => {
  const pending = [deferred<boolean>(), deferred<boolean>(), deferred<boolean>()]
  pending.forEach(operation => api.markNotificationRead.mockReturnValueOnce(operation.promise))
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(4)))
  const items = [40, 30, 20, 10].map(id => ({ ...item, id, topic: { ...item.topic!, title: `Notice ${id}` } }))
  await mountPage(4, items, 'unread')
  for (let index = 0; index < pending.length; index++) {
    await wrapper!.findAll('article button')[0]!.trigger('click')
  }
  const restored = [10]
  for (const index of failureOrder) {
    pending[index]!.reject(new Error('offline'))
    await flushPromises()
    restored.push(items[index]!.id)
    restored.sort((left, right) => right - left)
    const actual = wrapper!.findAll('article').map(row => Number(row.text().match(/Notice (\d+)/)![1]))
    expect(actual).toEqual(restored)
    expect(wrapper!.get('[data-unread-summary]').text()).toBe(`${restored.length} unread`)
  }
  expectCount(4)
  expect(api.fetchNotifications).not.toHaveBeenCalled()
})

test('unfiltered notification lists cannot overwrite a category-filtered badge count', async () => {
  status.stopPolling(true)
  status.startPolling({ ...initial, notificationCount: 2, badgeCategories: ['replies'] }, 1024)
  api.fetchNotifications.mockResolvedValueOnce(payload(8))
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ code: 0, result: { ...initial, notificationCount: 2, badgeCategories: ['replies'] } }) }))
  await mountPage(8)
  await click('Likes')
  await flushPromises()
  expect(wrapper!.get('[data-unread-summary]').text()).toBe('8 unread')
  expectCount(2)
})

test('a confirmed single read reconciles filtered badges instead of decrementing an excluded event', async () => {
  status.stopPolling(true)
  status.startPolling({ ...initial, notificationCount: 2, badgeCategories: ['replies'] }, 1024)
  api.markNotificationRead.mockResolvedValueOnce(true)
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: async () => ({ code: 0, result: { ...initial, notificationCount: 2, badgeCategories: ['replies'] } }) }))
  await mountPage(3, [{ ...item, eventType: 'like' }])
  await wrapper!.get('article button').trigger('click')
  await flushPromises()
  expectCount(2)
  expect(wrapper!.get('[data-unread-summary]').text()).toBe('2 unread')
})

test('a failed optimistic read restores only the decrement actually applied to a zero total', async () => {
  api.markNotificationRead.mockRejectedValueOnce(new Error('offline'))
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(0)))
  await mountPage(0)
  await wrapper!.get('article button').trigger('click')
  await flushPromises()
  expect(wrapper!.findAll('article button')).toHaveLength(1)
  expect(wrapper!.find('[data-unread-summary]').exists()).toBe(false)
  expectCount(0)
})
