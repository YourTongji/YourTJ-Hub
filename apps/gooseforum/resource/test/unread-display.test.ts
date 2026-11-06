// @vitest-environment happy-dom
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'

vi.mock('../src/runtime/browser-notification', () => ({ maybeNotifyUnread: vi.fn() }))
vi.mock('../src/runtime/document-title', () => ({ setUnreadMessagesDocumentTitle: vi.fn() }))

let status: ReturnType<typeof import('../src/runtime/unread-status').useUnreadStatus>
const initial = { notifications: true, messages: true, notificationCount: 3, messageCount: 12, badgeStyle: 'number' as const }
const response = (result: object) => ({ ok: true, json: async () => ({ code: 0, result }) })

beforeEach(async () => {
  vi.resetModules()
  vi.useFakeTimers()
  localStorage.clear()
  status = (await import('../src/runtime/unread-status')).useUnreadStatus()
})
afterEach(() => {
  status.stopPolling?.()
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

test('initial counts and account-wide dot style use the existing unread snapshot', () => {
  const fetch = vi.fn()
  vi.stubGlobal('fetch', fetch)
  status.startPolling({ ...initial, badgeStyle: 'dot' }, 1024)
  expect(status.badgeStyle.value).toBe('dot')
  expect(status.notificationCount.value).toBe(3)
  expect(status.messageCount.value).toBe(12)
  expect(fetch).not.toHaveBeenCalled()
})

test('a slow poll cannot revert a confirmed local style and another save is excluded', async () => {
  let finishPoll!: (value: unknown) => void
  let finishSave!: (value: unknown) => void
  const fetch = vi.fn((_path: string, init?: RequestInit) => new Promise(resolve => {
    if (init?.method === 'PUT') finishSave = resolve
    else finishPoll = resolve
  }))
  vi.stubGlobal('fetch', fetch)
  status.startPolling(initial, 1024)
  const poll = status.refresh(true)
  const save = status.updateBadgeStyle('dot')
  expect(status.badgeStyle.value).toBe('dot')
  expect(await status.updateBadgeStyle('number')).toBe(false)
  finishSave(response({ badgeStyle: 'dot' }))
  expect(await save).toBe(true)
  finishPoll(response(initial))
  await poll
  expect(status.badgeStyle.value).toBe('dot')
  expect(fetch).toHaveBeenCalledTimes(2)
})

test('save failure rolls back without changing unread facts and can retry', async () => {
  const fetch = vi.fn().mockRejectedValueOnce(new TypeError('offline')).mockResolvedValue(response({ badgeStyle: 'dot' }))
  vi.stubGlobal('fetch', fetch)
  status.startPolling(initial, 1024)
  expect(await status.updateBadgeStyle('dot')).toBe(false)
  expect(status.badgeStyle.value).toBe('number')
  expect(status.badgeStyleError.value).not.toBe('')
  expect(status.notificationCount.value).toBe(3)
  expect(status.messages.value).toBe(true)
  expect(await status.updateBadgeStyle('dot')).toBe(true)
  expect(status.badgeStyle.value).toBe('dot')
})

test('same-account shell navigation does not strand a pending preference save', async () => {
  let finish!: (value: unknown) => void
  vi.stubGlobal('fetch', vi.fn(() => new Promise(resolve => { finish = resolve })))
  status.startPolling(initial, 1024)
  const save = status.updateBadgeStyle('dot')
  status.stopPolling()
  status.startPolling(initial, 1024)
  finish(response({ badgeStyle: 'dot' }))
  expect(await save).toBe(true)
  expect(status.badgeStyleSaving.value).toBe(false)
  expect(status.badgeStyle.value).toBe('dot')
})

test('an old account response and cache cannot replace the next account snapshot', async () => {
  let finish!: (value: unknown) => void
  vi.stubGlobal('fetch', vi.fn(() => new Promise(resolve => { finish = resolve })))
  status.startPolling(initial, 1024)
  const old = status.refresh(true)
  status.startPolling({ notifications: false, messages: false, badgeStyle: 'dot' }, 2048)
  finish(response({ ...initial, notificationCount: 99 }))
  await old
  await flushPromises()
  expect(status.badgeStyle.value).toBe('dot')
  expect(status.notifications.value).toBe(false)
  expect(status.notificationCount.value).toBeUndefined()
  const cached = JSON.parse(localStorage.getItem('goose:unread-status:2048')!)
  expect(cached.data.notifications).toBe(false)
})


test('category filtering saves independently, refreshes counts, and keeps browser delivery', async () => {
  const fetch = vi.fn().mockResolvedValueOnce(response({ badgeStyle: 'dot', badgeCategories: ['replies'] }))
    .mockResolvedValue(response({ ...initial, notifications: false, notificationCount: 0, latestUnreadId: 22, badgeStyle: 'dot', badgeCategories: ['replies'] }))
  vi.stubGlobal('fetch', fetch)
  status.startPolling({ ...initial, badgeStyle: 'dot' }, 1024)
  expect(status.badgeCategories.value).toHaveLength(5)
  expect(await status.updateBadgeCategories(['replies'])).toBe(true)
  await flushPromises()
  expect(JSON.parse(fetch.mock.calls[0]![1]!.body)).toEqual({ badgeCategories: ['replies'] })
  expect(status.notificationCount.value).toBe(0)
  expect(status.badgeStyle.value).toBe('dot')
  const { maybeNotifyUnread } = await import('../src/runtime/browser-notification')
  expect(maybeNotifyUnread).toHaveBeenLastCalledWith(true, true, '', 22)
})

test('category save failure restores selection and account switching drops old saves', async () => {
  vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('offline')))
  status.startPolling(initial, 1024)
  expect(await status.updateBadgeCategories([])).toBe(false)
  expect(status.badgeCategories.value).toHaveLength(5)
  let finish!: (value: unknown) => void
  vi.stubGlobal('fetch', vi.fn(() => new Promise(resolve => { finish = resolve })))
  const save = status.updateBadgeCategories([])
  status.startPolling({ ...initial, badgeCategories: ['likes'] }, 2048)
  finish(response({badgeStyle: 'number', badgeCategories: []}))
  expect(await save).toBe(false)
  expect(status.badgeCategories.value).toEqual(['likes'])
})


