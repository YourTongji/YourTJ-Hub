// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { reconcileForYou } from '../src/runtime/feed-session-api'
import { feedAccount, resetFeedAccount } from '../src/runtime/feed-telemetry'

afterEach(() => { resetFeedAccount(0); vi.unstubAllGlobals() })
for (const status of [200, 401]) {
  test(`a late ${status} response cannot restore or log out a changed account`, async () => {
    resetFeedAccount(12)
    let resolve!: (value: unknown) => void
    vi.stubGlobal('fetch', vi.fn(() => new Promise((r) => { resolve = r })))
    const result = reconcileForYou([31])
    resetFeedAccount(13)
    resolve({ status, ok: status === 200, json: async () => ({ code: 0, result: { viewerId: 12 } }) })
    await expect(result).rejects.toThrow()
    expect(feedAccount()).toBe(13)
  })
}
