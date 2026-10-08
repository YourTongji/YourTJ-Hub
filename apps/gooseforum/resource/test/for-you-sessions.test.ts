// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import type { HomeProps, PagePayload } from '@gooseforum/client'
import { activeForYouSession, historySessionKey, restoreForYouSession, saveForYouSession, setSessionOwner, updateForYouSession, lostForYouPage } from '../src/runtime/for-you-sessions'

function page(ids = [1, 2]) {
  return { component: {}, payload: { component: 'home.index', url: '/?sort=for_you',
    props: { sort: 'for_you', snapshotId: 'snapshot', topics: ids.map((id) => ({ id })),
      pagination: { page: 2, nextPage: 3, hasNext: true, nextUrl: '/?sort=for_you&cursor=old' } } } as unknown as PagePayload }
}
afterEach(() => { setSessionOwner(0); vi.useRealTimers() })
test('history entries with identical URLs retain separate loaded batches and cursors', () => {
  setSessionOwner(12)
  const first = historySessionKey(1), second = historySessionKey(3)
  saveForYouSession(first, page())
  activeForYouSession.value = first
  updateForYouSession({ ...page().payload.props, topics: [{ id: 2 }, { id: 1 }, { id: 3 }] } as HomeProps)
  saveForYouSession(second, page([4, 5]))
  expect((restoreForYouSession(first)!.payload.props as HomeProps).topics.map((t) => t.id)).toEqual([2, 1, 3])
  expect((restoreForYouSession(second)!.payload.props as HomeProps).topics.map((t) => t.id)).toEqual([4, 5])
  expect((restoreForYouSession(first)!.payload.props as HomeProps).pagination.nextUrl).toContain('cursor=old')
})
test('cache is bounded to two sessions, expires when inactive and clears on account change', () => {
  vi.useFakeTimers(); setSessionOwner(12)
  for (let i = 1; i <= 3; i++) { vi.advanceTimersByTime(1); saveForYouSession(historySessionKey(i), page([i])) }
  expect(restoreForYouSession(historySessionKey(1))).toBeUndefined()
  expect((lostForYouPage('/?sort=for_you')!.payload.props as HomeProps).sessionLost).toBe(true)
  vi.advanceTimersByTime(30 * 60 * 1000)
  expect(restoreForYouSession(historySessionKey(3))).toBeUndefined()
  setSessionOwner(13)
  expect(lostForYouPage('/?sort=for_you')).toBeUndefined()
  expect(activeForYouSession.value).toBe('')
})
test('oversized payload is discarded rather than stored without a budget', () => {
  setSessionOwner(12)
  const large = page(); (large.payload.props as HomeProps).topics[0]!.description = 'x'.repeat(513 * 1024)
  saveForYouSession(historySessionKey(1), large)
  expect(restoreForYouSession(historySessionKey(1))).toBeUndefined()
})

test('the combined byte budget evicts an older session when two individual pages fit', () => {
  setSessionOwner(12)
  const large = page(); (large.payload.props as HomeProps).topics[0]!.description = 'x'.repeat(300 * 1024)
  saveForYouSession(historySessionKey(1), large)
  saveForYouSession(historySessionKey(2), large)
  expect(restoreForYouSession(historySessionKey(1))).toBeUndefined()
  expect(restoreForYouSession(historySessionKey(2))).toBeDefined()
})
