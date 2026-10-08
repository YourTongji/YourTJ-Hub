// @vitest-environment happy-dom
import { afterEach, expect, test, vi } from 'vitest'
import { reconcileForYou, refreshForYou } from '../src/runtime/feed-session-api'
import { feedAccount, resetFeedAccount } from '../src/runtime/feed-telemetry'
import * as telemetry from '../src/runtime/feed-telemetry'

afterEach(() => { resetFeedAccount(0); vi.restoreAllMocks(); vi.unstubAllGlobals() })

test('account change during pending confirmation cannot refresh the new account', async () => {
  resetFeedAccount(12)
  vi.spyOn(telemetry, 'pendingSeenPatches').mockReturnValueOnce(Array.from({ length: 51 }, () => ({ proof: 'pending', seen: { 0: 1000 } })))
  vi.spyOn(telemetry, 'confirmPendingSeen').mockImplementation(async () => { resetFeedAccount(13) })
  const fetchMock = vi.fn(async () => ({ status: 200, ok: true,
    json: async () => ({ code: 0, result: { viewerId: 13, seenConfirmed: true } }) }))
  vi.stubGlobal('fetch', fetchMock)
  await expect(refreshForYou(undefined, 'account-12-session')).rejects.toThrow()
  expect(fetchMock).not.toHaveBeenCalled()
  expect(feedAccount()).toBe(13)
})

test('refresh identifies the current snapshot without replacing another history session', async () => {
  resetFeedAccount(12)
  const fetchMock = vi.fn(async (_url: string, _options: RequestInit) => ({ status: 200, ok: true,
    json: async () => ({ code: 0, result: { viewerId: 12, seenConfirmed: true } }) }))
  vi.stubGlobal('fetch', fetchMock)
  await refreshForYou(undefined, 'newer-session')
  expect(JSON.parse(String(fetchMock.mock.calls[0][1].body))).toMatchObject({ replaceSnapshotId: 'newer-session', seenPatches: [] })
})
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
