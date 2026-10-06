// @vitest-environment happy-dom
import { beforeEach, afterEach, expect, test, vi } from 'vitest'
import type { TopicPayload } from '@gooseforum/client'
import {
  beginFeedDetail,
  endFeedDetail,
  resetFeedAccount,
  selectFeedTopic,
  feedRequestHeaders,
  feedFetch,
  observeFeedRows,
  flushFeedEvents,
} from '../src/runtime/feed-telemetry'
let intersection: (
  entries: Array<{ target: Element; isIntersecting: boolean; intersectionRatio: number }>,
) => void
beforeEach(() => {
  vi.useFakeTimers()
  resetFeedAccount(0)
  resetFeedAccount(12)
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true }))
  Object.defineProperty(document, 'hidden', { configurable: true, value: false })
  vi.stubGlobal(
    'IntersectionObserver',
    class {
      constructor(cb: typeof intersection) {
        intersection = cb
      }
      observe() {}
      disconnect() {}
    },
  )
})
afterEach(() => {
  resetFeedAccount(0)
  vi.useRealTimers()
  vi.unstubAllGlobals()
})
const topic = () => ({ id: 31, feedTrace: 'signed-trace', feedPosition: 0 }) as TopicPayload
test('context remains same-origin and account isolated', () => {
  selectFeedTopic(topic())
  expect(feedRequestHeaders('/p/post/31')['X-Goose-Feed-Trace']).toBe('signed-trace')
  expect(feedRequestHeaders('/p/post/310')).toEqual({})
  expect(feedRequestHeaders('https://other.test/api/forum/topic-like')).toEqual({})
  resetFeedAccount(13)
  expect(feedRequestHeaders('/api/forum/topic-like')).toEqual({})
})
test('requires 50% for one continuous foreground second and sends no opened claim', async () => {
  const root = document.createElement('section')
  const row = document.createElement('div')
  row.dataset.feedId = '31'
  root.append(row)
  const stop = observeFeedRows(root, () => [topic()])
  intersection([{ target: row, isIntersecting: true, intersectionRatio: 0.5 }])
  vi.advanceTimersByTime(999)
  await flushFeedEvents()
  expect(fetch).not.toHaveBeenCalled()
  vi.advanceTimersByTime(1)
  await flushFeedEvents()
  const body = JSON.parse((vi.mocked(fetch).mock.calls[0][1] as RequestInit).body as string)
  expect(body.patches[0].visibleMask).toBe(1)
  expect(body.patches[0].openedMask).toBeUndefined()
  stop()
})
test('background cancels the continuous visibility interval', async () => {
  const root = document.createElement('section')
  const row = document.createElement('div')
  row.dataset.feedId = '31'
  root.append(row)
  const stop = observeFeedRows(root, () => [topic()])
  intersection([{ target: row, isIntersecting: true, intersectionRatio: 0.6 }])
  vi.advanceTimersByTime(500)
  Object.defineProperty(document, 'hidden', { configurable: true, value: true })
  document.dispatchEvent(new Event('visibilitychange'))
  vi.advanceTimersByTime(2000)
  await flushFeedEvents()
  expect(fetch).not.toHaveBeenCalled()
  stop()
})

test('leaving a traced detail clears attribution for a later search or notification visit', () => {
  selectFeedTopic(topic())
  beginFeedDetail(31)
  expect(feedRequestHeaders('/api/forum/topics/like')['X-Goose-Feed-Trace']).toBe('signed-trace')
  endFeedDetail()
  expect(feedRequestHeaders('/p/post/31')).toEqual({})
  expect(feedRequestHeaders('/api/forum/topics/like')).toEqual({})
})

test('the real account-close endpoint clears queued exposure and attribution', async () => {
  const root = document.createElement('section')
  const row = document.createElement('div')
  row.dataset.feedId = '31'
  root.append(row)
  const stop = observeFeedRows(root, () => [topic()])
  intersection([{ target: row, isIntersecting: true, intersectionRatio: 0.6 }])
  vi.advanceTimersByTime(1000)
  selectFeedTopic(topic())
  await feedFetch('/api/forum/user/account-close', { method: 'POST' })
  expect(feedRequestHeaders('/p/post/31')).toEqual({})
  await flushFeedEvents()
  expect(fetch).toHaveBeenCalledTimes(1)
  stop()
})
