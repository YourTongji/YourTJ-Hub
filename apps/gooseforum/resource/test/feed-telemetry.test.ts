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
  pendingSeenPatches,
  seenConfirmationLost,
  invalidateSeen,
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
  document.body.innerHTML = ""
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
test('a card control does not select a detail attribution', () => {
  const root = document.createElement('section')
  root.innerHTML = '<div data-feed-id="31"><button>Like</button><a href="/p/post/31">Open</a></div>'
  document.body.append(root)
  const stop = observeFeedRows(root, () => [topic()])
  root.querySelector('button')!.click()
  expect(feedRequestHeaders('/p/post/31')).toEqual({})
  root.querySelector('a')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
  expect(feedRequestHeaders('/p/post/31')['X-Goose-Feed-Trace']).toBe('signed-trace')
  stop()
})
test('explicit request attribution is not overwritten by another selected detail', async () => {
  selectFeedTopic(topic())
  await feedFetch('/api/forum/topics/like', {
    headers: {
      'X-Goose-Feed-Trace': 'another-card',
      'X-Goose-Feed-Position': '2',
      'X-Goose-Feed-Topic': '32',
    },
  })
  const headers = (vi.mocked(fetch).mock.calls[0][1] as RequestInit).headers as Headers
  expect(headers.get('X-Goose-Feed-Trace')).toBe('another-card')
  expect(headers.get('X-Goose-Feed-Topic')).toBe('32')
})
test('requires 50% for one continuous foreground second and sends no opened claim', async () => {
  const root = document.createElement('section')
  const row = document.createElement('div')
  row.dataset.feedId = '31'
  root.append(row)
  document.body.append(root)
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
test('qualifying viewport exposure is confirmed without an analytics trace', async () => {
  const root = document.createElement('section')
  const row = document.createElement('div')
  row.dataset.feedId = '31'
  root.append(row)
  vi.mocked(fetch).mockResolvedValue({
    ok: true,
    json: async () => ({ code: 0, result: { seenConfirmed: true } }),
  } as Response)
  const proofs = [{ token: 'server-seen-proof', topicIds: [31], issuedAt: Date.now(), expiresAt: Date.now() + 1_800_000 }]
  document.body.append(root)
  const stop = observeFeedRows(root, () => [{ id: 31 } as TopicPayload], () => proofs)
  intersection([{ target: row, isIntersecting: true, intersectionRatio: 0.5 }])
  await vi.advanceTimersByTimeAsync(1000)
  await flushFeedEvents()
  expect(fetch).toHaveBeenCalledTimes(1)
  const body = JSON.parse((vi.mocked(fetch).mock.calls[0][1] as RequestInit).body as string)
  expect(body.seenPatches[0].proof).toBe('server-seen-proof')
  expect(body.seenPatches[0].seen['0']).toBeGreaterThanOrEqual(1000)
  expect(body.patches).toEqual([])
  stop()
})
test('background cancels the continuous visibility interval', async () => {
  const root = document.createElement('section')
  const row = document.createElement('div')
  row.dataset.feedId = '31'
  root.append(row)
  document.body.append(root)
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
  document.body.append(root)
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

test('functional visibility rejects partial, interrupted and masked cards', async () => {
  const root = document.createElement('section')
  root.innerHTML = '<div data-feed-id="31"></div>'
  document.body.append(root)
  const row = root.firstElementChild!
  const proofs = [{ token: 'functional-threshold', topicIds: [31], issuedAt: Date.now(), expiresAt: Date.now() + 1_800_000 }]
  const stop = observeFeedRows(root, () => [{ id: 31 } as TopicPayload], () => proofs)
  intersection([{ target: row, isIntersecting: true, intersectionRatio: 0.49 }])
  await vi.advanceTimersByTimeAsync(1100)
  expect(pendingSeenPatches()).toEqual([])
  intersection([{ target: row, isIntersecting: true, intersectionRatio: 0.5 }])
  await vi.advanceTimersByTimeAsync(999)
  expect(pendingSeenPatches()).toEqual([])
  intersection([{ target: row, isIntersecting: false, intersectionRatio: 0 }])
  await vi.advanceTimersByTimeAsync(1)
  root.setAttribute('inert', '')
  intersection([{ target: row, isIntersecting: true, intersectionRatio: 0.5 }])
  await vi.advanceTimersByTimeAsync(1100)
  expect(pendingSeenPatches()).toEqual([])
  root.removeAttribute('inert')
  const dialog = document.createElement('dialog'); dialog.setAttribute('open', ''); document.body.append(dialog)
  intersection([{ target: row, isIntersecting: true, intersectionRatio: 0.6 }])
  await vi.advanceTimersByTimeAsync(1100)
  expect(pendingSeenPatches()).toEqual([])
  stop()
})

test('failed functional confirmation retains the frozen claim for a later ACK', async () => {
  const root = document.createElement('section')
  root.innerHTML = '<div data-feed-id="31"></div>'; document.body.append(root)
  const stop = observeFeedRows(root, () => [{ id: 31 } as TopicPayload], () => [{
    token: 'retry-proof', topicIds: [31], issuedAt: Date.now(), expiresAt: Date.now() + 1_800_000,
  }])
  intersection([{ target: root.firstElementChild!, isIntersecting: true, intersectionRatio: 0.5 }])
  await vi.advanceTimersByTimeAsync(1000)
  const claim = pendingSeenPatches()
  vi.mocked(fetch).mockResolvedValueOnce({ ok: false } as Response)
  await expect(flushFeedEvents(false, true)).rejects.toThrow()
  expect(pendingSeenPatches()).toEqual(claim)
  vi.mocked(fetch).mockResolvedValueOnce({ ok: true, json: async () => ({ code: 0, result: { seenConfirmed: true } }) } as Response)
  await flushFeedEvents(false, true)
  expect(pendingSeenPatches()).toEqual([])
  stop()
})

test('expired unconfirmed claims cannot permanently block a fresh discovery batch', async () => {
  const root = document.createElement('section'); root.innerHTML = '<div data-feed-id="31"></div>'; document.body.append(root)
  const proof = { token: 'expired-pending', topicIds: [31], issuedAt: Date.now(), expiresAt: Date.now() + 1_800_000 }
  const stop = observeFeedRows(root, () => [{ id: 31 } as TopicPayload], () => [proof])
  intersection([{ target: root.firstElementChild!, isIntersecting: true, intersectionRatio: 0.5 }])
  await vi.advanceTimersByTimeAsync(1000)
  expect(pendingSeenPatches()).toHaveLength(1)
  vi.setSystemTime(proof.expiresAt)
  expect(pendingSeenPatches()).toEqual([])
  expect(seenConfirmationLost.value).toBe(true)
  stop()
})

test('a rejected process proof is discarded with feedback and cannot requalify', async () => {
  const root = document.createElement('section'); root.innerHTML = '<div data-feed-id="31"></div>'; document.body.append(root)
  const proof = { token: 'previous-process', topicIds: [31], issuedAt: Date.now(), expiresAt: Date.now() + 1_800_000 }
  const stop = observeFeedRows(root, () => [{ id: 31 } as TopicPayload], () => [proof])
  intersection([{ target: root.firstElementChild!, isIntersecting: true, intersectionRatio: 0.5 }])
  await vi.advanceTimersByTimeAsync(1000)
  invalidateSeen(pendingSeenPatches())
  expect(pendingSeenPatches()).toEqual([])
  expect(seenConfirmationLost.value).toBe(true)
  stop()
  const again = observeFeedRows(root, () => [{ id: 31 } as TopicPayload], () => [proof])
  intersection([{ target: root.firstElementChild!, isIntersecting: true, intersectionRatio: 0.5 }])
  await vi.advanceTimersByTimeAsync(1100)
  expect(pendingSeenPatches()).toEqual([])
  again()
})

test('a full 120-card buffer batches one request and requalifies the overflow card after ACK', async () => {
  const root = document.createElement('section')
  root.innerHTML = Array.from({ length: 121 }, (_, i) => `<div data-feed-id="${i+1}"></div>`).join('')
  document.body.append(root)
  const cards = Array.from({ length: 121 }, (_, i) => ({ id: i + 1 } as TopicPayload))
  const proofs = Array.from({ length: 7 }, (_, i) => ({ token: `bounded-${i}`, topicIds: cards.slice(i*20, i*20+20).map((c) => c.id), issuedAt: Date.now(), expiresAt: Date.now() + 1_800_000 }))
  const stop = observeFeedRows(root, () => cards, () => proofs)
  intersection([...root.children].map((target) => ({ target, isIntersecting: true, intersectionRatio: 0.6 })))
  await vi.advanceTimersByTimeAsync(1000)
  expect(pendingSeenPatches().reduce((sum, p) => sum + Object.keys(p.seen).length, 0)).toBe(120)
  vi.mocked(fetch).mockResolvedValue({ ok: true, json: async () => ({ code: 0, result: { seenConfirmed: true } }) } as Response)
  await flushFeedEvents(false, true)
  expect(fetch).toHaveBeenCalledTimes(1)
  expect(new TextEncoder().encode((vi.mocked(fetch).mock.calls[0][1] as RequestInit).body as string).length).toBeLessThanOrEqual(32768)
  await vi.advanceTimersByTimeAsync(2000)
  const overflow = pendingSeenPatches()
  expect(overflow).toHaveLength(1)
  expect(overflow[0]!.proof).toBe('bounded-6')
  stop()
})