test('category save fences a pre-save poll and starts a fresh filtered snapshot', async () => {
  let finishOld!: (value: unknown) => void
  const fetch = vi.fn().mockImplementationOnce(() => new Promise(resolve => { finishOld = resolve }))
    .mockResolvedValueOnce(response({ badgeStyle: 'number', badgeCategories: ['replies'] }))
    .mockResolvedValue(response({ ...initial, notifications: false, notificationCount: 0, latestNotificationType: 'comment', latestUnreadId: 22, badgeCategories: ['replies'] }))
  vi.stubGlobal('fetch', fetch)
  status.startPolling(initial, 1024)
  const oldPoll = status.refresh(true)
  expect(await status.updateBadgeCategories(['replies'])).toBe(true)
  await flushPromises()
  expect(status.notifications.value).toBe(false)
  expect(status.notificationCount.value).toBe(0)
  expect(status.notificationMessage.value).toBe((await import('../src/runtime/i18n')).i18n.global.t('notifications.noUnread'))
  finishOld(response(initial))
  await oldPoll
  expect(status.notifications.value).toBe(false)
  expect(status.notificationCount.value).toBe(0)
  expect(fetch).toHaveBeenCalledTimes(3)
})


test('legacy unread HTTP responses without IDs still trigger browser delivery', async () => {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ notifications: true, messages: false })))
  status.startPolling({ notifications: false, messages: false }, 1024)
  await status.refresh(true)
  const { maybeNotifyUnread } = await import('../src/runtime/browser-notification')
  expect(maybeNotifyUnread).toHaveBeenLastCalledWith(false, true, '', 0)
})
